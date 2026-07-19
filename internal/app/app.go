package app

import (
	coreconfig "Rfad-TUI-Backend/core/config"
	"context"
	"fmt"

	"Rfad-TUI-Backend/core/pkg/coretelemetry"

	"Rfad-TUI-Backend/core/pkg/health"

	"Rfad-TUI-Backend/migrations"

	"Rfad-TUI-Backend/core/pkg/logger"

	"Rfad-TUI-Backend/core/pkg/migrate"
	"Rfad-TUI-Backend/core/pkg/postgres"

	coreredis "Rfad-TUI-Backend/core/pkg/redis"

	"Rfad-TUI-Backend/core/pkg/storage"

	"Rfad-TUI-Backend/core/pkg/security"
	"Rfad-TUI-Backend/internal/middleware"
	"Rfad-TUI-Backend/internal/service"
	"Rfad-TUI-Backend/internal/worker"

	"Rfad-TUI-Backend/internal/telemetry"

	appconfig "Rfad-TUI-Backend/pkg/config"

	"Rfad-TUI-Backend/core/pkg/response"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Microservice struct {
	FiberApp *fiber.App

	CoreCfg *coreconfig.CoreConfig
	AppCfg  *appconfig.AppConfig

	Logger *slog.Logger

	S3Storage *storage.S3Client

	DBPool *pgxpool.Pool

	RedisClient     *coreredis.Wrapper
	Encryptor       *security.Encryptor
	MigrationsReady *atomic.Bool

	JWTManager *security.JWTManager

	TracerShutdown func(context.Context) error

	HealthCheckers []health.Checker

	WorkerShutdown func()
}

type Builder struct {
	app *Microservice
	err error
}

func NewBuilder(configPath string) *Builder {
	coreCfg, err := coreconfig.LoadCoreConfig(configPath)
	if err != nil {
		return &Builder{err: err}
	}

	appCfg, err := appconfig.LoadAppConfig(configPath)
	if err != nil {
		return &Builder{err: err}
	}

	isPrometheusEnabled := coreCfg.Prometheus.Enabled
	promServiceName := coreCfg.Prometheus.ServiceName

	coretelemetry.InitMetrics(promServiceName, isPrometheusEnabled)
	telemetry.InitAppMetrics(promServiceName, isPrometheusEnabled)

	fiberApp := fiber.New(fiber.Config{
		ErrorHandler: response.GlobalErrorHandler,
		BodyLimit:    500 * 1024 * 1024,
	})

	return &Builder{
		app: &Microservice{
			FiberApp:        fiberApp,
			CoreCfg:         coreCfg,
			AppCfg:          appCfg,
			MigrationsReady: &atomic.Bool{},
			HealthCheckers:  make([]health.Checker, 0),
		},
	}
}

func (b *Builder) WithLogger() *Builder {
	if b.err != nil {
		return b
	}

	logger.Init(b.app.AppCfg.App.IsProd)

	b.app.Logger = slog.Default()

	slog.Info("Логгер инициализирован", slog.Bool("is_prod", b.app.AppCfg.App.IsProd))

	return b
}

func (b *Builder) WithS3Storage() *Builder {
	if b.err != nil {
		return b
	}

	cfg := b.app.CoreCfg.S3
	if !cfg.Enabled {
		return b
	}

	s3Client, err := storage.NewS3Client(
		context.Background(),
		cfg.Endpoint,
		cfg.Region,
		cfg.AccessKey,
		cfg.SecretKey,
		cfg.Bucket,
	)
	if err != nil {
		if !b.app.AppCfg.App.IsProd {
			b.err = err
			return b
		}
		slog.Warn("Ошибка инициализации S3 (Prod)", slog.Any("error", err))
	}

	b.app.S3Storage = s3Client
	slog.Info("S3 Хранилище успешно подключено")

	if s3Client != nil {
		s3Checker := health.NewS3Checker(s3Client)
		b.app.HealthCheckers = append(b.app.HealthCheckers, s3Checker)
	}

	return b
}

//core:jwt
func (b *Builder) WithJWT() *Builder {
	if b.err != nil {
		return b
	}

	cfg := b.app.CoreCfg.JWT
	if !cfg.Enabled {
		return b
	}

	jwtManager, err := security.NewJWTManager(cfg.PrivateKeyPath, cfg.PublicKeyPath, cfg.AccessTTL)
	if err != nil {
		if !b.app.AppCfg.App.IsProd {
			b.err = err
			return b
		}
		slog.Warn("Ошибка инициализации JWT Manager (Prod)", slog.Any("error", err))
	}

	b.app.JWTManager = jwtManager
	if jwtManager != nil {
		slog.Info("JWT Manager успешно инициализирован")
	}

	return b
}

//core:jwt:end

func (b *Builder) WithMigrations() *Builder {
	if b.err != nil {
		return b
	}

	if err := migrate.Run(b.app.CoreCfg, migrations.FS); err != nil {
		if !b.app.AppCfg.App.IsProd {
			b.err = fmt.Errorf("критическая ошибка миграций (Dev): %w", err)
			return b
		}

		slog.Warn("Критическая ошибка миграций (Prod). Трафик заблокирован.", slog.Any("error", err))
		b.app.MigrationsReady.Store(false)
	} else {
		b.app.MigrationsReady.Store(true)
	}

	migChecker := health.NewMigrationChecker(b.app.MigrationsReady)
	b.app.HealthCheckers = append(b.app.HealthCheckers, migChecker)

	return b
}

func (b *Builder) WithDatabases() *Builder {
	if b.err != nil {
		return b
	}

	var mainDBName string
	if len(b.app.CoreCfg.Postgres.Names) > 0 {
		mainDBName = b.app.CoreCfg.Postgres.Names[0]
	} else {
		mainDBName = b.app.CoreCfg.Postgres.Name
	}

	if mainDBName == "" {
		b.err = fmt.Errorf("критическая ошибка: не указано имя базы данных (ни db_name, ни db_names)")
		return b
	}

	pool, err := postgres.NewPool(context.Background(), b.app.CoreCfg.Postgres, mainDBName)
	if err != nil {
		b.err = err
		return b
	}

	b.app.DBPool = pool

	pgChecker := health.NewPostgresChecker(b.app.DBPool)
	b.app.HealthCheckers = append(b.app.HealthCheckers, pgChecker)

	return b
}

func (b *Builder) WithRedis() *Builder {
	if b.err != nil {
		return b
	}

	ctx := context.Background()
	client, err := coreredis.NewRedisManager(ctx, b.app.CoreCfg.Redis)
	if err != nil {
		b.err = fmt.Errorf("ошибка инициализации пула Redis: %w", err)
		return b
	}

	b.app.RedisClient = client

	redisChecker := health.NewRedisChecker(b.app.RedisClient)
	b.app.HealthCheckers = append(b.app.HealthCheckers, redisChecker)

	return b
}

func (b *Builder) WithEncryptor() *Builder {
	if b.err != nil {
		return b
	}

	enc, err := security.NewEncryptor(b.app.CoreCfg.Security.MasterKey)
	if err != nil {
		b.err = fmt.Errorf("ошибка инициализации шифровальщика: %w", err)
		return b
	}

	b.app.Encryptor = enc
	return b
}

func (b *Builder) WithCORS() *Builder {
	if b.err != nil {
		return b
	}

	cfg := b.app.CoreCfg.CORS

	if !cfg.Enabled {
		return b
	}

	corsConfigRedis := cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			if origin == "" || origin == "http://localhost:3000" {
				return true
			}
			isAllowed, err := b.app.RedisClient.SIsMember(context.Background(), "cors:allowed_origins", origin).Result()

			if err != nil {
				slog.Error("Ошибка проверки CORS в Redis", "error", err, "origin", origin)
				return false
			}

			return isAllowed
		},

		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		AllowCredentials: cfg.AllowCredentials,
	}

	b.app.FiberApp.Use(cors.New(corsConfigRedis))

	slog.Info("CORS middleware успешно активирован (Redis режим)")

	return b
}

