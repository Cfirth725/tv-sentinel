-- ====================================================================
-- THE SENTINEL SUITE: UNIFIED TV EXTENSION SCHEMA (V2 - STANDARDIZED)
-- ====================================================================

-- --------------------------------------------------------------------
--          -- LOCAL USER IDENTITIES (SHARED SUITE ANCHOR) --
-- Included here via IF NOT EXISTS to guarantee cross-repo consistency.
-- Stores system account profiles responsible for active media tracking.
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- --------------------------------------------------------------------
--         -- NORMALIZED TV CATALOG (AUTONOMOUS CACHE LAYER) --
-- Acts as a read-through localized lookup layer to shield TMDB API quotas.
-- Flexible string definitions accommodate crowdsourced upstream metadata variations.
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tv_catalog (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    external_id TEXT NOT NULL UNIQUE,        -- TMDB serial string identifier
    cache_key TEXT NOT NULL UNIQUE,          -- Normalized lower-case base search key
    title_display TEXT NOT NULL,             -- Official presentation string title
    status TEXT,                             -- e.g., 'Returning Series', 'Ended', 'Post Production'
    type TEXT,                               -- e.g., 'Scripted', 'Reality', 'Miniseries'
    total_seasons_count INTEGER DEFAULT 1,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- --------------------------------------------------------------------
--    -- CATALOG SEASON STRUCTURAL DEPTHS (EPISODIC CACHE LAYER) --
-- Maps the expected total episode counts per season to facilitate local
-- progress delta logic without continuous upstream network round-trips.
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tv_catalog_season_counts (
    tv_id INTEGER,
    season_number INTEGER NOT NULL,
    total_episodes_count INTEGER NOT NULL,
    PRIMARY KEY (tv_id, season_number),
    FOREIGN KEY(tv_id) REFERENCES tv_catalog(id) ON DELETE CASCADE
);

-- --------------------------------------------------------------------
--    -- CATALOG METADATA WEIGHTS FOR AUTOMATED TASTE PROFILES --
-- Maps relational genre classifiers directly to physical TV catalog items.
-- Cascade deletions ensure that orphaned tags drop cleanly if a title is removed.
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tv_catalog_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tv_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY(tv_id) REFERENCES tv_catalog(id) ON DELETE CASCADE
);

-- --------------------------------------------------------------------
--               -- ISOLATED TV PROGRESS LEDGERS --
-- Evaluates real-time seasonal and episodic tracking checkpoints alongside
-- active affinity sentiment flags bound to individual unique user profiles.
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tv_watch_progress (
    user_id INTEGER,
    tv_id INTEGER,
    current_season INTEGER NOT NULL DEFAULT 1,
    current_episode INTEGER NOT NULL DEFAULT 1,
    sentiment INTEGER NOT NULL CHECK(sentiment IN (-1, 0, 1)) DEFAULT 0,
    last_watched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, tv_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(tv_id) REFERENCES tv_catalog(id) ON DELETE CASCADE
);

-- --------------------------------------------------------------------
--    -- HIGH-VOLUME INGESTION STAGING TABLE (ISOLATED TV SINK) --
-- Provides a dedicated staging sandbox to prevent cross-service lock contention.
-- Relaxes upstream relational constraints to maximize non-blocking input rates.
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tv_ingest_staging_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    series_title TEXT NOT NULL,
    season_number INTEGER NOT NULL,
    episode_number INTEGER NOT NULL,
    sentiment INTEGER NOT NULL CHECK (sentiment IN (-1, 0, 1)) DEFAULT 0,
    processed_status TEXT CHECK(processed_status IN ('PENDING', 'PROCESSED', 'FAILED')) DEFAULT 'PENDING',
    watched_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- --------------------------------------------------------------------
--              -- HIGH-PERFORMANCE QUERY OPTIMIZATIONS --
-- Explicitly constructed indexes to accelerate fast key scans, user profile updates,
-- and background processing engine task queries.
-- --------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_tv_catalog_lookup ON tv_catalog(cache_key);
CREATE INDEX IF NOT EXISTS idx_tv_progress_user ON tv_watch_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_tv_ingest_status ON tv_ingest_staging_history(processed_status);