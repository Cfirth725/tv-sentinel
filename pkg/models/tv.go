// Package models encapsulates the core domain types, bulk payload envelopes,
// API request/response frames, and database transfer objects used throughout the system.
package models

import "time"

// TvCatalog captures the specific metadata fields mapped from the external TMDB registry.
type TvCatalog struct {
	ID                int64     `json:"id"`
	ExternalID        string    `json:"external_id"`
	CacheKey          string    `json:"cache_key"`
	TitleDisplay      string    `json:"title_display"`
	Status            string    `json:"status"`
	Type              string    `json:"type"`
	TotalSeasonsCount int       `json:"total_seasons_count"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TvWatchProgress updates a user's running checkpoint progress ledger for a given television asset.
type TvWatchProgress struct {
	UserID         int64     `json:"user_id"`
	TvID           int64     `json:"tv_id"`
	CurrentSeason  int       `json:"current_season"`
	CurrentEpisode int       `json:"current_episode"`
	LastWatchedAt  time.Time `json:"last_watched_at"`
	Sentiment      int       `json:"sentiment"`
}

// TvIngestPayload represents an incoming raw tracking record forwarded from the ingestion streams.
type TvIngestPayload struct {
	Username      string    `json:"username"`
	SeriesTitle   string    `json:"series_title"`
	SeasonNumber  int       `json:"season_number"`
	EpisodeNumber int       `json:"episode_number"`
	Sentiment     int       `json:"sentiment"`
	WatchedAt     time.Time `json:"watched_at"`
}

// TvImportEnvelope acts as the array packet wrapper for batch historic data ingestion runs.
type TvImportEnvelope struct {
	Shows []TvIngestPayload `json:"shows"`
}

// User captures the unified system identity profile mapping shared across the suite.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}
