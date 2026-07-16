package model

type AppUpdate struct {
	ID            string `json:"id"`
	RemoteVersion string `json:"version"`
	URL           string `json:"url"`
	CreatedAt     int64  `json:"created_at"`
}
