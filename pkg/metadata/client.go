// Package metadata handles outbound REST synchronization protocols with upstream provider registries.
package metadata

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/Cfirth725/tv-sentinel/pkg/models"
)

const tmdbBaseURL = "https://api.themoviedb.org/3"

// ====================================================================
//                -- CLIENT CONFIGURATION & INTAKE --
// ====================================================================

// TmdbClient encapsulates network transport configurations and access keys for TMDB.
type TmdbClient struct {
	httpClient *http.Client
	bearerAuth string
}

// NewTmdbClient configures an outbound HTTP transport client with integrated request timeouts.
func NewTmdbClient(token string) *TmdbClient {
	return &TmdbClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Shield system threads from lingering network hangs
		},
		bearerAuth: "Bearer " + token,
	}
}

// ====================================================================
//             -- OUTBOUND EXTERNAL API SYNCHRONIZERS --
// ====================================================================

// SearchTvSeries queries TMDB for candidate shows matching a raw title string.
func (c *TmdbClient) SearchTvSeries(query string) (*models.TmdbSearchEnvelope, error) {
	start := time.Now()
	escapedQuery := url.QueryEscape(query)
	reqURL := fmt.Sprintf("%s/search/tv?query=%s&include_adult=false&language=en-US&page=1", tmdbBaseURL, escapedQuery)

	slog.Info("[REALTIME] Initiating upstream catalog search",
		"subsystem", "metadata",
		"provider", "tmdb",
		"query", query,
	)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		slog.Error("[ERROR] Failed to construct outbound search frame structure",
			"subsystem", "metadata",
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to construct search request: %w", err)
	}

	// Inject the security headers mandated by TMDB
	req.Header.Add("Authorization", c.bearerAuth)
	req.Header.Add("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("[ERROR] Outbound search network pipeline transport invocation failure",
			"subsystem", "metadata",
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err.Error(),
		)
		return nil, fmt.Errorf("outbound search network invocation failure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("[ERROR] Upstream search gateway returned anomalous HTTP status response",
			"subsystem", "metadata",
			"status_code", resp.StatusCode,
			"status", resp.Status,
		)
		return nil, fmt.Errorf("upstream API returned bad status: %s", resp.Status)
	}

	var envelope models.TmdbSearchEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		slog.Error("[ERROR] Upstream payload deserialization structural failure",
			"subsystem", "metadata",
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to decode search result JSON: %w", err)
	}

	slog.Info("[OK] Upstream catalog candidates resolved successfully",
		"subsystem", "metadata",
		"results_count", envelope.TotalResults,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &envelope, nil
}

// GetSeriesDetails queries TMDB for detailed metadata fields belonging to a specific series ID.
func (c *TmdbClient) GetSeriesDetails(seriesID int64) (*models.TmdbSeriesDetails, error) {
	start := time.Now()
	reqURL := fmt.Sprintf("%s/tv/%d?language=en-US", tmdbBaseURL, seriesID)

	slog.Info("[REALTIME] Pulling granular show metadata specifications",
		"subsystem", "metadata",
		"provider", "tmdb",
		"series_id", seriesID,
	)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		slog.Error("[ERROR] Failed to construct outbound detail frame configuration",
			"subsystem", "metadata",
			"series_id", seriesID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to construct series details request: %w", err)
	}

	req.Header.Add("Authorization", c.bearerAuth)
	req.Header.Add("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("[ERROR] Outbound detail network transport invocation failure",
			"subsystem", "metadata",
			"series_id", seriesID,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err.Error(),
		)
		return nil, fmt.Errorf("outbound details network invocation failure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("[ERROR] Upstream details gateway returned anomalous status code",
			"subsystem", "metadata",
			"series_id", seriesID,
			"status_code", resp.StatusCode,
			"status", resp.Status,
		)
		return nil, fmt.Errorf("upstream details API returned bad status: %s", resp.Status)
	}

	var details models.TmdbSeriesDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		slog.Error("[ERROR] Detail payload structural decoding exception",
			"subsystem", "metadata",
			"series_id", seriesID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to decode series details JSON: %w", err)
	}

	slog.Info("[OK] Upstream metadata profile fetched and parsed cleanly",
		"subsystem", "metadata",
		"series_id", seriesID,
		"title", details.Name,
		"seasons_count", details.NumberOfSeasons,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &details, nil
}
