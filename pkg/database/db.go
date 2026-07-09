// Package database manages persistent SQLite transaction handles, schemas,
// and domain-specific data access objects across the tracking suite.
package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/Cfirth725/tv-sentinel/pkg/models"
	_ "github.com/mattn/go-sqlite3" // Enforce physical C-binding registration
)

// Config holds the environment configuration parameters parsed from disk.
type Config struct {
	Port         string `json:"PORT"`
	DatabasePath string `json:"DATABASE_PATH"`
	WALMode      bool   `json:"SQLITE_WAL_MODE"`
	TmdbToken    string `json:"TMDB_TOKEN"`
}

// LoadConfig opens a target JSON file and decodes its environment properties.
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config JSON: %w", err)
	}
	return &config, nil
}

// InitDatabase opens a persistent connection handle to the target SQLite file,
// forces Write-Ahead Logging concurrency parameters, and executes table scaffolding.
func InitDatabase(dbPath string, schemaPath string) (*sql.DB, error) {
	slog.Info("[INIT] Initializing persistent storage subsystem...", "target_path", dbPath)

	// Open the SQLite file pointer with optimization flags tuned for local labs
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_sync=NORMAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open persistent database instance: %w", err)
	}

	// Verify the connection pool handle is active before running queries
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("database ping validation failure: %w", err)
	}

	slog.Info("[INIT] Storage connection verified. Synchronizing extensions schema layout...")

	// Read and parse the local schema extension file lines
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load schema execution instructions from target file: %w", err)
	}

	// Append tables safely using structural 'IF NOT EXISTS' gates
	if _, err := db.Exec(string(schemaBytes)); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply schema extension parameters to database file: %w", err)
	}

	slog.Info("[INIT] Database layout verified. Shared suite extensions active.")
	return db, nil
}

// GetUserByUsername checks the shared database identity table for an active account profile.
// It returns a nil model pointer if the user account record does not exist.
func GetUserByUsername(db *sql.DB, username string) (*models.User, error) {
	// Reusable model framework placeholder structure (Assumes models.User exists)
	var user models.User
	query := `SELECT id, username, created_at FROM users WHERE username = ?;`

	err := db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return explicit nil bounds safely if profile is missing
		}
		return nil, fmt.Errorf("failed to execute username identity scan transaction: %w", err)
	}

	return &user, nil
}

// GetTvByTitle checks the localized TV catalog cache using a lower-case normalized search key.
// It returns a nil model pointer safely if the series metadata has not been synchronized yet.
func GetTvByTitle(db *sql.DB, cleanTitle string) (*models.TvCatalog, error) {
	var tv models.TvCatalog
	query := `
		SELECT id, external_id, cache_key, title_display, status, type, total_seasons_count, updated_at 
		FROM tv_catalog 
		WHERE cache_key = LOWER(?);`

	err := db.QueryRow(query, cleanTitle).Scan(
		&tv.ID,
		&tv.ExternalID,
		&tv.CacheKey,
		&tv.TitleDisplay,
		&tv.Status,
		&tv.Type,
		&tv.TotalSeasonsCount,
		&tv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("local catalog cache query execution failure: %w", err)
	}

	return &tv, nil
}

// InsertTvCatalog commits fresh upstream metadata retrieved from TMDB directly into the local cache directory.
// It returns the newly assigned internal auto-incremented database row identifier.
func InsertTvCatalog(db *sql.DB, extID, cacheKey, title, status, tvType string, totalSeasons int) (int64, error) {
	query := `
		INSERT INTO tv_catalog (external_id, cache_key, title_display, status, type, total_seasons_count, updated_at)
		VALUES (?, LOWER(?), ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(external_id) DO UPDATE SET
			status = excluded.status,
			total_seasons_count = excluded.total_seasons_count,
			updated_at = CURRENT_TIMESTAMP;`

	res, err := db.Exec(query, extID, cacheKey, title, status, tvType, totalSeasons)
	if err != nil {
		return 0, fmt.Errorf("failed to execute catalog cache persistence operation: %w", err)
	}

	insertedID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve auto-allocated internal catalog row identification: %w", err)
	}

	return insertedID, nil
}

// UpsertTvWatchProgress commits a progressive seasonal and episodic watch marker milestone checkpoint.
// If an entry for the user and show already exists, it upserts the progress sequentially.
func UpsertTvWatchProgress(db *sql.DB, userID, tvID int64, season, episode, sentiment int) error {
	query := `
		INSERT INTO tv_watch_progress (user_id, tv_id, current_season, current_episode, sentiment, last_watched_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, tv_id) DO UPDATE SET
			current_season = excluded.current_season,
			current_episode = excluded.current_episode,
			sentiment = excluded.sentiment,
			last_watched_at = CURRENT_TIMESTAMP;`

	_, err := db.Exec(query, userID, tvID, season, episode, sentiment)
	if err != nil {
		return fmt.Errorf("failed to commit tracking milestone checkpoint to watch progress ledger: %w", err)
	}

	return nil
}
