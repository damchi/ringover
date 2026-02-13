package handlers

import (
	"context"
	"os"
	"ringover/internal/adapter/http/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

const (
	StatusOk        = "ok"
	StatusDown      = "down"
	healthDBTimeout = 2 * time.Second
)

type HealthBasic struct {
	AppName           string `json:"app_name"`
	AppVersion        string `json:"app_version"`
	CurrentSystemTime string `json:"current_system_time"`
	Message           string `json:"message"`
}

type HealthServices struct {
	Mysql string `json:"mysql"`
}

type HealthAdvanced struct {
	AppName           string         `json:"app_name"`
	AppVersion        string         `json:"app_version"`
	CurrentSystemTime string         `json:"current_system_time"`
	Language          string         `json:"language"`
	Status            HealthServices `json:"status"`
}

type HealthHandler struct {
	db *sqlx.DB
}

func NewHealthHandler(db *sqlx.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) CheckHealth(c *gin.Context) {
	ctx := c.Request.Context()
	statusCode := 200
	message := StatusOk

	if !h.checkConnectionToDatabase(ctx) {
		statusCode = 500
		message = StatusDown
	}

	c.JSON(statusCode, HealthBasic{
		AppName:           os.Getenv("APP_NAME"),
		AppVersion:        getAppVersion(),
		CurrentSystemTime: time.Now().Format("2006-01-02 15:04:05"),
		Message:           message,
	})
}

func (h *HealthHandler) CheckHealthReport(c *gin.Context) {
	ctx := c.Request.Context()

	databaseStatus := StatusDown
	if h.checkConnectionToDatabase(ctx) {
		databaseStatus = StatusOk
	}

	c.JSON(200, HealthAdvanced{
		AppName:           os.Getenv("APP_NAME"),
		AppVersion:        getAppVersion(),
		CurrentSystemTime: time.Now().Format("2006-01-02 15:04:05"),
		Language:          middleware.GetLang(c),
		Status: HealthServices{
			Mysql: databaseStatus,
		},
	})
}

func (h *HealthHandler) checkConnectionToDatabase(ctx context.Context) bool {
	if h.db == nil {
		return false
	}
	// Avoid hanging health checks if the database stalls.
	timeoutCtx, cancel := context.WithTimeout(ctx, healthDBTimeout)
	defer cancel()
	return h.db.PingContext(timeoutCtx) == nil
}

func getAppVersion() string {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		return "dev"
	}
	return version
}
