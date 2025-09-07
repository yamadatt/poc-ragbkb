package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDocumentsGetEndpointContract(t *testing.T) {
	// テストルーターをセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 文書取得エンドポイントは実装前なので、テンプレートハンドラーを設定
	router.GET("/documents/:id", func(c *gin.Context) {
		// このハンドラーは実装されるまで失敗する
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "文書取得エンドポイントはまだ実装されていません",
		})
	})

	tests := []struct {
		name           string
		documentId     string
		expectedStatus int
		description    string
	}{
		{
			name:           "文書取得 - 正常応答",
			documentId:     "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusOK,
			description:    "存在する文書の詳細を返すべき",
		},
		{
			name:           "文書取得 - 存在しない文書",
			documentId:     "550e8400-e29b-41d4-a716-446655440404",
			expectedStatus: http.StatusNotFound,
			description:    "存在しない文書IDの場合は404を返すべき",
		},
		{
			name:           "文書取得 - 無効なUUID",
			documentId:     "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			description:    "無効なUUID形式の場合は400を返すべき",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// リクエストを作成
			req, _ := http.NewRequest("GET", "/documents/"+tt.documentId, nil)
			w := httptest.NewRecorder()

			// リクエストを実行
			router.ServeHTTP(w, req)

			// ステータスコードの検証 - 実装前は失敗する
			assert.Equal(t, tt.expectedStatus, w.Code,
				"GET /documents/%s should return %d but got %d - implementation required",
				tt.documentId, tt.expectedStatus, w.Code)

			// レスポンスボディの検証（実装後）
			if w.Code == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				// 必須フィールドの存在確認
				expectedFields := []string{"id", "fileName", "fileSize", "fileType", "uploadedAt", "status"}
				for _, field := range expectedFields {
					assert.Contains(t, response, field, "Response should contain field: %s", field)
				}

				// フィールドの値検証
				if status, exists := response["status"]; exists {
					validStatuses := []string{"uploading", "processing", "ready", "error"}
					assert.Contains(t, validStatuses, status, "status should be one of: %v", validStatuses)
				}

				if fileType, exists := response["fileType"]; exists {
					validTypes := []string{"txt", "md"}
					assert.Contains(t, validTypes, fileType, "fileType should be one of: %v", validTypes)
				}

				// IDが要求したIDと一致することを確認
				if id, exists := response["id"]; exists {
					assert.Equal(t, tt.documentId, id, "Returned document ID should match requested ID")
				}

			} else if w.Code == http.StatusNotFound {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error", "Error response should contain error field")

			} else if w.Code == http.StatusBadRequest {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error", "Error response should contain error field")
			}
		})
	}
}
