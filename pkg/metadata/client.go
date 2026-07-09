// Package metadata handles outbound REST synchronization protocols with upstream provider registries.
package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Cfirth725/tv-sentinel/pkg/models"
)

const tmdbBaseURL = "https://api.themoviedb.org/3"

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

// SearchTvSeries queries TMDB for candidate shows matching a raw title string.
func (c *TmdbClient) SearchTvSeries(query string) (*models.TmdbSearchEnvelope, error) {
	escapedQuery := url.QueryEscape(query)
	reqURL := fmt.Sprintf("%s/search/tv?query=%s&include_adult=false&language=en-US&page=1", tmdbBaseURL, escapedQuery)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct search request: %w", err)
	}

	// Inject the security headers mandated by TMDB
	req.Header.Add("Authorization", c.bearerAuth)
	req.Header.Add("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("outbound search network invocation failure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream API returned bad status: %s", resp.Status)
	}

	var envelope models.TmdbSearchEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("failed to decode search result JSON: %w", err)
	}

	return &envelope, nil
}

// GetSeriesDetails queries TMDB for detailed metadata fields belonging to a specific series ID.
func (c *TmdbClient) GetSeriesDetails(seriesID int64) (*models.TmdbSeriesDetails, error) {
	reqURL := fmt.Sprintf("%s/tv/%d?language=en-US", tmdbBaseURL, seriesID)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct series details request: %w", err)
	}

	req.Header.Add("Authorization", c.bearerAuth)
	req.Header.Add("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("outbound details network invocation failure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream details API returned bad status: %s", resp.Status)
	}

	var details models.TmdbSeriesDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to decode series details JSON: %w", err)
	}

	return &details, nil
}
