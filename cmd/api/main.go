package main

import (
	"Rfad-TUI-Backend/core/pkg/corehandler"
	"Rfad-TUI-Backend/internal/app"
	"Rfad-TUI-Backend/internal/handler"
	"Rfad-TUI-Backend/internal/middleware"
	"Rfad-TUI-Backend/internal/repository"
	"Rfad-TUI-Backend/internal/router"
	"Rfad-TUI-Backend/internal/service"
	"log"
)

func main() {
	// 1. Собираем ядро со всеми новыми компонентами
	ms, err := app.NewBuilder("config.yml").
		WithLogger().
		WithCORS().
		WithTracing().
		WithMigrations().
		WithDatabases().
		WithRedis().
		WithS3Storage().
		Build()

	if err != nil {
		log.Fatalf("❌ Критическая ошибка при сборке: %v", err)
	}

	DefaultRepo := repository.NewDefaultRepository(
		ms.DBPool,
		ms.RedisClient,
		ms.S3Storage,
	)
	DefaultService := service.NewDefaultService(DefaultRepo, ms.Encryptor, ms.JWTManager, ms.AppCfg.App.IsProd)
	DefaultHandler := handler.NewDefaultHandler(DefaultService, ms.AppCfg.App.IsProd)

	CoreHandler := corehandler.NewDefaultHandler(DefaultService)

	credsFile := "credentials.json"
	WorkerRepo := repository.NewWorkerRepository(ms.DBPool, ms.S3Storage, credsFile, ms.RedisClient)
	WorkerSrv := service.NewWorkerService(WorkerRepo)

	ms.StartWorkers(WorkerSrv, credsFile)

	var midManager *middleware.Manager
	midManager = middleware.NewManager(
		ms.CoreCfg.Prometheus.Enabled,
		ms.CoreCfg.Jaeger.Enabled,
		ms.CoreCfg.Prometheus.Secure,
		ms.CoreCfg.Prometheus.User,
		ms.CoreCfg.Prometheus.Password,
		ms.JWTManager,
		ms.RedisClient,
		ms.AppCfg.App.IsProd,
	)

	router.SetupRoutes(
		ms.FiberApp,
		midManager,
		CoreHandler,
		DefaultHandler,
		ms.HealthCheckers,
		ms.CoreCfg.Prometheus.Enabled,
	)

	// 6. ЗАПУСК! (Эта функция блокирует поток и держит приложение живым)
	if err := ms.Run(); err != nil {
		log.Fatalf("❌ Ошибка при работе микросервиса: %v", err)
	}
}
