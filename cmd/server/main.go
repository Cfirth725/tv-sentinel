package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Cfirth725/tv-sentinel/pkg/database"
	"github.com/Cfirth725/tv-sentinel/pkg/ingest"
	"github.com/Cfirth725/tv-sentinel/pkg/intelligence"
	"github.com/Cfirth725/tv-sentinel/pkg/metadata"
)

func main() {
	// ====================================================================
	//         -- SERVICE INITIALIZATION & CONFIGURATION BOOTSTRAP --
	// ====================================================================

	// Set up structured text logging to standard output
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	slog.Info("[SERVER] Launching automated television media tracking service...",
		"subsystem", "server",
		"event", "boot_initiated",
	)

	// Load system configuration properties
	config, err := database.LoadConfig("config.json")
	if err != nil {
		slog.Error("[ERROR] Critical bootstrapping failure: unable to parse configuration parameters",
			"subsystem", "server",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	// Initialize the SQLite database connection with WAL mode enabled
	db, err := database.InitDatabase(config.DatabasePath, "pkg/database/schema.sql")
	if err != nil {
		slog.Error("[ERROR] Critical storage instantiation failure: pipeline blocked",
			"subsystem", "database",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	// Set up the TMDB metadata API client
	tmdbClient := metadata.NewTmdbClient(config.TmdbToken)

	// Initialize and start the asynchronous ingestion worker pool (4 parallel goroutines)
	engine := ingest.NewIngestionEngine(db, tmdbClient, 4)
	engine.StartWorkerPool()

	// Initialize the intelligence and analytical calculations engine
	intelEngine := intelligence.NewIntelligenceEngine(db)

	// ====================================================================
	//                -- HTTP ROUTING & PIPELINE MULTIPLEXER --
	// ====================================================================

	mux := http.NewServeMux()

	// Main async tracking ingestion route
	mux.HandleFunc("POST /api/v1/ingest", engine.HandleIngest)

	// User taste profile and Catch-Up Radar analytics route
	mux.HandleFunc("GET /api/v1/analytics/taste", intelEngine.HandleGetTasteProfile)

	// System health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "💚 TV Sentinel System Health: OPERATIONAL")
	})

	// ====================================================================
	//                -- RUNTIME SERVER CORE & OS SIGNAL LISTENERS --
	// ====================================================================

	// Format the listening port string correctly
	addr := ":" + config.Port
	if config.Port[0] == ':' {
		addr = config.Port
	}

	// Configure the HTTP server
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Listen for system interrupt signals to trigger graceful shutdown
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	// Start the HTTP server in a background goroutine
	go func() {
		slog.Info("[SERVER] Network socket successfully initialized and bound",
			"subsystem", "server",
			"event", "socket_active",
			"listen_address", addr,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("[ERROR] Critical network execution failure: system socket crashed",
				"subsystem", "server",
				"error", err.Error(),
			)
			os.Exit(1)
		}
	}()

	// Block main execution until a shutdown signal is received
	sig := <-shutdownSignal
	slog.Warn("[SHUTDOWN] Shutdown signal received! Initiating graceful pipeline teardown...", "signal", sig.String())

	// ====================================================================
	//               -- GRACEFUL PIPELINE TEARDOWN SEQUENCE --
	// ====================================================================

	// 1. Stop accepting new HTTP connections and drain in-flight requests
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("[ERROR] HTTP gateway server force-closed during teardown context", "error", err)
	} else {
		slog.Info("[SHUTDOWN] HTTP gateway server stopped successfully. Gateway locked.")
	}

	// 2. Stop the ingestion worker pool and let active background tasks finish
	slog.Info("[SHUTDOWN] Flushing asynchronous background worker threads...")
	engine.Stop()

	// 3. Close the database connection to let WAL checkpoint and collapse cleanly
	slog.Info("[SHUTDOWN] Flushing Write-Ahead Logs and closing state storage connection pool...")
	if err := db.Close(); err != nil {
		slog.Error("[ERROR] Error encountered while severing database pool connection", "error", err)
	} else {
		slog.Info("[SHUTDOWN] Database engine disconnected cleanly. All journal files collapsed!")
	}

	slog.Info("[SHUTDOWN] TV Sentinel shutdown complete. System offline.")
}
