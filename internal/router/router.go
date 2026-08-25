package router

import (
	"Rfad-TUI-Backend/core/pkg/corehandler"
	"Rfad-TUI-Backend/core/pkg/corerouter"
	"Rfad-TUI-Backend/core/pkg/health"
	"Rfad-TUI-Backend/internal/handler"
	"Rfad-TUI-Backend/internal/middleware"
	"time"

	"github.com/gofiber/fiber/v3"
)

// SetupRoutes настраивает все пути приложения
func SetupRoutes(
	app *fiber.App,
	midManager *middleware.Manager,
	coreHandler *corehandler.DefaultHandler,
	defaultHandler *handler.DefaultHandler,
	healthCheckers []health.Checker,
	promEnabled bool,
) {
	app.Use(midManager.Tracing())

	corerouter.RegisterSystemRoutes(app, healthCheckers, promEnabled, midManager.MetricsAuth())

	api := app.Group("/api/v1", midManager.Metrics())
	{
		api.Get("/updates/latest",
			midManager.RateLimit("get_latest_update", 10, 1*time.Minute),
			defaultHandler.GetLatestUpdate,
		)
		api.Get("/community/shaders",
			midManager.RateLimit("Community_Shaders", 5, 30*time.Second),
			defaultHandler.GetCommunityShaders,
		)
		api.Post("/community/shaders",
			midManager.RateLimit("Community_Shaders_Admin", 10, 5*time.Minute),
			midManager.AdminAuth(),
			defaultHandler.UploadCommunityShader,
		)
		api.Post("/save/config",
			midManager.RateLimit("Community_Shaders_Admin", 10, 5*time.Minute),
			midManager.AdminAuth(),
			defaultHandler.UploadConfig,
		)
		api.Post("/disable/config",
			midManager.RateLimit("Community_Shaders_Admin", 20, 5*time.Minute),
			midManager.AdminAuth(),
			defaultHandler.DisableConfig,
		)
		api.Post("/enable/config",
			midManager.RateLimit("Community_Shaders_Admin", 20, 5*time.Minute),
			midManager.AdminAuth(),
			defaultHandler.EnableConfig,
		)
		api.Get("/config/admin",
			midManager.RateLimit("Community_Shaders_Admin", 20, 20*time.Minute),
			midManager.AdminAuth(),
			defaultHandler.GetConfigs,
		)
		api.Get("/config",
			midManager.RateLimit("Community_Shaders_Admin", 20, 20*time.Minute),
			defaultHandler.GetConfigsAdmin,
		)
	}
}
