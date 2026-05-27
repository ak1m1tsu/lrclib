package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	baseURL   = "https://lrclib.net"
	userAgent = "lrclib-cli/dev (github.com/ak1m1tsu/lrclib)"

	maxRetries  = 3
	backoffBase = 500 * time.Millisecond
	backoffMax  = 10 * time.Second
)

// TrackMeta holds the search parameters used to look up a track.
type TrackMeta struct {
	TrackName  string
	ArtistName string
	AlbumName  string
	Duration   float64 // seconds
}

// Lyrics is the response returned by the API for a single track.
type Lyrics struct {
	ID           int     `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

// Client is the lrclib.net HTTP client.
type Client struct {
	http      *http.Client
	baseURL   string
	userAgent string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (useful in tests).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient replaces the underlying http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// New creates a Client with sensible production timeouts.
func New(opts ...Option) *Client {
	transport := &http.Transport{
		MaxIdleConnsPerHost: 10,
		DisableCompression:  false,
	}

	c := &Client{
		http: &http.Client{
			Timeout:   20 * time.Second,
			Transport: transport,
		},
		baseURL:   baseURL,
		userAgent: userAgent,
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

// Get fetches lyrics for the given track metadata using the fallback chain:
//  1. GET /api/get → prefer SyncedLyrics
//  2. No SyncedLyrics → use PlainLyrics from same response
//  3. No match → GET /api/search → first result
//  4. Returns an error if both paths fail.
func (c *Client) Get(ctx context.Context, meta TrackMeta) (*Lyrics, error) {
	// Step 1 & 2: direct lookup by metadata
	lyr, err := c.getByMeta(ctx, meta)
	if err == nil {
		return lyr, nil
	}

	// Step 3: search fallback
	lyr, err = c.search(ctx, meta.TrackName+" "+meta.ArtistName)
	if err == nil {
		return lyr, nil
	}

	return nil, fmt.Errorf("api: no lyrics found for %q by %q: %w", meta.TrackName, meta.ArtistName, err)
}

// GetByID fetches lyrics by the lrclib numeric track ID.
func (c *Client) GetByID(ctx context.Context, id int) (*Lyrics, error) {
	var lyr Lyrics
	if err := c.do(ctx, fmt.Sprintf("/api/get/%d", id), nil, &lyr); err != nil {
		return nil, err
	}
	return &lyr, nil
}

// Search returns the first result for a free-text query.
func (c *Client) Search(ctx context.Context, query string) (*Lyrics, error) {
	return c.search(ctx, query)
}

// ── internal helpers ──────────────────────────────────────────────────────────

func (c *Client) getByMeta(ctx context.Context, meta TrackMeta) (*Lyrics, error) {
	params := url.Values{}
	params.Set("track_name", meta.TrackName)
	params.Set("artist_name", meta.ArtistName)
	if meta.AlbumName != "" {
		params.Set("album_name", meta.AlbumName)
	}
	if meta.Duration > 0 {
		params.Set("duration", strconv.FormatFloat(meta.Duration, 'f', -1, 64))
	}

	var lyr Lyrics
	if err := c.do(ctx, "/api/get", params, &lyr); err != nil {
		return nil, err
	}

	return &lyr, nil
}

func (c *Client) search(ctx context.Context, query string) (*Lyrics, error) {
	params := url.Values{}
	params.Set("q", query)

	var results []Lyrics
	if err := c.do(ctx, "/api/search", params, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("api: search returned no results for %q", query)
	}

	return &results[0], nil
}

// do executes a GET request with retry/backoff and decodes the JSON response.
func (c *Client) do(ctx context.Context, path string, params url.Values, out any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("api: build url: %w", err)
	}
	if params != nil {
		u.RawQuery = params.Encode()
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			wait := backoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return fmt.Errorf("api: build request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("api: request failed: %w", err)
			continue
		}

		lastErr = c.handleResponse(resp, out)
		if lastErr == nil {
			return nil
		}

		// Only retry on 429 or 5xx
		var apiErr *Error
		if isRetryable(lastErr, &apiErr) {
			if apiErr != nil && apiErr.RetryAfter > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(apiErr.RetryAfter):
				}
			}
			continue
		}

		return lastErr
	}

	return lastErr
}

func (c *Client) handleResponse(resp *http.Response, out any) error {
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return &Error{StatusCode: resp.StatusCode, Message: "not found"}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		ra := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &Error{StatusCode: resp.StatusCode, Message: "rate limited", RetryAfter: ra}
	}

	if resp.StatusCode >= 500 {
		return &Error{StatusCode: resp.StatusCode, Message: "server error"}
	}

	if resp.StatusCode != http.StatusOK {
		return &Error{StatusCode: resp.StatusCode, Message: "unexpected status"}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("api: read body: %w", err)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("api: decode response: %w", err)
	}

	return nil
}

// Error represents an HTTP-level error from the API.
type Error struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	return fmt.Sprintf("api: HTTP %d: %s", e.StatusCode, e.Message)
}

// backoff returns exponential backoff with jitter for the given attempt (1-based).
func backoff(attempt int) time.Duration {
	exp := min(backoffBase*(1<<attempt), backoffMax)
	jitter := time.Duration(rand.Int64N(int64(exp) / 2))
	return exp + jitter
}

func parseRetryAfter(s string) time.Duration {
	if s == "" {
		return 0
	}
	if secs, err := strconv.Atoi(s); err == nil {
		return time.Duration(secs) * time.Second
	}
	return 0
}

func isRetryable(err error, out **Error) bool {
	var apiErr *Error
	if ok := errorAs(err, &apiErr); ok {
		*out = apiErr
		return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
	}
	return false
}

// errorAs is a thin wrapper kept here to avoid importing errors in a hot path.
func errorAs(err error, target **Error) bool {
	for err != nil {
		if e, ok := err.(*Error); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
