// Package intelligence handles out-of-band computations for taste profile anchors,
// watch progress completion metrics, and Catch-Up Radar generation.
package intelligence

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/Cfirth725/tv-sentinel/pkg/database"
	"github.com/Cfirth725/tv-sentinel/pkg/models"
)

// ====================================================================
//             -- CORE ANALYTICAL INTELLIGENCE SERVICE --
// ====================================================================

// IntelligenceEngine orchestrates the retrieval and mapping of raw database
// tracking ledgers into programmatic taste profile metrics.
type IntelligenceEngine struct {
	db *sql.DB
}

// NewIntelligenceEngine instantiates a new calculations service handle.
func NewIntelligenceEngine(db *sql.DB) *IntelligenceEngine {
	return &IntelligenceEngine{db: db}
}

// GenerateTasteProfile compiles completion depths, extracts genre anchors
// based on the 80% engagement limit, and merges catch-up delta vectors.
func (ie *IntelligenceEngine) GenerateTasteProfile(userID int64, username string) (*models.TasteProfile, error) {
	slog.Info("[REALTIME] Generating programmatic taste profile context...",
		"subsystem", "intelligence",
		"user_id", userID,
		"username", username,
	)

	// 1. Fetch current catch-up radar deltas via database DAO
	radar, err := database.GetPendingCatchUpRadar(ie.db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to gather catch-up radar components: %w", err)
	}

	// 2. Query all running watch progress milestones for active series
	progressRows, err := ie.db.Query(`
		SELECT p.tv_id, c.title_display, p.current_season, p.current_episode
		FROM tv_watch_progress p
		JOIN tv_catalog c ON p.tv_id = c.id
		WHERE p.user_id = ? AND p.sentiment >= 0;`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve core watch progress rows: %w", err)
	}
	defer progressRows.Close()

	var completions []models.ShowCompletion
	anchorShowIDs := make([]int64, 0)

	for progressRows.Next() {
		var tvID int64
		var title string
		var curSeason, curEpisode int

		if err := progressRows.Scan(&tvID, &title, &curSeason, &curEpisode); err != nil {
			return nil, fmt.Errorf("failed to scan progress row: %w", err)
		}

		// Calculate total available episodes versus total completed episodes
		totalEpisodes, watchedEpisodes, err := ie.calculateEpisodeDepths(tvID, curSeason, curEpisode)
		if err != nil {
			slog.Error("[ERROR] Analytical depth calculation failed for catalog item",
				"subsystem", "intelligence",
				"tv_id", tvID,
				"error", err.Error(),
			)
			continue
		}

		var engagement float64
		if totalEpisodes > 0 {
			engagement = (float64(watchedEpisodes) / float64(totalEpisodes)) * 100.0
		}

		// Flag series as anchor interests if they cross the 80% threshold
		isAnchor := engagement >= 80.0
		if isAnchor {
			anchorShowIDs = append(anchorShowIDs, tvID)
		}

		completions = append(completions, models.ShowCompletion{
			TvID:             tvID,
			TitleDisplay:     title,
			EpisodesWatched:  watchedEpisodes,
			TotalEpisodes:    totalEpisodes,
			EngagementScore:  engagement,
			IsAnchorInterest: isAnchor,
		})
	}

	// 3. Aggregate preferred genre tags for all anchor shows
	genres, err := ie.extractAnchorGenres(anchorShowIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to extract anchor genre signatures: %w", err)
	}

	slog.Info("[OK] Taste profile compilation complete",
		"subsystem", "intelligence",
		"user_id", userID,
		"anchors_found", len(anchorShowIDs),
		"radar_items", len(radar),
	)

	return &models.TasteProfile{
		UserID:          userID,
		Username:        username,
		AnchorGenres:    genres,
		CatchUpRadar:    radar,
		CompletionStats: completions,
		GeneratedAt:     time.Now(),
	}, nil
}

// ====================================================================
//             -- INTERNAL MATHEMATICAL HELPER METHOD --
// ====================================================================

// calculateEpisodeDepths queries the local seasonal cache to determine both
// the absolute total episodes inside the catalog and how many the user has completed.
func (ie *IntelligenceEngine) calculateEpisodeDepths(tvID int64, curSeason, curEpisode int) (int, int, error) {
	rows, err := ie.db.Query(`
		SELECT season_number, total_episodes_count 
		FROM tv_catalog_season_counts 
		WHERE tv_id = ? 
		ORDER BY season_number ASC;`, tvID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	totalEpisodes := 0
	watchedEpisodes := 0

	for rows.Next() {
		var sNum, epCount int
		if err := rows.Scan(&sNum, &epCount); err != nil {
			return 0, 0, err
		}

		totalEpisodes += epCount

		if sNum < curSeason {
			watchedEpisodes += epCount
		} else if sNum == curSeason {
			watchedEpisodes += curEpisode
		}
	}

	return totalEpisodes, watchedEpisodes, nil
}

// extractAnchorGenres runs a distinct aggregation query over the tags table
// to isolate top-performing genre profiles.
func (ie *IntelligenceEngine) extractAnchorGenres(tvIDs []int64) ([]string, error) {
	if len(tvIDs) == 0 {
		return []string{}, nil
	}

	query := `
		SELECT DISTINCT name 
		FROM tv_catalog_tags 
		WHERE tv_id IN (SELECT tv_id FROM tv_watch_progress WHERE sentiment >= 0) 
		ORDER BY name ASC;`

	rows, err := ie.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	genres := make([]string, 0)
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		genres = append(genres, g)
	}

	return genres, nil
}
