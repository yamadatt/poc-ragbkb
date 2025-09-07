package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

// RAGQueryIntegrationTestSuite はRAGクエリ統合テストスイート
type RAGQueryIntegrationTestSuite struct {
	suite.Suite
	router    *gin.Engine
	sessionId string
}

// SetupSuite はテストスイートの初期化
func (suite *RAGQueryIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.sessionId = "550e8400-e29b-41d4-a716-446655440000" // テスト用セッションID

	// 統合テスト用のテンプレートハンドラー（実装前）
	suite.setupRoutes()
}

// setupRoutes はテスト用ルートの設定
func (suite *RAGQueryIntegrationTestSuite) setupRoutes() {
	// 実装前はすべて501 Not Implementedを返すハンドラー
	suite.router.POST("/queries", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "RAGクエリエンドポイントは未実装",
		})
	})

	suite.router.GET("/queries/:sessionId/history", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "クエリ履歴エンドポイントは未実装",
		})
	})

	suite.router.POST("/documents", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "文書作成エンドポイントは未実装",
		})
	})

	suite.router.POST("/documents/:id/complete-upload", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "アップロード完了エンドポイントは未実装",
		})
	})
}

// TestRAGQueryFullFlow はRAGクエリの全フローをテスト
func (suite *RAGQueryIntegrationTestSuite) TestRAGQueryFullFlow() {
	// 前提条件: Knowledge Baseに文書が登録済みであることを確認
	// 実際のテストでは事前にテストデータを準備
	suite.prepareTestDocuments()

	// Step 1: 質問を送信
	queryRequest := map[string]interface{}{
		"question":  "AWS Bedrock Knowledge Baseの使い方を教えてください",
		"sessionId": suite.sessionId,
	}

	reqBody, _ := json.Marshal(queryRequest)
	req, _ := http.NewRequest("POST", "/queries", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// 実装前は501が返されることを確認
	suite.Equal(http.StatusNotImplemented, w.Code, "実装前は501 Not Implementedが返されるべき")

	if w.Code == http.StatusCreated {
		// 実装後のテストロジック
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		suite.NoError(err)

		// レスポンス構造の確認
		suite.Contains(response, "query", "クエリオブジェクトが必要")
		suite.Contains(response, "response", "レスポンスオブジェクトが必要")

		query := response["query"].(map[string]interface{})
		ragResponse := response["response"].(map[string]interface{})

		// クエリオブジェクトの検証
		suite.Equal(suite.sessionId, query["sessionId"], "セッションIDが正しく設定されている")
		suite.Equal(queryRequest["question"], query["question"], "質問が正しく記録されている")
		suite.Contains(query, "id", "クエリIDが生成されている")
		suite.Contains(query, "timestamp", "タイムスタンプが記録されている")
		suite.Contains([]string{"processing", "completed"}, query["status"],
			"クエリステータスが適切に設定されている")

		queryId := query["id"].(string)

		// レスポンスオブジェクトの検証
		suite.Contains(ragResponse, "id", "レスポンスIDが必要")
		suite.Contains(ragResponse, "answer", "回答が必要")
		suite.Contains(ragResponse, "sources", "情報源が必要")
		suite.Contains(ragResponse, "timestamp", "タイムスタンプが必要")
		suite.Contains(ragResponse, "processingTimeMs", "処理時間が必要")

		// 回答内容の検証
		answer := ragResponse["answer"].(string)
		suite.NotEmpty(answer, "回答が生成されている")
		suite.Greater(len(answer), 10, "十分な長さの回答が生成されている")

		// 情報源の検証
		sources := ragResponse["sources"].([]interface{})
		suite.Greater(len(sources), 0, "情報源が提供されている")
		suite.LessOrEqual(len(sources), 5, "情報源は最大5個まで")

		for i, source := range sources {
			sourceObj := source.(map[string]interface{})
			suite.Contains(sourceObj, "documentId", fmt.Sprintf("情報源%dにdocumentIdが必要", i))
			suite.Contains(sourceObj, "fileName", fmt.Sprintf("情報源%dにfileNameが必要", i))
			suite.Contains(sourceObj, "excerpt", fmt.Sprintf("情報源%dに抜粋が必要", i))
			suite.Contains(sourceObj, "confidence", fmt.Sprintf("情報源%dに信頼度が必要", i))

			// 信頼度の範囲確認
			confidence := sourceObj["confidence"].(float64)
			suite.GreaterOrEqual(confidence, 0.0, "信頼度は0.0以上")
			suite.LessOrEqual(confidence, 1.0, "信頼度は1.0以下")
		}

		// 処理時間の検証
		processingTime := ragResponse["processingTimeMs"].(float64)
		suite.Greater(processingTime, 0.0, "処理時間が記録されている")
		suite.Less(processingTime, 30000.0, "処理時間が30秒以内")

		// Step 2: クエリ履歴の確認
		historyReq, _ := http.NewRequest("GET",
			fmt.Sprintf("/queries/%s/history", suite.sessionId), nil)
		historyW := httptest.NewRecorder()
		suite.router.ServeHTTP(historyW, historyReq)

		if historyW.Code == http.StatusOK {
			var historyResponse map[string]interface{}
			err := json.Unmarshal(historyW.Body.Bytes(), &historyResponse)
			suite.NoError(err)

			queries := historyResponse["queries"].([]interface{})
			suite.Greater(len(queries), 0, "履歴にクエリが記録されている")

			// 最新のクエリが今回の質問であることを確認
			latestQuery := queries[0].(map[string]interface{})
			latestQueryObj := latestQuery["query"].(map[string]interface{})
			suite.Equal(queryId, latestQueryObj["id"], "履歴に最新のクエリが記録されている")
		}

		// Step 3: 継続的な会話のテスト
		suite.testContinuousConversation(queryId, ragResponse)
	}
}

// TestRAGQueryErrorCases はRAGクエリのエラーケースをテスト
func (suite *RAGQueryIntegrationTestSuite) TestRAGQueryErrorCases() {
	testCases := []struct {
		name        string
		request     map[string]interface{}
		expectedErr string
	}{
		{
			name: "空の質問",
			request: map[string]interface{}{
				"question":  "",
				"sessionId": suite.sessionId,
			},
			expectedErr: "質問は必須です",
		},
		{
			name: "長すぎる質問",
			request: map[string]interface{}{
				"question":  suite.generateLongString(1001),
				"sessionId": suite.sessionId,
			},
			expectedErr: "質問は1000文字以内で入力してください",
		},
		{
			name: "無効なセッションID",
			request: map[string]interface{}{
				"question":  "テスト質問です",
				"sessionId": "invalid-uuid",
			},
			expectedErr: "無効なセッションIDです",
		},
		{
			name: "セッションID未指定",
			request: map[string]interface{}{
				"question": "テスト質問です",
			},
			expectedErr: "セッションIDは必須です",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			reqBody, _ := json.Marshal(tc.request)
			req, _ := http.NewRequest("POST", "/queries", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			// 実装前は501が返される
			suite.Equal(http.StatusNotImplemented, w.Code)

			// 実装後は400 Bad Requestが期待される
			if w.Code == http.StatusBadRequest {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				suite.NoError(err)
				suite.Contains(response, "error")
			}
		})
	}
}

// TestRAGQueryNoRelevantDocuments は関連文書が見つからない場合のテスト
func (suite *RAGQueryIntegrationTestSuite) TestRAGQueryNoRelevantDocuments() {
	// 関連性の低い質問を送信
	queryRequest := map[string]interface{}{
		"question":  "量子力学の基礎について教えてください", // Knowledge Baseにない内容
		"sessionId": suite.sessionId,
	}

	reqBody, _ := json.Marshal(queryRequest)
	req, _ := http.NewRequest("POST", "/queries", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// 実装前は501が返される
	suite.Equal(http.StatusNotImplemented, w.Code)

	// 実装後は404 Not Foundが期待される（関連情報が見つからない）
	if w.Code == http.StatusNotFound {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		suite.NoError(err)

		suite.Contains(response, "error")
		suite.Contains(response, "query", "クエリオブジェクトは記録される")

		// クエリは記録されているが、レスポンスは空
		query := response["query"].(map[string]interface{})
		suite.Equal(queryRequest["question"], query["question"])
		suite.Equal(suite.sessionId, query["sessionId"])
	}
}

// prepareTestDocuments はテスト用文書を準備
func (suite *RAGQueryIntegrationTestSuite) prepareTestDocuments() {
	// 実際のテストではKnowledge Baseにテスト文書を事前登録
	// または専用のテスト環境を使用

	testDocuments := []string{
		"AWS Bedrock Knowledge Baseは、RAGシステムを構築するためのマネージドサービスです。",
		"ドキュメントの自動ベクトル化とOpenSearch Serverlessとの統合により、効率的な検索が可能です。",
		"企業の文書を安全に管理し、生成AIと組み合わせて高度な質問応答システムを構築できます。",
	}

	suite.T().Logf("テスト文書を準備: %d件", len(testDocuments))
	// 実装時は実際にS3アップロードとKnowledge Base同期を実行
}

// testContinuousConversation は継続的な会話をテスト
func (suite *RAGQueryIntegrationTestSuite) testContinuousConversation(previousQueryId string, previousResponse map[string]interface{}) {
	// 前の回答を参照したフォローアップ質問
	followupRequest := map[string]interface{}{
		"question":  "それはどのような利点がありますか？",
		"sessionId": suite.sessionId,
	}

	reqBody, _ := json.Marshal(followupRequest)
	req, _ := http.NewRequest("POST", "/queries", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	if w.Code == http.StatusCreated {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		suite.NoError(err)

		// フォローアップ質問への回答が前の文脈を理解していることを確認
		ragResponse := response["response"].(map[string]interface{})
		answer := ragResponse["answer"].(string)

		suite.NotEmpty(answer, "フォローアップ質問にも回答が生成される")
		// 実装時は文脈の一貫性をより詳細にテスト
	}
}

// generateLongString は指定した長さの文字列を生成
func (suite *RAGQueryIntegrationTestSuite) generateLongString(length int) string {
	result := ""
	for i := 0; i < length; i++ {
		result += "A"
	}
	return result
}

// TestRAGQueryIntegrationTestSuite はテストスイートを実行
func TestRAGQueryIntegrationTestSuite(t *testing.T) {
	// 統合テスト用の環境変数チェック
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("統合テストをスキップ: INTEGRATION_TEST環境変数が設定されていません")
	}

	suite.Run(t, new(RAGQueryIntegrationTestSuite))
}
