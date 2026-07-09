package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Cfirth725/tv-sentinel/pkg/database"
	"github.com/Cfirth725/tv-sentinel/pkg/metadata"
)

func main() {
	// Initialize the centralized text handler formatting for local terminal streaming
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
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
	defer db.Close()

	// 3. Instantiate Outbound Upstream Metadata Gateway Client
	_ = metadata.NewTmdbClient(config.TmdbToken) // Initialized for Phase 4 engine linkage

	// 4. Construct Modern High-Performance Routing Multiplexer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "💚 TV Sentinel System Health: OPERATIONAL")
	})

	// Format port target parameters cleanly to guarantee standard socket bindings (e.g., ":8081")
	addr := ":" + config.Port

	slog.Info("[SERVER] Network socket successfully initialized and bound",
		"subsystem", "server",
		"event", "socket_active",
		"listen_address", addr,
	)

	// 5. Fire Up Blocked Network Transport Loop
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("[ERROR] Critical network execution failure: system socket crashed",
			"subsystem", "server",
			"error", err.Error(),
		)
		os.Exit(1)
	}
}
