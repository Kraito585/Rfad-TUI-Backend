package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"Rfad-TUI-Backend/core/pkg/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	gDocID    = "17qsV5xDeJZyGZNFbxm3eZ50DYYm1URAvvhx588fAiSo"
	gDriveDir = "1JUOctbsugh2IIEUCWcBkupXYVYoJMg4G"
)

var workerRepoTracer = otel.Tracer("worker-repository")

type WorkerRepository struct {
	db              *pgxpool.Pool
	s3Client        *storage.S3Client
	credentialsFile string
}

// В конструктор передаем все внешние зависимости
func NewWorkerRepository(db *pgxpool.Pool, s3Client *storage.S3Client, credsFile string) *WorkerRepository {
	return &WorkerRepository{
		db:              db,
		s3Client:        s3Client,
		credentialsFile: credsFile,
	}
}

// FetchRemoteVersion скачивает текстовую версию из Google Docs
func (r *WorkerRepository) FetchRemoteVersion(ctx context.Context) (string, error) {
	ctx, span := workerRepoTracer.Start(ctx, "worker_repository.FetchRemoteVersion")
	defer span.End()

	docURL := fmt.Sprintf("https://docs.google.com/document/d/%s/export?format=txt", gDocID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, docURL, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса к Docs: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса к Google Docs: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения версии: %w", err)
	}

	remoteVersion := strings.TrimSpace(string(bodyBytes))
	return strings.Trim(remoteVersion, "\xef\xbb\xbf"), nil
}

// GetLatestInternalVersion получает версию из нашей БД
func (r *WorkerRepository) GetLatestInternalVersion(ctx context.Context) (string, error) {
	ctx, span := workerRepoTracer.Start(ctx, "worker_repository.GetLatestInternalVersion")
	defer span.End()

	var version string
	query := "SELECT remote_version FROM app_updates ORDER BY id DESC LIMIT 1"

	err := r.db.QueryRow(ctx, query).Scan(&version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("ошибка получения последней версии: %w", err)
	}

	return version, nil
}

// MirrorZipToS3 перекачивает архив с Google Drive в S3
func (r *WorkerRepository) MirrorZipToS3(ctx context.Context) (string, error) {
    ctx, span := workerRepoTracer.Start(ctx, "worker_repository.MirrorZipToS3")
    defer span.End()

    srv, err := drive.NewService(ctx, option.WithCredentialsFile(r.credentialsFile))
    if err != nil {
        return "", fmt.Errorf("ошибка GDrive клиента: %w", err)
    }

    query := fmt.Sprintf("'%s' in parents and mimeType contains 'zip' and trashed = false", gDriveDir)
    fileList, err := srv.Files.List().
        Q(query).OrderBy("createdTime desc").PageSize(1).Fields("files(id, name)").Do()
    if err != nil {
        return "", fmt.Errorf("ошибка поиска файла: %w", err)
    }

    if len(fileList.Files) == 0 {
        return "", fmt.Errorf("в папке не найдено ZIP архивов")
    }

    targetFile := fileList.Files[0]

    downloadResp, err := srv.Files.Get(targetFile.Id).Download()
    if err != nil {
        return "", fmt.Errorf("ошибка скачивания: %w", err)
    }
    defer downloadResp.Body.Close()

    newFileName := fmt.Sprintf("%s.zip", uuid.New().String())
    
    s3Key := fmt.Sprintf("rfad/%s", newFileName)

    err = r.s3Client.Upload(ctx, s3Key, downloadResp.Body, "application/zip")
    if err != nil {
        return "", fmt.Errorf("ошибка загрузки в S3: %w", err)
    }

    return s3Key, nil
}

// SaveUpdate сохраняет запись о версии в БД
func (r *WorkerRepository) SaveUpdate(ctx context.Context, id uuid.UUID, remoteVersion, url string) error {
	ctx, span := workerRepoTracer.Start(ctx, "worker_repository.SaveUpdate")
	defer span.End()

	query := "INSERT INTO app_updates (id, remote_version, url) VALUES ($1, $2, $3)"

	_, err := r.db.Exec(ctx, query, id, remoteVersion, url)
	if err != nil {
		return fmt.Errorf("ошибка сохранения обновления в БД: %w", err)
	}

	return nil
}
