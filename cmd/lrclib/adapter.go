package main

import (
	"context"
	"fmt"

	"github.com/ak1m1tsu/lrclib/internal/api"
	"github.com/ak1m1tsu/lrclib/internal/cache"
	"github.com/ak1m1tsu/lrclib/internal/errs"
	"github.com/ak1m1tsu/lrclib/internal/lrc"
	"github.com/ak1m1tsu/lrclib/internal/usecase"
)

// apiFetcher adapts api.Client to usecase.Fetcher.
type apiFetcher struct{ client *api.Client }

func (a *apiFetcher) Get(ctx context.Context, q usecase.Query) (*lrc.Track, error) {
	lyrics, err := a.client.Get(ctx, api.TrackMeta{
		TrackName:  q.TrackName,
		ArtistName: q.ArtistName,
		AlbumName:  q.AlbumName,
		Duration:   q.Duration,
	})
	if err != nil {
		return nil, err
	}
	return lyricsToTrack(lyrics)
}

// lyricsToTrack converts an api.Lyrics response to an lrc.Track.
// Prefers SyncedLyrics; falls back to PlainLyrics.
func lyricsToTrack(l *api.Lyrics) (*lrc.Track, error) {
	if l.SyncedLyrics != "" {
		t, err := lrc.ParseSynced(l.SyncedLyrics)
		if err == nil {
			t.Title = l.TrackName
			t.Artist = l.ArtistName
			t.Album = l.AlbumName
			t.Length = l.Duration
			return t, nil
		}
	}
	if l.PlainLyrics != "" {
		t, err := lrc.ParsePlain(l.PlainLyrics)
		if err == nil {
			t.Title = l.TrackName
			t.Artist = l.ArtistName
			t.Album = l.AlbumName
			t.Length = l.Duration
			return t, nil
		}
	}
	return nil, errs.NotFound(fmt.Sprintf("no usable lyrics for %q by %q", l.TrackName, l.ArtistName))
}

// apiSearcher adapts api.Client.SearchAll to usecase.Searcher.
type apiSearcher struct{ client *api.Client }

func (a *apiSearcher) Search(ctx context.Context, query string) ([]*lrc.Track, error) {
	results, err := a.client.SearchAll(ctx, query)
	if err != nil {
		return nil, err
	}
	tracks := make([]*lrc.Track, 0, len(results))
	for i := range results {
		t, err := lyricsToTrack(&results[i])
		if err != nil {
			continue
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}

// cacheAdapter adapts cache.Cache to usecase.LyricsCache.
type cacheAdapter struct{ c *cache.Cache }

func (a *cacheAdapter) Get(ctx context.Context, key string) (*lrc.Track, error) {
	entry, err := a.c.Get(ctx, key)
	if err != nil || entry == nil {
		return nil, err
	}
	return entryToTrack(entry)
}

func (a *cacheAdapter) Set(ctx context.Context, key string, track *lrc.Track) {
	a.c.Set(ctx, key, trackToEntry(track))
}

func entryToTrack(e *cache.Entry) (*lrc.Track, error) {
	if e.SyncedLyrics != "" {
		t, err := lrc.ParseSynced(e.SyncedLyrics)
		if err == nil {
			t.Title = e.TrackName
			t.Artist = e.ArtistName
			t.Album = e.AlbumName
			return t, nil
		}
	}
	if e.PlainLyrics != "" {
		t, err := lrc.ParsePlain(e.PlainLyrics)
		if err == nil {
			t.Title = e.TrackName
			t.Artist = e.ArtistName
			t.Album = e.AlbumName
			return t, nil
		}
	}
	return nil, nil
}

func trackToEntry(t *lrc.Track) *cache.Entry {
	entry := &cache.Entry{
		TrackName:  t.Title,
		ArtistName: t.Artist,
		AlbumName:  t.Album,
	}
	if t.IsPlain {
		entry.PlainLyrics = lrc.Format(t)
	} else {
		entry.SyncedLyrics = lrc.Format(t)
	}
	return entry
}
