package model

import (
	"time"

	"github.com/google/uuid"
)

// --- DTO: То, что мы ждем от клиента (Postman) ---

type UploadPresetMetadata struct {
	OriginURL         string              `json:"originUrl"`
	AuthorNickname    string              `json:"authorNickname"`
	PerformanceImpact int16               `json:"performance_impact"`
	Description       string              `json:"description"`
	OptionalMods      []UploadOptionalMod `json:"optional_mods"`
}

type UploadOptionalMod struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	FileName   string   `json:"file_name"`
	IsRequired bool     `json:"is_required"`
	DependsOn  []string `json:"depends_on,omitempty"`
}

// --- Entity: То, что ложится в базу данных ---

type CommunityShaderPreset struct {
	ID                uuid.UUID `json:"id"`
	URL               string    `json:"url"`
	Images            []string  `json:"images"`
	PerformanceImpact int16     `json:"performance_impact"`
	Metadata          Metadata  `json:"metadata"`
	CreatedAt         time.Time `json:"created_at"`
}

type Metadata struct {
	OriginURL      string        `json:"originUrl"`
	AuthorNickname string        `json:"authorNickname"`
	Description    string        `json:"description"`
	OptionalMods   []OptionalMod `json:"optional_mods"`
}

type OptionalMod struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	IsRequired bool     `json:"is_required"`
	DependsOn  []string `json:"depends_on,omitempty"`
}
