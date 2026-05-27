package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all runtime configuration for lrclib.
// Priority order: CLI flags > LRCLIB_* env vars > config.toml > zero values here.
type Config struct {
	Cache CacheConfig
	Log   LogConfig
	HTTP  HTTPConfig
}

// CacheConfig controls the local SQLite lyrics cache.
type CacheConfig struct {
	// TTL is how long a cached entry is considered fresh. Default: 7 days.
	TTL time.Duration
}

// LogConfig controls log output.
type LogConfig struct {
	// Level is the minimum log level: debug | info | warn | error. Default: info.
	Level string
	// Format is the log output format: text | json. Default: text.
	Format string
}

// HTTPConfig controls the outbound HTTP client.
type HTTPConfig struct {
	// Timeout is the read timeout for HTTP responses. Default: 15s.
	Timeout time.Duration
}

// defaults returns a Config with all built-in default values.
func defaults() Config {
	return Config{
		Cache: CacheConfig{TTL: 7 * 24 * time.Hour},
		Log:   LogConfig{Level: "info", Format: "text"},
		HTTP:  HTTPConfig{Timeout: 15 * time.Second},
	}
}

// Load reads configuration from the TOML file (if present) then overrides
// with any LRCLIB_* environment variables that are set.
// Missing config file is not an error.
func Load() (Config, error) {
	cfg := defaults()

	if err := loadFile(&cfg); err != nil {
		return cfg, fmt.Errorf("config: load file: %w", err)
	}

	applyEnv(&cfg)

	return cfg, nil
}

// FilePath returns the platform-appropriate path for config.toml.
func FilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve config dir: %w", err)
	}
	return filepath.Join(dir, "lrclib", "config.toml"), nil
}

// loadFile parses config.toml into cfg; a missing file is silently ignored.
func loadFile(cfg *Config) error {
	path, err := FilePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var raw fileSchema
	if err := toml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if raw.Cache.TTL > 0 {
		cfg.Cache.TTL = time.Duration(raw.Cache.TTL) * time.Second
	}
	if raw.Log.Level != "" {
		cfg.Log.Level = raw.Log.Level
	}
	if raw.Log.Format != "" {
		cfg.Log.Format = raw.Log.Format
	}
	if raw.HTTP.Timeout > 0 {
		cfg.HTTP.Timeout = time.Duration(raw.HTTP.Timeout) * time.Second
	}

	return nil
}

// applyEnv overrides cfg fields from LRCLIB_* environment variables.
func applyEnv(cfg *Config) {
	if v := os.Getenv("LRCLIB_CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Cache.TTL = d
		}
	}
	if v := os.Getenv("LRCLIB_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("LRCLIB_LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	if v := os.Getenv("LRCLIB_HTTP_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.HTTP.Timeout = d
		}
	}
}

// fileSchema mirrors config.toml structure with raw numeric types for TOML
// (seconds as integers, matching the example config).
type fileSchema struct {
	Cache struct {
		TTL int64 `toml:"ttl"`
	} `toml:"cache"`
	Log struct {
		Level  string `toml:"level"`
		Format string `toml:"format"`
	} `toml:"log"`
	HTTP struct {
		Timeout int64 `toml:"timeout"`
	} `toml:"http"`
}
