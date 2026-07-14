// Package intelligence coordinates analytical evaluation, tracking profile parsing,
// and mathematical taste anchor processing for active viewer accounts.
package intelligence

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

// HandleGetTasteProfile compiles and returns the complete programmatic taste profile metrics,
// including anchor genres, historical stats, and the active Catch-Up Radar.
func (ie *IntelligenceEngine) HandleGetTasteProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse target user ID parameter, defaulting to 1
	userIDStr := r.URL.Query().Get("user_id")
	userID := int64(1)
	if userIDStr != "" {
		id, err := strconv.ParseInt(userIDStr, 10, 64)
		if err == nil {
			userID = id
		}
	}

	// 2. Query user identity to resolve username for the profile payload
	var username string
	err := ie.db.QueryRow("SELECT username FROM users WHERE id = ?;", userID).Scan(&username)
	if err != nil {
		// Default to fallback system context if user profile has not been written yet
		username = "Default User"
	}

	// 3. Trigger the calculations engine pass
	profile, err := ie.GenerateTasteProfile(userID, username)
	if err != nil {
		slog.Error("[ERROR] Failed to compile programmatic taste profile analytics",
			"user_id", userID,
			"error", err.Error(),
		)
		http.Error(w, "Internal server error calculating analytics", http.StatusInternalServerError)
		return
	}

	// 4. Stream response payload cleanly back to client gateway
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		slog.Error("[ERROR] Failed to encode taste profile payload response",
			"user_id", userID,
			"error", err.Error(),
		)
	}
}
