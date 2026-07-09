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
