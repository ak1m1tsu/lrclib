// Package api provides an HTTP client for the lrclib.net public API.
// It implements the fallback chain: direct lookup by metadata → search →
// error, with exponential-backoff retries and Retry-After support on 429.
package api
