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

// DocumentServiceInterface はDocumentServiceのインターフェース
type DocumentServiceInterface interface {
	CreateDocument(ctx context.Context, req *models.CreateDocumentRequest) (*models.Document, error)
	GetDocument(ctx context.Context, id string) (*models.Document, error)
	ListDocuments(ctx context.Context, offset, limit int) (*models.DocumentListResponse, error)
	UpdateDocumentStatus(ctx context.Context, id string, status models.DocumentStatus) error
	UpdateDocumentPreview(ctx context.Context, id string, preview *string, previewLines int) error
	DeleteDocument(ctx context.Context, id string) error
	MarkDocumentAsReady(ctx context.Context, id string, kbDataSourceID string) error
	MarkDocumentAsError(ctx context.Context, id string, errorMsg string) error
	MarkDocumentAsKBSyncError(ctx context.Context, id string, errorMsg string) error
}

// DocumentService は文書管理サービス
type DocumentService struct {
	dynamoDB  *dynamodb.Client
	tableName string
}

// NewDocumentService はDocumentServiceの新しいインスタンスを作成
func NewDocumentService(dynamoDB *dynamodb.Client, tableName string) *DocumentService {
	return &DocumentService{
		dynamoDB:  dynamoDB,
		tableName: tableName,
	}
}

// CreateDocument は新しい文書を作成
func (s *DocumentService) CreateDocument(ctx context.Context, req *models.CreateDocumentRequest) (*models.Document, error) {
	// リクエストのバリデーション
	if err := req.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()
	document := &models.Document{
		ID:         uuid.New().String(),
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		FileType:   req.FileType,
		Status:     models.DocumentStatusUploading,
		UploadedAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// DynamoDBに保存
	item := document.ToDynamoDBItem()
	_, err := s.dynamoDB.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("文書の作成に失敗しました: %v", err))
	}

	return document, nil
}

// GetDocument は文書IDで文書を取得
func (s *DocumentService) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	if id == "" {
		return nil, models.NewValidationError("id", "文書IDは必須です")
	}

	result, err := s.dynamoDB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})

	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("文書の取得に失敗しました: %v", err))
	}

	if result.Item == nil {
		return nil, models.NewNotFoundError("文書")
	}

	document, err := s.dynamoDBItemToDocument(result.Item)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("文書データの変換に失敗しました: %v", err))
	}

	return document, nil
}

// ListDocuments は文書一覧を取得
func (s *DocumentService) ListDocuments(ctx context.Context, offset, limit int) (*models.DocumentListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20 // デフォルト値
	}

	// DynamoDBのScanを使用（実際のプロダクションではGSIを使用することを推奨）
	input := &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
		Limit:     aws.Int32(int32(limit + 1)), // hasMoreを判定するために+1
	}

	result, err := s.dynamoDB.Scan(ctx, input)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("文書一覧の取得に失敗しました: %v", err))
	}

	documents := make([]*models.DocumentResponse, 0, len(result.Items))
	for i, item := range result.Items {
		if i >= limit { // limitを超えた分はhasMoreの判定用
			break
		}

		document, err := s.dynamoDBItemToDocument(item)
		if err != nil {
			continue // エラーが発生したアイテムはスキップ
		}
		documents = append(documents, document.ToResponse())
	}

	response := &models.DocumentListResponse{
		Documents: documents,
		Total:     len(documents),
		Offset:    offset,
		Limit:     limit,
		HasMore:   len(result.Items) > limit,
	}

	return response, nil
}

// UpdateDocumentStatus は文書のステータスを更新
func (s *DocumentService) UpdateDocumentStatus(ctx context.Context, id string, status models.DocumentStatus) error {
	if id == "" {
		return models.NewValidationError("id", "文書IDは必須です")
	}

	now := time.Now()
	_, err := s.dynamoDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :status, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":    "status",
			"#updatedAt": "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":    &types.AttributeValueMemberS{Value: string(status)},
			":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("文書ステータスの更新に失敗しました: %v", err))
	}

	return nil
}

// UpdateDocumentPreview は文書のプレビュー情報を更新
func (s *DocumentService) UpdateDocumentPreview(ctx context.Context, id string, preview *string, previewLines int) error {
	if id == "" {
		return models.NewValidationError("id", "文書IDは必須です")
	}

	now := time.Now()
	updateExpr := "SET #updatedAt = :updatedAt, #previewLines = :previewLines"
	exprAttrNames := map[string]string{
		"#updatedAt":    "updatedAt",
		"#previewLines": "previewLines",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":updatedAt":    &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		":previewLines": &types.AttributeValueMemberN{Value: strconv.Itoa(previewLines)},
	}

	if preview != nil {
		updateExpr += ", #preview = :preview"
		exprAttrNames["#preview"] = "preview"
		exprAttrValues[":preview"] = &types.AttributeValueMemberS{Value: *preview}
	}

	_, err := s.dynamoDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(s.tableName),
		Key:                      map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:         aws.String(updateExpr),
		ExpressionAttributeNames: exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
		ConditionExpression:      aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("文書プレビュー情報の更新に失敗しました: %v", err))
	}

	return nil
}