func (b *Builder) WithTracing() *Builder {
	if b.err != nil {
		return b
	}

	cfg := b.app.CoreCfg.Jaeger

	shutdownFn, err := telemetry.InitJaeger(cfg.URL, cfg.ServiceName, cfg.Enabled)
	if err != nil {
		b.err = fmt.Errorf("ошибка инициализации Jaeger: %w", err)
		return b
	}

	if cfg.Enabled {
		slog.Info("Трейсинг Jaeger успешно активирован")
	}

	b.app.TracerShutdown = shutdownFn
	return b
}

func (m *Microservice) StartWorkers(workerSrv *service.WorkerService, credsFile string) {
	if _, err := os.Stat(credsFile); os.IsNotExist(err) {
		slog.Warn("Воркер обновлений отключен: файл credentials.json не найден")
		return
	}

	if m.S3Storage == nil {
		slog.Warn("Воркер обновлений отключен: хранилище S3 не инициализировано")
		return
	}

	// Инициализация и запуск менеджера
	workerManager := worker.NewManager(workerSrv)
	workerManager.Start()

	// Сохраняем метод для Graceful Shutdown
	m.WorkerShutdown = workerManager.Stop
}

func (b *Builder) Build() (*Microservice, error) {
	if b.err != nil {
		return nil, b.err
	}

	coreCfg := b.app.CoreCfg
	isPrometheusEnabled := coreCfg.Prometheus.Enabled
	isJaegerEnabled := coreCfg.Jaeger.Enabled

	midManager := middleware.NewManager(
		isPrometheusEnabled,
		isJaegerEnabled,
		coreCfg.Prometheus.Secure,
		coreCfg.Prometheus.User,
		coreCfg.Prometheus.Password,
		b.app.JWTManager,
		b.app.RedisClient,
		b.app.AppCfg.App.IsProd,
	)

	b.app.FiberApp.Use(midManager.Tracing())
	b.app.FiberApp.Use(midManager.Logging())
	b.app.FiberApp.Use(midManager.Metrics())

	if isPrometheusEnabled {
		slog.Info("Мониторинг Prometheus успешно активирован")
	} else {
		slog.Info("Мониторинг Prometheus отключен")
	}

	return b.app, nil
}

