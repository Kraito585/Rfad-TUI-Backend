package service

import (
	"Rfad-TUI-Backend/core/pkg/security"
	"Rfad-TUI-Backend/internal/model"
	"Rfad-TUI-Backend/internal/repository"
	"context"
	"fmt"
	"mime/multipart"

	"github.com/google/uuid"

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

	return update, nil
}

func (s *DefaultService) GetAllPresets(ctx context.Context) ([]model.CommunityShaderPreset, error) {
	presets, err := s.repo.GetAllPresets(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пресетов из репозитория: %w", err)
	}

	return presets, nil
}

func (s *DefaultService) UploadPreset(
	ctx context.Context,
	meta *model.UploadPresetMetadata,
	mainFile *multipart.FileHeader,
	images []*multipart.FileHeader,
	modFiles []*multipart.FileHeader,
) (uuid.UUID, error) {

	presetID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("генерация UUID: %w", err)
	}

	baseS3Path := fmt.Sprintf("rfad/shaders/presets/%s", presetID.String())

	// 1. Загрузка основного архива пресета
	mainFileSrc, _ := mainFile.Open()
	defer mainFileSrc.Close()

	mainS3Key := fmt.Sprintf("%s/main/%s", baseS3Path, mainFile.Filename)
	if err := s.repo.UploadFile(ctx, mainS3Key, mainFileSrc, "application/zip"); err != nil {
		return uuid.Nil, fmt.Errorf("ошибка загрузки основного файла: %w", err)
	}

	// 2. Загрузка скриншотов
	var imageUrls []string
	for _, img := range images {
		imgSrc, _ := img.Open()
		imgS3Key := fmt.Sprintf("%s/screens/%s", baseS3Path, img.Filename)
		if err := s.repo.UploadFile(ctx, imgS3Key, imgSrc, "image/jpeg"); err != nil {
			imgSrc.Close()
			return uuid.Nil, fmt.Errorf("ошибка загрузки картинки %s: %w", img.Filename, err)
		}
		imageUrls = append(imageUrls, imgS3Key)
		imgSrc.Close()
	}

	// 3. Создаем мапу для быстрого поиска опциональных файлов по имени
	modFilesMap := make(map[string]*multipart.FileHeader)
	for _, f := range modFiles {
		modFilesMap[f.Filename] = f
	}

	// 4. Обогащаем манифест опциональных модов
	var finalOptionalMods []model.OptionalMod
	for _, reqMod := range meta.OptionalMods {
		fileHeader, ok := modFilesMap[reqMod.FileName]
		if !ok {
			return uuid.Nil, fmt.Errorf("в запросе отсутствует прикрепленный файл: %s", reqMod.FileName)
		}

		modSrc, _ := fileHeader.Open()
		modS3Key := fmt.Sprintf("%s/mods/%s", baseS3Path, fileHeader.Filename)

		if err := s.repo.UploadFile(ctx, modS3Key, modSrc, "application/zip"); err != nil {
			modSrc.Close()
			return uuid.Nil, fmt.Errorf("ошибка загрузки мода %s: %w", fileHeader.Filename, err)
		}
		modSrc.Close()

		finalOptionalMods = append(finalOptionalMods, model.OptionalMod{
			ID:         reqMod.ID,
			Name:       reqMod.Name,
			URL:        modS3Key, // Обогатили ссылку!
			IsRequired: reqMod.IsRequired,
			DependsOn:  reqMod.DependsOn,
		})
	}

	// 5. Собираем финальную Entity и сохраняем
	finalPreset := model.CommunityShaderPreset{
		ID:                presetID,
		URL:               mainS3Key,
		Images:            imageUrls,
		PerformanceImpact: meta.PerformanceImpact,
		Metadata: model.Metadata{
			OriginURL:      meta.OriginURL,
			AuthorNickname: meta.AuthorNickname,
			OptionalMods:   finalOptionalMods,
		},
	}

	if err := s.repo.SavePreset(ctx, finalPreset); err != nil {
		return uuid.Nil, err
	}

	return presetID, nil
}
