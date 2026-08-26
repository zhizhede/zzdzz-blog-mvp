package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"zzdzz-blog/server/pkg/response"
)

type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Ping(c *gin.Context) {
	sqlDB, err := h.db.DB()
	dbOK := err == nil && sqlDB.Ping() == nil

	response.OK(c, gin.H{
		"server_time": time.Now().Format(time.RFC3339),
		"db_ok":       dbOK,
	})
}