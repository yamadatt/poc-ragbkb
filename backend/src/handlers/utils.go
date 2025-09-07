package handlers

import (
	"net/http"
	"strconv"

	"poc-ragbkb-backend/src/models"

	"github.com/gin-gonic/gin"
)

// SuccessResponse は成功レスポンスの共通構造
type SuccessResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// respondWithSuccess は成功レスポンスを返す
func respondWithSuccess(c *gin.Context, statusCode int, data interface{}, message ...string) {
	response := &SuccessResponse{
		Data: data,
	}

	if len(message) > 0 {
		response.Message = message[0]
	}

	c.JSON(statusCode, response)
}

// respondWithError はエラーレスポンスを返す
func respondWithError(c *gin.Context, err error) {
	if apiError, ok := err.(*models.APIError); ok {
		c.JSON(apiError.HTTPStatus(), &models.ErrorResponse{
			Error: apiError,
		})
		return
	}

	// その他のエラーは500として処理
	c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
		Error: models.NewInternalError("予期しないエラーが発生しました"),
	})
}

// getQueryParamInt はクエリパラメータを整数として取得
func getQueryParamInt(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// validateSessionID はセッションID形式の基本的なバリデーション
func validateSessionID(id string) error {
	if id == "" {
		return models.NewValidationError("sessionId", "セッションIDは必須です")
	}
	// カスタムセッション形式 (session_xxxxx_xxxxx) またはUUID形式を受け入れる
	if len(id) < 10 || len(id) > 50 {
		return models.NewValidationError("sessionId", "無効なセッションIDです")
	}
	return nil
}

// validateUUID はUUID形式の基本的なバリデーション
func validateUUID(id string) error {
	if id == "" {
		return models.NewValidationError("id", "IDは必須です")
	}
	if len(id) != 36 {
		return models.NewValidationError("id", "無効なID形式です")
	}
	return nil
}

// bindAndValidate はリクエストボディをバインドしてバリデーション
func bindAndValidate(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return models.NewValidationError("request", "リクエスト形式が不正です: "+err.Error())
	}

	// カスタムバリデーションがある場合は実行
	if validator, ok := obj.(interface{ Validate() error }); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// handleMethodNotAllowed は許可されていないHTTPメソッドのハンドラー
func handleMethodNotAllowed() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, &models.ErrorResponse{
			Error: &models.APIError{
				Code:    http.StatusMethodNotAllowed,
				Message: "このHTTPメソッドは許可されていません",
				Type:    "method_not_allowed",
			},
		})
	}
}

// handleNotFound は404エラーのハンドラー
func handleNotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, &models.ErrorResponse{
			Error: &models.APIError{
				Code:    http.StatusNotFound,
				Message: "リクエストされたリソースが見つかりません",
				Type:    "not_found",
			},
		})
	}
}
