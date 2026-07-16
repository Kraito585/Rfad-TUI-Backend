package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type JWTConfig struct {
	Enabled        bool   `yaml:"enabled" env:"JWT_ENABLED" env-default:"false"`
	PrivateKeyPath string `yaml:"private_key_path" env:"JWT_PRIVATE_KEY_PATH"`
	PublicKeyPath  string `yaml:"public_key_path" env:"JWT_PUBLIC_KEY_PATH"`
	AccessTTL      int    `yaml:"access_ttl" env:"JWT_ACCESS_TTL" env-default:"15"`
}

type S3Config struct {
	Enabled   bool   `yaml:"enabled" env:"S3_ENABLED" env-default:"false"`
	Endpoint  string `yaml:"endpoint" env:"S3_ENDPOINT"`
	Region    string `yaml:"region" env:"S3_REGION" env-default:"ru-1"`
	AccessKey string `yaml:"access_key" env:"S3_ACCESS_KEY"`
	SecretKey string `yaml:"secret_key" env:"S3_SECRET_KEY"`
	Bucket    string `yaml:"bucket" env:"S3_BUCKET"`
}

type PostgresConfig struct {
	Host     string   `yaml:"db_host" env:"DB_HOST"`
	Port     string   `yaml:"db_port" env:"DB_PORT"`
	Name     string   `yaml:"db_name" env:"DB_NAME"`
	Names    []string `yaml:"db_names" env:"DB_NAMES"`
	User     string   `yaml:"user" env:"DB_USER" env-required:"true"`
	Password string   `yaml:"password" env:"DB_PASS" env-required:"true"`

	MaxConns int32 `yaml:"max_conns" env:"DB_MAX_CONNS" env-default:"10"`
	MinConns int32 `yaml:"min_conns" env:"DB_MIN_CONNS" env-default:"2"`
}

type RedisConfig struct {
	Mode         string   `yaml:"mode" env-default:"standalone"`
	MasterAddrs  []string `yaml:"master_addrs"`
	ReplicaAddrs []string `yaml:"replica_addrs"`
	Password     string   `yaml:"password" env:"REDIS_PASS"`
	PoolSize     int      `yaml:"pool_size" env-default:"100"`
}

type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

type CoreConfig struct {
	Postgres PostgresConfig `yaml:"postgresql"`
	Redis    RedisConfig    `yaml:"redis"`
	CORS     CORSConfig     `yaml:"cors"`
	S3       S3Config       `yaml:"s3"`
	JWT      JWTConfig      `yaml:"jwt"`

	Prometheus struct {
		Enabled     bool   `yaml:"enabled" env:"PROMETHEUS_ENABLED"`
		Secure      bool   `yaml:"secure" env:"PROMETHEUS_SECURE"`
		User        string `yaml:"user" env:"METRICS_USER" env-required:"false"`
		Password    string `yaml:"password" env:"METRICS_PASS" env-required:"false"`
		ServiceName string `yaml:"service_name" env:"SERVICE_NAME" env-default:"unknown"`
	} `yaml:"prometheus"`

	Jaeger struct {
		Enabled     bool   `yaml:"enabled" env:"JAEGER_ENABLED"`
		ServiceName string `yaml:"service_name" env:"JAEGER_SERVICE_NAME"`
		URL         string `yaml:"url" env:"JAEGER_URL"`
	} `yaml:"jaeger"`

	Security struct {
		MasterKey string `yaml:"master_key" env:"MASTER_ENCRYPTION_KEY" env-required:"true"`
	} `yaml:"security"`
}

func (p *PostgresConfig) DSN(dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, dbName,
	)
}

func LoadCoreConfig(path string) (*CoreConfig, error) {
	var cfg CoreConfig
	err := cleanenv.ReadConfig(path, &cfg)
	return &cfg, err
}
