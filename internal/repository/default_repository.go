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
