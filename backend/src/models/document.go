package models

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DocumentStatus は文書の処理状態を表します
type DocumentStatus string

const (
	DocumentStatusUploading    DocumentStatus = "uploading"    // アップロード中
	DocumentStatusProcessing   DocumentStatus = "processing"   // 処理中（Knowledge Base同期中）
	DocumentStatusReady        DocumentStatus = "ready"        // 利用可能
	DocumentStatusError        DocumentStatus = "error"        // アップロードエラー
	DocumentStatusKBSyncError  DocumentStatus = "kb_sync_error" // Knowledge Base同期エラー（文書は利用可能）
)

// Document は文書エンティティです
type Document struct {
	ID           string         `json:"id" dynamodbav:"id"`                     // 文書ID（UUID）
	FileName     string         `json:"fileName" dynamodbav:"fileName"`         // ファイル名
	FileSize     int64          `json:"fileSize" dynamodbav:"fileSize"`         // ファイルサイズ（バイト）
	FileType     string         `json:"fileType" dynamodbav:"fileType"`         // ファイルタイプ（txt, md）
	S3Key        string         `json:"s3Key" dynamodbav:"s3Key"`               // S3オブジェクトキー
	S3Bucket     string         `json:"s3Bucket" dynamodbav:"s3Bucket"`         // S3バケット名
	Status       DocumentStatus `json:"status" dynamodbav:"status"`             // 処理状態
	Preview      *string        `json:"preview" dynamodbav:"preview"`           // 文書の冒頭部分（最大30行）
	PreviewLines int            `json:"previewLines" dynamodbav:"previewLines"` // プレビューの行数
	UploadedAt   time.Time      `json:"uploadedAt" dynamodbav:"uploadedAt"`     // アップロード日時
	ProcessedAt  *time.Time     `json:"processedAt" dynamodbav:"processedAt"`   // 処理完了日時
	ErrorMessage *string        `json:"errorMessage" dynamodbav:"errorMessage"` // エラーメッセージ
	KBDataSource *string        `json:"kbDataSource" dynamodbav:"kbDataSource"` // Knowledge BaseデータソースID
	CreatedAt    time.Time      `json:"createdAt" dynamodbav:"createdAt"`       // 作成日時
	UpdatedAt    time.Time      `json:"updatedAt" dynamodbav:"updatedAt"`       // 更新日時
}

// CreateDocumentRequest は文書作成リクエストです
type CreateDocumentRequest struct {
    FileName string `json:"fileName" binding:"required" example:"document.md"`
    FileSize int64  `json:"fileSize" binding:"required,min=1,max=52428800" example:"1024"` // 最大50MB
    FileType string `json:"fileType" binding:"required,oneof=txt md" example:"md"`
}

// Validate は文書作成リクエストのバリデーションを行います
func (req *CreateDocumentRequest) Validate() error {
	if req.FileName == "" {
		return NewValidationError("fileName", "ファイル名は必須です")
	}
	if req.FileSize <= 0 {
		return NewValidationError("fileSize", "ファイルサイズは1バイト以上である必要があります")
	}
    if req.FileSize > 52428800 { // 50MB
        return NewValidationError("fileSize", "ファイルサイズが制限を超えています（最大50MB）")
    }
	if req.FileType != "txt" && req.FileType != "md" {
		return NewValidationError("fileType", "サポートされていないファイルタイプです（txt, mdのみ）")
	}
	return nil
}

