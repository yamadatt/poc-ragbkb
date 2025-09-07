package services

import (
    "context"
    "fmt"
    "io"
    "log"
    "strconv"
    "strings"
    "time"

	"poc-ragbkb-backend/src/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
    dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
    "github.com/google/uuid"
)

// UploadServiceInterface はUploadServiceのインターフェース
type UploadServiceInterface interface {
    CreateUploadSession(ctx context.Context, document *models.Document) (*models.UploadSession, error)
    GetUploadSession(ctx context.Context, sessionID string) (*models.UploadSession, error)
    CompleteUpload(ctx context.Context, sessionID string) (*models.Document, error)
    CancelUploadSession(ctx context.Context, sessionID string) error
    CleanupExpiredSessions(ctx context.Context) error
    GeneratePresignedUploadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error)
    DeleteAllObjectsForDocument(ctx context.Context, documentID string) error
}

// UploadService はファイルアップロード管理サービス
type UploadService struct {
	dynamoDB             *dynamodb.Client
	s3Client             *s3.Client
	uploadTableName      string
	s3Bucket             string
	presignExpiration    time.Duration
	documentService      DocumentServiceInterface
	knowledgeBaseService KnowledgeBaseServiceInterface
}

// NewUploadService はUploadServiceの新しいインスタンスを作成
func NewUploadService(
	dynamoDB *dynamodb.Client,
	s3Client *s3.Client,
	uploadTableName string,
	s3Bucket string,
	presignExpiration time.Duration,
	documentService DocumentServiceInterface,
	knowledgeBaseService KnowledgeBaseServiceInterface,
) *UploadService {
	return &UploadService{
		dynamoDB:             dynamoDB,
		s3Client:             s3Client,
		uploadTableName:      uploadTableName,
		s3Bucket:             s3Bucket,
		presignExpiration:    presignExpiration,
		documentService:      documentService,
		knowledgeBaseService: knowledgeBaseService,
	}
}

// CreateUploadSession は新しいアップロードセッションを作成
func (s *UploadService) CreateUploadSession(ctx context.Context, document *models.Document) (*models.UploadSession, error) {
	if document == nil {
		return nil, models.NewValidationError("document", "文書情報は必須です")
	}

	now := time.Now()
	session := &models.UploadSession{
		ID:         uuid.New().String(),
		DocumentID: document.ID,
		FileName:   document.FileName,
		FileSize:   document.FileSize,
		FileType:   document.FileType,
		S3Bucket:   s.s3Bucket,
		Status:     models.UploadSessionStatusActive,
		ExpiresAt:  now.Add(s.presignExpiration),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// S3キーを生成
	session.S3Key = session.GenerateS3Key()

	// S3署名付きURLを生成
	uploadURL, err := s.GeneratePresignedUploadURL(ctx, session.S3Bucket, session.S3Key, s.presignExpiration)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("署名付きURL生成に失敗しました: %v", err))
	}
	session.UploadURL = uploadURL

	// DocumentエンティティにS3情報を設定
	document.S3Key = session.S3Key
	document.S3Bucket = session.S3Bucket

	// DynamoDBに保存
	item := session.ToDynamoDBItem()
	_, err = s.dynamoDB.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.uploadTableName),
		Item:      item,
	})

	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("アップロードセッションの作成に失敗しました: %v", err))
	}

	return session, nil
}

// GetUploadSession はセッションIDでアップロードセッションを取得
func (s *UploadService) GetUploadSession(ctx context.Context, sessionID string) (*models.UploadSession, error) {
	if sessionID == "" {
		return nil, models.NewValidationError("sessionId", "セッションIDは必須です")
	}

    result, err := s.dynamoDB.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String(s.uploadTableName),
        Key: map[string]dynamotypes.AttributeValue{
            "id": &dynamotypes.AttributeValueMemberS{Value: sessionID},
        },
    })

	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("アップロードセッションの取得に失敗しました: %v", err))
	}

	if result.Item == nil {
		return nil, models.NewNotFoundError("アップロードセッション")
	}

	session, err := s.dynamoDBItemToUploadSession(result.Item)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("アップロードセッションデータの変換に失敗しました: %v", err))
	}

	return session, nil
}

