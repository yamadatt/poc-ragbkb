package unit

import (
	"testing"
	"time"

	"poc-ragbkb-backend/src/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocument_Validation(t *testing.T) {
	tests := []struct {
		name     string
		document *models.Document
		wantErr  bool
		errMsg   string
	}{
		{
			name: "有効な文書",
			document: &models.Document{
				ID:        "doc123",
				FileName:  "test.txt",
				FileSize:  1024,
				FileType:  "txt",
				Status:    models.DocumentStatusReady,
				S3Key:     "documents/doc123/test.txt",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "IDが空",
			document: &models.Document{
				ID:        "",
				FileName:  "test.txt",
				FileSize:  1024,
				FileType:  "txt",
				Status:    models.DocumentStatusReady,
				S3Key:     "documents/doc123/test.txt",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "document ID is required",
		},
		{
			name: "ファイル名が空",
			document: &models.Document{
				ID:        "doc123",
				FileName:  "",
				FileSize:  1024,
				FileType:  "txt",
				Status:    models.DocumentStatusReady,
				S3Key:     "documents/doc123/test.txt",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "file name is required",
		},
		{
			name: "ファイルサイズが0",
			document: &models.Document{
				ID:        "doc123",
				FileName:  "test.txt",
				FileSize:  0,
				FileType:  "txt",
				Status:    models.DocumentStatusReady,
				S3Key:     "documents/doc123/test.txt",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "file size must be greater than 0",
		},
		{
        name: "ファイルサイズが制限超過",
			document: &models.Document{
				ID:        "doc123",
				FileName:  "test.txt",
                FileSize:  50*1024*1024 + 1, // 50MB + 1 byte
				FileType:  "txt",
				Status:    models.DocumentStatusReady,
				S3Key:     "documents/doc123/test.txt",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "file size exceeds maximum limit",
		},
		{
			name: "無効なファイルタイプ",
			document: &models.Document{
				ID:        "doc123",
				FileName:  "test.pdf",
				FileSize:  1024,
				FileType:  "pdf",
				Status:    models.DocumentStatusReady,
				S3Key:     "documents/doc123/test.pdf",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "unsupported file type",
		},
		{
			name: "無効なステータス",
			document: &models.Document{
				ID:        "doc123",
				FileName:  "test.txt",
				FileSize:  1024,
				FileType:  "txt",
				Status:    "invalid_status",
				S3Key:     "documents/doc123/test.txt",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid document status",
		},
		{
			name: "S3Keyが空",
			document: &models.Document{
				ID:        "doc123",
				FileName:  "test.txt",
				FileSize:  1024,
				FileType:  "txt",
				Status:    models.DocumentStatusReady,
				S3Key:     "",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "S3 key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.document.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDocumentStatus_Validation(t *testing.T) {
	validStatuses := []models.DocumentStatus{
		models.DocumentStatusUploading,
		models.DocumentStatusProcessing,
		models.DocumentStatusReady,
		models.DocumentStatusError,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			doc := &models.Document{
				ID:        "doc123",
				FileName:  "test.txt",
				FileSize:  1024,
				FileType:  "txt",
				Status:    status,
				S3Key:     "documents/doc123/test.txt",
				S3Bucket:  "test-bucket",
				UploadedAt: time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			err := doc.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestDocument_IsReady(t *testing.T) {
	tests := []struct {
		name     string
		document *models.Document
		expected bool
	}{
		{
			name: "ready状態",
			document: &models.Document{
				Status: models.DocumentStatusReady,
			},
			expected: true,
		},
		{
			name: "uploading状態",
			document: &models.Document{
				Status: models.DocumentStatusUploading,
			},
			expected: false,
		},
		{
			name: "processing状態",
			document: &models.Document{
				Status: models.DocumentStatusProcessing,
			},
			expected: false,
		},
		{
			name: "error状態",
			document: &models.Document{
				Status: models.DocumentStatusError,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.document.IsReady()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDocument_IsError(t *testing.T) {
	tests := []struct {
		name     string
		document *models.Document
		expected bool
	}{
		{
			name: "error状態",
			document: &models.Document{
				Status: models.DocumentStatusError,
			},
			expected: true,
		},
		{
			name: "ready状態",
			document: &models.Document{
				Status: models.DocumentStatusReady,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.document.IsError()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDocument_CanBeDeleted(t *testing.T) {
	tests := []struct {
		name     string
		document *models.Document
		expected bool
	}{
		{
			name: "ready状態 - 削除可能",
			document: &models.Document{
				Status: models.DocumentStatusReady,
			},
			expected: true,
		},
		{
			name: "error状態 - 削除可能",
			document: &models.Document{
				Status: models.DocumentStatusError,
			},
			expected: true,
		},
		{
			name: "uploading状態 - 削除不可",
			document: &models.Document{
				Status: models.DocumentStatusUploading,
			},
			expected: false,
		},
		{
			name: "processing状態 - 削除不可",
			document: &models.Document{
				Status: models.DocumentStatusProcessing,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.document.CanBeDeleted()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDocument_UpdateStatus(t *testing.T) {
	doc := &models.Document{
		ID:        "doc123",
		FileName:  "test.txt",
		FileSize:  1024,
		FileType:  "txt",
		Status:    models.DocumentStatusUploading,
		S3Key:     "documents/doc123/test.txt",
		S3Bucket:  "test-bucket",
		UploadedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// ステータス更新
	oldUpdated := doc.UpdatedAt
	time.Sleep(time.Millisecond) // 時間差を確保

	doc.UpdateStatus(models.DocumentStatusReady)

	assert.Equal(t, models.DocumentStatusReady, doc.Status)
	assert.True(t, doc.UpdatedAt.After(oldUpdated))
	assert.True(t, doc.ProcessedAt != nil)
}

func TestDocument_SetError(t *testing.T) {
	doc := &models.Document{
		ID:        "doc123",
		FileName:  "test.txt",
		FileSize:  1024,
		FileType:  "txt",
		Status:    models.DocumentStatusProcessing,
		S3Key:     "documents/doc123/test.txt",
		S3Bucket:  "test-bucket",
		UploadedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	errorMessage := "処理中にエラーが発生しました"
	oldUpdated := doc.UpdatedAt
	time.Sleep(time.Millisecond) // 時間差を確保

	doc.SetError(errorMessage)

	assert.Equal(t, models.DocumentStatusError, doc.Status)
	assert.True(t, doc.UpdatedAt.After(oldUpdated))
	assert.Equal(t, errorMessage, *doc.ErrorMessage)
}

func TestDocument_GetFileSizeFormatted(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		expected string
	}{
		{
			name:     "バイト",
			fileSize: 500,
			expected: "500 B",
		},
		{
			name:     "キロバイト",
			fileSize: 1024,
			expected: "1.0 KB",
		},
		{
			name:     "メガバイト",
			fileSize: 1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "10MB",
			fileSize: 10 * 1024 * 1024,
			expected: "10.0 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &models.Document{FileSize: tt.fileSize}
			result := doc.GetFileSizeFormatted()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ベンチマークテスト
func BenchmarkDocument_Validate(b *testing.B) {
	doc := &models.Document{
		ID:        "doc123",
		FileName:  "test.txt",
		FileSize:  1024,
		FileType:  "txt",
		Status:    models.DocumentStatusReady,
		S3Key:     "documents/doc123/test.txt",
		S3Bucket:  "test-bucket",
		UploadedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = doc.Validate()
	}
}
