package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ak1m1tsu/lrclib/internal/tui"
	"github.com/ak1m1tsu/lrclib/internal/usecase"
)

func newSearchCmd(svc *usecase.Service) *cobra.Command {
	var outDir string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Interactively search and download lyrics (TUI)",
		RunE: func(_ *cobra.Command, _ []string) error {
			if outDir == "" {
				var err error
				outDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("get working directory: %w", err)
				}
			}

			model := tui.New(svc, outDir)
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&outDir, "output-dir", "o", "", "Directory to save .lrc files (default: current directory)")

	return cmd
}
