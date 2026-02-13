package http

import (
	"ringover/internal/adapter/http/handlers"
	"ringover/internal/adapter/http/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, healthHandler *handlers.HealthHandler) {
	api := r.Group("/api")
	api.Use(middleware.LanguageMiddleware())
	{
		api.GET("/health", healthHandler.CheckHealth)
		api.GET("/health/report", healthHandler.CheckHealthReport)
	}

}
