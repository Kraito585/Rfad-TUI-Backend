package worker

import (
	"context"
	"log/slog"
	"time"

	"Rfad-TUI-Backend/internal/service"
)

// Manager агрегирует все фоновые задачи
type Manager struct {
	workerService *service.WorkerService
	quit          chan struct{}
}

func NewManager(workerService *service.WorkerService) *Manager {
	return &Manager{
		workerService: workerService,
		quit:          make(chan struct{}),
	}
}

// Start запускает все зарегистрированные фоновые процессы
func (m *Manager) Start() {
	slog.Info("Запуск пула фоновых воркеров...")

	go m.startUpdateScheduler()

	// В будущем:
	// go m.startSessionCleanupScheduler()
	// go m.startTelemetryExporter()
}

// Stop плавно завершает работу всех воркеров
func (m *Manager) Stop() {
	close(m.quit)
}

func (m *Manager) startUpdateScheduler() {
	slog.Info("Воркер обновлений запущен (интервал: 1 час)")

	// Делаем первый прогон сразу при старте
	m.runUpdateJob()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.runUpdateJob()
		case <-m.quit:
			slog.Info("Воркер обновлений плавно остановлен")
			return
		}
	}
}

func (m *Manager) runUpdateJob() {
	// Жесткий таймаут на одну итерацию обновления
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	if err := m.workerService.ProcessUpdates(ctx); err != nil {
		slog.Error("Ошибка в воркере обновлений", slog.Any("error", err))
	}
}