// DocumentResponse は文書レスポンスです
type DocumentResponse struct {
	ID           string         `json:"id"`
	FileName     string         `json:"fileName"`
	FileSize     int64          `json:"fileSize"`
	FileType     string         `json:"fileType"`
	Status       DocumentStatus `json:"status"`
	Preview      *string        `json:"preview,omitempty"`
	PreviewLines int            `json:"previewLines"`
	UploadedAt   time.Time      `json:"uploadedAt"`
	ProcessedAt  *time.Time     `json:"processedAt,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// ToResponse はDocumentをDocumentResponseに変換します
func (d *Document) ToResponse() *DocumentResponse {
	return &DocumentResponse{
		ID:           d.ID,
		FileName:     d.FileName,
		FileSize:     d.FileSize,
		FileType:     d.FileType,
		Status:       d.Status,
		Preview:      d.Preview,
		PreviewLines: d.PreviewLines,
		UploadedAt:   d.UploadedAt,
		ProcessedAt:  d.ProcessedAt,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

// DocumentListResponse は文書一覧レスポンスです
type DocumentListResponse struct {
	Documents  []*DocumentResponse `json:"documents"`
	Total      int                 `json:"total"`
	Offset     int                 `json:"offset"`
	Limit      int                 `json:"limit"`
	HasMore    bool                `json:"hasMore"`
	NextCursor *string             `json:"nextCursor,omitempty"`
}

// DynamoDB用のAttributeValue変換メソッド

// ToDynamoDBItem はDocumentをDynamoDB項目に変換します
func (d *Document) ToDynamoDBItem() map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"id":           &types.AttributeValueMemberS{Value: d.ID},
		"fileName":     &types.AttributeValueMemberS{Value: d.FileName},
		"fileSize":     &types.AttributeValueMemberN{Value: strconv.FormatInt(d.FileSize, 10)},
		"fileType":     &types.AttributeValueMemberS{Value: d.FileType},
		"s3Key":        &types.AttributeValueMemberS{Value: d.S3Key},
		"s3Bucket":     &types.AttributeValueMemberS{Value: d.S3Bucket},
		"status":       &types.AttributeValueMemberS{Value: string(d.Status)},
		"previewLines": &types.AttributeValueMemberN{Value: strconv.Itoa(d.PreviewLines)},
		"uploadedAt":   &types.AttributeValueMemberS{Value: d.UploadedAt.Format(time.RFC3339)},
		"createdAt":    &types.AttributeValueMemberS{Value: d.CreatedAt.Format(time.RFC3339)},
		"updatedAt":    &types.AttributeValueMemberS{Value: d.UpdatedAt.Format(time.RFC3339)},
	}

	if d.Preview != nil {
		item["preview"] = &types.AttributeValueMemberS{Value: *d.Preview}
	}
	if d.ProcessedAt != nil {
		item["processedAt"] = &types.AttributeValueMemberS{Value: d.ProcessedAt.Format(time.RFC3339)}
	}
	if d.ErrorMessage != nil {
		item["errorMessage"] = &types.AttributeValueMemberS{Value: *d.ErrorMessage}
	}
	if d.KBDataSource != nil {
		item["kbDataSource"] = &types.AttributeValueMemberS{Value: *d.KBDataSource}
	}

	return item
}

// IsProcessable は文書が処理可能な状態かを判定します
func (d *Document) IsProcessable() bool {
	return d.Status == DocumentStatusUploading || d.Status == DocumentStatusError
}

// MarkAsProcessing は文書のステータスを処理中に更新します
func (d *Document) MarkAsProcessing() {
	d.Status = DocumentStatusProcessing
	d.UpdatedAt = time.Now()
}

// MarkAsReady は文書のステータスを利用可能に更新します
func (d *Document) MarkAsReady(kbDataSourceID string) {
	now := time.Now()
	d.Status = DocumentStatusReady
	d.ProcessedAt = &now
	d.KBDataSource = &kbDataSourceID
	d.UpdatedAt = now
}

// MarkAsError は文書のステータスをエラーに更新します
func (d *Document) MarkAsError(errorMsg string) {
	d.Status = DocumentStatusError
	d.ErrorMessage = &errorMsg
	d.UpdatedAt = time.Now()
}

// Validate は文書の妥当性をバリデーションします
func (d *Document) Validate() error {
	if d.ID == "" {
		return NewValidationError("id", "document ID is required")
	}
	if d.FileName == "" {
		return NewValidationError("fileName", "file name is required")
	}
	if d.FileSize <= 0 {
		return NewValidationError("fileSize", "file size must be greater than 0")
	}
    if d.FileSize > 52428800 { // 50MB
        return NewValidationError("fileSize", "file size exceeds maximum limit")
    }
	if d.FileType != "txt" && d.FileType != "md" {
		return NewValidationError("fileType", "unsupported file type")
	}
	if d.Status != DocumentStatusUploading && d.Status != DocumentStatusProcessing && 
		d.Status != DocumentStatusReady && d.Status != DocumentStatusError {
		return NewValidationError("status", "invalid document status")
	}
	if d.S3Key == "" {
		return NewValidationError("s3Key", "S3 key is required")
	}
	if d.S3Bucket == "" {
		return NewValidationError("s3Bucket", "S3 bucket is required")
	}
	return nil
}

// IsReady は文書が利用可能状態かを判定します
func (d *Document) IsReady() bool {
	return d.Status == DocumentStatusReady
}

// IsError は文書がエラー状態かを判定します
func (d *Document) IsError() bool {
	return d.Status == DocumentStatusError
}

// CanBeDeleted は文書が削除可能な状態かを判定します
func (d *Document) CanBeDeleted() bool {
	return d.Status == DocumentStatusReady || d.Status == DocumentStatusError
}

// UpdateStatus は文書のステータスを更新します
func (d *Document) UpdateStatus(status DocumentStatus) {
	d.Status = status
	d.UpdatedAt = time.Now()
	if status == DocumentStatusReady {
		now := time.Now()
		d.ProcessedAt = &now
	}
}

// SetError は文書にエラーを設定します
func (d *Document) SetError(errorMsg string) {
	d.Status = DocumentStatusError
	d.ErrorMessage = &errorMsg
	d.UpdatedAt = time.Now()
}

// GetFileSizeFormatted は文書サイズをフォーマットして返します
func (d *Document) GetFileSizeFormatted() string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	size := float64(d.FileSize)
	switch {
	case d.FileSize >= GB:
		return fmt.Sprintf("%.1f GB", size/GB)
	case d.FileSize >= MB:
		return fmt.Sprintf("%.1f MB", size/MB)
	case d.FileSize >= KB:
		return fmt.Sprintf("%.1f KB", size/KB)
	default:
		return fmt.Sprintf("%d B", d.FileSize)
	}
}
