package usecase_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ak1m1tsu/lrclib/internal/lrc"
	"github.com/ak1m1tsu/lrclib/internal/usecase"
)

// ── mock implementations ──────────────────────────────────────────────────────

type mockFetcher struct {
	track *lrc.Track
	err   error
	calls int
}

func (m *mockFetcher) Get(_ context.Context, _ usecase.Query) (*lrc.Track, error) {
	m.calls++
	return m.track, m.err
}

type mockCache struct {
	store map[string]*lrc.Track
	sets  int
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]*lrc.Track)}
}

func (m *mockCache) Get(_ context.Context, key string) (*lrc.Track, error) {
	return m.store[key], nil
}

func (m *mockCache) Set(_ context.Context, key string, track *lrc.Track) {
	m.store[key] = track
	m.sets++
}

// ── helpers ───────────────────────────────────────────────────────────────────

func syncedTrack() *lrc.Track {
	return &lrc.Track{
		Title:  "Creep",
		Artist: "Radiohead",
		Lines:  []lrc.Line{{Timestamp: time.Second, Text: "When you were here before"}},
	}
}

// ── GetLyrics ─────────────────────────────────────────────────────────────────

func TestGetLyrics_FetchAndCache(t *testing.T) {
	fetcher := &mockFetcher{track: syncedTrack()}
	cache := newMockCache()
	svc := usecase.New(fetcher, cache, nil)

	q := usecase.Query{TrackName: "Creep", ArtistName: "Radiohead"}
	track, err := svc.GetLyrics(context.Background(), q)
	if err != nil {
		t.Fatalf("GetLyrics error: %v", err)
	}
	if track.Title != "Creep" {
		t.Errorf("Title = %q, want Creep", track.Title)
	}
	if fetcher.calls != 1 {
		t.Errorf("fetcher called %d times, want 1", fetcher.calls)
	}
	if cache.sets != 1 {
		t.Errorf("cache.Set called %d times, want 1", cache.sets)
	}
}

func TestGetLyrics_CacheHit_SkipsFetcher(t *testing.T) {
	fetcher := &mockFetcher{track: syncedTrack()}
	cache := newMockCache()
	svc := usecase.New(fetcher, cache, nil)

	q := usecase.Query{TrackName: "Creep", ArtistName: "Radiohead"}

	// First call — populates cache.
	if _, err := svc.GetLyrics(context.Background(), q); err != nil {
		t.Fatalf("first GetLyrics: %v", err)
	}

	// Second call — should hit cache.
	if _, err := svc.GetLyrics(context.Background(), q); err != nil {
		t.Fatalf("second GetLyrics: %v", err)
	}

	if fetcher.calls != 1 {
		t.Errorf("fetcher called %d times, want 1 (second call should use cache)", fetcher.calls)
	}
}

func TestGetLyrics_NilCache_StillWorks(t *testing.T) {
	fetcher := &mockFetcher{track: syncedTrack()}
	svc := usecase.New(fetcher, nil, nil)

	_, err := svc.GetLyrics(context.Background(), usecase.Query{TrackName: "Creep"})
	if err != nil {
		t.Fatalf("GetLyrics with nil cache: %v", err)
	}
}

func TestGetLyrics_FetcherError_Propagated(t *testing.T) {
	fetcher := &mockFetcher{err: errors.New("network error")}
	svc := usecase.New(fetcher, newMockCache(), nil)

	_, err := svc.GetLyrics(context.Background(), usecase.Query{TrackName: "X"})
	if err == nil {
		t.Error("expected error from fetcher, got nil")
	}
}

// ── CacheKey ─────────────────────────────────────────────────────────────────

func TestQuery_CacheKey_Lowercase(t *testing.T) {
	q := usecase.Query{ArtistName: "Radiohead", TrackName: "Creep", AlbumName: "Pablo Honey"}
	key := q.CacheKey()
	if key != "radiohead:creep:pablo honey" {
		t.Errorf("CacheKey = %q, want %q", key, "radiohead:creep:pablo honey")
	}
}

func TestQuery_CacheKey_EmptyAlbum(t *testing.T) {
	q1 := usecase.Query{ArtistName: "A", TrackName: "B"}
	q2 := usecase.Query{ArtistName: "A", TrackName: "B", AlbumName: ""}
	if q1.CacheKey() != q2.CacheKey() {
		t.Errorf("keys should match when album is empty: %q vs %q", q1.CacheKey(), q2.CacheKey())
	}
}

// ── SaveLRC ───────────────────────────────────────────────────────────────────

func TestSaveLRC_WritesFile(t *testing.T) {
	svc := usecase.New(&mockFetcher{}, nil, nil)
	track := syncedTrack()
	path := filepath.Join(t.TempDir(), "out", "creep.lrc")

	if err := svc.SaveLRC(track, path); err != nil {
		t.Fatalf("SaveLRC error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if len(data) == 0 {
		t.Error("written file is empty")
	}
}

func TestSaveLRC_CreatesParentDirs(t *testing.T) {
	svc := usecase.New(&mockFetcher{}, nil, nil)
	path := filepath.Join(t.TempDir(), "a", "b", "c", "creep.lrc")

	if err := svc.SaveLRC(syncedTrack(), path); err != nil {
		t.Fatalf("SaveLRC error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}
