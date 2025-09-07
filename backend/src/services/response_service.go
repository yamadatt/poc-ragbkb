package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"poc-ragbkb-backend/src/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// ResponseServiceInterface はResponseServiceのインターフェース
type ResponseServiceInterface interface {
	CreateResponse(ctx context.Context, queryID string, answer string, sources []models.Source, processingTimeMs int64, modelUsed string, tokensUsed int32) (*models.Response, error)
	GetResponse(ctx context.Context, id string) (*models.Response, error)
	GetResponseByQueryID(ctx context.Context, queryID string) (*models.Response, error)
}

// ResponseService はレスポンス管理サービス
type ResponseService struct {
	dynamoDB  *dynamodb.Client
	tableName string
}

// NewResponseService はResponseServiceの新しいインスタンスを作成
func NewResponseService(dynamoDB *dynamodb.Client, tableName string) *ResponseService {
	return &ResponseService{
		dynamoDB:  dynamoDB,
		tableName: tableName,
	}
}

// CreateResponse は新しいレスポンスを作成
func (s *ResponseService) CreateResponse(ctx context.Context, queryID string, answer string, sources []models.Source, processingTimeMs int64, modelUsed string, tokensUsed int32) (*models.Response, error) {
	if queryID == "" {
		return nil, models.NewValidationError("queryId", "クエリIDは必須です")
	}
	if answer == "" {
		return nil, models.NewValidationError("answer", "回答は必須です")
	}

    now := time.Now()

    // セーフガード: 情報源の必須フィールドをフォールバックで補完
    for i := range sources {
        if sources[i].DocumentID == "" {
            sources[i].DocumentID = fmt.Sprintf("doc-%d", i+1)
        }
        if sources[i].FileName == "" {
            sources[i].FileName = fmt.Sprintf("document-%d", i+1)
        }
        // 抜粋が長すぎる場合は500文字に丸める（ルーン長ベース）
        r := []rune(sources[i].Excerpt)
        if len(r) > 500 {
            sources[i].Excerpt = string(r[:500])
        }
    }
	response := &models.Response{
		ID:               uuid.New().String(),
		QueryID:          queryID,
		Answer:           answer,
		Sources:          sources,
		ProcessingTimeMs: processingTimeMs,
		ModelUsed:        modelUsed,
		TokensUsed:       tokensUsed,
		CreatedAt:        now,
	}

    // 追加の安全策：Response側のトランケーションユーティリティも適用
    // （将来の変更に備え二重で丸め込み）
    respCopy := *response
    respCopy.TruncateExcerpts(500)
    response.Sources = respCopy.Sources

    // バリデーション
    if err := response.ValidateSources(); err != nil {
        return nil, err
    }

	// DynamoDBに保存
	item := response.ToDynamoDBItem()
	_, err := s.dynamoDB.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("レスポンスの作成に失敗しました: %v", err))
	}

	return response, nil
}

// GetResponse はレスポンスIDでレスポンスを取得
func (s *ResponseService) GetResponse(ctx context.Context, id string) (*models.Response, error) {
	if id == "" {
		return nil, models.NewValidationError("id", "レスポンスIDは必須です")
	}

	result, err := s.dynamoDB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})

	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("レスポンスの取得に失敗しました: %v", err))
	}

	if result.Item == nil {
		return nil, models.NewNotFoundError("レスポンス")
	}

	response, err := s.dynamoDBItemToResponse(result.Item)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("レスポンスデータの変換に失敗しました: %v", err))
	}

	return response, nil
}

// GetResponseByQueryID はクエリIDでレスポンスを取得
func (s *ResponseService) GetResponseByQueryID(ctx context.Context, queryID string) (*models.Response, error) {
	if queryID == "" {
		return nil, models.NewValidationError("queryId", "クエリIDは必須です")
	}

	// DynamoDBのScanを使用してqueryIDでフィルタ
	// 実際のプロダクションではGSIを使用することを推奨
	input := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("queryId = :queryId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":queryId": &types.AttributeValueMemberS{Value: queryID},
		},
		Limit: aws.Int32(1), // 1つのレスポンスのみを期待
	}

	result, err := s.dynamoDB.Scan(ctx, input)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("レスポンスの取得に失敗しました: %v", err))
	}

	if len(result.Items) == 0 {
		return nil, models.NewNotFoundError("レスポンス")
	}

	response, err := s.dynamoDBItemToResponse(result.Items[0])
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("レスポンスデータの変換に失敗しました: %v", err))
	}

	return response, nil
}

// dynamoDBItemToResponse はDynamoDB項目をResponseに変換
func (s *ResponseService) dynamoDBItemToResponse(item map[string]types.AttributeValue) (*models.Response, error) {
	response := &models.Response{}

	if id, ok := item["id"].(*types.AttributeValueMemberS); ok {
		response.ID = id.Value
	}
	if queryID, ok := item["queryId"].(*types.AttributeValueMemberS); ok {
		response.QueryID = queryID.Value
	}
	if answer, ok := item["answer"].(*types.AttributeValueMemberS); ok {
		response.Answer = answer.Value
	}
	if modelUsed, ok := item["modelUsed"].(*types.AttributeValueMemberS); ok {
		response.ModelUsed = modelUsed.Value
	}
	if tokensUsed, ok := item["tokensUsed"].(*types.AttributeValueMemberN); ok {
		if tokens, err := strconv.ParseInt(tokensUsed.Value, 10, 32); err == nil {
			response.TokensUsed = int32(tokens)
		}
	}
	if processingTimeMs, ok := item["processingTimeMs"].(*types.AttributeValueMemberN); ok {
		if timeMs, err := strconv.ParseInt(processingTimeMs.Value, 10, 64); err == nil {
			response.ProcessingTimeMs = timeMs
		}
	}
	if createdAt, ok := item["createdAt"].(*types.AttributeValueMemberS); ok {
		if t, err := time.Parse(time.RFC3339, createdAt.Value); err == nil {
			response.CreatedAt = t
		}
	}

	// Sourcesの変換
	if sourcesAttr, ok := item["sources"].(*types.AttributeValueMemberL); ok {
		sources := make([]models.Source, len(sourcesAttr.Value))
		for i, sourceAttr := range sourcesAttr.Value {
			if sourceMap, ok := sourceAttr.(*types.AttributeValueMemberM); ok {
				source := models.Source{}

				if documentID, ok := sourceMap.Value["documentId"].(*types.AttributeValueMemberS); ok {
					source.DocumentID = documentID.Value
				}
				if fileName, ok := sourceMap.Value["fileName"].(*types.AttributeValueMemberS); ok {
					source.FileName = fileName.Value
				}
				if excerpt, ok := sourceMap.Value["excerpt"].(*types.AttributeValueMemberS); ok {
					source.Excerpt = excerpt.Value
				}
				if confidence, ok := sourceMap.Value["confidence"].(*types.AttributeValueMemberN); ok {
					if conf, err := strconv.ParseInt(confidence.Value, 10, 32); err == nil {
						source.Confidence = float64(conf) / 1000.0 // 1000で割って元の0-1の範囲に戻す
					}
				}

				sources[i] = source
			}
		}
		response.Sources = sources
	}

	return response, nil
}
