package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

// KnowledgeBaseSyncIntegrationTestSuite はKnowledge Base同期統合テストスイート
type KnowledgeBaseSyncIntegrationTestSuite struct {
	suite.Suite
	router *gin.Engine
}

// SetupSuite はテストスイートの初期化
func (suite *KnowledgeBaseSyncIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// 統合テスト用のテンプレートハンドラー（実装前）
	suite.setupRoutes()
}

// setupRoutes はテスト用ルートの設定
func (suite *KnowledgeBaseSyncIntegrationTestSuite) setupRoutes() {
	// 実装前はすべて501 Not Implementedを返すハンドラー
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

	suite.router.GET("/documents/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "文書取得エンドポイントは未実装",
		})
	})

	suite.router.POST("/queries", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "RAGクエリエンドポイントは未実装",
		})
	})
}

// TestKnowledgeBaseSyncFlow はKnowledge Base同期フローをテスト
func (suite *KnowledgeBaseSyncIntegrationTestSuite) TestKnowledgeBaseSyncFlow() {
	// Step 1: 文書アップロードの開始
	uploadRequest := map[string]interface{}{
		"fileName": "knowledge-base-guide.md",
		"fileSize": 5120, // 5KB
		"fileType": "md",
	}

	reqBody, _ := json.Marshal(uploadRequest)
	req, _ := http.NewRequest("POST", "/documents", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// 実装前は501が返されることを確認
	suite.Equal(http.StatusNotImplemented, w.Code, "実装前は501 Not Implementedが返されるべき")

	if w.Code == http.StatusCreated {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		suite.NoError(err)

		documentId := response["id"].(string)
		uploadUrl := response["uploadUrl"].(string)

		// Step 2: S3への文書アップロード（モック）
		suite.mockS3UploadWithKnowledgeBaseContent(uploadUrl)

		// Step 3: アップロード完了通知
		completeReq, _ := http.NewRequest("POST", fmt.Sprintf("/documents/%s/complete-upload", documentId), nil)
		completeW := httptest.NewRecorder()
		suite.router.ServeHTTP(completeW, completeReq)

		if completeW.Code == http.StatusOK {
			// Step 4: Knowledge Base同期処理の開始を確認
			suite.verifyKnowledgeBaseSyncStarted(documentId)

			// Step 5: Knowledge Base同期完了まで待機
			suite.waitForKnowledgeBaseSync(documentId)

			// Step 6: 同期後の文書ステータス確認
			suite.verifyDocumentAfterSync(documentId)

			// Step 7: Knowledge Baseでの検索可能性を確認
			suite.testKnowledgeBaseSearchability(documentId)
		}
	}
}

// TestKnowledgeBaseSyncErrorHandling はKnowledge Base同期エラーをテスト
func (suite *KnowledgeBaseSyncIntegrationTestSuite) TestKnowledgeBaseSyncErrorHandling() {
	testCases := []struct {
		name        string
		fileName    string
		fileContent string
		expectError bool
		errorType   string
	}{
		{
			name:        "正常なマークダウン文書",
			fileName:    "valid-document.md",
			fileContent: "# AWS Bedrock Knowledge Base\n\nThis is a valid markdown document.",
			expectError: false,
		},
		{
			name:        "空の文書",
			fileName:    "empty-document.txt",
			fileContent: "",
			expectError: true,
			errorType:   "EMPTY_CONTENT",
		},
		{
			name:        "不正な文字エンコーディング",
			fileName:    "invalid-encoding.txt",
			fileContent: "\xff\xfe\x00Invalid UTF-8 content",
			expectError: true,
			errorType:   "ENCODING_ERROR",
		},
		{
			name:        "大容量文書（Knowledge Base制限テスト）",
			fileName:    "large-document.md",
			fileContent: suite.generateLargeContent(10000), // 10KB のコンテンツ
			expectError: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 文書アップロードの開始
			uploadRequest := map[string]interface{}{
				"fileName": tc.fileName,
				"fileSize": int64(len(tc.fileContent)),
				"fileType": "txt",
			}
			if tc.fileName[len(tc.fileName)-2:] == "md" {
				uploadRequest["fileType"] = "md"
			}

			reqBody, _ := json.Marshal(uploadRequest)
			req, _ := http.NewRequest("POST", "/documents", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			// 実装前のテスト
			suite.Equal(http.StatusNotImplemented, w.Code)

			// 実装後のテストロジック
			if w.Code == http.StatusCreated {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				documentId := response["id"].(string)

				// S3アップロードのモック
				suite.mockS3UploadWithContent(response["uploadUrl"].(string), tc.fileContent)

				// アップロード完了通知
				completeReq, _ := http.NewRequest("POST", fmt.Sprintf("/documents/%s/complete-upload", documentId), nil)
				completeW := httptest.NewRecorder()
				suite.router.ServeHTTP(completeW, completeReq)

				// Knowledge Base同期の結果を確認
				if tc.expectError {
					suite.waitForKnowledgeBaseSyncError(documentId, tc.errorType)
				} else {
					suite.waitForKnowledgeBaseSync(documentId)
				}
			}
		})
	}
}

// TestKnowledgeBaseMultiDocumentSync は複数文書の同時同期をテスト
func (suite *KnowledgeBaseSyncIntegrationTestSuite) TestKnowledgeBaseMultiDocumentSync() {
	documents := []struct {
		fileName string
		content  string
	}{
		{
			fileName: "bedrock-overview.md",
			content:  "# AWS Bedrock Overview\n\nBedrock is a fully managed service...",
		},
		{
			fileName: "knowledge-base-guide.md",
			content:  "# Knowledge Base User Guide\n\nKnowledge Base allows you to...",
		},
		{
			fileName: "rag-implementation.txt",
			content:  "RAG Implementation Guide\n\nRetrieval Augmented Generation...",
		},
	}

	documentIds := make([]string, 0, len(documents))

	// Step 1: 複数文書を並行してアップロード
	for _, doc := range documents {
		uploadRequest := map[string]interface{}{
			"fileName": doc.fileName,
			"fileSize": int64(len(doc.content)),
			"fileType": "md",
		}
		if doc.fileName[len(doc.fileName)-3:] == "txt" {
			uploadRequest["fileType"] = "txt"
		}

		reqBody, _ := json.Marshal(uploadRequest)
		req, _ := http.NewRequest("POST", "/documents", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		if w.Code == http.StatusCreated {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			documentIds = append(documentIds, response["id"].(string))

			// S3アップロードのモック
			suite.mockS3UploadWithContent(response["uploadUrl"].(string), doc.content)

			// アップロード完了通知
			completeReq, _ := http.NewRequest("POST",
				fmt.Sprintf("/documents/%s/complete-upload", response["id"].(string)), nil)
			completeW := httptest.NewRecorder()
			suite.router.ServeHTTP(completeW, completeReq)
		}
	}

	// Step 2: すべての文書の同期完了を待機
	for _, docId := range documentIds {
		suite.waitForKnowledgeBaseSync(docId)
	}

	// Step 3: 複数文書にまたがる検索のテスト
	suite.testCrossDocumentSearch(documentIds)
}

// verifyKnowledgeBaseSyncStarted はKnowledge Base同期開始を確認
func (suite *KnowledgeBaseSyncIntegrationTestSuite) verifyKnowledgeBaseSyncStarted(documentId string) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/documents/%s", documentId), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		status := response["status"].(string)
		suite.Contains([]string{"processing"}, status,
			"Knowledge Base同期開始後はprocessingステータスになる")

		// Knowledge Base同期に関する追加フィールドの確認
		suite.Contains(response, "kbStatus", "Knowledge Base同期ステータスフィールドが必要")
	}
}

