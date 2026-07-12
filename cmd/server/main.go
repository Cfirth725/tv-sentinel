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
	"github.com/Cfirth725/tv-sentinel/pkg/metadata"
)

func main() {
	// Initialize the centralized text handler formatting for local terminal streaming
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	slog.Info("[SERVER] Launching automated television media tracking service...",
		"subsystem", "server",
		"event", "boot_initiated",
	)

	// 1. Ingest Configuration Parameters
	config, err := database.LoadConfig("config.json")
	if err != nil {
		slog.Error("[ERROR] Critical bootstrapping failure: unable to parse configuration parameters",
			"subsystem", "server",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	// 2. Initialize Shared Persistent Database Concurrency Engine (WAL Pool)
	db, err := database.InitDatabase(config.DatabasePath, "pkg/database/schema.sql")
	if err != nil {
		slog.Error("[ERROR] Critical storage instantiation failure: pipeline blocked",
			"subsystem", "database",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	// 3. Instantiate Outbound Upstream Metadata Gateway Client
	tmdbClient := metadata.NewTmdbClient(config.TmdbToken)

	// 4. Allocate and start the high-capacity ingestion background worker routine pool (4 parallel threads)
	engine := ingest.NewIngestionEngine(db, tmdbClient, 4)
	engine.StartWorkerPool()

	// 5. Construct Modern High-Performance Routing Multiplexer
	mux := http.NewServeMux()

	// Core high-throughput streaming ingest gateway route
	mux.HandleFunc("POST /api/v1/ingest", engine.HandleIngest)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "💚 TV Sentinel System Health: OPERATIONAL")
	})

	// Format port target parameters cleanly to guarantee standard socket bindings (e.g., ":8093")
	addr := ":" + config.Port
	if config.Port[0] == ':' {
		addr = config.Port
	}

	// Configure the underlying HTTP Server wrapper to support controlled lifecycle shutdowns.
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Listen explicitly for operating system lifecycle terminal interrupts.
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	// Ignite the network listener socket in a non-blocking background routine.
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

	// Execution pauses here, unblocking only when an active OS signal drops into the channel.
	sig := <-shutdownSignal
	slog.Warn("[SHUTDOWN] Shutdown signal received! Initiating graceful pipeline teardown...", "signal", sig.String())

	// Execute the lifecycle teardown sequence in strict reverse order of instantiation.

	// 1. Force the gateway to stop accepting new connection threads and drain active requests.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("[ERROR] HTTP gateway server force-closed during teardown context", "error", err)
	} else {
		slog.Info("[SHUTDOWN] HTTP gateway server stopped successfully. Gateway locked.")
	}

	// 2. Shut down and drain the ingestion worker pools before severing database pools.
	slog.Info("[SHUTDOWN] Flushing asynchronous background worker threads...")
	engine.Stop()

	// 3. Sever connection links to the storage layer, forcing a final database WAL checkpoint
	// to cleanly collapse active journal files back into the core file on disk.
	slog.Info("[SHUTDOWN] Flushing Write-Ahead Logs and closing state storage connection pool...")
	if err := db.Close(); err != nil {
		slog.Error("[ERROR] Error encountered while severing database pool connection", "error", err)
	} else {
		slog.Info("[SHUTDOWN] Database engine disconnected cleanly. All journal files collapsed!")
	}

	slog.Info("[SHUTDOWN] TV Sentinel shutdown complete. System offline.")
}
