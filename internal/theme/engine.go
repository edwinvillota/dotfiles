package theme

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/edwinvillota/dotfiles/internal/fsx"
	"github.com/edwinvillota/dotfiles/internal/ledger"
	"github.com/edwinvillota/dotfiles/internal/manifest"
)

// Default is the theme assumed when state.toml has none and the theme the
// repo's committed configs are normalized to.
const Default = "ayu-dark"

// Units are the manifest units the theme engine writes into. Installing any
// of them triggers a theme re-apply (see apply's postInstall hook).
var Units = []string{"wezterm", "nvim", "zsh", "zellij", "yazi", "btop", "gh-dash", "visidata"}

// Result reports what Apply changed and what the user must do to see it.
type Result struct {
	Written int
	Notices []string
	Reload  []string
}

// Apply switches the machine to theme `name`. Every write is ledgered so
// `dotfiles uninstall` still restores the machine fully. Re-applying the
// same theme is a no-op; A→B→A round-trips byte-identically.
// Pass a shared ledger when calling from inside apply.Run (which saves it
// afterwards); nil loads and saves one here.
func Apply(m *manifest.Manifest, name string, led *ledger.Ledger, log io.Writer) (*Result, error) {
	if log == nil {
		log = io.Discard
	}
	p, err := Load(name)
	if err != nil {
		return nil, err
	}
	ownLedger := led == nil
	if ownLedger {
		if led, err = ledger.Load(m.Home); err != nil {
			return nil, err
		}
	}
	now := time.Now()
	res := &Result{}

	write := func(unit, path string, content []byte, mustExist bool) error {
		old, rerr := os.ReadFile(path)
		if rerr != nil && mustExist {
			res.Notices = append(res.Notices, fmt.Sprintf("skip %s (not installed)", path))
			return nil
		}
		if rerr == nil && string(old) == string(content) {
			return nil
		}
		backup := ""
		if e := led.Find(path); e != nil {
			backup = e.Backup
		} else if rerr == nil {
			backup = ledger.BackupPath(m.Home, now, path)
			if err := fsx.CopyFile(path, backup); err != nil {
				return fmt.Errorf("preserve original: %w", err)
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, content, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			return err
		}
		kind := ledger.Created
		if backup != "" {
			kind = ledger.Replaced
		}
		led.Add(ledger.Entry{Path: path, Kind: kind, Backup: backup, Unit: unit, Time: now})
		fmt.Fprintf(log, "  theme    %s\n", path)
		res.Written++
		return nil
	}
	mutate := func(unit, path string, f func([]byte) ([]byte, error)) error {
		old, rerr := os.ReadFile(path)
		if rerr != nil {
			res.Notices = append(res.Notices, fmt.Sprintf("skip %s (not installed)", path))
			return nil
		}
		out, err := f(old)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return write(unit, path, out, true)
	}
	dest := func(unit string) string {
		if u, ok := m.Units[unit]; ok {
			return m.DestPath(u)
		}
		return ""
	}

	// Machine-local generated files (each read by a repo config with a
	// built-in fallback, so a missing file is never fatal for the tool).
	if d := dest("wezterm"); d != "" {
		if err := write("wezterm", filepath.Join(d, "theme.lua"), []byte(Wezterm(p)), false); err != nil {
			return res, err
		}
	}
	if d := dest("nvim"); d != "" {
		if err := write("nvim", filepath.Join(d, "lua/config/theme-active.lua"), []byte(NvimActive(p)), false); err != nil {
			return res, err
		}
	}
	if d := dest("zsh"); d != "" {
		if err := write("zsh", filepath.Join(d, "00-theme.zsh"), []byte(ZshEnv(p)), false); err != nil {
			return res, err
		}
	}
	if d := dest("zellij"); d != "" {
		if err := write("zellij", filepath.Join(d, "layouts/default.kdl"), []byte(ZjstatusLayout(p)), false); err != nil {
			return res, err
		}
		if err := mutate("zellij", filepath.Join(d, "config.kdl"), func(b []byte) ([]byte, error) {
			return SetZellijTheme(b, p.Name)
		}); err != nil {
			return res, err
		}
	}
	if d := dest("yazi"); d != "" {
		if err := mutate("yazi", filepath.Join(d, "theme.toml"), func(b []byte) ([]byte, error) {
			return SetYaziFlavor(b, p.Name)
		}); err != nil {
			return res, err
		}
	}
	if d := dest("btop"); d != "" {
		if err := mutate("btop", filepath.Join(d, "btop.conf"), func(b []byte) ([]byte, error) {
			return SetBtopTheme(b, p.Name)
		}); err != nil {
			return res, err
		}
	}
	if d := dest("gh-dash"); d != "" {
		if err := mutate("gh-dash", filepath.Join(d, "config.yml"), func(b []byte) ([]byte, error) {
			return SetGhDashTheme(b, p)
		}); err != nil {
			return res, err
		}
	}

	// VisiData's unit dest is the rc file itself, so the generated theme goes
	// next to it and .visidatarc execs it when present.
	if d := dest("visidata"); d != "" {
		if _, serr := os.Stat(d); serr != nil {
			res.Notices = append(res.Notices, fmt.Sprintf("skip %s (not installed)", d))
		} else if err := write("visidata", filepath.Join(m.Home, ".visidata", "theme.py"), []byte(VisiData(p)), false); err != nil {
			return res, err
		}
	}

	if ownLedger {
		if err := led.Save(); err != nil {
			return res, err
		}
	}
	res.Reload = []string{
		"wezterm: reloads live (watches its config)",
		"zellij: restart the session to pick up theme + status bar",
		"nvim: running instances keep the old colors; new ones use " + p.Label,
		"shell (fzf/bat): open a new shell or `source ~/.config/zsh/00-theme.zsh`",
		"yazi / btop / gh-dash / visidata: restart the app",
	}
	return res, nil
}

// RenderAssets regenerates the committed per-theme asset files under the
// repo root (zellij themes, btop themes, yazi flavors) from the palettes.
// A unit test asserts the working tree matches, so palettes stay the single
// source of truth.
func RenderAssets(root string) error {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			return err
		}
		files := map[string]string{
			filepath.Join(root, "zellij/themes", name+".kdl"):                ZellijTheme(p),
			filepath.Join(root, "btop/themes", name+".theme"):                BtopTheme(p),
			filepath.Join(root, "yazi/flavors", name+".yazi", "flavor.toml"): YaziFlavor(p),
		}
		for path, content := range files {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// NormalizeRepo resets theme-selecting lines in the REPO's configs back to
// the default theme. Backup (live -> repo) calls this so a machine's theme
// choice never leaks into a commit.
func NormalizeRepo(root string) error {
	p, err := Load(Default)
	if err != nil {
		return err
	}
	norm := func(rel string, f func([]byte) ([]byte, error)) error {
		path := filepath.Join(root, rel)
		old, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		out, err := f(old)
		if err != nil || string(out) == string(old) {
			return err
		}
		return os.WriteFile(path, out, 0o644)
	}
	steps := []struct {
		rel string
		f   func([]byte) ([]byte, error)
	}{
		{"zellij/config.kdl", func(b []byte) ([]byte, error) { return SetZellijTheme(b, Default) }},
		{"zellij/layouts/default.kdl", func([]byte) ([]byte, error) { return []byte(ZjstatusLayout(p)), nil }},
		{"yazi/theme.toml", func(b []byte) ([]byte, error) { return SetYaziFlavor(b, Default) }},
		{"btop/btop.conf", func(b []byte) ([]byte, error) { return SetBtopTheme(b, Default) }},
		{"gh-dash/config.yml", func(b []byte) ([]byte, error) { return SetGhDashTheme(b, p) }},
	}
	for _, s := range steps {
		if err := norm(s.rel, s.f); err != nil {
			return fmt.Errorf("normalize %s: %w", s.rel, err)
		}
	}
	return nil
}
