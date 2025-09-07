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

// QueryServiceInterface はQueryServiceのインターフェース
type QueryServiceInterface interface {
	CreateQuery(ctx context.Context, req *models.CreateQueryRequest) (*models.Query, error)
	GetQuery(ctx context.Context, id string) (*models.Query, error)
	GetQueryHistory(ctx context.Context, sessionID string, offset, limit int) (*models.QueryHistoryResponse, error)
	UpdateQueryStatus(ctx context.Context, id string, status models.QueryStatus) error
	MarkQueryAsCompleted(ctx context.Context, id string, processingTimeMs int64) error
	MarkQueryAsFailed(ctx context.Context, id string, errorMsg string, processingTimeMs int64) error
}

// QueryService はクエリ管理サービス
type QueryService struct {
	dynamoDB        *dynamodb.Client
	queryTableName  string
	responseService ResponseServiceInterface
}

// NewQueryService はQueryServiceの新しいインスタンスを作成
func NewQueryService(dynamoDB *dynamodb.Client, queryTableName string, responseService ResponseServiceInterface) *QueryService {
	return &QueryService{
		dynamoDB:        dynamoDB,
		queryTableName:  queryTableName,
		responseService: responseService,
	}
}

// CreateQuery は新しいクエリを作成
func (s *QueryService) CreateQuery(ctx context.Context, req *models.CreateQueryRequest) (*models.Query, error) {
	// リクエストのバリデーション
	if err := req.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()
	query := &models.Query{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		Question:  req.Question,
		Status:    models.QueryStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// DynamoDBに保存
	item := query.ToDynamoDBItem()
	_, err := s.dynamoDB.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.queryTableName),
		Item:      item,
	})

	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("クエリの作成に失敗しました: %v", err))
	}

	return query, nil
}

// GetQuery はクエリIDでクエリを取得
func (s *QueryService) GetQuery(ctx context.Context, id string) (*models.Query, error) {
	if id == "" {
		return nil, models.NewValidationError("id", "クエリIDは必須です")
	}

	result, err := s.dynamoDB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.queryTableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})

	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("クエリの取得に失敗しました: %v", err))
	}

	if result.Item == nil {
		return nil, models.NewNotFoundError("クエリ")
	}

	query, err := s.dynamoDBItemToQuery(result.Item)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("クエリデータの変換に失敗しました: %v", err))
	}

	return query, nil
}

// GetQueryHistory はセッションIDでクエリ履歴を取得
func (s *QueryService) GetQueryHistory(ctx context.Context, sessionID string, offset, limit int) (*models.QueryHistoryResponse, error) {
	if sessionID == "" {
		return nil, models.NewValidationError("sessionId", "セッションIDは必須です")
	}

	// UUIDの基本的なバリデーション
	if len(sessionID) != 36 {
		return nil, models.NewValidationError("sessionId", "無効なセッションIDです")
	}

	if limit <= 0 || limit > 50 {
		limit = 10 // デフォルト値
	}

	// DynamoDBのQueryを使用してセッションIDでフィルタ
	// 実際のプロダクションではGSIを使用することを推奨
	input := &dynamodb.ScanInput{
		TableName:        aws.String(s.queryTableName),
		FilterExpression: aws.String("sessionId = :sessionId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sessionId": &types.AttributeValueMemberS{Value: sessionID},
		},
		Limit: aws.Int32(int32(limit + 1)), // hasMoreを判定するために+1
	}

	result, err := s.dynamoDB.Scan(ctx, input)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("クエリ履歴の取得に失敗しました: %v", err))
	}

	queriesWithResponse := make([]*models.QueryWithResponse, 0, len(result.Items))
	for i, item := range result.Items {
		if i >= limit { // limitを超えた分はhasMoreの判定用
			break
		}

		query, err := s.dynamoDBItemToQuery(item)
		if err != nil {
			continue // エラーが発生したアイテムはスキップ
		}

		queryWithResponse := &models.QueryWithResponse{
			Query: query.ToResponse(),
		}

		// レスポンスがある場合は取得
		if query.IsCompleted() && s.responseService != nil {
			response, err := s.responseService.GetResponseByQueryID(ctx, query.ID)
			if err == nil {
				queryWithResponse.Response = response.ToResponse()
			}
		}

		queriesWithResponse = append(queriesWithResponse, queryWithResponse)
	}

	response := &models.QueryHistoryResponse{
		Queries:   queriesWithResponse,
		Total:     len(queriesWithResponse),
		SessionID: sessionID,
		Offset:    offset,
		Limit:     limit,
		HasMore:   len(result.Items) > limit,
	}

	return response, nil
}

