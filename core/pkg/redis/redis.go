package redis

import (
	"context"
	"fmt"
	"time"

	"Rfad-TUI-Backend/core/config"
	//core:telemetry
	"Rfad-TUI-Backend/core/pkg/coretelemetry"
	//core:telemetry:end

	"github.com/redis/go-redis/v9"
)

// Wrapper скрывает сложность роутинга между мастером и репликами
type Wrapper struct {
	writer redis.Cmdable
	reader redis.Cmdable
}

// NewRedisManager инициализирует подключения в зависимости от режима
func NewRedisManager(ctx context.Context, cfg config.RedisConfig) (*Wrapper, error) {
	wrapper := &Wrapper{}

	switch cfg.Mode {
	case "cluster":
		// В режиме Cluster сам Redis роутит запросы.
		clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.MasterAddrs,
			Password: cfg.Password,
			PoolSize: cfg.PoolSize,
			ReadOnly: true,
		})

		// Навешиваем хук до первого запроса
		//core:telemetry
		clusterClient.AddHook(coretelemetry.NewRedisHook())
		//core:telemetry:end

		if err := clusterClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("ошибка подключения к Redis Cluster: %w", err)
		}

		wrapper.writer = clusterClient
		wrapper.reader = clusterClient

	case "master_replica":
		// Создаем конкретный тип клиента для мастера
		masterClient := redis.NewClient(&redis.Options{
			Addr:     cfg.MasterAddrs[0],
			Password: cfg.Password,
			PoolSize: cfg.PoolSize,
		})

		//core:telemetry
		masterClient.AddHook(coretelemetry.NewRedisHook())
		//core:telemetry:end

		// Вызываем Ping напрямую от masterClient (без type assertion)
		if err := masterClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("redis master недоступен: %w", err)
		}
		wrapper.writer = masterClient

		// Создаем конкретный тип клиента для реплики
		replicaClient := redis.NewClient(&redis.Options{
			Addr:     cfg.ReplicaAddrs[0],
			Password: cfg.Password,
			PoolSize: cfg.PoolSize,
		})

		//core:telemetry
		replicaClient.AddHook(coretelemetry.NewRedisHook())
		//core:telemetry:end

		if err := replicaClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("redis replica недоступна: %w", err)
		}
		wrapper.reader = replicaClient

	default: // "standalone"
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.MasterAddrs[0],
			Password: cfg.Password,
		})

		//core:telemetry
		client.AddHook(coretelemetry.NewRedisHook())
		//core:telemetry:end

		wrapper.writer = client
		wrapper.reader = client
	}

	return wrapper, nil
}

// =========================
// МЕТОДЫ ЗАПИСИ (Идут в Master)
// =========================
func (w *Wrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return w.writer.Set(ctx, key, value, expiration)
}

func (w *Wrapper) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return w.writer.Del(ctx, keys...)
}

func (w *Wrapper) Incr(ctx context.Context, key string) *redis.IntCmd {
	return w.writer.Incr(ctx, key)
}

// =========================
// МЕТОДЫ ЧТЕНИЯ (Идут в Replicas)
// =========================
func (w *Wrapper) Get(ctx context.Context, key string) *redis.StringCmd {
	return w.reader.Get(ctx, key)
}

// Метод для плавного завершения работы
func (w *Wrapper) Close() error {
	var err error
	if closer, ok := w.writer.(interface{ Close() error }); ok {
		err = closer.Close()
	}
	if w.writer != w.reader {
		if closer, ok := w.reader.(interface{ Close() error }); ok {
			err = closer.Close()
		}
	}
	return err
}

func (w *Wrapper) Expire(ctx context.Context, key string, expire time.Duration) *redis.BoolCmd {
	return w.writer.Expire(ctx, key, expire)
}

func (w *Wrapper) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return w.reader.HGet(ctx, key, field)
}

func (w *Wrapper) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return w.reader.SIsMember(ctx, key, member)
}

func (w *Wrapper) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return w.reader.Exists(ctx, keys...)
}

func (w *Wrapper) GetDel(ctx context.Context, key string) *redis.StringCmd {
	return w.writer.GetDel(ctx, key)
}

func (w *Wrapper) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return w.writer.HSet(ctx, key, values...)
}

func (w *Wrapper) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return w.writer.SAdd(ctx, key, members...)
}

func (w *Wrapper) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return w.writer.HDel(ctx, key, fields...)
}

func (w *Wrapper) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return w.writer.SRem(ctx, key, members...)
}

func (w *Wrapper) Ping(ctx context.Context) *redis.StatusCmd {
	return w.writer.Ping(ctx)
}
