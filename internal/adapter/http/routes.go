package http

import (
	"ringover/internal/adapter/http/handlers"
	"ringover/internal/adapter/http/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, healthHandler *handlers.HealthHandler, taskHandler *handlers.TaskHandler) {
	api := r.Group("/api")
	api.Use(middleware.LanguageMiddleware())
	{
		api.GET("/health", healthHandler.CheckHealth)
		api.GET("/health/report", healthHandler.CheckHealthReport)
		api.POST("/tasks", taskHandler.CreateTask)
		api.GET("/tasks", taskHandler.ListRootTasks)
		api.GET("/tasks/:id/subtasks", taskHandler.ListRootSubTasks)
	}
}