// DeleteDocument は文書を削除
func (s *DocumentService) DeleteDocument(ctx context.Context, id string) error {
	if id == "" {
		return models.NewValidationError("id", "文書IDは必須です")
	}

	_, err := s.dynamoDB.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("文書の削除に失敗しました: %v", err))
	}

	return nil
}

// MarkDocumentAsReady は文書を処理完了状態にマーク
func (s *DocumentService) MarkDocumentAsReady(ctx context.Context, id string, kbDataSourceID string) error {
	if id == "" {
		return models.NewValidationError("id", "文書IDは必須です")
	}

	now := time.Now()
	_, err := s.dynamoDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :status, #processedAt = :processedAt, #kbDataSource = :kbDataSource, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":       "status",
			"#processedAt":  "processedAt",
			"#kbDataSource": "kbDataSource",
			"#updatedAt":    "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":       &types.AttributeValueMemberS{Value: string(models.DocumentStatusReady)},
			":processedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":kbDataSource": &types.AttributeValueMemberS{Value: kbDataSourceID},
			":updatedAt":    &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("文書のReady状態への更新に失敗しました: %v", err))
	}

	return nil
}

// MarkDocumentAsError は文書をエラー状態にマーク
func (s *DocumentService) MarkDocumentAsError(ctx context.Context, id string, errorMsg string) error {
	if id == "" {
		return models.NewValidationError("id", "文書IDは必須です")
	}

	now := time.Now()
	_, err := s.dynamoDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :status, #errorMessage = :errorMessage, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":       "status",
			"#errorMessage": "errorMessage",
			"#updatedAt":    "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":       &types.AttributeValueMemberS{Value: string(models.DocumentStatusError)},
			":errorMessage": &types.AttributeValueMemberS{Value: errorMsg},
			":updatedAt":    &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("文書のエラー状態への更新に失敗しました: %v", err))
	}

	return nil
}

// MarkDocumentAsKBSyncError は文書をKnowledge Base同期エラー状態にマーク
func (s *DocumentService) MarkDocumentAsKBSyncError(ctx context.Context, id string, errorMsg string) error {
	if id == "" {
		return models.NewValidationError("id", "文書IDは必須です")
	}

	now := time.Now()
	_, err := s.dynamoDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :status, errorMessage = :errorMessage, updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":       &types.AttributeValueMemberS{Value: string(models.DocumentStatusKBSyncError)},
			":errorMessage": &types.AttributeValueMemberS{Value: errorMsg},
			":updatedAt":    &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return models.NewInternalError(fmt.Sprintf("文書のKB同期エラー状態への更新に失敗しました: %v", err))
	}

	return nil
}

// dynamoDBItemToDocument はDynamoDB項目をDocumentに変換
func (s *DocumentService) dynamoDBItemToDocument(item map[string]types.AttributeValue) (*models.Document, error) {
	document := &models.Document{}

	if id, ok := item["id"].(*types.AttributeValueMemberS); ok {
		document.ID = id.Value
	}
	if fileName, ok := item["fileName"].(*types.AttributeValueMemberS); ok {
		document.FileName = fileName.Value
	}
	if fileSize, ok := item["fileSize"].(*types.AttributeValueMemberN); ok {
		if size, err := strconv.ParseInt(fileSize.Value, 10, 64); err == nil {
			document.FileSize = size
		}
	}
	if fileType, ok := item["fileType"].(*types.AttributeValueMemberS); ok {
		document.FileType = fileType.Value
	}
	if s3Key, ok := item["s3Key"].(*types.AttributeValueMemberS); ok {
		document.S3Key = s3Key.Value
	}
	if s3Bucket, ok := item["s3Bucket"].(*types.AttributeValueMemberS); ok {
		document.S3Bucket = s3Bucket.Value
	}
	if status, ok := item["status"].(*types.AttributeValueMemberS); ok {
		document.Status = models.DocumentStatus(status.Value)
	}
	if preview, ok := item["preview"].(*types.AttributeValueMemberS); ok {
		document.Preview = &preview.Value
	}
	if previewLines, ok := item["previewLines"].(*types.AttributeValueMemberN); ok {
		if lines, err := strconv.Atoi(previewLines.Value); err == nil {
			document.PreviewLines = lines
		}
	}
	if uploadedAt, ok := item["uploadedAt"].(*types.AttributeValueMemberS); ok {
		if t, err := time.Parse(time.RFC3339, uploadedAt.Value); err == nil {
			document.UploadedAt = t
		}
	}
	if processedAt, ok := item["processedAt"].(*types.AttributeValueMemberS); ok {
		if t, err := time.Parse(time.RFC3339, processedAt.Value); err == nil {
			document.ProcessedAt = &t
		}
	}
	if errorMessage, ok := item["errorMessage"].(*types.AttributeValueMemberS); ok {
		document.ErrorMessage = &errorMessage.Value
	}
	if kbDataSource, ok := item["kbDataSource"].(*types.AttributeValueMemberS); ok {
		document.KBDataSource = &kbDataSource.Value
	}
	if createdAt, ok := item["createdAt"].(*types.AttributeValueMemberS); ok {
		if t, err := time.Parse(time.RFC3339, createdAt.Value); err == nil {
			document.CreatedAt = t
		}
	}
	if updatedAt, ok := item["updatedAt"].(*types.AttributeValueMemberS); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt.Value); err == nil {
			document.UpdatedAt = t
		}
	}

	return document, nil
}
