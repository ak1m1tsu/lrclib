// Package cache implements a local SQLite-backed lyrics cache using
// modernc.org/sqlite (pure Go, no CGO). Cache misses and errors are
// non-fatal — callers receive a WARN log and fall back to the network.
package cache
