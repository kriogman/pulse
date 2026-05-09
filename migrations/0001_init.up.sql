CREATE TABLE IF NOT EXISTS monitors (
    id           TEXT    PRIMARY KEY,                        -- ULID
    name         TEXT    NOT NULL,
    type         TEXT    NOT NULL,                           -- http, tcp, ping...
    target       TEXT    NOT NULL,                           -- URL o host
    interval_sec INTEGER NOT NULL CHECK (interval_sec >= 5),
    timeout_ms   INTEGER NOT NULL CHECK (timeout_ms > 0),
    config_json  TEXT    NOT NULL DEFAULT '{}',              -- params específicos del tipo
    enabled      INTEGER NOT NULL DEFAULT 1,                 -- 0/1 (SQLite no tiene BOOLEAN)
    created_at   INTEGER NOT NULL,                           -- unix epoch segundos
    updated_at   INTEGER NOT NULL
    -- futuro: user_id TEXT REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS checks (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id    TEXT    NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at    INTEGER NOT NULL,                          -- unix epoch milisegundos
    duration_ms   INTEGER NOT NULL,
    status        TEXT    NOT NULL,                          -- up, down, degraded
    status_code   INTEGER,                                   -- HTTP status, si aplica
    error         TEXT,                                      -- mensaje si status=down
    metadata_json TEXT    NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_checks_monitor_started ON checks(monitor_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_monitors_enabled       ON monitors(enabled);
