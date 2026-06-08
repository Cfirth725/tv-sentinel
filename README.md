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

## Core Philosophy & Constraints
1. **Zero External Dependencies:** Built strictly using the Go standard library (`net/http`, `slog`, `regexp`, `database/sql`) to minimize container footprint and maximize execution velocity.
2. **Implicit Engagement Tracking:** Eliminates explicit user rating matrices. Taste anchors and enjoyment metrics are calculated programmatically through completion depth (**Engagement Score** $\ge$ 80%).
3. **Decoupled Data Infrastructure:** Configuration parameters point to a central, un-tracked SQLite file using Write-Ahead Logging (`WAL` mode) to allow multi-process concurrency across the suite.

---

## 🛠️ Tech Stack & Runtime
* **Language Runtime:** Go 1.22+ (Enhanced standard routing patterns)
* **Database Engine:** SQLite 3 via `github.com/mattn/go-sqlite3`
* **Metadata Authority:** TMDB (The Movie Database) API
* **Deployment Target:** Docker Multi-stage scratch container on the Milford Node

---

## 📂 System Topology
```text
tv-sentinel/
├── cmd/
│   └── server/
│       └── main.go       # HTTP Router & dependency injection entry point
├── pkg/
│   ├── database/
│   │   └── db.go         # SQLite connection layer & WAL configuration
│   └── parser/
│       └── normalizer.go # TV title scrubbing & season/episode regex engine
├── config.json           # Local execution configuration
└── README.md             # Project roadmap & technical specification
```

## 🗺️ Project Roadmap
### Phase 1: Core Scaffolding & Infrastructure (In Progress)
- [ ] Establish isolated repository workspace and module layout (`go mod init tv-sentinel`).
- [ ] Construct standard library network health router loop on custom port `8093`.
- [ ] Configure local environment bridging via `config.json` targeting the shared database file.

### Phase 2: Metadata Matching & Parsing 
- [ ] Author regex title normalizer to cleanly strip structural tokens (e.g., "Season 1", "UK Edition") and isolate base series names.
- [ ] Build robust database schemas extensions if necessary or link cleanly to the unified core `media_catalog`.
- [ ] Model core data access objects (DAOs) inside `pkg/models`.

### Phase 3: Ingestion Channels & Co-Viewing Intelligence
- [ ] Build high-volume asynchronous ingestion routes utilizing buffered Go channels (`chan`) to absorb historical user data.
- [ ] Implement the television-specific taste profile calculator using the mathematical engagement index.