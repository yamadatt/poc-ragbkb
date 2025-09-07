package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDocumentsDeleteEndpointContract(t *testing.T) {
	// テストルーターをセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 文書削除エンドポイントは実装前なので、テンプレートハンドラーを設定
	router.DELETE("/documents/:id", func(c *gin.Context) {
		// このハンドラーは実装されるまで失敗する
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "文書削除エンドポイントはまだ実装されていません",
		})
	})

	tests := []struct {
		name           string
		documentId     string
		expectedStatus int
		description    string
	}{
		{
			name:           "文書削除 - 正常削除",
			documentId:     "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNoContent,
			description:    "存在する文書の削除で204 No Contentを返すべき",
		},
		{
			name:           "文書削除 - 存在しない文書",
			documentId:     "550e8400-e29b-41d4-a716-446655440404",
			expectedStatus: http.StatusNotFound,
			description:    "存在しない文書IDの場合は404を返すべき",
		},
		{
			name:           "文書削除 - 無効なUUID",
			documentId:     "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			description:    "無効なUUID形式の場合は400を返すべき",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// リクエストを作成
			req, _ := http.NewRequest("DELETE", "/documents/"+tt.documentId, nil)
			w := httptest.NewRecorder()

			// リクエストを実行
			router.ServeHTTP(w, req)

			// ステータスコードの検証 - 実装前は失敗する
			assert.Equal(t, tt.expectedStatus, w.Code,
				"DELETE /documents/%s should return %d but got %d - implementation required",
				tt.documentId, tt.expectedStatus, w.Code)

			// レスポンスボディの検証
			if w.Code == http.StatusNoContent {
				// 204 No Contentの場合、レスポンスボディは空であるべき
				assert.Empty(t, w.Body.String(), "No Content response should have empty body")

			} else if w.Code == http.StatusNotFound || w.Code == http.StatusBadRequest {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error", "Error response should contain error field")
			}
		})
	}
}
