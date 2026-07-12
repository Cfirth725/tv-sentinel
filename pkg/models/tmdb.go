// Package models encapsulates the core domain types, bulk payload envelopes,
// API request/response frames, and database transfer objects used throughout the system.
package models

// TmdbSearchRow handles an individual structural show result from a general search array.
type TmdbSearchRow struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"` // TMDB uses 'name' for TV series titles instead of 'title'
	OriginalName  string   `json:"original_name"`
	FirstAirDate  string   `json:"first_air_date"`
	OriginCountry []string `json:"origin_country"`
}

// TmdbSearchEnvelope handles the top-level pagination array returned from an HTTP search query.
type TmdbSearchEnvelope struct {
	Page         int             `json:"page"`
	Results      []TmdbSearchRow `json:"results"`
	TotalPages   int             `json:"total_pages"`
	TotalResults int             `json:"total_results"`
}

// TmdbSeriesDetails handles the targeted response payload when looking up an exact series ID.
type TmdbSeriesDetails struct {
	ID               int64           `json:"id"`
	Name             string          `json:"name"`
	Status           string          `json:"status"` // E.g., 'Returning Series', 'Ended', 'Canceled'
	Type             string          `json:"type"`   // E.g., 'Scripted', 'Reality', 'Miniseries'
	NumberOfEpisodes int             `json:"number_of_episodes"`
	NumberOfSeasons  int             `json:"number_of_seasons"`
	Overview         string          `json:"overview"`
	Seasons          []TmdbSeasonRow `json:"seasons"`
}

// TmdbSeasonRow handles the nested structural metadata for individual seasons
// within a detailed series response envelope.
type TmdbSeasonRow struct {
	SeasonNumber int `json:"season_number"`
	EpisodeCount int `json:"episode_count"`
}
