package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ak1m1tsu/lrclib/internal/errs"
	"github.com/ak1m1tsu/lrclib/internal/lrc"
)

// Query holds the parameters for a lyrics lookup.
type Query struct {
	TrackName  string
	ArtistName string
	AlbumName  string
	Duration   float64 // seconds; zero means unknown
}

// CacheKey returns a lowercase canonical key for cache storage.
func (q Query) CacheKey() string {
	return strings.ToLower(q.ArtistName + ":" + q.TrackName + ":" + q.AlbumName)
}

// Fetcher retrieves lyrics from an external source (e.g. lrclib.net).
// Implemented by an adapter wrapping internal/api.Client.
type Fetcher interface {
	Get(ctx context.Context, q Query) (*lrc.Track, error)
}

// LyricsCache stores and retrieves previously fetched tracks.
// Implemented by an adapter wrapping internal/cache.Cache.
type LyricsCache interface {
	Get(ctx context.Context, key string) (*lrc.Track, error)
	Set(ctx context.Context, key string, track *lrc.Track)
}

// Service implements the lyrics lookup and file-save business logic.
type Service struct {
	fetcher Fetcher
	cache   LyricsCache
	log     *slog.Logger
}

// New creates a Service. cache may be nil (cache will be skipped).
func New(fetcher Fetcher, cache LyricsCache, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{fetcher: fetcher, cache: cache, log: log}
}

// GetLyrics returns lyrics for the given query.
// It checks the cache first; on a miss it calls the Fetcher and populates the cache.
func (s *Service) GetLyrics(ctx context.Context, q Query) (*lrc.Track, error) {
	key := q.CacheKey()

	if s.cache != nil {
		if track, _ := s.cache.Get(ctx, key); track != nil {
			s.log.Debug("cache hit", "key", key)
			return track, nil
		}
	}

	track, err := s.fetcher.Get(ctx, q)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.Set(ctx, key, track)
	}

	return track, nil
}

// SaveLRC writes track as an .lrc file to path.
// Parent directories are created automatically.
func (s *Service) SaveLRC(track *lrc.Track, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errs.Internal(fmt.Sprintf("create output directory for %s", path), err)
	}
	content := lrc.Format(track)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return errs.Internal(fmt.Sprintf("write %s", path), err)
	}
	return nil
}
