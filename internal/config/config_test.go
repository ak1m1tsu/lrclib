package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ak1m1tsu/lrclib/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Cache.TTL != 7*24*time.Hour {
		t.Errorf("Cache.TTL = %v, want %v", cfg.Cache.TTL, 7*24*time.Hour)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want %q", cfg.Log.Format, "text")
	}
	if cfg.HTTP.Timeout != 15*time.Second {
		t.Errorf("HTTP.Timeout = %v, want %v", cfg.HTTP.Timeout, 15*time.Second)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("LRCLIB_CACHE_TTL", "24h")
	t.Setenv("LRCLIB_LOG_LEVEL", "debug")
	t.Setenv("LRCLIB_LOG_FORMAT", "json")
	t.Setenv("LRCLIB_HTTP_TIMEOUT", "30s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Cache.TTL != 24*time.Hour {
		t.Errorf("Cache.TTL = %v, want 24h", cfg.Cache.TTL)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want debug", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want json", cfg.Log.Format)
	}
	if cfg.HTTP.Timeout != 30*time.Second {
		t.Errorf("HTTP.Timeout = %v, want 30s", cfg.HTTP.Timeout)
	}
}

func TestLoad_InvalidEnvIgnored(t *testing.T) {
	clearEnv(t)
	t.Setenv("LRCLIB_CACHE_TTL", "not-a-duration")
	t.Setenv("LRCLIB_HTTP_TIMEOUT", "???")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Invalid values should fall back to defaults
	if cfg.Cache.TTL != 7*24*time.Hour {
		t.Errorf("Cache.TTL = %v, want default 168h", cfg.Cache.TTL)
	}
	if cfg.HTTP.Timeout != 15*time.Second {
		t.Errorf("HTTP.Timeout = %v, want default 15s", cfg.HTTP.Timeout)
	}
}

func TestLoad_TOMLFile(t *testing.T) {
	clearEnv(t)

	// Point XDG config dir to a temp directory
	dir := t.TempDir()
	lrclibDir := filepath.Join(dir, "lrclib")
	if err := os.MkdirAll(lrclibDir, 0o755); err != nil {
		t.Fatal(err)
	}

	toml := "[cache]\nttl = 3600\n\n[log]\nlevel = \"warn\"\nformat = \"json\"\n\n[http]\ntimeout = 5\n"
	if err := os.WriteFile(filepath.Join(lrclibDir, "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AppData", dir)         // Windows
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux/macOS

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Cache.TTL != time.Hour {
		t.Errorf("Cache.TTL = %v, want 1h", cfg.Cache.TTL)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("Log.Level = %q, want warn", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want json", cfg.Log.Format)
	}
	if cfg.HTTP.Timeout != 5*time.Second {
		t.Errorf("HTTP.Timeout = %v, want 5s", cfg.HTTP.Timeout)
	}
}

func TestLoad_MissingFileIsOK(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	_, err := config.Load()
	if err != nil {
		t.Errorf("Load() with missing config file should not error, got: %v", err)
	}
}

func TestFilePath_ReturnsNonEmpty(t *testing.T) {
	p, err := config.FilePath()
	if err != nil {
		t.Fatalf("FilePath() error: %v", err)
	}
	if p == "" {
		t.Error("FilePath() returned empty string")
	}
}

// clearEnv unsets all LRCLIB_* vars for the duration of the test.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"LRCLIB_CACHE_TTL",
		"LRCLIB_LOG_LEVEL",
		"LRCLIB_LOG_FORMAT",
		"LRCLIB_HTTP_TIMEOUT",
	} {
		t.Setenv(key, "")
	}
}
