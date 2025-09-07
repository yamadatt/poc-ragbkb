package handlers

import (
	"net/http"

	"poc-ragbkb-backend/src/models"
	"poc-ragbkb-backend/src/services"

	"github.com/gin-gonic/gin"
)

// QueriesHandler はクエリ関連エンドポイントのハンドラー
type QueriesHandler struct {
	queryService         services.QueryServiceInterface
	responseService      services.ResponseServiceInterface
	knowledgeBaseService services.KnowledgeBaseServiceInterface
}

// NewQueriesHandler はQueriesHandlerの新しいインスタンスを作成
func NewQueriesHandler(
	queryService services.QueryServiceInterface,
	responseService services.ResponseServiceInterface,
	knowledgeBaseService services.KnowledgeBaseServiceInterface,
) *QueriesHandler {
	return &QueriesHandler{
		queryService:         queryService,
		responseService:      responseService,
		knowledgeBaseService: knowledgeBaseService,
	}
}

// CreateQuery はRAGクエリ実行エンドポイント
// @Summary RAGクエリ実行
// @Description Knowledge BaseにRAGクエリを送信して回答を生成
// @Tags queries
// @Accept json
// @Produce json
// @Param request body models.CreateQueryRequest true "クエリリクエスト"
// @Success 201 {object} SuccessResponse{data=models.QueryWithCompleteResponse}
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /queries [post]
func (h *QueriesHandler) CreateQuery(c *gin.Context) {
	var req models.CreateQueryRequest
	if err := bindAndValidate(c, &req); err != nil {
		respondWithError(c, err)
		return
	}

	// クエリエンティティを作成
	query, err := h.queryService.CreateQuery(c.Request.Context(), &req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	// クエリを処理中状態に更新
	if err := h.queryService.UpdateQueryStatus(c.Request.Context(), query.ID, models.QueryStatusProcessing); err != nil {
		respondWithError(c, err)
		return
	}

	// Knowledge BaseにRAGクエリを実行
	ragResponse, err := h.knowledgeBaseService.QueryKnowledgeBase(c.Request.Context(), req.Question, req.SessionID)
	if err != nil {
		// クエリを失敗状態に更新
		h.queryService.MarkQueryAsFailed(c.Request.Context(), query.ID, err.Error(), 0)

		// Knowledge Baseに関連情報がない場合は404を返す
		if _, ok := err.(*models.APIError); ok {
			if err.(*models.APIError).Code == http.StatusNotFound {
				respondWithError(c, models.NewNotFoundError("関連する情報"))
				return
			}
		}

		respondWithError(c, err)
		return
	}

	// レスポンスを保存
	response, err := h.responseService.CreateResponse(
		c.Request.Context(),
		query.ID,
		ragResponse.Answer,
		ragResponse.Sources,
		ragResponse.ProcessingTimeMs,
		ragResponse.ModelUsed,
		ragResponse.TokensUsed,
	)
	if err != nil {
		// レスポンス保存に失敗してもクエリは成功として処理
		h.queryService.MarkQueryAsCompleted(c.Request.Context(), query.ID, ragResponse.ProcessingTimeMs)
		respondWithError(c, err)
		return
	}

	// クエリを完了状態に更新
	if err := h.queryService.MarkQueryAsCompleted(c.Request.Context(), query.ID, ragResponse.ProcessingTimeMs); err != nil {
		respondWithError(c, err)
		return
	}

	// 更新されたクエリを取得
	updatedQuery, err := h.queryService.GetQuery(c.Request.Context(), query.ID)
	if err != nil {
		respondWithError(c, err)
		return
	}

	// 統合レスポンスを作成
	completeResponse := &models.QueryWithCompleteResponse{
		Query:    updatedQuery.ToResponse(),
		Response: response.ToResponse(),
	}

	respondWithSuccess(c, http.StatusCreated, completeResponse)
}

// GetQueryHistory はクエリ履歴取得エンドポイント
// @Summary クエリ履歴取得
// @Description セッションIDでクエリ履歴を取得
// @Tags queries
// @Produce json
// @Param sessionId path string true "セッションID"
// @Param offset query int false "オフセット" default(0)
// @Param limit query int false "取得件数" default(10)
// @Success 200 {object} SuccessResponse{data=models.QueryHistoryResponse}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /queries/{sessionId}/history [get]
func (h *QueriesHandler) GetQueryHistory(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if err := validateSessionID(sessionID); err != nil {
		respondWithError(c, err)
		return
	}

	offset := getQueryParamInt(c, "offset", 0)
	limit := getQueryParamInt(c, "limit", 10)

	if offset < 0 {
		respondWithError(c, models.NewValidationError("offset", "オフセットは0以上である必要があります"))
		return
	}
	if limit <= 0 || limit > 50 {
		respondWithError(c, models.NewValidationError("limit", "取得件数は1以上50以下である必要があります"))
		return
	}

	history, err := h.queryService.GetQueryHistory(c.Request.Context(), sessionID, offset, limit)
	if err != nil {
		respondWithError(c, err)
		return
	}

	respondWithSuccess(c, http.StatusOK, history)
}
