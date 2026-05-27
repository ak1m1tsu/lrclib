// Package tui implements the interactive terminal UI using Bubble Tea.
// It follows the strict TEA pattern (Model / Update / View). Long-running
// operations (HTTP, file I/O) are always dispatched as tea.Cmd — Update
// must never block.
package tui
