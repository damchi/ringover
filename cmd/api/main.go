package main

import (
	dbadapter "ringover/internal/adapter/db"
	"ringover/pkg/translator"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpadapter "ringover/internal/adapter/http"
	"ringover/internal/adapter/http/handlers"
	httpmiddleware "ringover/internal/adapter/http/middleware"
	appservice "ringover/internal/app/service"
	"ringover/internal/config"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	// Make zap available to packages that log through zap.L().
	zap.ReplaceGlobals(logger)
	defer func() {
		if err := logger.Sync(); err != nil {
			zap.L().Debug("failed to sync logger", zap.Error(err))
		}
	}()

	translator.InitTranslator(translator.Config{
		TranslationFolder:  "pkg/translator/translation",
		SupportedLanguages: []string{translator.LanguageFr, translator.LanguageEn},
	})

	cfg := config.LoadConfig()
	db, err := dbadapter.ConnectDB(cfg)
	if err != nil {
		logger.Fatal("failed to connect to mysql", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn("failed to close mysql connection", zap.Error(err))
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery(), httpmiddleware.GinZapMiddleware(logger))
	healthHandler := handlers.NewHealthHandler(db)

	taskRepository := dbadapter.NewTaskRepository(db)
	taskService := appservice.NewTaskService(taskRepository)
	taskHandler := handlers.NewTaskHandler(taskService)

	httpadapter.RegisterRoutes(r, healthHandler, taskHandler)

	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	logger.Info("starting server", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("could not start server", zap.Error(err))
	}
}
