package config

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type AppConfig struct {
	App struct {
		Port              string `yaml:"port" env:"PORT" env-default:"7910"`
		MetricsEnabled    bool   `yaml:"metrics_enabled" env:"METRICS_ENABLED"`
		IsProd            bool   `yaml:"is_prod" env:"IS_PROD"`
		EnablePodTracking bool   `yaml:"enable_pod_tracking"`
		Cluster           bool   `yaml:"cluster" env:"CLUSTER_MODE"`
		PodID             string `yaml:"pod_id" env:"POD_ID"`
	} `yaml:"app"`

	Core struct {
		StrictMode bool `yaml:"strict_mode"`
		// В YAML написано modulse, тег считывает именно его
		Modules []string `yaml:"modules"`
	} `yaml:"core"`

	Cors struct {
		URL  string   `yaml:"url" env:"CORS_URL"`
		URLs []string `yaml:"urls" env:"CORS_URLS"`
	} `yaml:"cors"`
}

func LoadAppConfig(path string) (*AppConfig, error) {
	var cfg AppConfig
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, err
	}

	// ====================================================
	// АВТОМАТИЧЕСКОЕ ИЗВЛЕЧЕНИЕ ИЛИ ГЕНЕРАЦИЯ POD_ID
	// ====================================================
	if cfg.App.PodID == "" {
		// Если POD_ID не задан явно в env, проверяем стандартный K8s HOSTNAME
		if k8sHost := os.Getenv("HOSTNAME"); k8sHost != "" {
			cfg.App.PodID = k8sHost
		} else {
			// Локальный дев-запуск: генерируем случайный 8-символьный суффикс
			bytes := make([]byte, 4)
			if _, randErr := rand.Read(bytes); randErr == nil {
				cfg.App.PodID = fmt.Sprintf("dev-pod-%x", bytes)
			} else {
				cfg.App.PodID = "fallback-dev-pod"
			}
		}
	}

	return &cfg, err
}
