package models

import (
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// UploadSessionStatus はアップロードセッションの状態を表します
type UploadSessionStatus string

const (
	UploadSessionStatusActive   UploadSessionStatus = "active"   // アクティブ（アップロード可能）
	UploadSessionStatusUsed     UploadSessionStatus = "used"     // 使用済み（アップロード完了）
	UploadSessionStatusExpired  UploadSessionStatus = "expired"  // 期限切れ
	UploadSessionStatusCanceled UploadSessionStatus = "canceled" // キャンセル済み
)

// UploadSession は文書アップロードセッションエンティティです
type UploadSession struct {
	ID         string              `json:"id" dynamodbav:"id"`                 // セッションID（UUID）
	DocumentID string              `json:"documentId" dynamodbav:"documentId"` // 関連する文書ID
	FileName   string              `json:"fileName" dynamodbav:"fileName"`     // ファイル名
	FileSize   int64               `json:"fileSize" dynamodbav:"fileSize"`     // ファイルサイズ
	FileType   string              `json:"fileType" dynamodbav:"fileType"`     // ファイルタイプ
	UploadURL  string              `json:"uploadUrl" dynamodbav:"uploadUrl"`   // S3署名付きURL
	S3Key      string              `json:"s3Key" dynamodbav:"s3Key"`           // S3オブジェクトキー
	S3Bucket   string              `json:"s3Bucket" dynamodbav:"s3Bucket"`     // S3バケット名
	Status     UploadSessionStatus `json:"status" dynamodbav:"status"`         // セッション状態
	ExpiresAt  time.Time           `json:"expiresAt" dynamodbav:"expiresAt"`   // 有効期限
	CreatedAt  time.Time           `json:"createdAt" dynamodbav:"createdAt"`   // 作成日時
	UpdatedAt  time.Time           `json:"updatedAt" dynamodbav:"updatedAt"`   // 更新日時
	UsedAt     *time.Time          `json:"usedAt" dynamodbav:"usedAt"`         // 使用日時
}

// UploadSessionResponse はアップロードセッションレスポンスです
type UploadSessionResponse struct {
	ID        string    `json:"id"`
	FileName  string    `json:"fileName"`
	FileSize  int64     `json:"fileSize"`
	FileType  string    `json:"fileType"`
	UploadURL string    `json:"uploadUrl"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ToResponse はUploadSessionをUploadSessionResponseに変換します
func (us *UploadSession) ToResponse() *UploadSessionResponse {
	return &UploadSessionResponse{
		ID:        us.ID,
		FileName:  us.FileName,
		FileSize:  us.FileSize,
		FileType:  us.FileType,
		UploadURL: us.UploadURL,
		ExpiresAt: us.ExpiresAt,
	}
}

// CompleteUploadRequest はアップロード完了リクエストです
type CompleteUploadRequest struct {
	SessionID string `json:"sessionId" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// Validate はアップロード完了リクエストのバリデーションを行います
func (req *CompleteUploadRequest) Validate() error {
	if req.SessionID == "" {
		return NewValidationError("sessionId", "セッションIDは必須です")
	}
	return nil
}

// CompleteUploadResponse はアップロード完了レスポンスです
type CompleteUploadResponse struct {
	ID       string         `json:"id"`
	FileName string         `json:"fileName"`
	FileSize int64          `json:"fileSize"`
	FileType string         `json:"fileType"`
	Status   DocumentStatus `json:"status"`
}

// DynamoDB用のAttributeValue変換メソッド

// ToDynamoDBItem はUploadSessionをDynamoDB項目に変換します
func (us *UploadSession) ToDynamoDBItem() map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"id":         &types.AttributeValueMemberS{Value: us.ID},
		"documentId": &types.AttributeValueMemberS{Value: us.DocumentID},
		"fileName":   &types.AttributeValueMemberS{Value: us.FileName},
		"fileSize":   &types.AttributeValueMemberN{Value: strconv.FormatInt(us.FileSize, 10)},
		"fileType":   &types.AttributeValueMemberS{Value: us.FileType},
		"uploadUrl":  &types.AttributeValueMemberS{Value: us.UploadURL},
		"s3Key":      &types.AttributeValueMemberS{Value: us.S3Key},
		"s3Bucket":   &types.AttributeValueMemberS{Value: us.S3Bucket},
		"status":     &types.AttributeValueMemberS{Value: string(us.Status)},
		"expiresAt":  &types.AttributeValueMemberS{Value: us.ExpiresAt.Format(time.RFC3339)},
		"createdAt":  &types.AttributeValueMemberS{Value: us.CreatedAt.Format(time.RFC3339)},
		"updatedAt":  &types.AttributeValueMemberS{Value: us.UpdatedAt.Format(time.RFC3339)},
	}

	if us.UsedAt != nil {
		item["usedAt"] = &types.AttributeValueMemberS{Value: us.UsedAt.Format(time.RFC3339)}
	}

	return item
}

// IsActive はセッションがアクティブかを判定します
func (us *UploadSession) IsActive() bool {
	return us.Status == UploadSessionStatusActive && time.Now().Before(us.ExpiresAt)
}

// IsExpired はセッションが期限切れかを判定します
func (us *UploadSession) IsExpired() bool {
	return time.Now().After(us.ExpiresAt) || us.Status == UploadSessionStatusExpired
}

// MarkAsUsed はセッションを使用済みに更新します
func (us *UploadSession) MarkAsUsed() {
	now := time.Now()
	us.Status = UploadSessionStatusUsed
	us.UsedAt = &now
	us.UpdatedAt = now
}

// MarkAsExpired はセッションを期限切れに更新します
func (us *UploadSession) MarkAsExpired() {
	us.Status = UploadSessionStatusExpired
	us.UpdatedAt = time.Now()
}

// MarkAsCanceled はセッションをキャンセル済みに更新します
func (us *UploadSession) MarkAsCanceled() {
	us.Status = UploadSessionStatusCanceled
	us.UpdatedAt = time.Now()
}

// GenerateS3Key はS3オブジェクトキーを生成します
func (us *UploadSession) GenerateS3Key() string {
	// documents/{documentId}/{timestamp}_{fileName}の形式
	timestamp := us.CreatedAt.Format("20060102150405")
	return "documents/" + us.DocumentID + "/" + timestamp + "_" + us.FileName
}

// GetRemainingTTL は残りの有効期限を秒単位で返します
func (us *UploadSession) GetRemainingTTL() int64 {
	remaining := us.ExpiresAt.Unix() - time.Now().Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}
