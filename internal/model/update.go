package model

import (
	"encoding/json"
	"time"
)

type AppUpdate struct {
	ID            string `json:"id"`
	RemoteVersion string `json:"version"`
	URL           string `json:"url"`
	CreatedAt     int64  `json:"created_at"`
}

type UploadConfig struct {
	Config []json.RawMessage `json:"config"`
}

type SwitchConfig struct {
	UUID string `json:"UUID"`
}

type ConfigAdminResponse struct {
	ID        string          `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	Active    bool            `json:"active"`
	Config    json.RawMessage `json:"config"`
}