// UpdateQueryStatus はクエリのステータスを更新
func (s *QueryService) UpdateQueryStatus(ctx context.Context, id string, status models.QueryStatus) error {
	if id == "" {
		return models.NewValidationError("id", "クエリIDは必須です")
	}

	now := time.Now()
	updateExpression := "SET #status = :status, #updatedAt = :updatedAt"
	expressionAttributeNames := map[string]string{
		"#status":    "status",
		"#updatedAt": "updatedAt",
	}
	expressionAttributeValues := map[string]types.AttributeValue{
		":status":    &types.AttributeValueMemberS{Value: string(status)},
		":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	// 完了状態の場合は完了日時も更新
	if status == models.QueryStatusCompleted || status == models.QueryStatusFailed {
		updateExpression += ", #completedAt = :completedAt"
		expressionAttributeNames["#completedAt"] = "completedAt"
		expressionAttributeValues[":completedAt"] = &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)}
	}

	_, err := s.dynamoDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.queryTableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
		ConditionExpression:       aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("クエリステータスの更新に失敗しました: %v", err))
	}

	return nil
}

// MarkQueryAsCompleted はクエリを完了状態にマーク
func (s *QueryService) MarkQueryAsCompleted(ctx context.Context, id string, processingTimeMs int64) error {
	if id == "" {
		return models.NewValidationError("id", "クエリIDは必須です")
	}

	now := time.Now()
	_, err := s.dynamoDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.queryTableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :status, #processingTimeMs = :processingTimeMs, #completedAt = :completedAt, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":           "status",
			"#processingTimeMs": "processingTimeMs",
			"#completedAt":      "completedAt",
			"#updatedAt":        "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":           &types.AttributeValueMemberS{Value: string(models.QueryStatusCompleted)},
			":processingTimeMs": &types.AttributeValueMemberN{Value: strconv.FormatInt(processingTimeMs, 10)},
			":completedAt":      &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":updatedAt":        &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("クエリの完了状態への更新に失敗しました: %v", err))
	}

	return nil
}

// MarkQueryAsFailed はクエリを失敗状態にマーク
func (s *QueryService) MarkQueryAsFailed(ctx context.Context, id string, errorMsg string, processingTimeMs int64) error {
	if id == "" {
		return models.NewValidationError("id", "クエリIDは必須です")
	}

	now := time.Now()
	_, err := s.dynamoDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.queryTableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :status, #errorMessage = :errorMessage, #processingTimeMs = :processingTimeMs, #completedAt = :completedAt, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":           "status",
			"#errorMessage":     "errorMessage",
			"#processingTimeMs": "processingTimeMs",
			"#completedAt":      "completedAt",
			"#updatedAt":        "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":           &types.AttributeValueMemberS{Value: string(models.QueryStatusFailed)},
			":errorMessage":     &types.AttributeValueMemberS{Value: errorMsg},
			":processingTimeMs": &types.AttributeValueMemberN{Value: strconv.FormatInt(processingTimeMs, 10)},
			":completedAt":      &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":updatedAt":        &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("クエリの失敗状態への更新に失敗しました: %v", err))
	}

	return nil
}

// dynamoDBItemToQuery はDynamoDB項目をQueryに変換
func (s *QueryService) dynamoDBItemToQuery(item map[string]types.AttributeValue) (*models.Query, error) {
	query := &models.Query{}

	if id, ok := item["id"].(*types.AttributeValueMemberS); ok {
		query.ID = id.Value
	}
	if sessionID, ok := item["sessionId"].(*types.AttributeValueMemberS); ok {
		query.SessionID = sessionID.Value
	}
	if question, ok := item["question"].(*types.AttributeValueMemberS); ok {
		query.Question = question.Value
	}
	if status, ok := item["status"].(*types.AttributeValueMemberS); ok {
		query.Status = models.QueryStatus(status.Value)
	}
	if errorMessage, ok := item["errorMessage"].(*types.AttributeValueMemberS); ok {
		query.ErrorMessage = &errorMessage.Value
	}
	if processingTimeMs, ok := item["processingTimeMs"].(*types.AttributeValueMemberN); ok {
		if timeMs, err := strconv.ParseInt(processingTimeMs.Value, 10, 64); err == nil {
			query.ProcessingTimeMs = timeMs
		}
	}
	if createdAt, ok := item["createdAt"].(*types.AttributeValueMemberS); ok {
		if t, err := time.Parse(time.RFC3339, createdAt.Value); err == nil {
			query.CreatedAt = t
		}
	}
	if updatedAt, ok := item["updatedAt"].(*types.AttributeValueMemberS); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt.Value); err == nil {
			query.UpdatedAt = t
		}
	}
	if completedAt, ok := item["completedAt"].(*types.AttributeValueMemberS); ok {
		if t, err := time.Parse(time.RFC3339, completedAt.Value); err == nil {
			query.CompletedAt = &t
		}
	}

	return query, nil
}
