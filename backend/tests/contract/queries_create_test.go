package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestQueriesCreateEndpointContract(t *testing.T) {
	// テストルーターをセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// クエリ作成エンドポイントは実装前なので、テンプレートハンドラーを設定
	router.POST("/queries", func(c *gin.Context) {
		// このハンドラーは実装されるまで失敗する
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "クエリ作成エンドポイントはまだ実装されていません",
		})
	})

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		description    string
	}{
		{
			name: "RAGクエリ - 正常リクエスト",
			requestBody: map[string]interface{}{
				"question":  "AWS Bedrock Knowledge Baseの使い方を教えてください",
				"sessionId": "550e8400-e29b-41d4-a716-446655440000",
			},
			expectedStatus: http.StatusCreated,
			description:    "有効な質問でRAG処理を実行",
		},
		{
			name: "RAGクエリ - 空の質問",
			requestBody: map[string]interface{}{
				"question":  "",
				"sessionId": "550e8400-e29b-41d4-a716-446655440000",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "空の質問はバリデーションエラー",
		},
		{
			name: "RAGクエリ - 長すぎる質問",
			requestBody: map[string]interface{}{
				"question":  repeatString("A", 1001), // 1001文字
				"sessionId": "550e8400-e29b-41d4-a716-446655440000",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "1000文字を超える質問はエラー",
		},
		{
			name: "RAGクエリ - 無効なセッションID",
			requestBody: map[string]interface{}{
				"question":  "テスト質問です",
				"sessionId": "invalid-uuid",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "無効なUUID形式のセッションIDはエラー",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// リクエストボディをJSON化
			reqBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/queries", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// リクエストを実行
			router.ServeHTTP(w, req)

			// ステータスコードの検証 - 実装前は失敗する
			assert.Equal(t, tt.expectedStatus, w.Code,
				"POST /queries should return %d but got %d - implementation required",
				tt.expectedStatus, w.Code)

			// レスポンスボディの検証（実装後）
			if w.Code == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				// 必須フィールドの存在確認
				assert.Contains(t, response, "query", "Response should contain query object")
				assert.Contains(t, response, "response", "Response should contain response object")

				// queryオブジェクトの確認
				if query, ok := response["query"].(map[string]interface{}); ok {
					queryFields := []string{"id", "question", "timestamp", "sessionId", "status"}
					for _, field := range queryFields {
						assert.Contains(t, query, field, "Query should contain field: %s", field)
					}

					// ステータスの確認
					if status, exists := query["status"]; exists {
						validStatuses := []string{"processing", "completed"}
						assert.Contains(t, validStatuses, status, "query status should be valid")
					}
				}

				// responseオブジェクトの確認
				if resp, ok := response["response"].(map[string]interface{}); ok {
					respFields := []string{"id", "answer", "sources", "timestamp", "processingTimeMs"}
					for _, field := range respFields {
						assert.Contains(t, resp, field, "Response should contain field: %s", field)
					}

					// sourcesの構造確認
					if sources, ok := resp["sources"].([]interface{}); ok {
						for i, source := range sources {
							if sourceObj, ok := source.(map[string]interface{}); ok {
								sourceFields := []string{"documentId", "fileName", "excerpt", "confidence"}
								for _, field := range sourceFields {
									assert.Contains(t, sourceObj, field, "Source %d should contain field: %s", i, field)
								}
							}
						}
					}
				}

			} else if w.Code == http.StatusBadRequest {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error", "Error response should contain error field")

			} else if w.Code == http.StatusNotFound {
				// 関連情報が見つからない場合
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error", "404 response should contain error field")
				assert.Contains(t, response, "query", "404 response should still contain query object")
			}
		})
	}
}

// 文字列を指定した回数繰り返すヘルパー関数
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
