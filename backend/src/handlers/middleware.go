package handlers

import (
	"net/http"

	"poc-ragbkb-backend/src/models"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware はCORS設定を追加するミドルウェア
func CORSMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "false")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Amz-Date, X-Api-Key, X-Amz-Security-Token, X-Amz-User-Agent")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD")
		c.Header("Access-Control-Expose-Headers", "ETag, x-amz-server-side-encryption, x-amz-request-id, x-amz-id-2")
		c.Header("Access-Control-Max-Age", "3600")

		// プリフライトリクエストの処理
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})
}

// ErrorHandlerMiddleware はエラーハンドリングミドルウェア
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// エラーが発生した場合の処理
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			if apiError, ok := err.Err.(*models.APIError); ok {
				c.JSON(apiError.HTTPStatus(), &models.ErrorResponse{
					Error: apiError,
				})
				return
			}

			// その他のエラーの場合は500を返す
			c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
				Error: models.NewInternalError("予期しないエラーが発生しました"),
			})
		}
	}
}

// RequestLoggerMiddleware はリクエストログ出力ミドルウェア
func RequestLoggerMiddleware() gin.HandlerFunc {
	return gin.Logger()
}

// RecoveryMiddleware はパニック回復ミドルウェア
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
			Error: models.NewInternalError("サーバー内部エラーが発生しました"),
		})
	})
}