// CompleteUpload はアップロードを完了
func (s *UploadService) CompleteUpload(ctx context.Context, sessionID string) (*models.Document, error) {
	// アップロードセッションを取得
	session, err := s.GetUploadSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// セッションがアクティブで期限内かを確認
	if !session.IsActive() {
		if session.IsExpired() {
			return nil, models.NewValidationError("sessionId", "アップロードセッションの有効期限が切れています")
		}
		return nil, models.NewValidationError("sessionId", "アップロードセッションは既に使用済みまたは無効です")
	}

	// S3にファイルが存在するかを確認
	exists, err := s.checkS3ObjectExists(ctx, session.S3Bucket, session.S3Key)
	if err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("S3オブジェクトの存在確認に失敗しました: %v", err))
	}
	if !exists {
		return nil, models.NewValidationError("file", "ファイルがアップロードされていません")
	}

	// アップロードセッションを使用済みにマーク
	session.MarkAsUsed()
	if err := s.updateUploadSession(ctx, session); err != nil {
		return nil, models.NewInternalError(fmt.Sprintf("アップロードセッションの更新に失敗しました: %v", err))
	}

	// プレビューを生成して文書に設定
	preview, previewLines, err := s.generateDocumentPreview(ctx, session.S3Bucket, session.S3Key)
	if err != nil {
		// プレビュー生成に失敗してもアップロード自体は成功させる
		log.Printf("プレビュー生成に失敗: DocumentID=%s, Error=%v", session.DocumentID, err)
	} else {
		// 文書にプレビュー情報を設定
		if err := s.documentService.UpdateDocumentPreview(ctx, session.DocumentID, preview, previewLines); err != nil {
			log.Printf("プレビュー情報の保存に失敗: DocumentID=%s, Error=%v", session.DocumentID, err)
		}
	}

	// 文書のステータスを処理中に更新
	if err := s.documentService.UpdateDocumentStatus(ctx, session.DocumentID, models.DocumentStatusProcessing); err != nil {
		return nil, err
	}

	// Knowledge Baseに同期（バックグラウンド処理）
	go func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		kbDataSourceID := s.knowledgeBaseService.GetDataSourceID() // 実際のデータソースIDを取得
		
		if err := s.knowledgeBaseService.SyncDocumentToKnowledgeBase(syncCtx, session.DocumentID, session.S3Key); err != nil {
			// Knowledge Base同期に失敗した場合、専用のエラー状態に設定
			// 文書自体は正常にアップロードされているため、Knowledge Base検索はできないが閲覧は可能
			log.Printf("Knowledge Base同期に失敗: DocumentID=%s, Error=%v", session.DocumentID, err)
			s.documentService.MarkDocumentAsKBSyncError(syncCtx, session.DocumentID, fmt.Sprintf("Knowledge Base同期に失敗: %v", err))
			return
		}

		// 同期に成功した場合は文書を利用可能状態にマーク
		s.documentService.MarkDocumentAsReady(syncCtx, session.DocumentID, kbDataSourceID)
	}()

	// 更新された文書情報を取得
	document, err := s.documentService.GetDocument(ctx, session.DocumentID)
	if err != nil {
		return nil, err
	}

	return document, nil
}

// CancelUploadSession はアップロードセッションをキャンセル
func (s *UploadService) CancelUploadSession(ctx context.Context, sessionID string) error {
	session, err := s.GetUploadSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.MarkAsCanceled()
	return s.updateUploadSession(ctx, session)
}

// CleanupExpiredSessions は期限切れのセッションをクリーンアップ
func (s *UploadService) CleanupExpiredSessions(ctx context.Context) error {
	// 実装を簡略化: 実際にはScanでexpiredなセッションを検索してクリーンアップ
	// DynamoDBのTTL機能を使用することを推奨
	return nil
}

// GeneratePresignedUploadURL はS3署名付きアップロードURLを生成
func (s *UploadService) GeneratePresignedUploadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.s3Client)

	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String("application/octet-stream"),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", fmt.Errorf("署名付きURL生成に失敗: %w", err)
	}

	return request.URL, nil
}

// checkS3ObjectExists はS3オブジェクトの存在を確認
func (s *UploadService) checkS3ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// S3オブジェクトが存在しない場合（簡略化）
		return false, nil
	}

	return true, nil
}

// updateUploadSession はアップロードセッションを更新
func (s *UploadService) updateUploadSession(ctx context.Context, session *models.UploadSession) error {
    item := session.ToDynamoDBItem()
    _, err := s.dynamoDB.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String(s.uploadTableName),
        Item:      item,
    })

	if err != nil {
		return fmt.Errorf("アップロードセッションの更新に失敗: %w", err)
	}

	return nil
}

// generateDocumentPreview はS3からファイル内容を読み取ってプレビューを生成
func (s *UploadService) generateDocumentPreview(ctx context.Context, bucket, key string) (preview *string, previewLines int, err error) {
	// S3からファイル内容を読み取り
	result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("S3オブジェクト取得に失敗: %w", err)
	}
	defer result.Body.Close()

	// ファイル内容を読み取り（最大100KB）
	const maxReadSize = 100 * 1024 // 100KB
	content := make([]byte, maxReadSize)
	n, err := result.Body.Read(content)
	if err != nil && err != io.EOF {
		return nil, 0, fmt.Errorf("ファイル内容の読み取りに失敗: %w", err)
	}

	// 文字列に変換
	contentStr := string(content[:n])
	
	// 改行で分割して最初の30行を取得
	lines := strings.Split(contentStr, "\n")
	const maxPreviewLines = 30
	
	actualLines := len(lines)
	if actualLines > maxPreviewLines {
		actualLines = maxPreviewLines
	}
	
	previewContent := strings.Join(lines[:actualLines], "\n")
	
	// DynamoDBの項目サイズ制限（400KB）を考慮して切り詰め
	const maxPreviewSize = 50000 // 50KB（安全な範囲）
	if len(previewContent) > maxPreviewSize {
		previewContent = previewContent[:maxPreviewSize] + "\n...(以下省略)"
	}
	
	return &previewContent, actualLines, nil
}

