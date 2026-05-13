package storage

// migration001 is the original reviewed_merge_requests table.
const migration001 = `
CREATE TABLE IF NOT EXISTS reviewed_merge_requests (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    client_name   TEXT     NOT NULL,
    project_ref   TEXT     NOT NULL,
    mr_iid        INTEGER  NOT NULL,
    mr_id         INTEGER,
    reviewed_user TEXT     NOT NULL,
    reviewed_at   DATETIME NOT NULL,
    UNIQUE(client_name, project_ref, mr_iid, reviewed_user)
);`

// migration002 adds the clients table and the review_events event log.
const migration002 = `
CREATE TABLE IF NOT EXISTS clients (
    id             INTEGER  PRIMARY KEY AUTOINCREMENT,
    name           TEXT     NOT NULL UNIQUE,
    platform       TEXT     NOT NULL,
    base_url       TEXT     NOT NULL,
    token          TEXT     NOT NULL,
    webhook_secret TEXT     NOT NULL,
    username       TEXT     NOT NULL,
    created_at     DATETIME NOT NULL,
    updated_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS review_events (
    id                INTEGER  PRIMARY KEY AUTOINCREMENT,
    client_name       TEXT     NOT NULL,
    project_ref       TEXT     NOT NULL,
    mr_iid            INTEGER  NOT NULL,
    mr_id             INTEGER,
    mr_title          TEXT     NOT NULL DEFAULT '',
    mr_url            TEXT     NOT NULL DEFAULT '',
    author_username   TEXT     NOT NULL DEFAULT '',
    prompt_tokens     INTEGER  NOT NULL DEFAULT 0,
    completion_tokens INTEGER  NOT NULL DEFAULT 0,
    total_tokens      INTEGER  NOT NULL DEFAULT 0,
    cost_usd          REAL     NOT NULL DEFAULT 0.0,
    duration_ms       INTEGER  NOT NULL DEFAULT 0,
    status            TEXT     NOT NULL DEFAULT 'success',
    error_message     TEXT,
    reviewed_at       DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_review_events_client
    ON review_events(client_name, reviewed_at DESC);

CREATE INDEX IF NOT EXISTS idx_review_events_time
    ON review_events(reviewed_at DESC);`
