package health

import (
	"context"
	"fmt"
	"Rfad-TUI-Backend/core/pkg/storage"
)

type S3Checker struct {
	client *storage.S3Client
}

func NewS3Checker(client *storage.S3Client) *S3Checker {
	return &S3Checker{
		client: client,
	}
}

func (c *S3Checker) Name() string {
	return "s3_storage"
}

func (c *S3Checker) Check(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("S3 клиент не инициализирован")
	}

	// Вызываем наш легковесный пинг
	return c.client.Ping(ctx)
}
