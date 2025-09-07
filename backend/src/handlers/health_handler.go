package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthResponse はヘルスチェックレスポンス
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
}

// HealthHandler はヘルスチェックエンドポイントのハンドラー
type HealthHandler struct {
	startTime time.Time
	version   string
}

// NewHealthHandler はHealthHandlerの新しいインスタンスを作成
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		version:   version,
	}
}

// Health はヘルスチェックエンドポイント
// @Summary ヘルスチェック
// @Description アプリケーションの健全性を確認
// @Tags health
// @Produce json
// @Success 200 {object} SuccessResponse{data=HealthResponse}
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	uptime := time.Since(h.startTime)

	healthData := &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    uptime.String(),
	}

	respondWithSuccess(c, http.StatusOK, healthData)
}
