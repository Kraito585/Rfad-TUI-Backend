package service

import (
	"Rfad-TUI-Backend/core/pkg/security"
	"Rfad-TUI-Backend/internal/model"
	"Rfad-TUI-Backend/internal/repository"
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
)

type TokenEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(cryptoText string) (string, error)
}

type DefaultService struct {
	repo       *repository.DefaultRepository
	encryptor  TokenEncryptor
	jwtManager *security.JWTManager
	isProd     bool
}

func NewDefaultService(repo *repository.DefaultRepository, enc TokenEncryptor, jwtManager *security.JWTManager, isProd bool) *DefaultService {
	return &DefaultService{
		repo:       repo,
		encryptor:  enc,
		jwtManager: jwtManager,
		isProd:     isProd,
	}
}

var tracer = otel.Tracer("default-service")

func (s *DefaultService) GetLatestUpdate(ctx context.Context) (*model.AppUpdate, error) {
	ctx, span := tracer.Start(ctx, "service.GetLatestUpdate")
	defer span.End()

	update, err := s.repo.GetLatestUpdate(ctx)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить данные об обновлении: %w", err)
	}

	if update == nil {
		return nil, fmt.Errorf("обновления еще не были опубликованы")
	}

	// Можно дополнить URL, если в БД хранится только имя файла
	// В нашем случае воркер сохранял url как "/downloads/файл.zip"
	// update.URL = "https://ваш-s3-бакет.selectel.ru" + update.URL

	return update, nil
}
