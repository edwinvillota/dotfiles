package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/edwinvillota/dotfiles/internal/manifest"
	"github.com/edwinvillota/dotfiles/internal/plan"
	"github.com/edwinvillota/dotfiles/internal/state"
)

func write(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) (*manifest.Manifest, *state.State) {
	root, home := t.TempDir(), t.TempDir()
	write(t, filepath.Join(root, "dotfiles.toml"), `
[unit.nvim]
src = "nvim"
dest = "~/.config/nvim"
granular = ["lua/plugins/*.lua"]
[unit.zsh]
src = "zsh/config"
dest = "~/.config/zsh"
secret = ["databases.zsh"]
[profile.work]
exclude = ["nvim/lua/plugins/obsidian.lua"]
`)
	write(t, filepath.Join(root, "nvim/init.lua"), "repo")
	write(t, filepath.Join(root, "nvim/lua/plugins/lsp.lua"), "same")
	write(t, filepath.Join(root, "nvim/lua/plugins/obsidian.lua"), "repo")
	write(t, filepath.Join(home, ".config/nvim/init.lua"), "live")
	write(t, filepath.Join(home, ".config/nvim/lua/plugins/lsp.lua"), "same")
	write(t, filepath.Join(home, ".config/nvim/lua/plugins/obsidian.lua"), "live")
	write(t, filepath.Join(home, ".config/nvim/lua/plugins/csv.lua"), "live")
	write(t, filepath.Join(home, ".config/zsh/databases.zsh"), "export X=1\n")
	m, err := manifest.Load(filepath.Join(root, "dotfiles.toml"), home)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := state.Load(home)
	st.Save() // state file exists -> TUI starts in normal mode, not first-run help
	return m, st
}

func press(md *Model, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		md.Update(msg)
	}
}

func TestTreeNavigationAndToggle(t *testing.T) {
	m, st := fixture(t)
	md := New(m, st)
	md.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if len(md.rows) != 2 || md.rows[0].key != "nvim" || md.rows[1].key != "zsh" {
		t.Fatalf("rows = %+v", md.rows)
	}
	// nvim backup: init ~1, obsidian ~1, csv +1
	if r := md.rows[0]; r.update != 2 || r.create != 1 || r.children != 3 {
		t.Errorf("nvim counts %+v", r)
	}
	press(md, "l") // expand
	if len(md.rows) != 5 || md.rows[1].key != "nvim/lua/plugins/csv.lua" {
		t.Fatalf("after expand rows = %v", keys(md.rows))
	}
	press(md, "j", " ") // disable csv.lua
	if !st.IsDisabled("nvim/lua/plugins/csv.lua") {
		t.Error("granule not disabled")
	}
	if md.rows[0].create != 0 || md.rows[0].skipped == 0 {
		t.Errorf("unit counts not recomputed: %+v", md.rows[0])
	}
	press(md, " ") // re-enable
	if st.IsDisabled("nvim/lua/plugins/csv.lua") {
		t.Error("granule not re-enabled")
	}
	press(md, "G")
	if md.cursor != len(md.rows)-1 {
		t.Error("G")
	}
	press(md, "g", "g")
	if md.cursor != 0 {
		t.Error("gg")
	}
	press(md, "h") // collapse from unit
	if len(md.rows) != 2 {
		t.Error("collapse")
	}
	press(md, " ") // disable whole nvim
	if !st.IsDisabled("nvim") {
		t.Error("unit not disabled")
	}
	for _, a := range md.plan.Changes() {
		if a.Unit == "nvim" {
			t.Errorf("disabled unit still has change: %+v", a)
		}
	}
	press(md, " ")
	// saved to disk
	st2, _ := state.Load(m.Home)
	if len(st2.Disabled) != 0 {
		t.Errorf("state not persisted: %v", st2.Disabled)
	}
}

func TestFilterProfileDirectionAndViews(t *testing.T) {
	m, st := fixture(t)
	md := New(m, st)
	md.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	press(md, "/", "o", "b", "s")
	if len(md.rows) != 1 || md.rows[0].key != "nvim/lua/plugins/obsidian.lua" {
		t.Errorf("filter rows = %v", keys(md.rows))
	}
	press(md, "esc")
	if len(md.rows) != 2 {
		t.Error("filter not cleared")
	}
	press(md, "p") // -> work
	if md.prof != "work" || st.Profile != "work" {
		t.Errorf("profile = %q", md.prof)
	}
	press(md, "l", "j", "j", "j")
	v := stripANSI(md.View())
	if !strings.Contains(v, "excluded by profile") {
		t.Error("preview should show profile exclusion")
	}
	press(md, "m")
	if md.dir != plan.Install {
		t.Error("direction toggle")
	}
	v = stripANSI(md.View())
	if !strings.Contains(v, "INSTALL") || !strings.Contains(v, "profile: work") {
		t.Errorf("header wrong:\n%s", v[:200])
	}
	press(md, "?")
	if !strings.Contains(stripANSI(md.View()), "switch backup") {
		t.Error("help overlay")
	}
	press(md, "esc", "x")
	if md.mode != modeConfirm || !strings.Contains(stripANSI(md.confirm), "Proceed?") {
		t.Errorf("confirm modal missing: mode=%v", md.mode)
	}
	press(md, "n")
	if md.mode != modeNormal {
		t.Error("n should cancel")
	}
}

func TestApplyFromTUI(t *testing.T) {
	m, st := fixture(t)
	md := New(m, st)
	md.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	press(md, "i") // install -> confirm
	if md.mode != modeConfirm {
		t.Fatal("expected confirm")
	}
	press(md, "y")
	cmd := md.runApply()
	msg := cmd()
	md.Update(msg)
	if md.mode != modeResult || !md.resultOK {
		t.Fatalf("apply failed: %s", md.result)
	}
	if b, _ := os.ReadFile(filepath.Join(m.Home, ".config/nvim/init.lua")); string(b) != "repo" {
		t.Error("install did not write")
	}
	// plan is rebuilt: nothing left to do
	if n := len(md.plan.Changes()); n != 0 {
		t.Errorf("post-install plan has %d changes", n)
	}
}

func keys(rs []row) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.key)
	}
	return out
}

func stripANSI(s string) string {
	var sb strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			in = true
		case in && r == 'm':
			in = false
		case !in:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func TestThemePickerApplies(t *testing.T) {
	m, st := fixture(t)
	md := New(m, st)
	md.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	press(md, "t")
	if md.mode != modeTheme {
		t.Fatal("t should open the theme picker")
	}
	v := stripANSI(md.View())
	if !strings.Contains(v, "Ayu Dark") || !strings.Contains(v, "Tokyo Night") {
		t.Fatalf("picker missing themes:\n%s", v)
	}
	press(md, "j") // move to second theme
	sel := md.themeNames[md.themeIdx]
	press(md, "enter")
	if md.mode != modeResult {
		t.Fatalf("apply should end in result modal, log=%s", md.result)
	}
	if !md.resultOK {
		t.Fatalf("theme apply failed: %s", md.result)
	}
	if st.Theme != sel {
		t.Fatalf("state theme = %q, want %q", st.Theme, sel)
	}
	if _, err := os.Stat(filepath.Join(m.Home, ".config/nvim/lua/config/theme-active.lua")); err != nil {
		t.Fatal("nvim theme-active.lua not written")
	}
	press(md, "t")
	if md.themeNames[md.themeIdx] != sel {
		t.Fatal("picker should highlight the active theme")
	}
	press(md, "esc")
	if md.mode != modeNormal {
		t.Fatal("esc should close the picker")
	}
}
