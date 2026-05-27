// Package main is the composition root for the lrclib CLI.
// It wires up dependencies and delegates to rootCmd.Execute().
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information injected at build time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lrclib",
		Short: "Search and download song lyrics from lrclib.net",
		Long: `lrclib is a cross-platform CLI for searching and downloading
song lyrics from lrclib.net and saving them as .lrc files.`,
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
	}

	return cmd
}
