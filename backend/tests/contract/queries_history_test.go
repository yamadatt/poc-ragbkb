package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestQueriesHistoryEndpointContract(t *testing.T) {
	// テストルーターをセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// クエリ履歴エンドポイントは実装前なので、テンプレートハンドラーを設定
	router.GET("/queries/:sessionId/history", func(c *gin.Context) {
		// このハンドラーは実装されるまで失敗する
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "クエリ履歴エンドポイントはまだ実装されていません",
		})
	})

	tests := []struct {
		name           string
		sessionId      string
		queryParams    string
		expectedStatus int
		description    string
	}{
		{
			name:           "クエリ履歴取得 - 正常応答",
			sessionId:      "550e8400-e29b-41d4-a716-446655440000",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			description:    "セッションの質問応答履歴を取得",
		},
		{
			name:           "クエリ履歴取得 - limit指定",
			sessionId:      "550e8400-e29b-41d4-a716-446655440000",
			queryParams:    "?limit=10",
			expectedStatus: http.StatusOK,
			description:    "limit指定で履歴件数を制限",
		},
		{
			name:           "クエリ履歴取得 - limit上限超過",
			sessionId:      "550e8400-e29b-41d4-a716-446655440000",
			queryParams:    "?limit=100",
			expectedStatus: http.StatusBadRequest,
			description:    "limit上限(50)を超えた場合はエラー",
		},
		{
			name:           "クエリ履歴取得 - 無効なセッションID",
			sessionId:      "invalid-uuid",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			description:    "無効なUUID形式のセッションIDはエラー",
		},
		{
			name:           "クエリ履歴取得 - 存在しないセッション",
			sessionId:      "550e8400-e29b-41d4-a716-446655440404",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			description:    "存在しないセッションでも空配列を返すべき",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// リクエストを作成
			path := "/queries/" + tt.sessionId + "/history" + tt.queryParams
			req, _ := http.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			// リクエストを実行
			router.ServeHTTP(w, req)

			// ステータスコードの検証 - 実装前は失敗する
			assert.Equal(t, tt.expectedStatus, w.Code,
				"GET /queries/%s/history should return %d but got %d - implementation required",
				tt.sessionId, tt.expectedStatus, w.Code)

			// レスポンスボディの検証（実装後）
			if w.Code == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				// 必須フィールドの存在確認
				assert.Contains(t, response, "queries", "Response should contain queries array")
				assert.Contains(t, response, "total", "Response should contain total count")

				// queriesフィールドの型確認
				queries, ok := response["queries"].([]interface{})
				assert.True(t, ok, "queries should be an array")

				// totalフィールドの型確認
				total, ok := response["total"].(float64)
				assert.True(t, ok, "total should be a number")
				assert.Equal(t, float64(len(queries)), total, "total should match queries count")

				// 各クエリオブジェクトの構造確認
				for i, queryItem := range queries {
					queryObj, ok := queryItem.(map[string]interface{})
					assert.True(t, ok, "query at index %d should be an object", i)

					// query-response ペアの構造確認
					assert.Contains(t, queryObj, "query", "Query item should contain query object")
					assert.Contains(t, queryObj, "response", "Query item should contain response object")

					// queryオブジェクトの確認
					if query, ok := queryObj["query"].(map[string]interface{}); ok {
						queryFields := []string{"id", "question", "timestamp", "sessionId"}
						for _, field := range queryFields {
							assert.Contains(t, query, field, "Query should contain field: %s", field)
						}

						// セッションIDが要求したIDと一致することを確認
						if sessionId, exists := query["sessionId"]; exists {
							assert.Equal(t, tt.sessionId, sessionId, "Query sessionId should match requested sessionId")
						}
					}

					// responseオブジェクトの確認
					if response, ok := queryObj["response"].(map[string]interface{}); ok {
						responseFields := []string{"id", "answer", "sources", "timestamp", "processingTimeMs"}
						for _, field := range responseFields {
							assert.Contains(t, response, field, "Response should contain field: %s", field)
						}

						// sourcesの構造確認
						if sources, ok := response["sources"].([]interface{}); ok {
							for j, source := range sources {
								if sourceObj, ok := source.(map[string]interface{}); ok {
									sourceFields := []string{"documentId", "fileName", "excerpt", "confidence"}
									for _, field := range sourceFields {
										assert.Contains(t, sourceObj, field, "Source %d should contain field: %s", j, field)
									}
								}
							}
						}
					}
				}

				// limit指定がある場合の件数確認
				if tt.queryParams == "?limit=10" {
					assert.LessOrEqual(t, len(queries), 10, "Queries count should not exceed limit")
				}

			} else if w.Code == http.StatusBadRequest {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error", "Error response should contain error field")
			}
		})
	}
}

func TestQueriesHistoryEndpointServerError(t *testing.T) {
	// サーバーエラーケースのテスト
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// エラーをシミュレートするハンドラー（実装前）
	router.GET("/queries/:sessionId/history", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "クエリ履歴エンドポイントはまだ実装されていません",
		})
	})

	req, _ := http.NewRequest("GET", "/queries/550e8400-e29b-41d4-a716-446655440000/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 実装前は501の代わりに500が返される可能性をテスト
	if w.Code == http.StatusInternalServerError {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Contains(t, response, "error", "Error response should contain error field")
		assert.Contains(t, response, "requestId", "Error response should contain requestId field")
	}
}
