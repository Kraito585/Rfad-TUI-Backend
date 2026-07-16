package migrate

import (
	"embed"
	"errors"
	"fmt"
	"Rfad-TUI-Backend/core/config"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// УДАЛИЛИ: var FS embed.FS

// ДОБАВИЛИ: аргумент fs embed.FS
func Run(cfg *config.CoreConfig, fs embed.FS) error {
	log.Println("Ожидание проверки миграций базы данных...")

	dsn := cfg.Postgres.DSN(cfg.Postgres.Name)

	// Используем переданную файловую систему fs
	d, err := iofs.New(fs, ".")
	if err != nil {
		return fmt.Errorf("ошибка чтения файлов миграций: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return fmt.Errorf("ошибка инициализации мигратора: %w", err)
	}

	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			log.Printf("⚠️ Предупреждение при закрытии источника миграций: %v", sourceErr)
		}
		if dbErr != nil {
			log.Printf("⚠️ Предупреждение при закрытии БД мигратора: %v", dbErr)
		}
	}()

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("✅ База данных актуальна, новых миграций нет.")
			return nil
		}
		return fmt.Errorf("ошибка при выполнении миграций: %w", err)
	}

	log.Println("🚀 Миграции успешно применены!")
	return nil
}
