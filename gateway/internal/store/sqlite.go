package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db        *sql.DB
	dataDir   string
	writeLock sync.Mutex
}

type migration struct {
	version int
	name    string
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		name:    "initial_room_schema",
		sql: `
CREATE TABLE rooms (
    room_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    owner_username TEXT NOT NULL,
    owner_display_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
    queue_limit INTEGER NOT NULL CHECK (queue_limit BETWEEN 1 AND 100),
    playback_mode TEXT NOT NULL CHECK (playback_mode IN ('fifo', 'fair_random')),
    music_folder_ids_json TEXT NOT NULL,
    preload_next_track INTEGER NOT NULL DEFAULT 1,
    revision INTEGER NOT NULL DEFAULT 0,
    playback_status TEXT NOT NULL DEFAULT 'stopped' CHECK (playback_status IN ('stopped', 'playing', 'paused')),
    paused_for_empty INTEGER NOT NULL DEFAULT 0,
    position_seconds REAL NOT NULL DEFAULT 0,
    anchor_unix_ms INTEGER,
    current_track_json TEXT,
    current_contributor TEXT NOT NULL DEFAULT '',
    current_contributor_name TEXT NOT NULL DEFAULT '',
    current_history_id TEXT NOT NULL DEFAULT '',
    last_contributor TEXT NOT NULL DEFAULT '',
    created_unix_ms INTEGER NOT NULL,
    updated_unix_ms INTEGER NOT NULL
);

CREATE TABLE members (
    room_id TEXT NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    username TEXT NOT NULL COLLATE NOCASE,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    active INTEGER NOT NULL DEFAULT 1,
    joined_unix_ms INTEGER NOT NULL,
    last_seen_unix_ms INTEGER NOT NULL,
    PRIMARY KEY (room_id, username)
);

CREATE TABLE invites (
    invite_id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL DEFAULT '',
    expires_unix_ms INTEGER NOT NULL,
    max_uses INTEGER NOT NULL CHECK (max_uses BETWEEN 1 AND 10000),
    use_count INTEGER NOT NULL DEFAULT 0,
    revoked_unix_ms INTEGER,
    created_by TEXT NOT NULL,
    created_unix_ms INTEGER NOT NULL
);

CREATE TABLE queue (
    queue_id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    track_json TEXT NOT NULL,
    contributor_username TEXT NOT NULL COLLATE NOCASE,
    contributor_display_name TEXT NOT NULL,
    created_unix_ms INTEGER NOT NULL,
    UNIQUE (room_id, position)
);

CREATE TABLE playback_history (
    history_id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    track_json TEXT NOT NULL,
    contributor_username TEXT NOT NULL,
    contributor_display_name TEXT NOT NULL,
    started_unix_ms INTEGER NOT NULL,
    finished_unix_ms INTEGER,
    played_seconds REAL NOT NULL DEFAULT 0
);

CREATE TABLE security_audit (
    audit_id INTEGER PRIMARY KEY AUTOINCREMENT,
    occurred_unix_ms INTEGER NOT NULL,
    username TEXT NOT NULL,
    room_id TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE plugin_state (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    plugin_version TEXT NOT NULL,
    generation INTEGER NOT NULL,
    navidrome_public_url TEXT NOT NULL,
    gateway_public_url TEXT NOT NULL,
    users_json TEXT NOT NULL,
    last_heartbeat_unix_ms INTEGER NOT NULL
);

CREATE TABLE update_state (
    key TEXT PRIMARY KEY,
    value_json TEXT NOT NULL,
    updated_unix_ms INTEGER NOT NULL
);
`,
	},
	{
		version: 2,
		name:    "query_indexes",
		sql: `
CREATE INDEX idx_members_username ON members(username, active);
CREATE INDEX idx_invites_room ON invites(room_id, created_unix_ms DESC);
CREATE INDEX idx_queue_room_position ON queue(room_id, position);
CREATE INDEX idx_queue_contributor ON queue(room_id, contributor_username);
CREATE INDEX idx_history_room_started ON playback_history(room_id, started_unix_ms DESC);
CREATE INDEX idx_audit_room_time ON security_audit(room_id, occurred_unix_ms DESC);
`,
	},
	{
		version: 3,
		name:    "plugin_license_file",
		sql: `
ALTER TABLE plugin_state ADD COLUMN license_file TEXT NOT NULL DEFAULT '';
`,
	},
	{
		version: 4,
		name:    "plugin_update_channel",
		sql: `
ALTER TABLE plugin_state ADD COLUMN update_channel TEXT NOT NULL DEFAULT 'stable';
`,
	},
}

func Open(ctx context.Context, databasePath, dataDir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(databasePath) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	store := &Store{db: db, dataDir: dataDir}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := os.Chmod(databasePath, 0o600); err != nil {
		db.Close()
		return nil, fmt.Errorf("secure database: %w", err)
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_unix_ms INTEGER NOT NULL
)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	var current int
	if err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&current); err != nil {
		return err
	}
	pending := false
	for _, item := range migrations {
		if item.version > current {
			pending = true
			break
		}
	}
	if current > 0 && pending {
		if _, err := s.Backup(ctx, "pre-migration"); err != nil {
			return fmt.Errorf("backup before migration: %w", err)
		}
	}
	for _, item := range migrations {
		if item.version <= current {
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, item.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d (%s): %w", item.version, item.name, err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations(version, name, applied_unix_ms) VALUES (?, ?, ?)",
			item.version, item.name, unixMillis(time.Now().UTC()),
		); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Backup(ctx context.Context, reason string) (string, error) {
	backupDir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", err
	}
	cleanReason := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, reason)
	path := filepath.Join(backupDir, fmt.Sprintf("rooms-%s-%s.sqlite3", time.Now().UTC().Format("20060102T150405.000000000Z"), cleanReason))
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func unixMillis(value time.Time) int64 { return value.UTC().UnixMilli() }

func fromUnixMillis(value int64) time.Time { return time.UnixMilli(value).UTC() }

func nullableTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	result := fromUnixMillis(value.Int64)
	return &result
}

func rollback(tx *sql.Tx, err *error) {
	if *err != nil {
		_ = tx.Rollback()
	}
}

func isNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
