package cache_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ak1m1tsu/lrclib/internal/cache"
)

func openTemp(t *testing.T, opts ...cache.Option) *cache.Cache {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cache.db")
	c, err := cache.OpenAt(path, opts...)
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestGetSet_RoundTrip(t *testing.T) {
	c := openTemp(t)
	ctx := context.Background()

	want := &cache.Entry{
		ID:           1,
		TrackName:    "Creep",
		ArtistName:   "Radiohead",
		SyncedLyrics: "[00:00.00]When you were here before",
	}

	c.Set(ctx, "track:1", want)

	got, err := c.Get(ctx, "track:1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.TrackName != want.TrackName {
		t.Errorf("TrackName = %q, want %q", got.TrackName, want.TrackName)
	}
	if got.SyncedLyrics != want.SyncedLyrics {
		t.Errorf("SyncedLyrics = %q, want %q", got.SyncedLyrics, want.SyncedLyrics)
	}
}

func TestGet_Miss(t *testing.T) {
	c := openTemp(t)
	got, err := c.Get(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil on miss, got %+v", got)
	}
}

func TestGet_ExpiredEntryMiss(t *testing.T) {
	// TTL of 1 nanosecond — entry expires immediately.
	c := openTemp(t, cache.WithTTL(time.Nanosecond))
	ctx := context.Background()

	c.Set(ctx, "track:exp", &cache.Entry{TrackName: "Expired"})

	// Give the TTL time to pass.
	time.Sleep(2 * time.Millisecond)

	got, err := c.Get(ctx, "track:exp")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for expired entry, got a result")
	}
}

func TestSet_Upsert(t *testing.T) {
	c := openTemp(t)
	ctx := context.Background()

	c.Set(ctx, "track:1", &cache.Entry{TrackName: "Original"})
	c.Set(ctx, "track:1", &cache.Entry{TrackName: "Updated"})

	got, err := c.Get(ctx, "track:1")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry after upsert")
	}
	if got.TrackName != "Updated" {
		t.Errorf("TrackName = %q, want Updated", got.TrackName)
	}
}

func TestDelete(t *testing.T) {
	c := openTemp(t)
	ctx := context.Background()

	c.Set(ctx, "track:del", &cache.Entry{TrackName: "ToDelete"})

	if err := c.Delete(ctx, "track:del"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	got, err := c.Get(ctx, "track:del")
	if err != nil {
		t.Fatalf("Get after delete error: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestClear(t *testing.T) {
	c := openTemp(t)
	ctx := context.Background()

	c.Set(ctx, "track:1", &cache.Entry{TrackName: "A"})
	c.Set(ctx, "track:2", &cache.Entry{TrackName: "B"})

	if err := c.Clear(ctx); err != nil {
		t.Fatalf("Clear error: %v", err)
	}

	stats, err := c.Stat(ctx)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if stats.Entries != 0 {
		t.Errorf("Entries = %d after clear, want 0", stats.Entries)
	}
}

func TestStat(t *testing.T) {
	c := openTemp(t)
	ctx := context.Background()

	c.Set(ctx, "track:1", &cache.Entry{TrackName: "A"})
	c.Set(ctx, "track:2", &cache.Entry{TrackName: "B"})

	stats, err := c.Stat(ctx)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if stats.Entries != 2 {
		t.Errorf("Entries = %d, want 2", stats.Entries)
	}
}

func TestDBPath_NonEmpty(t *testing.T) {
	p, err := cache.DBPath()
	if err != nil {
		t.Fatalf("DBPath error: %v", err)
	}
	if p == "" {
		t.Error("DBPath returned empty string")
	}
}
