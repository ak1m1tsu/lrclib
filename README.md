# lrclib

Cross-platform CLI for searching and downloading song lyrics from [lrclib.net](https://lrclib.net) and saving them as `.lrc` files.

## Install

**Download a binary** from the [Releases](https://github.com/ak1m1tsu/lrclib/releases) page — available for Linux, macOS, and Windows.

**Or build from source** (requires Go 1.22+):

```sh
CGO_ENABLED=0 go install github.com/ak1m1tsu/lrclib/cmd/lrclib@latest
```

## Usage

### `get` — fetch lyrics for a specific track

```sh
lrclib get "Creep" --artist "Radiohead"
# Saved: Radiohead - Creep.lrc

lrclib get "Creep" --artist "Radiohead" --album "Pablo Honey" --duration 238
# Saved: Radiohead - Creep.lrc

# Custom output path
lrclib get "Creep" --artist "Radiohead" --output ~/Music/Radiohead/Creep.lrc

# Print to stdout instead of saving
lrclib get "Creep" --artist "Radiohead" --stdout
```

| Flag | Short | Description |
|------|-------|-------------|
| `--artist` | `-a` | Artist name **(required)** |
| `--album` | `-A` | Album name (improves accuracy) |
| `--duration` | `-d` | Track duration in seconds |
| `--output` | `-o` | Output file path (default: `<artist> - <track>.lrc`) |
| `--stdout` | | Print to stdout instead of saving |

Lyrics are fetched with a cache-first strategy: if the track was already downloaded it is returned immediately from the local cache.

### `search` — interactive TUI search

```sh
lrclib search
lrclib search --output-dir ~/Music
```

Type a query, press **Enter** to search, select a result with **Enter** to save it as an `.lrc` file. Press **Esc** to go back to the search prompt, **Ctrl+C** to quit.

| Flag | Short | Description |
|------|-------|-------------|
| `--output-dir` | `-o` | Directory to save `.lrc` files (default: current directory) |

## Configuration

lrclib reads `config.toml` from the platform config directory:

| Platform | Path |
|----------|------|
| Linux | `~/.config/lrclib/config.toml` |
| macOS | `~/Library/Application Support/lrclib/config.toml` |
| Windows | `%AppData%\lrclib\config.toml` |

```toml
[cache]
ttl = 604800      # Cache TTL in seconds (default: 7 days)

[log]
level  = "info"   # debug | info | warn | error
format = "text"   # text | json

[http]
timeout = 15      # HTTP read timeout in seconds
```

Settings can also be set via environment variables (override the config file):

| Variable | Description |
|----------|-------------|
| `LRCLIB_CACHE_TTL` | Cache TTL as a Go duration, e.g. `168h` |
| `LRCLIB_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` |
| `LRCLIB_LOG_FORMAT` | Log format: `text`, `json` |
| `LRCLIB_HTTP_TIMEOUT` | HTTP timeout as a Go duration, e.g. `15s` |

Priority: **CLI flags > env vars > config.toml > built-in defaults**

## Data locations

| Data | Path |
|------|------|
| Cache DB | `os.UserCacheDir()/lrclib/cache.db` |
| Config | `os.UserConfigDir()/lrclib/config.toml` |

## Build from source

```sh
git clone https://github.com/ak1m1tsu/lrclib.git
cd lrclib
CGO_ENABLED=0 go build -o lrclib ./cmd/lrclib
```

## License

MIT — see [LICENSE](LICENSE).
