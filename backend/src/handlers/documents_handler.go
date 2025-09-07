package handlers

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "poc-ragbkb-backend/src/models"
    "poc-ragbkb-backend/src/services"

    "github.com/gin-gonic/gin"
)

// DocumentsHandler は文書関連エンドポイントのハンドラー
type DocumentsHandler struct {
    documentService services.DocumentServiceInterface
    uploadService   services.UploadServiceInterface
    knowledgeBaseService services.KnowledgeBaseServiceInterface
}

// NewDocumentsHandler はDocumentsHandlerの新しいインスタンスを作成
func NewDocumentsHandler(
    documentService services.DocumentServiceInterface,
    uploadService services.UploadServiceInterface,
    knowledgeBaseService services.KnowledgeBaseServiceInterface,
) *DocumentsHandler {
    return &DocumentsHandler{
        documentService:     documentService,
        uploadService:       uploadService,
        knowledgeBaseService: knowledgeBaseService,
    }
}

// CreateDocument は文書アップロード開始エンドポイント
// @Summary 文書アップロード開始
// @Description 新しい文書のアップロードセッションを開始
// @Tags documents
// @Accept json
// @Produce json
// @Param request body models.CreateDocumentRequest true "文書作成リクエスト"
// @Success 201 {object} SuccessResponse{data=models.UploadSessionResponse}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /documents [post]
func (h *DocumentsHandler) CreateDocument(c *gin.Context) {
	var req models.CreateDocumentRequest
	if err := bindAndValidate(c, &req); err != nil {
		respondWithError(c, err)
		return
	}

	// 文書エンティティを作成
	document, err := h.documentService.CreateDocument(c.Request.Context(), &req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	// アップロードセッションを作成
	session, err := h.uploadService.CreateUploadSession(c.Request.Context(), document)
	if err != nil {
		respondWithError(c, err)
		return
	}

	respondWithSuccess(c, http.StatusCreated, session.ToResponse())
}

// GetDocument は文書詳細取得エンドポイント
// @Summary 文書詳細取得
// @Description 文書IDで文書詳細を取得
// @Tags documents
// @Produce json
// @Param id path string true "文書ID"
// @Success 200 {object} SuccessResponse{data=models.DocumentResponse}
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /documents/{id} [get]
func (h *DocumentsHandler) GetDocument(c *gin.Context) {
	id := c.Param("documentId")
	if err := validateUUID(id); err != nil {
		respondWithError(c, err)
		return
	}

	document, err := h.documentService.GetDocument(c.Request.Context(), id)
	if err != nil {
		respondWithError(c, err)
		return
	}

	respondWithSuccess(c, http.StatusOK, document.ToResponse())
}

// ListDocuments は文書一覧取得エンドポイント
// @Summary 文書一覧取得
// @Description 登録されている文書の一覧を取得
// @Tags documents
// @Produce json
// @Param offset query int false "オフセット" default(0)
// @Param limit query int false "取得件数" default(20)
// @Success 200 {object} SuccessResponse{data=models.DocumentListResponse}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /documents [get]
func (h *DocumentsHandler) ListDocuments(c *gin.Context) {
	offset := getQueryParamInt(c, "offset", 0)
	limit := getQueryParamInt(c, "limit", 20)

	if offset < 0 {
		respondWithError(c, models.NewValidationError("offset", "オフセットは0以上である必要があります"))
		return
	}
	if limit <= 0 || limit > 100 {
		respondWithError(c, models.NewValidationError("limit", "取得件数は1以上100以下である必要があります"))
		return
	}

	documents, err := h.documentService.ListDocuments(c.Request.Context(), offset, limit)
	if err != nil {
		respondWithError(c, err)
		return
	}

	respondWithSuccess(c, http.StatusOK, documents)
}

// CompleteUpload はアップロード完了エンドポイント
// @Summary アップロード完了
// @Description 文書のアップロードを完了し、Knowledge Baseへの同期を開始
// @Tags documents
// @Produce json
// @Param id path string true "文書ID"
// @Success 200 {object} SuccessResponse{data=models.CompleteUploadResponse}
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /documents/{id}/complete-upload [post]
func (h *DocumentsHandler) CompleteUpload(c *gin.Context) {
    // 新/旧両対応: sessionId 優先、なければ documentId を使用
    sessionID := c.Param("sessionId")
    if sessionID == "" {
        sessionID = c.Param("documentId")
    }

    if err := validateUUID(sessionID); err != nil {
        respondWithError(c, err)
        return
    }

    // アップロードセッションを完了（ここではパスの値をセッションIDとして使用）
    document, err := h.uploadService.CompleteUpload(c.Request.Context(), sessionID)
	if err != nil {
		respondWithError(c, err)
		return
	}

	response := &models.CompleteUploadResponse{
		ID:       document.ID,
		FileName: document.FileName,
		FileSize: document.FileSize,
		FileType: document.FileType,
		Status:   document.Status,
	}

	respondWithSuccess(c, http.StatusOK, response)
}

// DeleteDocument は文書削除エンドポイント
// @Summary 文書削除
// @Description 文書を削除（S3ファイルとKnowledge Baseからも削除）
// @Tags documents
// @Param id path string true "文書ID"
// @Success 204 "削除成功"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /documents/{id} [delete]
func (h *DocumentsHandler) DeleteDocument(c *gin.Context) {
    id := c.Param("documentId")
    if err := validateUUID(id); err != nil {
        respondWithError(c, err)
        return
    }

    // 文書を取得して存在確認
    _, err := h.documentService.GetDocument(c.Request.Context(), id)
    if err != nil {
        respondWithError(c, err)
        return
    }

    // S3から該当文書のオブジェクトを削除（ベストエフォート）
    if err := h.uploadService.DeleteAllObjectsForDocument(c.Request.Context(), id); err != nil {
        respondWithError(c, err)
        return
    }

    // Knowledge Baseのインデックス更新（S3削除後の同期）。設定があれば非同期で実行
    if dsID := h.knowledgeBaseService.GetDataSourceID(); dsID != "" {
        go func() {
            ctx := context.Background()
            jobID, err := h.knowledgeBaseService.StartIngestionJob(ctx, dsID)
            if err != nil {
                // ログにエラーを記録（削除処理自体は成功として扱う）
                fmt.Printf("Knowledge Base ingestion job start failed for document deletion %s: %v\n", id, err)
                return
            }
            fmt.Printf("Knowledge Base ingestion job started for document deletion %s, job ID: %s\n", id, jobID)
            
            // ジョブの完了を待機してログ出力（オプション：デバッグ時に有効）
            // NOTE: 本格運用時はジョブ監視を別途実装することを推奨
            time.Sleep(2 * time.Second) // 短時間待機してからステータス確認
            if status, failureReasons, statusErr := h.knowledgeBaseService.GetIngestionJobDetails(ctx, jobID); statusErr == nil {
                if len(failureReasons) > 0 {
                    fmt.Printf("Knowledge Base ingestion job status for document deletion %s: %s (errors: %v)\n", id, status, failureReasons)
                } else {
                    fmt.Printf("Knowledge Base ingestion job status for document deletion %s: %s\n", id, status)
                }
            }
        }()
    }

    // DynamoDBから文書レコードを削除
    if err := h.documentService.DeleteDocument(c.Request.Context(), id); err != nil {
        respondWithError(c, err)
        return
    }

	// 204 No Contentを返す
	c.Status(http.StatusNoContent)
}
