package tui

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ak1m1tsu/lrclib/internal/lrc"
	"github.com/ak1m1tsu/lrclib/internal/usecase"
)

type state int

const (
	stateInput   state = iota // user typing query
	stateLoading              // search in progress
	stateList                 // showing results
	stateSaving               // saving selected track
	stateDone                 // saved successfully
	stateError                // unrecoverable error
)

// searchResultMsg carries results from a background search.
type searchResultMsg struct {
	tracks []*lrc.Track
	err    error
}

// saveResultMsg carries the outcome of a background save.
type saveResultMsg struct {
	path string
	err  error
}

// trackItem wraps *lrc.Track for the bubbles list.
type trackItem struct{ t *lrc.Track }

func (i trackItem) Title() string {
	if i.t.Title != "" {
		return i.t.Title
	}
	return "(untitled)"
}

func (i trackItem) Description() string {
	desc := i.t.Artist
	if i.t.Album != "" {
		desc += " — " + i.t.Album
	}
	kind := "plain"
	if !i.t.IsPlain {
		kind = "synced"
	}
	return fmt.Sprintf("%s  [%s]", desc, kind)
}
func (i trackItem) FilterValue() string { return i.t.Title + " " + i.t.Artist }

type uiStyles struct {
	title  lipgloss.Style
	status lipgloss.Style
	err    lipgloss.Style
	ok     lipgloss.Style
}

func newStyles() uiStyles {
	return uiStyles{
		title:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
		status: lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		err:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		ok:     lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	}
}

// Model is the root Bubble Tea model.
type Model struct {
	svc     *usecase.Service
	outDir  string
	state   state
	input   textinput.Model
	list    list.Model
	spinner spinner.Model
	tracks  []*lrc.Track
	savedAt string
	err     error
	width   int
	height  int
	styles  uiStyles
}

// New creates a Model wired to svc. outDir is where .lrc files are saved.
func New(svc *usecase.Service, outDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "artist - track name…"
	ti.Focus()
	ti.CharLimit = 200

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Results"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return Model{
		svc:     svc,
		outDir:  outDir,
		state:   stateInput,
		input:   ti,
		list:    l,
		spinner: sp,
		styles:  newStyles(),
	}
}

// Init satisfies tea.Model; starts the spinner tick.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update is the pure state-transition function.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-6)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case searchResultMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		items := make([]list.Item, len(msg.tracks))
		for i, t := range msg.tracks {
			items[i] = trackItem{t}
		}
		m.tracks = msg.tracks
		m.list.SetItems(items)
		m.state = stateList
		return m, nil

	case saveResultMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.savedAt = msg.path
		m.state = stateDone
		return m, nil

	case spinner.TickMsg:
		if m.state == stateLoading || m.state == stateSaving {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m.delegateToChild(msg)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		if m.state == stateList {
			// ESC from list → back to input
			m.state = stateInput
			m.input.Focus()
			return m, textinput.Blink
		}
		return m, tea.Quit

	case tea.KeyEnter:
		switch m.state {
		case stateInput:
			query := m.input.Value()
			if query == "" {
				return m, nil
			}
			m.state = stateLoading
			return m, tea.Batch(m.spinner.Tick, m.doSearch(query))

		case stateList:
			selected, ok := m.list.SelectedItem().(trackItem)
			if !ok {
				return m, nil
			}
			m.state = stateSaving
			return m, tea.Batch(m.spinner.Tick, m.doSave(selected.t))

		case stateDone, stateError:
			// Reset to input for a new search
			m.state = stateInput
			m.input.Reset()
			m.input.Focus()
			m.err = nil
			return m, textinput.Blink
		}
	}

	return m.delegateToChild(msg)
}

func (m Model) delegateToChild(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.state {
	case stateInput:
		m.input, cmd = m.input.Update(msg)
	case stateList:
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

// View renders the current state.
func (m Model) View() string {
	switch m.state {
	case stateInput:
		return fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			m.styles.title.Render("lrclib — interactive search"),
			m.input.View(),
			m.styles.status.Render("Enter to search • Ctrl+C to quit"),
		)

	case stateLoading:
		return fmt.Sprintf("\n  %s Searching…", m.spinner.View())

	case stateList:
		if len(m.tracks) == 0 {
			return m.styles.status.Render("No results. Press Esc to search again.")
		}
		return fmt.Sprintf("%s\n%s",
			m.list.View(),
			m.styles.status.Render("Enter to save • Esc to search again • / to filter"),
		)

	case stateSaving:
		return fmt.Sprintf("\n  %s Saving…", m.spinner.View())

	case stateDone:
		return fmt.Sprintf("\n  %s\n\n%s",
			m.styles.ok.Render("Saved: "+m.savedAt),
			m.styles.status.Render("Enter for new search • Ctrl+C to quit"),
		)

	case stateError:
		return fmt.Sprintf("\n  %s\n\n%s",
			m.styles.err.Render("Error: "+m.err.Error()),
			m.styles.status.Render("Enter for new search • Ctrl+C to quit"),
		)

	default:
		return ""
	}
}

// doSearch dispatches a background search command.
func (m Model) doSearch(query string) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		tracks, err := svc.SearchTracks(context.Background(), query)
		return searchResultMsg{tracks: tracks, err: err}
	}
}

// doSave dispatches a background save command.
func (m Model) doSave(t *lrc.Track) tea.Cmd {
	svc := m.svc
	outDir := m.outDir
	return func() tea.Msg {
		filename := sanitize(t.Artist+" - "+t.Title) + ".lrc"
		path := filepath.Join(outDir, filename)
		err := svc.SaveLRC(t, path)
		return saveResultMsg{path: path, err: err}
	}
}

// sanitize removes characters invalid in file names.
func sanitize(s string) string {
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
	if len(out) == 0 {
		return "track"
	}
	return string(out)
}
