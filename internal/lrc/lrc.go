package lrc

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Track holds all data parsed from an LRC source or API response.
type Track struct {
	Title  string  // [ti:] tag
	Artist string  // [ar:] tag
	Album  string  // [al:] tag
	By     string  // [by:] tag
	Length float64 // [length:] tag value in seconds; zero means absent
	Lines  []Line
	// IsPlain is true when the track was built from plainLyrics (no timestamps).
	IsPlain bool
}

// Line is a single lyric line.
// Timestamp is the zero value for plain (non-synced) tracks.
type Line struct {
	Timestamp time.Duration
	Text      string
}

// ParseSynced parses a synced LRC string (the syncedLyrics field from the API).
// Each input line is either a metadata tag [key:value] or a timestamped lyric
// [mm:ss.xx] text. Malformed lines are silently skipped.
// Returns a non-nil error only when src is empty.
func ParseSynced(src string) (*Track, error) {
	if strings.TrimSpace(src) == "" {
		return nil, fmt.Errorf("lrc: synced lyrics source is empty")
	}

	t := &Track{}

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		// Must start with '['
		if !strings.HasPrefix(line, "[") {
			continue
		}

		closeIdx := strings.Index(line, "]")
		if closeIdx < 0 {
			continue
		}

		tag := line[1:closeIdx]
		rest := strings.TrimSpace(line[closeIdx+1:])

		if colon := strings.IndexByte(tag, ':'); colon >= 0 {
			key := strings.ToLower(strings.TrimSpace(tag[:colon]))
			val := strings.TrimSpace(tag[colon+1:])

			switch key {
			case "ti":
				t.Title = val
				continue
			case "ar":
				t.Artist = val
				continue
			case "al":
				t.Album = val
				continue
			case "by":
				t.By = val
				continue
			case "length":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					t.Length = f
				}
				continue
			}

			// Try to parse as a timestamp [mm:ss.xx] or [mm:ss.xxx]
			if d, ok := parseTimestamp(tag); ok {
				t.Lines = append(t.Lines, Line{Timestamp: d, Text: rest})
			}
			// Unknown key:value tags are silently ignored
			continue
		}

		// Tag without colon — not a valid LRC tag, skip
	}

	return t, nil
}

// ParsePlain parses a plain-text lyrics string (the plainLyrics field from the API).
// Each non-empty line becomes a Line with a zero Timestamp.
// Returns a non-nil error only when src is empty.
func ParsePlain(src string) (*Track, error) {
	if strings.TrimSpace(src) == "" {
		return nil, fmt.Errorf("lrc: plain lyrics source is empty")
	}

	t := &Track{IsPlain: true}

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			t.Lines = append(t.Lines, Line{}) // preserve blank lines
			continue
		}
		t.Lines = append(t.Lines, Line{Text: line})
	}

	return t, nil
}

// Format generates an .lrc file string from a Track.
// Metadata tags are written first (only non-empty / non-zero fields), followed
// by lyric lines. For synced tracks each line is prefixed [mm:ss.xx]; for plain
// tracks the bare text is written without a timestamp prefix.
func Format(t *Track) string {
	var sb strings.Builder

	if t.Title != "" {
		fmt.Fprintf(&sb, "[ti:%s]\n", t.Title)
	}
	if t.Artist != "" {
		fmt.Fprintf(&sb, "[ar:%s]\n", t.Artist)
	}
	if t.Album != "" {
		fmt.Fprintf(&sb, "[al:%s]\n", t.Album)
	}
	if t.By != "" {
		fmt.Fprintf(&sb, "[by:%s]\n", t.By)
	}
	if t.Length > 0 {
		fmt.Fprintf(&sb, "[length:%.2f]\n", t.Length)
	}

	// Blank line between tags and lyrics when tags are present and there are lines.
	if sb.Len() > 0 && len(t.Lines) > 0 {
		sb.WriteByte('\n')
	}

	for _, l := range t.Lines {
		if t.IsPlain {
			sb.WriteString(l.Text)
		} else {
			fmt.Fprintf(&sb, "[%s]%s", formatTimestamp(l.Timestamp), l.Text)
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// parseTimestamp parses "mm:ss.xx" or "mm:ss.xxx" into a time.Duration.
func parseTimestamp(s string) (time.Duration, bool) {
	// Expect exactly one ':' and one '.'
	colonIdx := strings.IndexByte(s, ':')
	dotIdx := strings.IndexByte(s, '.')
	if colonIdx < 0 || dotIdx < 0 || dotIdx < colonIdx {
		return 0, false
	}

	mm, err := strconv.Atoi(s[:colonIdx])
	if err != nil {
		return 0, false
	}

	ss, err := strconv.Atoi(s[colonIdx+1 : dotIdx])
	if err != nil {
		return 0, false
	}

	fracStr := s[dotIdx+1:]
	frac, err := strconv.Atoi(fracStr)
	if err != nil {
		return 0, false
	}

	var fracDur time.Duration
	switch len(fracStr) {
	case 2: // centiseconds
		fracDur = time.Duration(frac) * 10 * time.Millisecond
	case 3: // milliseconds
		fracDur = time.Duration(frac) * time.Millisecond
	default:
		return 0, false
	}

	d := time.Duration(mm)*time.Minute +
		time.Duration(ss)*time.Second +
		fracDur

	return d, true
}

// formatTimestamp formats a time.Duration as "mm:ss.xx" (centiseconds).
func formatTimestamp(d time.Duration) string {
	totalCS := d.Milliseconds() / 10
	cs := totalCS % 100
	totalSec := totalCS / 100
	sec := totalSec % 60
	mins := totalSec / 60

	return fmt.Sprintf("%02d:%02d.%02d", mins, sec, cs)
}