// waitForKnowledgeBaseSync はKnowledge Base同期完了を待機
func (suite *KnowledgeBaseSyncIntegrationTestSuite) waitForKnowledgeBaseSync(documentId string) {
	timeout := time.After(60 * time.Second) // Knowledge Base同期は時間がかかる
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			suite.Fail("Knowledge Base同期がタイムアウトしました")
			return
		case <-ticker.C:
			req, _ := http.NewRequest("GET", fmt.Sprintf("/documents/%s", documentId), nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)

				status := response["status"].(string)
				kbStatus := response["kbStatus"].(string)

				if status == "ready" && kbStatus == "synced" {
					suite.T().Logf("Knowledge Base同期が完了しました: %s", documentId)
					return
				}

				if status == "error" || kbStatus == "error" {
					errorMsg := ""
					if msg, exists := response["errorMsg"]; exists {
						errorMsg = msg.(string)
					}
					suite.Fail(fmt.Sprintf("Knowledge Base同期でエラーが発生: %s", errorMsg))
					return
				}
			}
		}
	}
}

// waitForKnowledgeBaseSyncError はKnowledge Base同期エラーを待機
func (suite *KnowledgeBaseSyncIntegrationTestSuite) waitForKnowledgeBaseSyncError(documentId string, expectedErrorType string) {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			suite.Fail("Knowledge Base同期エラーのタイムアウト")
			return
		case <-ticker.C:
			req, _ := http.NewRequest("GET", fmt.Sprintf("/documents/%s", documentId), nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)

				if status := response["status"].(string); status == "error" {
					if errorMsg, exists := response["errorMsg"]; exists {
						suite.Contains(errorMsg.(string), expectedErrorType,
							"期待されるエラータイプが含まれている")
					}
					return
				}
			}
		}
	}
}

// verifyDocumentAfterSync は同期後の文書を確認
func (suite *KnowledgeBaseSyncIntegrationTestSuite) verifyDocumentAfterSync(documentId string) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/documents/%s", documentId), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		suite.Equal("ready", response["status"], "同期完了後はreadyステータス")
		suite.Equal("synced", response["kbStatus"], "Knowledge Baseステータスはsynced")
		suite.Contains(response, "uploadedAt", "アップロード日時が記録されている")
	}
}

