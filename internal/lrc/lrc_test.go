package lrc_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ak1m1tsu/lrclib/internal/lrc"
)

// ── ParseSynced ──────────────────────────────────────────────────────────────

func TestParseSynced_HappyPath(t *testing.T) {
	src := "[ti:Creep]\n[ar:Radiohead]\n[al:Pablo Honey]\n[by:lrclib]\n[length:236.0]\n" +
		"[00:00.00]When you were here before\n" +
		"[00:04.50]Couldn't look you in the eye\n"

	tr, err := lrc.ParseSynced(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tr.Title != "Creep" {
		t.Errorf("Title = %q, want %q", tr.Title, "Creep")
	}
	if tr.Artist != "Radiohead" {
		t.Errorf("Artist = %q, want %q", tr.Artist, "Radiohead")
	}
	if tr.Album != "Pablo Honey" {
		t.Errorf("Album = %q, want %q", tr.Album, "Pablo Honey")
	}
	if tr.By != "lrclib" {
		t.Errorf("By = %q, want %q", tr.By, "lrclib")
	}
	if tr.Length != 236.0 {
		t.Errorf("Length = %v, want 236.0", tr.Length)
	}
	if tr.IsPlain {
		t.Error("IsPlain should be false for synced lyrics")
	}
	if len(tr.Lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(tr.Lines))
	}
	if tr.Lines[0].Text != "When you were here before" {
		t.Errorf("Lines[0].Text = %q", tr.Lines[0].Text)
	}
	want0 := 0 * time.Second
	if tr.Lines[0].Timestamp != want0 {
		t.Errorf("Lines[0].Timestamp = %v, want %v", tr.Lines[0].Timestamp, want0)
	}
	want1 := 4*time.Second + 500*time.Millisecond
	if tr.Lines[1].Timestamp != want1 {
		t.Errorf("Lines[1].Timestamp = %v, want %v", tr.Lines[1].Timestamp, want1)
	}
}

func TestParseSynced_MillisecondTimestamp(t *testing.T) {
	src := "[00:09.100]Your skin makes me cry\n"

	tr, err := lrc.ParseSynced(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tr.Lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(tr.Lines))
	}
	want := 9*time.Second + 100*time.Millisecond
	if tr.Lines[0].Timestamp != want {
		t.Errorf("Timestamp = %v, want %v", tr.Lines[0].Timestamp, want)
	}
}

func TestParseSynced_MalformedLinesSkipped(t *testing.T) {
	src := "[ti:Test]\nnot a tag\n[broken\n[00:01.00]valid line\n"

	tr, err := lrc.ParseSynced(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tr.Lines) != 1 {
		t.Errorf("got %d lines, want 1 (malformed lines must be skipped)", len(tr.Lines))
	}
}

func TestParseSynced_EmptyInputError(t *testing.T) {
	_, err := lrc.ParseSynced("   ")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestParseSynced_MetadataOnly(t *testing.T) {
	src := "[ti:Instrumental]\n[ar:Artist]\n"

	tr, err := lrc.ParseSynced(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Title != "Instrumental" {
		t.Errorf("Title = %q", tr.Title)
	}
	if len(tr.Lines) != 0 {
		t.Errorf("got %d lines, want 0", len(tr.Lines))
	}
}

// ── ParsePlain ───────────────────────────────────────────────────────────────

func TestParsePlain_HappyPath(t *testing.T) {
	src := "When you were here before\nCouldn't look you in the eye\n"

	tr, err := lrc.ParsePlain(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tr.IsPlain {
		t.Error("IsPlain should be true")
	}
	// Two content lines + one empty trailing line from the trailing newline split
	found := 0
	for _, l := range tr.Lines {
		if l.Text != "" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("got %d non-empty lines, want 2", found)
	}
	for _, l := range tr.Lines {
		if l.Timestamp != 0 {
			t.Errorf("plain line has non-zero timestamp %v", l.Timestamp)
		}
	}
}

func TestParsePlain_EmptyInputError(t *testing.T) {
	_, err := lrc.ParsePlain("\n\n  \n")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

// ── Format ───────────────────────────────────────────────────────────────────

func TestFormat_Synced_GoldenFile(t *testing.T) {
	tr := &lrc.Track{
		Title:  "Creep",
		Artist: "Radiohead",
		Album:  "Pablo Honey",
		By:     "lrclib",
		Length: 236.0,
		Lines: []lrc.Line{
			{Timestamp: 0, Text: "When you were here before"},
			{Timestamp: 4*time.Second + 500*time.Millisecond, Text: "Couldn't look you in the eye"},
			{Timestamp: 9*time.Second + 100*time.Millisecond, Text: "You're just like an angel"},
			{Timestamp: 13*time.Second + 600*time.Millisecond, Text: "Your skin makes me cry"},
		},
	}

	got := lrc.Format(tr)
	golden, err := os.ReadFile("testdata/synced.lrc")
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}

	want := strings.ReplaceAll(string(golden), "\r\n", "\n")
	got = strings.ReplaceAll(got, "\r\n", "\n")

	if got != want {
		t.Errorf("Format() output does not match golden file.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormat_Plain_GoldenFile(t *testing.T) {
	tr := &lrc.Track{
		Title:   "Creep",
		Artist:  "Radiohead",
		IsPlain: true,
		Lines: []lrc.Line{
			{Text: "When you were here before"},
			{Text: "Couldn't look you in the eye"},
			{Text: "You're just like an angel"},
			{Text: "Your skin makes me cry"},
		},
	}

	got := lrc.Format(tr)
	golden, err := os.ReadFile("testdata/plain.lrc")
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}

	want := strings.ReplaceAll(string(golden), "\r\n", "\n")
	got = strings.ReplaceAll(got, "\r\n", "\n")

	if got != want {
		t.Errorf("Format() output does not match golden file.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormat_OmitsEmptyMetadata(t *testing.T) {
	tr := &lrc.Track{
		Lines: []lrc.Line{{Timestamp: time.Second, Text: "hello"}},
	}
	out := lrc.Format(tr)
	for _, tag := range []string{"[ti:", "[ar:", "[al:", "[by:", "[length:"} {
		if strings.Contains(out, tag) {
			t.Errorf("output should not contain %q when field is empty/zero", tag)
		}
	}
}

// ── Timestamp round-trip ─────────────────────────────────────────────────────

func TestTimestampRoundTrip(t *testing.T) {
	cases := []struct {
		input string
		text  string
	}{
		{"[00:00.00]line", "line"},
		{"[01:23.45]line", "line"},
		{"[99:59.99]line", "line"},
	}

	for _, tc := range cases {
		tr, err := lrc.ParseSynced(tc.input)
		if err != nil {
			t.Fatalf("ParseSynced(%q): %v", tc.input, err)
		}
		if len(tr.Lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(tr.Lines))
		}
		out := lrc.Format(tr)
		if !strings.Contains(out, tc.input[1:strings.Index(tc.input, "]")+1]) {
			t.Errorf("formatted output %q does not contain original timestamp from %q", out, tc.input)
		}
	}
}
