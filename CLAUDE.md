# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## TL;DR

1. **Branches are required** — every task gets its own branch from `master`; commit after each micro-task
2. **Before every PR** — `make fmt && make lint && make test` must be green
3. **CGO_ENABLED=0 everywhere** — no exceptions; GoReleaser cross-compilation breaks otherwise
4. **Cache errors = WARN, not fatal** — log as WARN and continue without cache
5. **`context.Context` is always the first argument** in any function doing IO

---

## Project Overview

**lrclib** — cross-platform Go CLI for searching and downloading song lyrics from [lrclib.net](https://lrclib.net) and saving them as `.lrc` files. Includes a TUI for interactive search.

**Stack:** Go 1.26.3+, Cobra (CLI), Bubble Tea (TUI), `log/slog`, `modernc.org/sqlite` (cache, CGO-free), GoReleaser, GitHub Actions.

---

## Build & Commands

```bash
make build          # go build ./cmd/lrclib
make run            # go run ./cmd/lrclib
make test           # go test ./...  (unit only; integration excluded by build tag)
make test-int       # go test -tags=integration ./...
make lint           # golangci-lint run
make fmt            # gofumpt -w .
make clean          # rm -rf dist/ tmp/
make release-dry    # goreleaser release --snapshot --clean
```

**Run a single test:**
```bash
go test ./internal/api/ -run TestFunctionName -v
go test -tags=integration ./internal/cache/ -run TestIntegrationName -v
```

**Build with version info (ldflags):**
```bash
CGO_ENABLED=0 go build -ldflags "-X main.version=dev -X main.commit=abc -X main.date=2024-01-01" ./cmd/lrclib
```

GoReleaser targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`.

---

## Architecture

```
cmd/lrclib/             ← composition root: DI wiring + rootCmd.Execute() only
  ↓
internal/tui/           ← Bubble Tea presentation layer
  ↓
internal/usecase/       ← business logic; knows nothing about Cobra, HTTP, or SQLite
  ↓  (via consumer-defined interfaces)
internal/api/           ← HTTP client for lrclib.net
internal/cache/         ← SQLite cache (modernc.org/sqlite, CGO-free)
internal/config/        ← config loading (XDG/APPDATA), env vars
  ↓
internal/lrc/           ← domain: LRC format parsing and generation
internal/errs/          ← error kinds and exit codes
```

**Key rules:**
- `cmd/lrclib/main.go` — only dependency wiring and `rootCmd.Execute()`; no business logic
- Everything is `internal/`; there is no `pkg/` (this is a binary, not a library)
- `internal/usecase/` does **not** import `internal/api` or `internal/cache` directly — only the interfaces it defines
- Interfaces are defined on the **consumer** side, not the implementation side

---

## Workflow

### Branch naming
```
feature/<name>   fix/<name>   chore/<name>   refactor/<name>
```
One branch per task → one commit per micro-task → PR → review → merge to `master`. **Never commit directly to `master`.**

### Conventional Commits
```
feat(api): add synced lyrics endpoint
fix(cache): handle WAL lock on concurrent writes
perf(http): enable connection pooling
```
Types: `feat` `fix` `refactor` `test` `docs` `ci` `chore` `perf`

### When to ask the user before acting
- Adding a new dependency
- Changing the public CLI interface (flags, commands, output format)
- Deleting files
- Modifying CI workflows

**Act autonomously:** refactoring within a package, adding tests, fixing linter errors, updating docs.

### Definition of Done (before each PR)
- `make fmt` — no diff
- `make lint` — no errors
- `make test` — all green
- New logic has ≥ 70% test coverage
- GoDoc comments on all new exported symbols
- `CHANGELOG.md` updated for user-facing features

---

## Key Design Decisions

### API fallback chain (`internal/api/`)
```
1. GET /api/get (by metadata) → prefer syncedLyrics
2. No synced lyrics → use plainLyrics from same response
3. No match → GET /api/search → take first result
4. API unavailable → check local cache
5. Cache miss → user-friendly error (exit code 1)
```

HTTP client: 5s connect / 15s read / 90s idle timeouts; exponential backoff with jitter, max 3 retries; respects `Retry-After` on 429.

### Cache (`internal/cache/`)
- Backend: `modernc.org/sqlite` — pure Go, no CGO
- Location: `os.UserCacheDir()/lrclib/cache.db` — **never hardcode paths**
- Required PRAGMAs on open: `journal_mode=WAL`, `busy_timeout=5000`, `synchronous=NORMAL`, `foreign_keys=ON`
- TTL: 7 days default; write with `INSERT OR REPLACE`
- Cache errors → log as `WARN`, continue without cache

### Error handling (`internal/errs/`)
Error kinds: `KindNotFound` (1), `KindNetwork` (2), `KindRateLimited` (3), `KindInternal` (4), `KindBadInput` (5). Exit code 0 = success.

- `os.Exit` only in `main.go`
- `panic` forbidden in production code
- Use `errors.Is` / `errors.As` — never compare error strings

### TUI (`internal/tui/`)
Strict TEA pattern: `Model`, `Update`, `View`. Long operations (HTTP, file IO) run as `tea.Cmd` — never block `Update`. No global state in the TUI package.

### Config priority
CLI flags > `LRCLIB_*` env vars > `config.toml` > defaults

Config file: `os.UserConfigDir()/lrclib/config.toml`

### Logging
`log/slog` with `TextHandler` at INFO by default. Use `--log-format=json` / `--log-level=debug` or `LRCLIB_LOG_LEVEL=debug`.

---

## Hard Rules

| Rule | Reason |
|------|---------|
| `CGO_ENABLED=0` always | GoReleaser cross-compilation |
| `os.Exit` only in `main.go` | Untestable otherwise |
| No `panic` in production | No cleanup possible |
| No real HTTP calls in unit tests | Flaky, network-dependent |
| No hardcoded file paths | Use `os.UserCacheDir()` / `os.UserConfigDir()` |
| No global variables | Except `version`, `commit`, `date` (ldflags) |
| Cache errors logged as WARN only | Cache is degraded, not fatal |
