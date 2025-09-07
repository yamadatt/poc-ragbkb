package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// QueryStatus はクエリの処理状態を表します
type QueryStatus string

const (
	QueryStatusPending    QueryStatus = "pending"    // 処理待ち
	QueryStatusProcessing QueryStatus = "processing" // 処理中
	QueryStatusCompleted  QueryStatus = "completed"  // 完了
	QueryStatusFailed     QueryStatus = "failed"     // 失敗
)

// Query はクエリエンティティです
type Query struct {
	ID                  string      `json:"id" dynamodbav:"id"`                             // クエリID（UUID）
	SessionID           string      `json:"sessionId" dynamodbav:"sessionId"`               // セッションID（UUID）
	Question            string      `json:"question" dynamodbav:"question"`                 // 質問内容
	Status              QueryStatus `json:"status" dynamodbav:"status"`                     // 処理状態
	ErrorMessage        *string     `json:"errorMessage" dynamodbav:"errorMessage"`         // エラーメッセージ
	ProcessingTimeMs    int64       `json:"processingTimeMs" dynamodbav:"processingTimeMs"` // 処理時間（ミリ秒）
	ProcessingStartedAt *time.Time  `json:"processingStartedAt" dynamodbav:"processingStartedAt"` // 処理開始日時
	CreatedAt           time.Time   `json:"createdAt" dynamodbav:"createdAt"`               // 作成日時
	UpdatedAt           time.Time   `json:"updatedAt" dynamodbav:"updatedAt"`               // 更新日時
	CompletedAt         *time.Time  `json:"completedAt" dynamodbav:"completedAt"`           // 完了日時
}

// CreateQueryRequest はクエリ作成リクエストです
type CreateQueryRequest struct {
	Question  string `json:"question" binding:"required" example:"AWS Bedrock Knowledge Baseの使い方を教えてください"`
	SessionID string `json:"sessionId" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// Validate はクエリ作成リクエストのバリデーションを行います
func (req *CreateQueryRequest) Validate() error {
	if strings.TrimSpace(req.Question) == "" {
		return NewValidationError("question", "質問は必須です")
	}
	if len(req.Question) > 1000 {
		return NewValidationError("question", "質問は1000文字以内で入力してください")
	}
	if req.SessionID == "" {
		return NewValidationError("sessionId", "セッションIDは必須です")
	}
	// カスタムセッション形式 (session_xxxxx_xxxxx) またはUUID形式を受け入れる
	if len(req.SessionID) < 10 || len(req.SessionID) > 50 {
		return NewValidationError("sessionId", "無効なセッションIDです")
	}
	return nil
}

// QueryResponse はクエリレスポンスです
type QueryResponse struct {
	ID               string      `json:"id"`
	SessionID        string      `json:"sessionId"`
	Question         string      `json:"question"`
	Status           QueryStatus `json:"status"`
	ProcessingTimeMs int64       `json:"processingTimeMs"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
	CompletedAt      *time.Time  `json:"completedAt,omitempty"`
}

// ToResponse はQueryをQueryResponseに変換します
func (q *Query) ToResponse() *QueryResponse {
	return &QueryResponse{
		ID:               q.ID,
		SessionID:        q.SessionID,
		Question:         q.Question,
		Status:           q.Status,
		ProcessingTimeMs: q.ProcessingTimeMs,
		CreatedAt:        q.CreatedAt,
		UpdatedAt:        q.UpdatedAt,
		CompletedAt:      q.CompletedAt,
	}
}

// QueryHistoryResponse はクエリ履歴レスポンスです
type QueryHistoryResponse struct {
	Queries   []*QueryWithResponse `json:"queries"`
	Total     int                  `json:"total"`
	SessionID string               `json:"sessionId"`
	Offset    int                  `json:"offset"`
	Limit     int                  `json:"limit"`
	HasMore   bool                 `json:"hasMore"`
}

// QueryWithResponse はクエリとレスポンスを組み合わせた構造体です
type QueryWithResponse struct {
	Query    *QueryResponse    `json:"query"`
	Response *ResponseResponse `json:"response,omitempty"`
}

// DynamoDB用のAttributeValue変換メソッド

// ToDynamoDBItem はQueryをDynamoDB項目に変換します
func (q *Query) ToDynamoDBItem() map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"id":               &types.AttributeValueMemberS{Value: q.ID},
		"sessionId":        &types.AttributeValueMemberS{Value: q.SessionID},
		"question":         &types.AttributeValueMemberS{Value: q.Question},
		"status":           &types.AttributeValueMemberS{Value: string(q.Status)},
		"processingTimeMs": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", q.ProcessingTimeMs)},
		"createdAt":        &types.AttributeValueMemberS{Value: q.CreatedAt.Format(time.RFC3339)},
		"updatedAt":        &types.AttributeValueMemberS{Value: q.UpdatedAt.Format(time.RFC3339)},
	}

	if q.ErrorMessage != nil {
		item["errorMessage"] = &types.AttributeValueMemberS{Value: *q.ErrorMessage}
	}
	if q.CompletedAt != nil {
		item["completedAt"] = &types.AttributeValueMemberS{Value: q.CompletedAt.Format(time.RFC3339)}
	}

	return item
}

