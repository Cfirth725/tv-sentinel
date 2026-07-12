# 📺 TV Sentinel 
An independent, offline-first background service and REST API built in Go. It operates on local home lab hardware to aggregate, normalize, and track television series consumption for multiple isolated user profiles.

This repository serves as an integral component of **The Sentinel Suite**—a tri-repository microservice architecture designed to share a single data backend via infrastructure configuration rather than tightly coupled code.

---

## 🏗️ The Sentinel Suite Architecture

```
┌───────────────────────────────────────────────────┐
│            Media Sentinel (Future UI)             │
└──────────────────────────┬────────────────────────┘
                           │ (Aggregates API Feeds)
  ┌────────────────────────┼────────────────────────┐
  ▼                        ▼                        ▼
┌────────────────┐ ┌───────────────┐ ┌────────────────┐
│ Anime Sentinel │ │  TV Sentinel  │ │ Movie Sentinel │
│   (Repo #1)    │ │   (Repo #2)   │ │   (Repo #3)    │
└───────┬────────┘ └──────┬────────┘ └───────┬────────┘
        │                 │                  │
   └────────────────────┐ │ ┌──────────────────────┘
					    ▼ ▼ ▼
		┌───────────────────────────────────┐
		│      shared-sentinel-data/        │
		│        sentinel_suite.db          │
		└───────────────────────────────────┘
```

## Concurrent Background Processing Flow & Stampede Mitigation
```
[Gateway Intake]
 │  POST /api/v1/ingest (Sub-2ms validation)
 ▼
[Buffered Channel]
 │  Capacity: 10,000 tasks
 ▼
[Worker Routine Pool]
 │  4 Parallel Goroutines draining the channel
 ▼
[Step 1: Staging Log]
 │  Commit raw entry directly to 'ingest_staging_history'
 ▼
[Step 2: Read-Through Cache Lookup]
 │  Check local 'tv_catalog' table via Normalized lower-case CacheKey
 │
 ├───► (Cache HIT) ─────────────────────────────────────────────────┐
 │                                                                  │
 └───► (Cache MISS)                                                 │
       ▼                                                            │
 [Step 3: Double-Check Verification]                                │
       │  Query local DB catalog one more time                      │
       │                                                            │
       ├───► (Stampede Mitigated: HIT) ─────────────────────────────┤
       │                                                            │
       └───► (True Miss: Hit Network)                               │
             ▼                                                      │
       [TMDB REST API v4]                                           │
             │  Outbound query dispatch                             │
             ▼                                                      │
       [Cache Populate Layer]                                       │
             │  Insert fresh metadata row                           │
             ▼                                                      │
 [Update Progress State Engine] ────────────────────────────────────┴─► [TV Watch Progress Ledger]
```

## Core Philosophy & Constraints
1. **Zero External Runtime Dependencies:** Built strictly using the Go standard library (`net/http`, `slog`, `regexp`, `database/sql`, `sync/atomic`) to minimize container footprint and maximize execution velocity.
2. **Implicit Engagement Tracking:** Eliminates explicit user rating matrices. Taste anchors and enjoyment metrics are calculated programmatically through completion depth (**Engagement Score** $\ge$ 80%).
3. **Decoupled Data Infrastructure:** Configuration parameters point to a central, un-tracked SQLite file using Write-Ahead Logging (`WAL` mode) to allow multi-process concurrency across the suite.
4. **Structured DevOps Telemetry:** Built with consistent, scannable log tokens (`[INIT]`, `[SECURE]`, `[IDLE]`, `[REALTIME]`, `[SERVER]`, `[OK]`, `[ERROR]`, `[SHUTDOWN]`) for clean, production-grade terminal visibility.
5. **Graceful Pipeline Teardown:** Listens explicitly for OS lifecycle interrupts (`SIGINT`, `SIGTERM`). On capture, the API gateway locks down instantly, background workers fully drain existing channels, and SQLite connection pools execute an exclusive final checkpoint—collapsing active disk fragments cleanly.

---

## 🛠️ Tech Stack & Runtime
* **Language Runtime:** Go 1.24+ (Native structured logging, atomic concurrency primitives, and enhanced HTTP routing)
* **Database Engine:** SQLite 3 via `github.com/mattn/go-sqlite3`
* **Metadata Authority:** TMDB (The Movie Database) API via Header-Based Bearer Token Auth
* **Deployment Target:** Docker Multi-stage scratch container on the Milford Node

---

## 📂 System Topology
```
tv-sentinel/
├── cmd/
│ └── server/
│ └── main.go # HTTP Router, telemetry initialization, & application entry point
├── pkg/
│ ├── database/
│ │ ├── db.go # SQLite connection layer, WAL handlers, and core database DAOs
│ │ └── schema.sql # Relational schema extension rules & staging configurations
│ ├── ingest/
│ │ └── engine.go # Asynchronous worker pools, buffer queues, & HTTP gateway routines
│ ├── metadata/
│ │ └── client.go # Outbound TMDB REST client featuring authenticated bearer contexts
│ ├── models/
│ │ ├── tmdb.go # Upstream REST API JSON DTO payload data shapes
│ │ └── tv.go # Local domain core and ingestion transaction models
│ └── parser/
│ └── regex.go # Cascading TV title normalizer & episodic sequence extraction engine
├── config.json # Local execution configurations (Port, path targets, secret auth strings)
└── README.md # Project roadmap & technical specification
```

## 🗺️ Project Roadmap
### Phase 1: Core Scaffolding & Infrastructure (Completed)
- [x] Establish isolated repository workspace and module layout (`go mod init tv-sentinel`).
- [x] Construct standard library network health router loop on custom port architectures.
- [x] Configure local environment bridging via `config.json` targeting the shared database file.

### Phase 2: Metadata Matching & Parsing (Completed)
- [x] Author regex title normalizer to cleanly strip structural tokens (e.g., "SxxExx", "1x02") and isolate base series names.
- [x] Build robust database schemas extensions omitting strict status constraints for crowd-sourced metadata resilience.
- [x] Model core data access objects (DAOs) and wire shapes separated across cleanly mapped package structures.
- [x] Implement outbound REST pipeline with robust 10-second thread protection timeouts and header injection profiles.

### Phase 3: Ingestion Channels & Background Orchestrator Engine (Completed)
- [x] Build a high-volume, non-blocking asynchronous ingestion route utilizing a 10,000-capacity buffered channel to absorb tracking data.
- [x] Construct a concurrent background worker routine pool (4 workers) to process ingested payloads out-of-band using method receiver patterns.
- [x] Connect background workers to `parser.NormalizeTvEntry` for live title transformation out-of-band.
- [x] Integrate a lock-free **Double-Check Mechanism** to dynamically eliminate cache stampedes over concurrent ingestion bursts.
- [x] Introduce an **OS Signal Interceptor** to safely close background processes and enforce explicit SQLite WAL log file collapse during server shutdowns.

### Phase 4: Taste Analytical Intelligence (In Progress)
- [x] Design relational table structures (`tv_catalog_season_counts`) to cache episodic seasonal depths.
- [x] Establish defensive database DAO functions (`GetPendingCatchUpRadar`) to isolate unwatched episodes while excluding dropped series.
- [ ] Code the television-specific taste profile engine and calculate completion metrics based on the 80% engagement rule.
- [ ] Expose an analytical intelligence endpoint (`GET /api/v1/analytics/taste`) to resolve user preference anchors and the Catch-Up Radar.