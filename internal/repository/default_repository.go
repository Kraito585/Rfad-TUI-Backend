package repository

import (
	coreredis "Rfad-TUI-Backend/core/pkg/redis"
	"Rfad-TUI-Backend/internal/model"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
)

type DefaultRepository struct {
	db *pgxpool.Pool
	r  *coreredis.Wrapper
}

func NewDefaultRepository(
	db *pgxpool.Pool,
	r *coreredis.Wrapper,
) *DefaultRepository {
	return &DefaultRepository{
		db: db,
		r:  r,
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