// testKnowledgeBaseSearchability はKnowledge Baseでの検索可能性をテスト
func (suite *KnowledgeBaseSyncIntegrationTestSuite) testKnowledgeBaseSearchability(documentId string) {
	// 同期された文書に関連する質問を送信
	queryRequest := map[string]interface{}{
		"question":  "Knowledge Baseについて教えてください",
		"sessionId": "550e8400-e29b-41d4-a716-446655440000",
	}

	reqBody, _ := json.Marshal(queryRequest)
	req, _ := http.NewRequest("POST", "/queries", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	if w.Code == http.StatusCreated {
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		ragResponse := response["response"].(map[string]interface{})
		sources := ragResponse["sources"].([]interface{})

		// 同期された文書が情報源として使用されていることを確認
		found := false
		for _, source := range sources {
			sourceObj := source.(map[string]interface{})
			if sourceObj["documentId"].(string) == documentId {
				found = true
				suite.Greater(sourceObj["confidence"].(float64), 0.5,
					"同期された文書の信頼度が十分に高い")
				break
			}
		}

		suite.True(found, "同期された文書がRAGの情報源として使用されている")
	}
}

// testCrossDocumentSearch は複数文書にまたがる検索をテスト
func (suite *KnowledgeBaseSyncIntegrationTestSuite) testCrossDocumentSearch(documentIds []string) {
	queryRequest := map[string]interface{}{
		"question":  "Bedrockのオーバービューと実装ガイドを教えてください",
		"sessionId": "550e8400-e29b-41d4-a716-446655440000",
	}

	reqBody, _ := json.Marshal(queryRequest)
	req, _ := http.NewRequest("POST", "/queries", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	if w.Code == http.StatusCreated {
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		ragResponse := response["response"].(map[string]interface{})
		sources := ragResponse["sources"].([]interface{})

		// 複数の文書から情報が取得されていることを確認
		sourceDocIds := make(map[string]bool)
		for _, source := range sources {
			sourceObj := source.(map[string]interface{})
			docId := sourceObj["documentId"].(string)
			sourceDocIds[docId] = true
		}

		suite.Greater(len(sourceDocIds), 1, "複数の文書から情報が取得されている")

		// 同期したすべての文書が検索対象に含まれていることを確認
		for _, docId := range documentIds {
			if sourceDocIds[docId] {
				suite.T().Logf("文書 %s が検索結果に含まれています", docId)
			}
		}
	}
}

// mockS3UploadWithKnowledgeBaseContent はKnowledge Base用コンテンツでS3アップロードをモック
func (suite *KnowledgeBaseSyncIntegrationTestSuite) mockS3UploadWithKnowledgeBaseContent(uploadUrl string) {
	content := `# AWS Bedrock Knowledge Base ユーザーガイド

## 概要
AWS Bedrock Knowledge Baseは、企業の文書を自動的にベクトル化し、
RAG（Retrieval Augmented Generation）システムを構築するためのマネージドサービスです。

## 主な機能
- 自動的なドキュメント処理とベクトル化
- OpenSearch Serverlessとの統合
- リアルタイムな検索とレスポンス生成

## 利点
1. 運用負荷の軽減
2. スケーラブルな検索システム
3. セキュアな文書管理
`

	suite.T().Logf("S3アップロードをモック: URL=%s, Content length=%d",
		uploadUrl, len(content))
}

// mockS3UploadWithContent は指定されたコンテンツでS3アップロードをモック
func (suite *KnowledgeBaseSyncIntegrationTestSuite) mockS3UploadWithContent(uploadUrl, content string) {
	suite.T().Logf("S3アップロードをモック: URL=%s, Content length=%d",
		uploadUrl, len(content))
}

// generateLargeContent は大容量コンテンツを生成
func (suite *KnowledgeBaseSyncIntegrationTestSuite) generateLargeContent(sizeKB int) string {
	content := "# Large Document Test\n\n"

	paragraph := "This is a test paragraph for generating large content. " +
		"It contains information about AWS Bedrock Knowledge Base integration. " +
		"This paragraph will be repeated to create the desired file size. "

	// 指定されたサイズになるまで段落を追加
	for len(content) < sizeKB*1024 {
		content += paragraph + "\n\n"
	}

	return content[:sizeKB*1024] // 正確なサイズに切り詰め
}

// TestKnowledgeBaseSyncIntegrationTestSuite はテストスイートを実行
func TestKnowledgeBaseSyncIntegrationTestSuite(t *testing.T) {
	// 統合テスト用の環境変数チェック
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("統合テストをスキップ: INTEGRATION_TEST環境変数が設定されていません")
	}

	suite.Run(t, new(KnowledgeBaseSyncIntegrationTestSuite))
}