// dynamoDBItemToUploadSession はDynamoDB項目をUploadSessionに変換
func (s *UploadService) dynamoDBItemToUploadSession(item map[string]dynamotypes.AttributeValue) (*models.UploadSession, error) {
	session := &models.UploadSession{}

    if id, ok := item["id"].(*dynamotypes.AttributeValueMemberS); ok {
        session.ID = id.Value
    }
    if documentID, ok := item["documentId"].(*dynamotypes.AttributeValueMemberS); ok {
        session.DocumentID = documentID.Value
    }
    if fileName, ok := item["fileName"].(*dynamotypes.AttributeValueMemberS); ok {
        session.FileName = fileName.Value
    }
    if fileSize, ok := item["fileSize"].(*dynamotypes.AttributeValueMemberN); ok {
        if size, err := strconv.ParseInt(fileSize.Value, 10, 64); err == nil {
            session.FileSize = size
        }
    }
    if fileType, ok := item["fileType"].(*dynamotypes.AttributeValueMemberS); ok {
        session.FileType = fileType.Value
    }
    if uploadURL, ok := item["uploadUrl"].(*dynamotypes.AttributeValueMemberS); ok {
        session.UploadURL = uploadURL.Value
    }
    if s3Key, ok := item["s3Key"].(*dynamotypes.AttributeValueMemberS); ok {
        session.S3Key = s3Key.Value
    }
    if s3Bucket, ok := item["s3Bucket"].(*dynamotypes.AttributeValueMemberS); ok {
        session.S3Bucket = s3Bucket.Value
    }
    if status, ok := item["status"].(*dynamotypes.AttributeValueMemberS); ok {
        session.Status = models.UploadSessionStatus(status.Value)
    }
    if expiresAt, ok := item["expiresAt"].(*dynamotypes.AttributeValueMemberS); ok {
        if t, err := time.Parse(time.RFC3339, expiresAt.Value); err == nil {
            session.ExpiresAt = t
        }
    }
    if createdAt, ok := item["createdAt"].(*dynamotypes.AttributeValueMemberS); ok {
        if t, err := time.Parse(time.RFC3339, createdAt.Value); err == nil {
            session.CreatedAt = t
        }
    }
    if updatedAt, ok := item["updatedAt"].(*dynamotypes.AttributeValueMemberS); ok {
        if t, err := time.Parse(time.RFC3339, updatedAt.Value); err == nil {
            session.UpdatedAt = t
        }
    }
    if usedAt, ok := item["usedAt"].(*dynamotypes.AttributeValueMemberS); ok {
        if t, err := time.Parse(time.RFC3339, usedAt.Value); err == nil {
            session.UsedAt = &t
        }
    }

	return session, nil
}

// DeleteAllObjectsForDocument は指定したDocumentID配下のS3オブジェクトを全削除
func (s *UploadService) DeleteAllObjectsForDocument(ctx context.Context, documentID string) error {
    if documentID == "" {
        return models.NewValidationError("documentId", "文書IDは必須です")
    }

    prefix := "documents/" + documentID + "/"

    // リストしてまとめて削除（1000件単位）
    var continuationToken *string
    for {
        listOut, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
            Bucket:            aws.String(s.s3Bucket),
            Prefix:            aws.String(prefix),
            ContinuationToken: continuationToken,
        })
        if err != nil {
            return fmt.Errorf("S3オブジェクト一覧取得に失敗: %w", err)
        }

        if len(listOut.Contents) > 0 {
            // DeleteObjects は1回で最大1000件
            objects := make([]s3types.ObjectIdentifier, 0, len(listOut.Contents))
            for _, obj := range listOut.Contents {
                objects = append(objects, s3types.ObjectIdentifier{Key: obj.Key})
            }

            _, err := s.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
                Bucket: aws.String(s.s3Bucket),
                Delete: &s3types.Delete{
                    Objects: objects,
                    Quiet:   aws.Bool(true),
                },
            })
            if err != nil {
                return fmt.Errorf("S3オブジェクト削除に失敗: %w", err)
            }
        }

        if aws.ToBool(listOut.IsTruncated) && listOut.NextContinuationToken != nil {
            continuationToken = listOut.NextContinuationToken
            continue
        }
        break
    }

    return nil
}
