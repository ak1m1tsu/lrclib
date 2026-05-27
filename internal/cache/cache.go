package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // CGO-free SQLite driver
)

const (
	defaultTTL = 7 * 24 * time.Hour
	driver     = "sqlite"
)

// Entry is what gets stored and retrieved from the cache.
type Entry struct {
	ID           int
	TrackName    string
	ArtistName   string
	AlbumName    string
	PlainLyrics  string
	SyncedLyrics string
}

// Cache is a SQLite-backed lyrics cache.
type Cache struct {
	db  *sql.DB
	ttl time.Duration
	log *slog.Logger
}

// Option configures a Cache.
type Option func(*Cache)

// WithTTL overrides the default 7-day TTL.
func WithTTL(d time.Duration) Option {
	return func(c *Cache) { c.ttl = d }
}

// WithLogger sets the logger used for WARN-level cache errors.
func WithLogger(l *slog.Logger) Option {
	return func(c *Cache) { c.log = l }
}

// Open opens (or creates) the cache database at the platform-default location.
// The caller must call Close when done.
func Open(opts ...Option) (*Cache, error) {
	path, err := DBPath()
	if err != nil {
		return nil, fmt.Errorf("cache: resolve db path: %w", err)
	}
	return OpenAt(path, opts...)
}

// OpenAt opens (or creates) the cache database at a specific path.
// Useful for tests.
func OpenAt(path string, opts ...Option) (*Cache, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("cache: create db dir: %w", err)
	}

	db, err := sql.Open(driver, path)
	if err != nil {
		return nil, fmt.Errorf("cache: open db: %w", err)
	}

	if err := applyPragmas(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cache: apply pragmas: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cache: migrate: %w", err)
	}

	c := &Cache{
		db:  db,
		ttl: defaultTTL,
		log: slog.Default(),
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// Close releases the database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}

// Get returns a cached entry for the given track ID.
// Returns (nil, nil) on a cache miss or if the entry is expired.
// Cache errors are logged as WARN and treated as a miss.
func (c *Cache) Get(ctx context.Context, trackID string) (*Entry, error) {
	const q = `
		SELECT data FROM lyrics_cache
		WHERE track_id = ?
		  AND (cached_at + ttl) > ?`

	row := c.db.QueryRowContext(ctx, q, trackID, time.Now().Unix())

	var blob []byte
	if err := row.Scan(&blob); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		c.log.Warn("cache get failed", "track_id", trackID, "err", err)
		return nil, nil //nolint:nilerr // cache errors are non-fatal
	}

	var entry Entry
	if err := json.Unmarshal(blob, &entry); err != nil {
		c.log.Warn("cache decode failed", "track_id", trackID, "err", err)
		return nil, nil //nolint:nilerr // cache errors are non-fatal
	}

	return &entry, nil
}

// Set stores or updates an entry in the cache.
// Cache errors are logged as WARN and not propagated.
func (c *Cache) Set(ctx context.Context, trackID string, entry *Entry) {
	blob, err := json.Marshal(entry)
	if err != nil {
		c.log.Warn("cache encode failed", "track_id", trackID, "err", err)
		return
	}

	const q = `
		INSERT OR REPLACE INTO lyrics_cache (track_id, data, cached_at, ttl)
		VALUES (?, ?, ?, ?)`

	if _, err := c.db.ExecContext(ctx, q, trackID, blob, time.Now().Unix(), int64(c.ttl.Seconds())); err != nil {
		c.log.Warn("cache set failed", "track_id", trackID, "err", err)
	}
}

// Delete removes a specific entry from the cache.
func (c *Cache) Delete(ctx context.Context, trackID string) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM lyrics_cache WHERE track_id = ?`, trackID)
	if err != nil {
		return fmt.Errorf("cache: delete %s: %w", trackID, err)
	}
	return nil
}

// Clear removes all entries from the cache.
func (c *Cache) Clear(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM lyrics_cache`)
	if err != nil {
		return fmt.Errorf("cache: clear: %w", err)
	}
	return nil
}

// Stats returns the number of entries and total size of the cache database.
type Stats struct {
	Entries   int
	SizeBytes int64
	DBPath    string
}

// Stat returns basic cache statistics.
func (c *Cache) Stat(ctx context.Context) (Stats, error) {
	var count int
	if err := c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lyrics_cache`).Scan(&count); err != nil {
		return Stats{}, fmt.Errorf("cache: stat count: %w", err)
	}
	return Stats{Entries: count}, nil
}

// DBPath returns the platform-appropriate path for the cache database.
func DBPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cache: resolve cache dir: %w", err)
	}
	return filepath.Join(dir, "lrclib", "cache.db"), nil
}

// applyPragmas sets the required SQLite connection pragmas.
func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA busy_timeout=5000`,
		`PRAGMA synchronous=NORMAL`,
		`PRAGMA foreign_keys=ON`,
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}
	return nil
}

// migrate creates the schema if it does not exist.
func migrate(db *sql.DB) error {
	const ddl = `
		CREATE TABLE IF NOT EXISTS lyrics_cache (
			track_id  TEXT    PRIMARY KEY,
			data      BLOB    NOT NULL,
			cached_at INTEGER NOT NULL,
			ttl       INTEGER NOT NULL DEFAULT 604800
		);
		CREATE INDEX IF NOT EXISTS idx_lyrics_cache_expiry
			ON lyrics_cache (cached_at + ttl);`

	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}
