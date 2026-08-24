package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSetupWizardFlow(t *testing.T) {
	m, st := fixture(t)
	s := NewSetup(m, st)
	s.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	v := stripANSI(s.View())
	if !strings.Contains(v, "██") || !strings.Contains(v, "d o t f i l e s") {
		t.Fatal("welcome screen missing DEV banner")
	}
	pressS := func(keys ...string) {
		for _, k := range keys {
			var msg tea.KeyMsg
			switch k {
			case " ":
				msg = tea.KeyMsg{Type: tea.KeySpace}
			case "enter":
				msg = tea.KeyMsg{Type: tea.KeyEnter}
			case "esc":
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}
			s.Update(msg)
		}
	}
	pressS("enter") // -> profile
	if s.step != 1 || !strings.Contains(stripANSI(s.View()), "Which machine") {
		t.Fatal("profile step")
	}
	pressS("enter") // -> theme
	v = stripANSI(s.View())
	if s.step != 2 || !strings.Contains(v, "Pick a theme") || !strings.Contains(v, "Nord") {
		t.Fatalf("theme step:\n%s", v)
	}
	pressS("j", "j", "enter") // pick a non-default theme -> features
	picked := s.themes[s.themeIdx]
	if picked == "ayu-dark" {
		t.Fatal("j/j should have moved off the default theme")
	}
	v = stripANSI(s.View())
	if s.step != 3 || !strings.Contains(v, "[x] nvim") || !strings.Contains(v, "configs") {
		t.Fatalf("features step:\n%s", v)
	}
	if !strings.Contains(v, "ghostty") || !strings.Contains(v, "coming soon") {
		t.Fatalf("ghostty coming-soon row missing:\n%s", v)
	}
	// ghostty is not toggleable
	for i, f := range s.feats {
		if f.key == "__ghostty" {
			s.cursor = i
		}
	}
	pressS(" ")
	if st.IsDisabled("__ghostty") {
		t.Fatal("coming-soon row must not toggle")
	}
	s.cursor = 0
	for i, f := range s.feats {
		if f.key == "nvim" {
			s.cursor = i
		}
	}
	pressS(" ") // toggle nvim off
	if !st.IsDisabled("nvim") {
		t.Fatal("toggle should disable nvim")
	}
	pressS(" ")     // back on
	pressS("enter") // -> review (single confirmation)
	v = stripANSI(s.View())
	if s.step != 4 || !strings.Contains(v, "Review") || !strings.Contains(v, "create") {
		t.Fatalf("review step:\n%s", v)
	}
	// single confirm runs everything
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if s.step != 5 || cmd == nil {
		t.Fatal("enter on review should run")
	}
	s.Update(cmd())
	if s.step != 6 || s.err != nil {
		t.Fatalf("done step, err=%v log=%s", s.err, s.log)
	}
	if st.Theme != picked {
		t.Fatalf("state theme = %q, want %q", st.Theme, picked)
	}
	if b, _ := os.ReadFile(filepath.Join(m.Home, ".config/nvim/init.lua")); string(b) != "repo" {
		t.Fatal("configs not applied")
	}
	if !strings.Contains(stripANSI(s.View()), "all done") {
		t.Fatal("done screen")
	}
}
