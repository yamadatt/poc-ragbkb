package unit

import (
	"testing"
	"time"

	"poc-ragbkb-backend/src/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery_Validation(t *testing.T) {
	tests := []struct {
		name    string
		query   *models.Query
		wantErr bool
		errMsg  string
	}{
		{
			name: "有効なクエリ",
			query: &models.Query{
				ID:               "query123",
				SessionID:        "session456",
				Question:         "AWS Bedrockについて教えてください",
				Status:           models.QueryStatusPending,
				ProcessingTimeMs: 0,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			wantErr: false,
		},
		{
			name: "IDが空",
			query: &models.Query{
				ID:               "",
				SessionID:        "session456",
				Question:         "AWS Bedrockについて教えてください",
				Status:           models.QueryStatusPending,
				ProcessingTimeMs: 0,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			wantErr: true,
			errMsg:  "query ID is required",
		},
		{
			name: "セッションIDが空",
			query: &models.Query{
				ID:               "query123",
				SessionID:        "",
				Question:         "AWS Bedrockについて教えてください",
				Status:           models.QueryStatusPending,
				ProcessingTimeMs: 0,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			wantErr: true,
			errMsg:  "session ID is required",
		},
		{
			name: "質問が空",
			query: &models.Query{
				ID:               "query123",
				SessionID:        "session456",
				Question:         "",
				Status:           models.QueryStatusPending,
				ProcessingTimeMs: 0,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			wantErr: true,
			errMsg:  "question is required",
		},
		{
			name: "質問が長すぎる",
			query: &models.Query{
				ID:               "query123",
				SessionID:        "session456",
				Question:         generateLongString(1001), // 1000文字制限を超える
				Status:           models.QueryStatusPending,
				ProcessingTimeMs: 0,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			wantErr: true,
			errMsg:  "question exceeds maximum length",
		},
		{
			name: "無効なステータス",
			query: &models.Query{
				ID:               "query123",
				SessionID:        "session456",
				Question:         "AWS Bedrockについて教えてください",
				Status:           "invalid_status",
				ProcessingTimeMs: 0,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid query status",
		},
		{
			name: "処理時間が負の値",
			query: &models.Query{
				ID:               "query123",
				SessionID:        "session456",
				Question:         "AWS Bedrockについて教えてください",
				Status:           models.QueryStatusCompleted,
				ProcessingTimeMs: -100,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			wantErr: true,
			errMsg:  "processing time cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestQuery_StatusValidation(t *testing.T) {
	validStatuses := []models.QueryStatus{
		models.QueryStatusPending,
		models.QueryStatusProcessing,
		models.QueryStatusCompleted,
		models.QueryStatusFailed,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			query := &models.Query{
				ID:               "query123",
				SessionID:        "session456",
				Question:         "テスト質問",
				Status:           status,
				ProcessingTimeMs: 100,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}

			err := query.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestQuery_StartProcessing(t *testing.T) {
	query := &models.Query{
		ID:               "query123",
		SessionID:        "session456",
		Question:         "テスト質問",
		Status:           models.QueryStatusPending,
		ProcessingTimeMs: 0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	oldUpdatedAt := query.UpdatedAt
	time.Sleep(time.Millisecond) // 時間差を確保

	query.StartProcessing()

	assert.Equal(t, models.QueryStatusProcessing, query.Status)
	assert.True(t, query.UpdatedAt.After(oldUpdatedAt))
	assert.True(t, query.ProcessingStartedAt != nil)
}

func TestQuery_CompleteProcessing(t *testing.T) {
	query := &models.Query{
		ID:                  "query123",
		SessionID:           "session456",
		Question:            "テスト質問",
		Status:              models.QueryStatusProcessing,
		ProcessingTimeMs:    0,
		ProcessingStartedAt: &time.Time{},
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	startTime := time.Now()
	query.ProcessingStartedAt = &startTime
	time.Sleep(time.Millisecond * 100) // 処理時間をシミュレート

	oldUpdatedAt := query.UpdatedAt
	query.CompleteProcessing()

	assert.Equal(t, models.QueryStatusCompleted, query.Status)
	assert.True(t, query.UpdatedAt.After(oldUpdatedAt))
	assert.True(t, query.ProcessingTimeMs > 0)
	assert.True(t, query.ProcessingTimeMs >= 100) // 最低100msは経過している
}

func TestQuery_FailProcessing(t *testing.T) {
	query := &models.Query{
		ID:                  "query123",
		SessionID:           "session456",
		Question:            "テスト質問",
		Status:              models.QueryStatusProcessing,
		ProcessingTimeMs:    0,
		ProcessingStartedAt: &time.Time{},
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	errorMessage := "処理中にエラーが発生しました"
	oldUpdatedAt := query.UpdatedAt
	time.Sleep(time.Millisecond) // 時間差を確保

	query.FailProcessing(errorMessage)

	assert.Equal(t, models.QueryStatusFailed, query.Status)
	assert.True(t, query.UpdatedAt.After(oldUpdatedAt))
	assert.Equal(t, errorMessage, *query.ErrorMessage)
}

func TestQuery_IsProcessable(t *testing.T) {
	tests := []struct {
		name     string
		query    *models.Query
		expected bool
	}{
		{
			name: "pending - 処理可能",
			query: &models.Query{
				Status: models.QueryStatusPending,
			},
			expected: true,
		},
		{
			name: "processing - 処理不可",
			query: &models.Query{
				Status: models.QueryStatusProcessing,
			},
			expected: false,
		},
		{
			name: "completed - 処理不可",
			query: &models.Query{
				Status: models.QueryStatusCompleted,
			},
			expected: false,
		},
		{
			name: "failed - 処理不可（ただし再試行は可能）",
			query: &models.Query{
				Status: models.QueryStatusFailed,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.query.IsProcessable()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuery_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		query    *models.Query
		expected bool
	}{
		{
			name: "failed - 再試行可能",
			query: &models.Query{
				Status: models.QueryStatusFailed,
			},
			expected: true,
		},
		{
			name: "pending - 再試行不要",
			query: &models.Query{
				Status: models.QueryStatusPending,
			},
			expected: false,
		},
		{
			name: "processing - 再試行不可",
			query: &models.Query{
				Status: models.QueryStatusProcessing,
			},
			expected: false,
		},
		{
			name: "completed - 再試行不要",
			query: &models.Query{
				Status: models.QueryStatusCompleted,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.query.IsRetryable()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuery_GetProcessingDuration(t *testing.T) {
	tests := []struct {
		name              string
		processingTimeMs  int64
		expectedDuration  time.Duration
		expectedFormatted string
	}{
		{
			name:              "ミリ秒",
			processingTimeMs:  500,
			expectedDuration:  500 * time.Millisecond,
			expectedFormatted: "500ms",
		},
		{
			name:              "秒",
			processingTimeMs:  2500,
			expectedDuration:  2500 * time.Millisecond,
			expectedFormatted: "2.5s",
		},
		{
			name:              "分",
			processingTimeMs:  65000, // 65秒
			expectedDuration:  65000 * time.Millisecond,
			expectedFormatted: "1m5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &models.Query{
				ProcessingTimeMs: tt.processingTimeMs,
			}

			duration := query.GetProcessingDuration()
			assert.Equal(t, tt.expectedDuration, duration)

			formatted := query.GetFormattedProcessingTime()
			assert.Equal(t, tt.expectedFormatted, formatted)
		})
	}
}

func TestQuery_SanitizeQuestion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "通常のテキスト",
			input:    "AWS Bedrockについて教えてください",
			expected: "AWS Bedrockについて教えてください",
		},
		{
			name:     "HTMLタグの除去",
			input:    "<script>alert('xss')</script>AWS について",
			expected: "AWS について",
		},
		{
			name:     "SQLインジェクション対策",
			input:    "'; DROP TABLE users; --",
			expected: "'; DROP TABLE users; --", // クエリ文字列として保持（処理時に適切にエスケープ）
		},
		{
			name:     "改行文字の正規化",
			input:    "Line 1\r\nLine 2\rLine 3\nLine 4",
			expected: "Line 1\nLine 2\nLine 3\nLine 4",
		},
		{
			name:     "余分な空白の除去",
			input:    "   AWS   Bedrock   について   ",
			expected: "AWS Bedrock について",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &models.Query{Question: tt.input}
			query.SanitizeQuestion()
			assert.Equal(t, tt.expected, query.Question)
		})
	}
}

func TestQuery_EstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		question string
		expected int
	}{
		{
			name:     "短いテキスト",
			question: "Hello",
			expected: 1,
		},
		{
			name:     "中程度のテキスト",
			question: "AWS Bedrockについて詳しく教えてください",
			expected: 4, // 大まかな見積もり
		},
		{
			name:     "長いテキスト",
			question: generateLongString(200),
			expected: 50, // 4文字 ≈ 1トークンの概算
		},
		{
			name:     "空文字列",
			question: "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &models.Query{Question: tt.question}
			result := query.EstimateTokenCount()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuery_GetAgeInMinutes(t *testing.T) {
	now := time.Now()
	query := &models.Query{
		CreatedAt: now.Add(-5 * time.Minute),
	}

	age := query.GetAgeInMinutes()
	assert.True(t, age >= 5.0)
	assert.True(t, age < 6.0) // 若干の誤差を許容
}

func TestQuery_IsStale(t *testing.T) {
	tests := []struct {
		name      string
		createdAt time.Time
		status    models.QueryStatus
		expected  bool
	}{
		{
			name:      "新しいpending",
			createdAt: time.Now().Add(-1 * time.Minute),
			status:    models.QueryStatusPending,
			expected:  false,
		},
		{
			name:      "古いpending - stale",
			createdAt: time.Now().Add(-11 * time.Minute),
			status:    models.QueryStatusPending,
			expected:  true,
		},
		{
			name:      "古いprocessing - stale",
			createdAt: time.Now().Add(-16 * time.Minute),
			status:    models.QueryStatusProcessing,
			expected:  true,
		},
		{
			name:      "完了済み - not stale",
			createdAt: time.Now().Add(-30 * time.Minute),
			status:    models.QueryStatusCompleted,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &models.Query{
				CreatedAt: tt.createdAt,
				Status:    tt.status,
			}

			result := query.IsStale()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ヘルパー関数
func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}

// ベンチマークテスト
func BenchmarkQuery_Validate(b *testing.B) {
	query := &models.Query{
		ID:               "query123",
		SessionID:        "session456",
		Question:         "AWS Bedrockについて教えてください",
		Status:           models.QueryStatusPending,
		ProcessingTimeMs: 0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = query.Validate()
	}
}

func BenchmarkQuery_SanitizeQuestion(b *testing.B) {
	query := &models.Query{
		Question: "   AWS   Bedrock   について   教えて  ください   ",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.SanitizeQuestion()
	}
}

func BenchmarkQuery_EstimateTokenCount(b *testing.B) {
	query := &models.Query{
		Question: "AWS Bedrockについて詳しく教えてください。特にRAGシステムの実装方法について知りたいです。",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = query.EstimateTokenCount()
	}
}
