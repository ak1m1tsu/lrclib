// Package main is the composition root for the lrclib CLI.
// It wires up dependencies and delegates to rootCmd.Execute().
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/ak1m1tsu/lrclib/internal/api"
	"github.com/ak1m1tsu/lrclib/internal/cache"
	"github.com/ak1m1tsu/lrclib/internal/config"
	"github.com/ak1m1tsu/lrclib/internal/errs"
	"github.com/ak1m1tsu/lrclib/internal/usecase"
)

// Version information injected at build time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		var appErr *errs.AppError
		if ok := errsAs(err, &appErr); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", appErr.Message)
			os.Exit(appErr.Kind.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(errs.KindInternal))
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		slog.Warn("failed to load config, using defaults", "err", err)
		cfg, _ = config.Load()
	}

	log := buildLogger(cfg)

	apiClient := api.New()

	var lyricsCache usecase.LyricsCache
	if c, err := cache.Open(cache.WithTTL(cfg.Cache.TTL), cache.WithLogger(log)); err != nil {
		log.Warn("cache unavailable", "err", err)
	} else {
		lyricsCache = &cacheAdapter{c: c}
	}

	svc := usecase.New(
		&apiFetcher{client: apiClient},
		lyricsCache,
		log,
		usecase.WithSearcher(&apiSearcher{client: apiClient}),
	)

	return newRootCmd(svc).ExecuteContext(ctx)
}

func newRootCmd(svc *usecase.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lrclib",
		Short: "Search and download song lyrics from lrclib.net",
		Long: `lrclib is a cross-platform CLI for searching and downloading
song lyrics from lrclib.net and saving them as .lrc files.`,
		Version:      fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		SilenceUsage: true,
	}

	cmd.AddCommand(newGetCmd(svc))
	cmd.AddCommand(newSearchCmd(svc))

	return cmd
}

func buildLogger(cfg config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Log.Format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

// errsAs is a local errors.As for *errs.AppError to avoid importing errors.
func errsAs(err error, target **errs.AppError) bool {
	for err != nil {
		if e, ok := err.(*errs.AppError); ok {
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