func (m *Microservice) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = ctx
	_ = cancel

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		port := m.AppCfg.App.Port
		slog.Info("HTTP Сервер запускается", slog.String("port", port))

		if err := m.FiberApp.Listen(":" + port); err != nil {
			slog.Error("Ошибка HTTP сервера", slog.Any("error", err))
		}
	}()

	<-sigChan
	slog.Info("Получен сигнал остановки, начинаем Graceful Shutdown...")

	if err := m.FiberApp.Shutdown(); err != nil {
		slog.Warn("Ошибка при остановке Fiber", slog.Any("error", err))
	}

	if m.TracerShutdown != nil {
		slog.Info("Остановка трейсера Jaeger...")
		ctxTimeout, cancelTrace := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelTrace()

		if err := m.TracerShutdown(ctxTimeout); err != nil {
			slog.Warn("Ошибка при остановке Jaeger", slog.Any("error", err))
		}
	}

	if m.DBPool != nil {
		slog.Info("Закрытие пула PostgreSQL...")
		m.DBPool.Close()
	}

	if m.RedisClient != nil {
		slog.Info("Закрытие соединений Redis...")
		if err := m.RedisClient.Close(); err != nil {
			slog.Warn("Ошибка при остановке Redis", slog.Any("error", err))
		}
	}

	if m.WorkerShutdown != nil {
		slog.Info("Остановка фоновых процессов...")
		m.WorkerShutdown()
	}

	slog.Info("Микросервис успешно остановлен. До свидания!")
	return nil
}
