package main

import (
	"fmt"
	"github.com/Cfirth725/tv-sentinel/pkg/database"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("🚀 Launching TV Sentinel...")

	config, err := database.LoadConfig("config.json")
	if err != nil {
		slog.Error("❌ Critical Failure: Unable to parse config.json", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "💚 TV Sentinel System Health: OPERATIONAL")
	})

	slog.Info("📡 Network socket successfully bound", "port", config.Port)
	if err := http.ListenAndServe(config.Port, mux); err != nil {
		slog.Error("❌ Critical Failure: Network socket server crashed", "error", err)
		os.Exit(1)
	}
}
