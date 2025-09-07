package main

import (
	"context"
	"log"
	"os"
	"time"

	"poc-ragbkb-backend/src/handlers"
	"poc-ragbkb-backend/src/services"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	ginEngine "github.com/gin-gonic/gin"
)

const (
    // デフォルト値
    DefaultPort                = ":8080"
    DefaultVersion             = "1.0.0"
    DefaultS3Bucket            = "poc-ragbkb-documents"
    DefaultDocumentsTable      = "Documents"
    DefaultQueriesTable        = "Queries"
    DefaultResponsesTable      = "Responses"
    DefaultUploadSessionsTable = "UploadSessions"
    // KB/DS は未設定時は空にし、明示設定を必須にする
    DefaultKnowledgeBaseID     = ""
    DefaultDataSourceID        = ""
    DefaultModelID             = "amazon.titan-text-express-v1"
    DefaultPresignExpiration   = 15 * time.Minute
)

var ginLambda *ginadapter.GinLambda

func main() {
	// 環境変数から設定を読み込み
	version := getEnv("VERSION", DefaultVersion)
	s3Bucket := getEnv("S3_BUCKET_NAME", DefaultS3Bucket)
	documentsTable := getEnv("DOCUMENTS_TABLE_NAME", DefaultDocumentsTable)
	queriesTable := getEnv("QUERIES_TABLE_NAME", DefaultQueriesTable)
	responsesTable := getEnv("RESPONSES_TABLE_NAME", DefaultResponsesTable)
	uploadSessionsTable := getEnv("UPLOAD_SESSIONS_TABLE_NAME", DefaultUploadSessionsTable)
	knowledgeBaseID := getEnv("KNOWLEDGE_BASE_ID", DefaultKnowledgeBaseID)
	dataSourceID := getEnv("DATA_SOURCE_ID", DefaultDataSourceID)
	modelID := getEnv("MODEL_ID", DefaultModelID)

	// AWS設定をロード
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("AWS設定の読み込みに失敗: %v", err)
	}

	// AWSクライアントを初期化
	dynamoClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	bedrockAgentClient := bedrockagent.NewFromConfig(cfg)
	bedrockRuntimeClient := bedrockruntime.NewFromConfig(cfg)
	bedrockAgentRuntimeClient := bedrockagentruntime.NewFromConfig(cfg)

	// サービスを初期化
	documentService := services.NewDocumentService(dynamoClient, documentsTable)
	responseService := services.NewResponseService(dynamoClient, responsesTable)
	queryService := services.NewQueryService(dynamoClient, queriesTable, responseService)
	knowledgeBaseService := services.NewKnowledgeBaseService(
		bedrockAgentClient,
		bedrockRuntimeClient,
		bedrockAgentRuntimeClient,
		knowledgeBaseID,
		dataSourceID,
		modelID,
	)
	uploadService := services.NewUploadService(
		dynamoClient,
		s3Client,
		uploadSessionsTable,
		s3Bucket,
		DefaultPresignExpiration,
		documentService,
		knowledgeBaseService,
	)

	// ハンドラーを初期化
	healthHandler := handlers.NewHealthHandler(version)
    documentsHandler := handlers.NewDocumentsHandler(documentService, uploadService, knowledgeBaseService)
	queriesHandler := handlers.NewQueriesHandler(queryService, responseService, knowledgeBaseService)

	// Ginエンジンをセットアップ
	r := setupRouter(healthHandler, documentsHandler, queriesHandler)

    log.Printf("Lambda starting...")
    log.Printf("Version: %s", version)
    log.Printf("Knowledge Base ID: %s", knowledgeBaseID)
    if knowledgeBaseID == "" {
        log.Printf("WARNING: KNOWLEDGE_BASE_ID is not configured. Running in mock KB mode.")
    }
    if dataSourceID == "" {
        log.Printf("WARNING: DATA_SOURCE_ID is not configured. Ingestion will be skipped.")
    }

	// Lambda環境では、aws-lambda-go-api-proxyを使用
	ginLambda = ginadapter.New(r)
	lambda.Start(Handler)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}

// setupRouter はGinルーターをセットアップ
func setupRouter(
	healthHandler *handlers.HealthHandler,
	documentsHandler *handlers.DocumentsHandler,
	queriesHandler *handlers.QueriesHandler,
) *ginEngine.Engine {
	// プロダクションモードではGinを本番モードに設定
	if os.Getenv("GIN_MODE") == "release" {
		ginEngine.SetMode(ginEngine.ReleaseMode)
	}

	r := ginEngine.New()

	// ミドルウェアを追加
	r.Use(handlers.RequestLoggerMiddleware())
	r.Use(handlers.RecoveryMiddleware())
	r.Use(handlers.CORSMiddleware())
	r.Use(handlers.ErrorHandlerMiddleware())

	// ヘルスチェックエンドポイント
	r.GET("/health", healthHandler.Health)

    // 文書関連エンドポイント（API Gatewayの定義と一致させる）
    r.POST("/documents", documentsHandler.CreateDocument)
    r.GET("/documents", documentsHandler.ListDocuments)
    r.GET("/documents/:documentId", documentsHandler.GetDocument)
    r.POST("/documents/:documentId/complete-upload", documentsHandler.CompleteUpload)
    // 新パラメータ名（互換維持のため同一ハンドラで対応）
    // ルートパターンは同一のため追加は不要。ハンドラ側でsessionId/docId両対応。
	r.DELETE("/documents/:documentId", documentsHandler.DeleteDocument)

	// クエリ関連エンドポイント（API Gatewayの定義と一致させる）
	r.POST("/queries", queriesHandler.CreateQuery)
	r.GET("/queries/:sessionId/history", queriesHandler.GetQueryHistory)

	// 404ハンドラー
	r.NoRoute(func(c *ginEngine.Context) {
		c.JSON(404, ginEngine.H{
			"error": ginEngine.H{
				"code":    404,
				"message": "リクエストされたエンドポイントが見つかりません",
				"type":    "not_found",
			},
		})
	})

	return r
}

// getEnv は環境変数を取得（デフォルト値付き）
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
