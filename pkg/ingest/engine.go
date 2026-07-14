// Package ingest manages asynchronous worker pools, background task queuing,
// and transactional data streaming gates for media payload synchronization.
package ingest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Cfirth725/tv-sentinel/pkg/database"
	"github.com/Cfirth725/tv-sentinel/pkg/metadata"
	"github.com/Cfirth725/tv-sentinel/pkg/models"
	"github.com/Cfirth725/tv-sentinel/pkg/parser"
)

// ====================================================================
//                -- ENGINE STRUCT & INITIALIZATION --
// ====================================================================

// IngestionEngine manages the thread-safe async buffer queue and orchestrates
// background database workers, decoupling the API from the storage layer.
type IngestionEngine struct {
	db          *sql.DB
	tmdbClient  *metadata.TmdbClient
	queue       chan models.TvIngestPayload
	workerCount int
	wg          sync.WaitGroup
	activeTasks int64
	isShutting  int32
	idlePrinted uint32
}

// NewIngestionEngine constructs the core pipeline component, initializing the thread-safe
// buffered channel and binding the upstream synchronization client.
func NewIngestionEngine(db *sql.DB, client *metadata.TmdbClient, workerCount int) *IngestionEngine {
	return &IngestionEngine{
		db:          db,
		tmdbClient:  client,
		queue:       make(chan models.TvIngestPayload, 10000),
		workerCount: workerCount,
	}
}

// StartWorkerPool ignites the background routines using a pointer receiver to ensure
// references map to the active memory context instead of duplicating the queue.
func (e *IngestionEngine) StartWorkerPool() {
	slog.Info("[INIT] Activating asynchronous worker pool...", "workers", e.workerCount)
	for i := 1; i <= e.workerCount; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}
}

// Stop safely flushes remaining queued frames, locks out incoming requests, and tears down worker channels.
func (e *IngestionEngine) Stop() {
	if !atomic.CompareAndSwapInt32(&e.isShutting, 0, 1) {
		return
	}

	slog.Info("[SHUTDOWN] Terminating ingestion entry pipeline, flushing channels...",
		"subsystem", "engine",
		"remaining_in_buffer", len(e.queue),
	)

	close(e.queue)
	e.wg.Wait()

	slog.Info("[SHUTDOWN] Ingestion processor pool successfully terminated and unallocated",
		"subsystem", "engine",
	)
}

// ====================================================================
//             -- BACKGROUND ASYNC ROUTINE WORKER POOLS --
// ====================================================================

// worker represents an autonomous background consumer method that continuously drains the central channel.
// It incorporates a lock-free double-check read-through cache layer to shield upstream API resources.
func (e *IngestionEngine) worker(workerID int) {
	defer e.wg.Done()
	slog.Debug("[INIT] Background routine worker pool listener initialized", "worker_id", workerID)

	for payload := range e.queue {
		e.processPayload(workerID, payload)
		atomic.AddInt64(&e.activeTasks, -1)

		if len(e.queue) == 0 {
			if atomic.CompareAndSwapUint32(&e.idlePrinted, 0, 1) {
				fmt.Println("\n\033[1;32m================ MIGRATION COMPLETELY FINISHED ================\033[0m")
				slog.Info("[IDLE] Processing stream exhausted. Pipeline idle state achieved.")
			}
		} else {
			atomic.StoreUint32(&e.idlePrinted, 0)
		}
	}
}

