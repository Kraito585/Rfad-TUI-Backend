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

	// 1. Спрашиваем репозиторий: "Какая версия в Google?"
	remoteVersion, err := s.repo.FetchRemoteVersion(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения удаленной версии: %w", err)
	}

	// 2. Спрашиваем репозиторий: "Какая версия у нас в БД?"
	currentVersion, err := s.repo.GetLatestInternalVersion(ctx)
	if err != nil {
		return fmt.Errorf("ошибка чтения версии из БД: %w", err)
	}

	// 3. Бизнес-логика: сравнение версий
	if remoteVersion != currentVersion {
		slog.Info("Найдено обновление, начинаем зеркалирование", slog.String("new_version", remoteVersion))

		// 4. Командуем репозиторию перелить файл
		fileName, err := s.repo.MirrorZipToS3(ctx)
		if err != nil {
			return fmt.Errorf("ошибка зеркалирования архива: %w", err)
		}

		// 5. Генерируем нашу внутреннюю версию
		internalVersionID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("ошибка генерации UUIDv7: %w", err)
		}

		downloadURL := fmt.Sprintf("/downloads/%s", fileName)

		// 6. Командуем репозиторию сохранить результат
		err = s.repo.SaveUpdate(ctx, internalVersionID, remoteVersion, downloadURL)
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
