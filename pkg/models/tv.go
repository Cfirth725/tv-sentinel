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

// TvCatchUpDelta represents a television series where upstream catalog availability
// outpaces the user's current localized watch state tracking records.
type TvCatchUpDelta struct {
	TvID              int64     `json:"tv_id"`
	TitleDisplay      string    `json:"title_display"`
	CurrentSeason     int       `json:"current_season"`
	CurrentEpisode    int       `json:"current_episode"`
	TotalSeasonsCount int       `json:"total_seasons_count"`
	LastWatchedAt     time.Time `json:"last_watched_at"`
}

// ====================================================================
//          -- PROGRAMMATIC TASTE PROFILE ANALYTICS MODELS --
// ====================================================================

// ShowCompletion represents the computed tracking depth and engagement
// metrics for a single television series.
type ShowCompletion struct {
	TvID             int64   `json:"tv_id"`
	TitleDisplay     string  `json:"title_display"`
	EpisodesWatched  int     `json:"episodes_watched"`
	TotalEpisodes    int     `json:"total_episodes"`
	EngagementScore  float64 `json:"engagement_score"`   // Percentage: (Watched / Total) * 100
	IsAnchorInterest bool    `json:"is_anchor_interest"` // True if EngagementScore >= 80%
}

// TasteProfile aggregates calculated user metrics, preferred genre anchors,
// and the current catch-up delta queue.
type TasteProfile struct {
	UserID          int64            `json:"user_id"`
	Username        string           `json:"username"`
	AnchorGenres    []string         `json:"anchor_genres"` // Aggregated from tags of shows >= 80% engagement
	CatchUpRadar    []TvCatchUpDelta `json:"catch_up_radar"`
	CompletionStats []ShowCompletion `json:"completion_stats"`
	GeneratedAt     time.Time        `json:"generated_at"`
}
