package tui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestRenderFrames writes real frames to $TUI_RENDER_DIR for eyeballing.
func TestRenderFrames(t *testing.T) {
	dir := os.Getenv("TUI_RENDER_DIR")
	if dir == "" {
		t.Skip("set TUI_RENDER_DIR to dump frames")
	}
	m, st := fixture(t)
	md := New(m, st)
	md.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	dump := func(name string) {
		os.WriteFile(dir+"/"+name+".txt", []byte(stripANSI(md.View())), 0o644)
		os.WriteFile(dir+"/"+name+".ansi", []byte(md.View()), 0o644)
	}
	dump("01-main")
	press(md, "l", "j", "j", "j", "d")
	dump("02-expanded-diff")
	press(md, "m")
	dump("03-install")
	press(md, "x")
	dump("04-confirm")
	press(md, "n", "?")
	dump("05-help")
	press(md, "esc", "/", "csv")
	dump("06-filter")
}

func TestRenderSetupFrames(t *testing.T) {
	dir := os.Getenv("TUI_RENDER_DIR")
	if dir == "" {
		t.Skip("set TUI_RENDER_DIR to dump frames")
	}
	m, st := fixture(t)
	s := NewSetup(m, st)
	s.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	dump := func(n string) { os.WriteFile(dir+"/setup-"+n+".txt", []byte(stripANSI(s.View())), 0o644) }
	dump("0-welcome")
	s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dump("1-profile")
	s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dump("2-features")
	s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dump("3-review")
}
