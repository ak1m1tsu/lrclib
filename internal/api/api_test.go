package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ak1m1tsu/lrclib/internal/api"
)

// serve is a helper that registers a handler on path and returns a test server + client.
func serve(t *testing.T, path string, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := api.New(api.WithBaseURL(srv.URL))
	return srv, client
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestGet_DirectLookupSuccess(t *testing.T) {
	want := api.Lyrics{
		TrackName:    "Creep",
		ArtistName:   "Radiohead",
		SyncedLyrics: "[00:00.00]When you were here before",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/get", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, want)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := api.New(api.WithBaseURL(srv.URL))
	got, err := c.Get(context.Background(), api.TrackMeta{TrackName: "Creep", ArtistName: "Radiohead"})
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.SyncedLyrics != want.SyncedLyrics {
		t.Errorf("SyncedLyrics = %q, want %q", got.SyncedLyrics, want.SyncedLyrics)
	}
}

func TestGet_FallsBackToSearch(t *testing.T) {
	searchResult := api.Lyrics{TrackName: "Creep", PlainLyrics: "plain text"}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/get", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []api.Lyrics{searchResult})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := api.New(api.WithBaseURL(srv.URL))
	got, err := c.Get(context.Background(), api.TrackMeta{TrackName: "Creep", ArtistName: "Radiohead"})
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.PlainLyrics != searchResult.PlainLyrics {
		t.Errorf("PlainLyrics = %q, want %q", got.PlainLyrics, searchResult.PlainLyrics)
	}
}

func TestGet_BothPathsFail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/get", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []api.Lyrics{})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := api.New(api.WithBaseURL(srv.URL))
	_, err := c.Get(context.Background(), api.TrackMeta{TrackName: "unknown"})
	if err == nil {
		t.Error("expected error when both paths fail, got nil")
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestGetByID_Success(t *testing.T) {
	want := api.Lyrics{ID: 42, TrackName: "Test"}
	_, c := serve(t, "/api/get/42", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, want)
	})

	got, err := c.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %d, want %d", got.ID, want.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	_, c := serve(t, "/api/get/99", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetByID(context.Background(), 99)
	if err == nil {
		t.Error("expected error for 404, got nil")
	}
}

// ── Search ────────────────────────────────────────────────────────────────────

func TestSearch_ReturnsFirstResult(t *testing.T) {
	results := []api.Lyrics{
		{TrackName: "First"},
		{TrackName: "Second"},
	}
	_, c := serve(t, "/api/search", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, results)
	})

	got, err := c.Search(context.Background(), "query")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if got.TrackName != "First" {
		t.Errorf("TrackName = %q, want %q", got.TrackName, "First")
	}
}

func TestSearch_EmptyResults(t *testing.T) {
	_, c := serve(t, "/api/search", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []api.Lyrics{})
	})

	_, err := c.Search(context.Background(), "nothing")
	if err == nil {
		t.Error("expected error for empty results, got nil")
	}
}

// ── Error handling ────────────────────────────────────────────────────────────

func TestGet_ContextCancelled(t *testing.T) {
	_, c := serve(t, "/api/get", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Get(ctx, api.TrackMeta{TrackName: "x"})
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

func TestError_Message(t *testing.T) {
	_, c := serve(t, "/api/get/1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}