// MarkAsProcessing はクエリのステータスを処理中に更新します
func (q *Query) MarkAsProcessing() {
	q.Status = QueryStatusProcessing
	q.UpdatedAt = time.Now()
}

// MarkAsCompleted はクエリのステータスを完了に更新します
func (q *Query) MarkAsCompleted(processingTimeMs int64) {
	now := time.Now()
	q.Status = QueryStatusCompleted
	q.ProcessingTimeMs = processingTimeMs
	q.CompletedAt = &now
	q.UpdatedAt = now
}

// MarkAsFailed はクエリのステータスを失敗に更新します
func (q *Query) MarkAsFailed(errorMsg string, processingTimeMs int64) {
	now := time.Now()
	q.Status = QueryStatusFailed
	q.ErrorMessage = &errorMsg
	q.ProcessingTimeMs = processingTimeMs
	q.CompletedAt = &now
	q.UpdatedAt = now
}

// IsCompleted はクエリが完了したかを判定します
func (q *Query) IsCompleted() bool {
	return q.Status == QueryStatusCompleted || q.Status == QueryStatusFailed
}

// Validate はクエリの妥当性をバリデーションします
func (q *Query) Validate() error {
	if q.ID == "" {
		return NewValidationError("id", "query ID is required")
	}
	if q.SessionID == "" {
		return NewValidationError("sessionId", "session ID is required")
	}
	if strings.TrimSpace(q.Question) == "" {
		return NewValidationError("question", "question is required")
	}
	if len(q.Question) > 1000 {
		return NewValidationError("question", "question exceeds maximum length")
	}
	if q.Status != QueryStatusPending && q.Status != QueryStatusProcessing && 
		q.Status != QueryStatusCompleted && q.Status != QueryStatusFailed {
		return NewValidationError("status", "invalid query status")
	}
	if q.ProcessingTimeMs < 0 {
		return NewValidationError("processingTimeMs", "processing time cannot be negative")
	}
	return nil
}

// StartProcessing はクエリの処理を開始します
func (q *Query) StartProcessing() {
	now := time.Now()
	q.Status = QueryStatusProcessing
	q.ProcessingStartedAt = &now
	q.UpdatedAt = now
}

// CompleteProcessing はクエリの処理を完了します
func (q *Query) CompleteProcessing() {
	if q.ProcessingStartedAt != nil {
		processingTime := time.Since(*q.ProcessingStartedAt)
		q.ProcessingTimeMs = processingTime.Milliseconds()
	}
	q.Status = QueryStatusCompleted
	now := time.Now()
	q.CompletedAt = &now
	q.UpdatedAt = now
}

// FailProcessing はクエリの処理を失敗として完了します
func (q *Query) FailProcessing(errorMessage string) {
	if q.ProcessingStartedAt != nil {
		processingTime := time.Since(*q.ProcessingStartedAt)
		q.ProcessingTimeMs = processingTime.Milliseconds()
	}
	q.Status = QueryStatusFailed
	q.ErrorMessage = &errorMessage
	now := time.Now()
	q.CompletedAt = &now
	q.UpdatedAt = now
}

// IsProcessable はクエリが処理可能な状態かを判定します
func (q *Query) IsProcessable() bool {
	return q.Status == QueryStatusPending
}

// IsRetryable はクエリが再試行可能な状態かを判定します
func (q *Query) IsRetryable() bool {
	return q.Status == QueryStatusFailed
}

// GetProcessingDuration は処理時間をtime.Durationで返します
func (q *Query) GetProcessingDuration() time.Duration {
	return time.Duration(q.ProcessingTimeMs) * time.Millisecond
}

// GetFormattedProcessingTime はフォーマットされた処理時間を返します
func (q *Query) GetFormattedProcessingTime() string {
	duration := q.GetProcessingDuration()
	return duration.String()
}

// SanitizeQuestion は質問内容をサニタイズします
func (q *Query) SanitizeQuestion() {
	// HTMLタグの除去
	q.Question = strings.ReplaceAll(q.Question, "<", "&lt;")
	q.Question = strings.ReplaceAll(q.Question, ">", "&gt;")
	
	// 改行文字の正規化
	q.Question = strings.ReplaceAll(q.Question, "\r\n", "\n")
	q.Question = strings.ReplaceAll(q.Question, "\r", "\n")
	
	// 余分な空白の除去
	q.Question = strings.TrimSpace(q.Question)
}

// EstimateTokenCount はトークン数を見積もります（大まかな計算）
func (q *Query) EstimateTokenCount() int {
	if len(q.Question) == 0 {
		return 0
	}
	// 日本語の場合、約4文字で1トークン程度と仮定
	return len([]rune(q.Question)) / 4
}

// GetAgeInMinutes はクエリの経過時間を分単位で返します
func (q *Query) GetAgeInMinutes() float64 {
	return time.Since(q.CreatedAt).Minutes()
}

// IsStale はクエリが期限切れかを判定します
func (q *Query) IsStale() bool {
	age := q.GetAgeInMinutes()
	switch q.Status {
	case QueryStatusPending:
		return age > 10 // 10分以上pending状態の場合はstale
	case QueryStatusProcessing:
		return age > 15 // 15分以上processing状態の場合はstale
	default:
		return false // 完了済みの場合はstaleではない
	}
}
