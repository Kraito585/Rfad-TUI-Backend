package repository

import (
	coreredis "Rfad-TUI-Backend/core/pkg/redis"
	"Rfad-TUI-Backend/core/pkg/storage"
	"Rfad-TUI-Backend/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
)

type DefaultRepository struct {
	db       *pgxpool.Pool
	r        *coreredis.Wrapper
	s3Client *storage.S3Client
}

func NewDefaultRepository(
	db *pgxpool.Pool,
	r *coreredis.Wrapper,
	s3Client *storage.S3Client,
) *DefaultRepository {
	return &DefaultRepository{
		db:       db,
		r:        r,
		s3Client: s3Client,
	}
}

var defaultRepoTracer = otel.Tracer("default-repository")

func (r *DefaultRepository) GetLatestUpdate(ctx context.Context) (*model.AppUpdate, error) {
	ctx, span := defaultRepoTracer.Start(ctx, "repository.GetLatestUpdate")
	defer span.End()

	var update model.AppUpdate
	var createdAt time.Time

	// Вся магия UUIDv7 здесь: сортировка по убыванию (DESC) ID гарантирует получение самой новой записи
	query := `
		SELECT id, remote_version, url, created_at 
		FROM app_updates 
		ORDER BY id DESC 
		LIMIT 1
	`

	err := r.db.QueryRow(ctx, query).Scan(
		&update.ID,
		&update.RemoteVersion,
		&update.URL,
		&createdAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Ошибки нет, просто таблица пока пустая
		}
		return nil, fmt.Errorf("ошибка запроса к БД: %w", err)
	}

	update.CreatedAt = createdAt.UnixMilli()

	return &update, nil
}

// GetAllPresets возвращает все доступные пресеты разом
func (r *DefaultRepository) GetAllPresets(ctx context.Context) ([]model.CommunityShaderPreset, error) {
	ctx, span := defaultRepoTracer.Start(ctx, "default_repository.GetAllPresets")
	defer span.End()

	query := `
        SELECT id, url, images, performance_impact, metadata, created_at 
        FROM community_shader_presets 
        ORDER BY performance_impact ASC, created_at DESC
    `

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса пресетов: %w", err)
	}
	defer rows.Close()

	presets := make([]model.CommunityShaderPreset, 0, 9)

	for rows.Next() {
		var p model.CommunityShaderPreset
		var metaBytes []byte

		err := rows.Scan(
			&p.ID,
			&p.URL,
			&p.Images,
			&p.PerformanceImpact,
			&metaBytes,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга строки пресета: %w", err)
		}

		if err := json.Unmarshal(metaBytes, &p.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка анмаршалинга метадаты (id=%s): %w", p.ID, err)
		}

		presets = append(presets, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по пресетам: %w", err)
	}

	return presets, nil
}

// UploadFile отправляет поток байт напрямую в S3 бакет
func (r *DefaultRepository) UploadFile(ctx context.Context, s3Key string, data io.Reader, contentType string) error {
	ctx, span := defaultRepoTracer.Start(ctx, "default_repository.UploadFile")
	defer span.End()

	return r.s3Client.Upload(ctx, s3Key, data, contentType)
}

// SavePreset сохраняет полностью сформированную модель в БД
func (r *DefaultRepository) SavePreset(ctx context.Context, p model.CommunityShaderPreset) error {
	ctx, span := defaultRepoTracer.Start(ctx, "default_repository.SavePreset")
	defer span.End()

	metaBytes, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга метадаты: %w", err)
	}

	query := `
        INSERT INTO community_shader_presets (id, url, images, performance_impact, metadata) 
        VALUES ($1, $2, $3, $4, $5)
    `

	_, err = r.db.Exec(ctx, query, p.ID, p.URL, p.Images, p.PerformanceImpact, metaBytes)
	if err != nil {
		return fmt.Errorf("ошибка сохранения пресета в БД: %w", err)
	}

	return nil
}

func (r *DefaultRepository) SaveConfig(ctx context.Context, body []byte) error {
	ctx, span := defaultRepoTracer.Start(ctx, "default_repository.SaveConfig")
	defer span.End()

	UUID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("Ошибка генерации UUID: %w", err)
	}

	query := `INSERT INTO configs_patches (id, config) VALUES ($1, $2)`

	_, err = r.db.Exec(ctx, query, UUID, body)
	if err != nil {
		return fmt.Errorf("Ошибка сохранение конфига: %w", err)
	}

	return nil
}

func (r *DefaultRepository) DisableConfig(ctx context.Context, UUID string) error {
	ctx, span := defaultRepoTracer.Start(ctx, "default_repository.DisableConfig")
	defer span.End()

	query := `UPDATE configs_patches SET active = $1 WHERE id = $2`

	_, err := r.db.Exec(ctx, query, false, UUID)
	if err != nil {
		return fmt.Errorf("не удалось отключить конфиг: %w", err)
	}
	return nil
}

func (r *DefaultRepository) EnableConfig(ctx context.Context, UUID string) error {
	ctx, span := defaultRepoTracer.Start(ctx, "default_repository.EnableConfig")
	defer span.End()

	query := `UPDATE configs_patches SET active = $1 WHERE id = $2`

	_, err := r.db.Exec(ctx, query, true, UUID)
	if err != nil {
		return fmt.Errorf("не удалось включить конфиг: %w", err)
	}
	return nil
}

func (r *DefaultRepository) GetConfig(ctx context.Context) ([]json.RawMessage, error) {
	ctx, span := defaultRepoTracer.Start(ctx, "default_repository.GetConfig")
	defer span.End()

	query := `SELECT config FROM configs_patches WHERE active = true`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса на загрузку конфигов: %w", err)
	}
	defer rows.Close()

	var configs []json.RawMessage

	for rows.Next() {
		var configData json.RawMessage
		if err := rows.Scan(&configData); err != nil {
			return nil, fmt.Errorf("ошибка чтения данных строки: %w", err)
		}

		configs = append(configs, configData)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам БД: %w", err)
	}

	if configs == nil {
		configs = make([]json.RawMessage, 0)
	}

	return configs, nil
}
