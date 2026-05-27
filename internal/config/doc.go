// Package config handles loading and validating lrclib configuration.
// Sources are applied in priority order: CLI flags > LRCLIB_* env vars >
// config.toml (XDG/APPDATA location) > built-in defaults.
package config
