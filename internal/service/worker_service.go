package service

import (
	"context"
	"fmt"
	"log/slog"

	"Rfad-TUI-Backend/internal/repository"

	"github.com/google/uuid"
)

type WorkerService struct {
	repo *repository.WorkerRepository
}

// В конструктор передаем ТОЛЬКО репозиторий
func NewWorkerService(repo *repository.WorkerRepository) *WorkerService {
	return &WorkerService{
		repo: repo,
	}
}

func (s *WorkerService) ProcessUpdates(ctx context.Context) error {
	slog.Info("Проверка обновлений в Google Docs...")

	remoteVersion, err := s.repo.FetchRemoteVersion(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения удаленной версии: %w", err)
	}

	currentVersion, err := s.repo.GetLatestInternalVersion(ctx)
	if err != nil {
		return fmt.Errorf("ошибка чтения версии из БД: %w", err)
	}

	if remoteVersion != currentVersion {
		slog.Info("Найдено обновление, пытаемся захватить задачу...", slog.String("new_version", remoteVersion))

		// 1. ЗАХВАТЫВАЕМ ЛОК ЗДЕСЬ
		acquired, err := s.repo.AcquireLock(ctx)
		if err != nil {
			return fmt.Errorf("ошибка работы с Redis: %w", err)
		}
		if !acquired {
			slog.Debug("Другая реплика уже зеркалирует архив, отступаем")
			return nil
		}

		// 2. ГАРАНТИРУЕМ СНЯТИЕ ЛОКА ПРИ ЛЮБОЙ ОШИБКЕ ВНИЗУ
		defer s.repo.ReleaseLock(ctx)

		// Теперь мы в полной безопасности. Если любой из шагов ниже упадет,
		// функция прервется (return err), но defer сработает и лок снимется моментально.

		fileName, err := s.repo.MirrorZipToS3(ctx)
		if err != nil {
			return fmt.Errorf("ошибка зеркалирования архива: %w", err)
		}

		internalVersionID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("ошибка генерации UUIDv7: %w", err)
		}

		err = s.repo.SaveUpdate(ctx, internalVersionID, remoteVersion, fileName)
		if err != nil {
			return fmt.Errorf("ошибка сохранения новой версии в БД: %w", err)
		}

		slog.Info("Зеркалирование успешно завершено",
			slog.String("internal_version", internalVersionID.String()),
			slog.String("remote_version", remoteVersion),
		)
	} else {
		slog.Debug("Обновлений не найдено, версия актуальна", slog.String("version", currentVersion))
	}

	return nil
}