// processPayload handles the core transactional tracking sequence.
func (e *IngestionEngine) processPayload(workerID int, payload models.TvIngestPayload) {
	normalized := parser.NormalizeTvEntry(payload.SeriesTitle)
	title := normalized.BaseTitle
	if title == "" {
		slog.Error("[ERROR] Processing exception: payload text string resolves to empty base token",
			"subsystem", "engine",
			"worker_id", workerID,
			"raw_input", payload.SeriesTitle,
		)
		return
	}

	stagingQuery := `
		INSERT INTO tv_ingest_staging_history (
			username, series_title, season_number, episode_number, sentiment, watched_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`

	_, err := e.db.Exec(stagingQuery,
		payload.Username,
		title,
		normalized.SeasonNumber,
		normalized.EpisodeNumber,
		payload.Sentiment,
		payload.WatchedAt,
	)
	if err != nil {
		slog.Error("[ERROR] Local staging ledger persistence failure",
			"subsystem", "engine",
			"worker_id", workerID,
			"user", payload.Username,
			"error", err.Error(),
		)
		return
	}

	slog.Debug("[REALTIME] Stream payload entry staged successfully to history logs",
		"subsystem", "engine",
		"worker_id", workerID,
		"user", payload.Username,
		"normalized_title", title,
	)

	user, err := database.GetUserByUsername(e.db, payload.Username)
	if err != nil {
		slog.Error("[ERROR] Core profile lookup failure", "subsystem", "engine", "username", payload.Username, "error", err.Error())
		return
	}
	if user == nil {
		slog.Warn("[ERROR] Action rejected: Target username profile missing from local engine database", "subsystem", "engine", "username", payload.Username)
		return
	}

	var localTvID int64

	cachedTv, err := database.GetTvByTitle(e.db, title)
	if err != nil {
		slog.Error("[ERROR] Database cache lookup runtime failure", "subsystem", "engine", "title", title, "error", err.Error())
	}

	if cachedTv != nil {
		slog.Info("[REALTIME] Cache HIT: Storage catalog cache verified. Bypassing upstream networking.",
			"subsystem", "engine",
			"worker_id", workerID,
			"catalog_id", cachedTv.ID,
			"title", cachedTv.TitleDisplay,
		)
		localTvID = cachedTv.ID
	} else {
		slog.Debug("[REALTIME] Cache MISS: Target missing from local catalog. Fetching external allocation.", "subsystem", "engine", "title", title)

		doubleCheckTv, checkErr := database.GetTvByTitle(e.db, title)
		if checkErr == nil && doubleCheckTv != nil {
			slog.Info("[REALTIME] Cache STAMPEDE MITIGATED: Catalog populated. Network call aborted.",
				"subsystem", "engine",
				"worker_id", workerID,
				"catalog_id", doubleCheckTv.ID,
				"title", doubleCheckTv.TitleDisplay,
			)
			localTvID = doubleCheckTv.ID
			goto updateProgress
		}

		searchEnvelope, err := e.tmdbClient.SearchTvSeries(title)
		if err != nil {
			slog.Warn("[REALTIME] Upstream metadata synchronization network request dropped", "subsystem", "engine", "title", title, "error", err.Error())
			return
		}

		if len(searchEnvelope.Results) == 0 {
			slog.Warn("[REALTIME] External match pass returned clean zero bounds", "subsystem", "engine", "title", title)
			return
		}

		topCandidate := searchEnvelope.Results[0]
		details, err := e.tmdbClient.GetSeriesDetails(topCandidate.ID)
		if err != nil {
			slog.Error("[ERROR] Upstream metadata specification pull failure", "subsystem", "engine", "series_id", topCandidate.ID, "error", err.Error())
			return
		}

		insertedID, err := database.InsertTvCatalog(
			e.db,
			strconv.FormatInt(details.ID, 10),
			title,
			details.Name,
			details.Status,
			details.Type,
			details.NumberOfSeasons,
		)
		if err != nil {
			slog.Error("[ERROR] Cache mapping persistence layer fault", "subsystem", "engine", "title", title, "error", err.Error())
			return
		}
		localTvID = insertedID

		// Cache structural episodic depths for each season returned by TMDB
		for _, s := range details.Seasons {
			if s.SeasonNumber < 1 {
				continue
			}
			err := database.InsertTvSeasonCount(e.db, localTvID, s.SeasonNumber, s.EpisodeCount)
			if err != nil {
				slog.Error("[ERROR] Failed to populate localized season episodic cache",
					"subsystem", "engine",
					"tv_id", localTvID,
					"season_number", s.SeasonNumber,
					"error", err.Error(),
				)
			}
		}
	}

updateProgress:
	err = database.UpsertTvWatchProgress(e.db, user.ID, localTvID, normalized.SeasonNumber, normalized.EpisodeNumber, payload.Sentiment)
	if err != nil {
		slog.Error("[ERROR] User state tracking engine checkpoint failure", "subsystem", "engine", "username", user.Username, "error", err.Error())
		return
	}

	slog.Info("[OK] Progress tracker update transaction committed",
		"subsystem", "engine",
		"username", user.Username,
		"title", title,
		"season", normalized.SeasonNumber,
		"episode", normalized.EpisodeNumber,
	)
}

// ====================================================================
//                -- HTTP PUBLIC ENTRYWAYS & GATES --
// ====================================================================

// HandleIngest serves as the high-performance HTTP gateway loop. It decodes body structures,
// executes baseline validation filters, and immediately offloads elements to the queue.
func (e *IngestionEngine) HandleIngest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if atomic.LoadInt32(&e.isShutting) == 1 {
		http.Error(w, "Server shutting down", http.StatusServiceUnavailable)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Missing required 'username' query parameter", http.StatusBadRequest)
		return
	}

	sentimentVal := 0
	if sentimentParam := r.URL.Query().Get("sentiment"); sentimentParam != "" {
		fmt.Sscanf(sentimentParam, "%d", &sentimentVal)
	}

	var envelope models.TvImportEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		slog.Error("[SERVER] JSON stream decoding fault on payload envelope drop", "error", err.Error())
		http.Error(w, "Invalid JSON structure", http.StatusBadRequest)
		return
	}

	var acceptedCount int
	for _, p := range envelope.Shows {
		p.Username = username
		p.Sentiment = sentimentVal

		if p.SeriesTitle == "" {
			continue
		}

		select {
		case e.queue <- p:
			atomic.AddInt64(&e.activeTasks, 1)
			acceptedCount++
		default:
			slog.Error("[ERROR] Resource bottleneck: Async core buffer channel max capacity hit. Dropping incoming packets.")
			http.Error(w, "Server Resource Saturation: Storage Buffer Exhausted", http.StatusServiceUnavailable)
			return
		}
	}

	duration := time.Since(start)
	slog.Info("[SERVER] Batch ingestion intake dispatch successful", "count", acceptedCount, "duration", duration)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "accepted",
		"received": len(envelope.Shows),
		"queued":   acceptedCount,
		"elapsed":  duration.String(),
	})
}
