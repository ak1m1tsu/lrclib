package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ak1m1tsu/lrclib/internal/errs"
	"github.com/ak1m1tsu/lrclib/internal/lrc"
	"github.com/ak1m1tsu/lrclib/internal/usecase"
)

func newGetCmd(svc *usecase.Service) *cobra.Command {
	var (
		artist   string
		album    string
		duration float64
		output   string
		stdout   bool
	)

	cmd := &cobra.Command{
		Use:   "get <track>",
		Short: "Fetch lyrics for a track and save as .lrc",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			trackName := args[0]

			if artist == "" {
				return errs.BadInput("--artist is required")
			}

			q := usecase.Query{
				TrackName:  trackName,
				ArtistName: artist,
				AlbumName:  album,
				Duration:   duration,
			}

			track, err := svc.GetLyrics(cmd.Context(), q)
			if err != nil {
				return err
			}

			if stdout {
				_, err = fmt.Fprint(cmd.OutOrStdout(), lrc.Format(track))
				return err
			}

			dest := output
			if dest == "" {
				dest = sanitizeFilename(artist+" - "+trackName) + ".lrc"
			}
			dest = filepath.Clean(dest)

			if err := svc.SaveLRC(track, dest); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Saved: %s\n", dest)
			return err
		},
	}

	cmd.Flags().StringVarP(&artist, "artist", "a", "", "Artist name (required)")
	cmd.Flags().StringVarP(&album, "album", "A", "", "Album name (optional, improves accuracy)")
	cmd.Flags().Float64VarP(&duration, "duration", "d", 0, "Track duration in seconds (optional)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: <artist> - <track>.lrc)")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Print LRC to stdout instead of saving to file")

	return cmd
}

// sanitizeFilename removes characters that are invalid in file names on common OSes.
func sanitizeFilename(s string) string {
	const invalid = `/\:*?"<>|`
	out := make([]byte, 0, len(s))
	for i := range len(s) {
		c := s[i]
		keep := true
		for j := range len(invalid) {
			if c == invalid[j] {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, c)
		}
	}
	return string(out)
}
