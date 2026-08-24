package theme

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/edwinvillota/dotfiles/internal/manifest"
)

func TestAllPalettesLoadAndValidate(t *testing.T) {
	names := Names()
	if len(names) != 9 {
		t.Fatalf("expected 9 themes, got %d: %v", len(names), names)
	}
	for _, n := range names {
		p, err := Load(n)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if p.Roles.Accent == "" || p.Roles.Panel == "" {
			t.Errorf("%s: roles not derived", n)
		}
	}
}

func TestLoadUnknown(t *testing.T) {
	if _, err := Load("solarized"); err == nil {
		t.Fatal("expected error for unknown theme")
	}
}

func TestAyuRoleOverridesMatchLegacyTUI(t *testing.T) {
	p, _ := Load("ayu-dark")
	for want, got := range map[string]string{
		"#e6b450": p.Roles.Accent, "#0d1017": p.Roles.Panel,
		"#1f2430": p.Roles.Line, "#131721": p.Roles.Sel,
	} {
		if got != want {
			t.Errorf("role = %s, want %s", got, want)
		}
	}
}

func TestMix(t *testing.T) {
	if got := Mix("#000000", "#ffffff", 0.5); got != "#808080" {
		t.Errorf("Mix = %s", got)
	}
	if got := Mix("#102030", "#102030", 0.3); got != "#102030" {
		t.Errorf("Mix identity = %s", got)
	}
}

func TestRenderersContainPaletteColors(t *testing.T) {
	p, _ := Load("nord")
	for name, out := range map[string]string{
		"wezterm": Wezterm(p), "zellij": ZellijTheme(p), "zjstatus": ZjstatusLayout(p),
		"zsh": ZshEnv(p), "yazi": YaziFlavor(p), "btop": BtopTheme(p), "ghdash": GhDashThemeBlock(p),
	} {
		if name != "zjstatus" && !strings.Contains(out, p.Primary.Background) {
			t.Errorf("%s output missing background %s", name, p.Primary.Background)
		}
		if strings.Contains(out, "{{") {
			t.Errorf("%s output has unexpanded tokens", name)
		}
	}
	if !strings.Contains(NvimActive(p), `"nord"`) {
		t.Error("nvim renderer missing colorscheme")
	}
}

func TestSwitchers(t *testing.T) {
	z := []byte("// x\ntheme \"ayu-dark\"\nrest\n")
	out, err := SetZellijTheme(z, "nord")
	if err != nil || string(out) != "// x\ntheme \"nord\"\nrest\n" {
		t.Fatalf("zellij switch: %q %v", out, err)
	}
	back, _ := SetZellijTheme(out, "ayu-dark")
	if string(back) != string(z) {
		t.Error("zellij switch does not round-trip")
	}
	if _, err := SetZellijTheme([]byte("no line here"), "nord"); err == nil {
		t.Error("expected error without theme line")
	}

	y := []byte("[flavor]\ndark = \"ayu-dark\"\n")
	out, err = SetYaziFlavor(y, "iceberg")
	if err != nil || !strings.Contains(string(out), `dark = "iceberg"`) {
		t.Fatalf("yazi switch: %q %v", out, err)
	}

	b := []byte("#c\ncolor_theme = \"Default\"\ntheme_background = true\n")
	out, err = SetBtopTheme(b, "jellybeans")
	if err != nil || !strings.Contains(string(out), `color_theme = "jellybeans"`) {
		t.Fatalf("btop switch: %q %v", out, err)
	}
}

func TestGhDashBlockReplaceAndAppend(t *testing.T) {
	p, _ := Load("nord")
	in := []byte("prSections:\n  - a\n\ntheme:\n  colors:\n    text:\n      primary: \"#x\"\nkeybindings:\n  prs: []\n")
	out, err := SetGhDashTheme(in, p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, p.Primary.Foreground) || strings.Contains(s, "#x") {
		t.Errorf("block not replaced:\n%s", s)
	}
	if !strings.Contains(s, "keybindings:") || !strings.Contains(s, "prSections:") {
		t.Errorf("surrounding config damaged:\n%s", s)
	}
	// no block yet: appended
	out, err = SetGhDashTheme([]byte("prSections:\n  - a\n"), p)
	if err != nil || !strings.Contains(string(out), "theme:") {
		t.Errorf("append failed: %v", err)
	}
	// idempotent
	again, err := SetGhDashTheme(out, p)
	if err != nil || string(again) != string(out) {
		t.Error("gh-dash switch not idempotent")
	}
}

// repoRoot finds the checkout root from this source file.
func repoRoot(t *testing.T) string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(f)))
}

// TestCommittedAssetsNotStale asserts the committed zellij/btop/yazi theme
// assets match what the renderers produce, so palette.toml stays the single
// source of truth. Regenerate with `dotfiles theme render`.
func TestCommittedAssetsNotStale(t *testing.T) {
	root := repoRoot(t)
	for _, n := range Names() {
		p, _ := Load(n)
		for path, want := range map[string]string{
			filepath.Join(root, "zellij/themes", n+".kdl"):                ZellijTheme(p),
			filepath.Join(root, "btop/themes", n+".theme"):                BtopTheme(p),
			filepath.Join(root, "yazi/flavors", n+".yazi", "flavor.toml"): YaziFlavor(p),
		} {
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s: %v (run `dotfiles theme render`)", path, err)
			}
			if string(got) != want {
				t.Errorf("%s is stale — run `dotfiles theme render`", path)
			}
		}
	}
}

func testManifest(t *testing.T) *manifest.Manifest {
	root, home := t.TempDir(), t.TempDir()
	os.Setenv("XDG_STATE_HOME", "") // ledger under home
	write := func(p, s string) {
		t.Helper()
		os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755)
		if err := os.WriteFile(filepath.Join(root, p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("dotfiles.toml", `
[unit.wezterm]
src = "wezterm"
dest = "~/.config/wezterm"
[unit.zellij]
src = "zellij"
dest = "~/.config/zellij"
[unit.btop]
src = "btop"
dest = "~/.config/btop"
`)
	m, err := manifest.Load(filepath.Join(root, "dotfiles.toml"), home)
	if err != nil {
		t.Fatal(err)
	}
	// "live" files for the mutating switchers
	for p, s := range map[string]string{
		".config/zellij/config.kdl": "theme \"ayu-dark\"\nrest\n",
		".config/btop/btop.conf":    "color_theme = \"ayu-dark\"\n",
	} {
		fp := filepath.Join(home, p)
		os.MkdirAll(filepath.Dir(fp), 0o755)
		os.WriteFile(fp, []byte(s), 0o644)
	}
	return m
}

func TestApplyIdempotentAndRoundTrip(t *testing.T) {
	m := testManifest(t)
	snap := func() map[string]string {
		out := map[string]string{}
		filepath.Walk(filepath.Join(m.Home, ".config"), func(p string, fi os.FileInfo, err error) error {
			if err == nil && !fi.IsDir() {
				b, _ := os.ReadFile(p)
				out[p] = string(b)
			}
			return nil
		})
		return out
	}
	if _, err := Apply(m, "ayu-dark", nil, nil); err != nil {
		t.Fatal(err)
	}
	base := snap()
	if len(base) < 4 {
		t.Fatalf("expected theme files written, got %v", base)
	}
	// re-apply same theme: no writes
	res, err := Apply(m, "ayu-dark", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 0 {
		t.Errorf("re-apply wrote %d files, want 0", res.Written)
	}
	// A -> B -> A is byte-identical
	if _, err := Apply(m, "tokyo-night", nil, nil); err != nil {
		t.Fatal(err)
	}
	mid := snap()
	if mid[filepath.Join(m.Home, ".config/zellij/config.kdl")] != "theme \"tokyo-night\"\nrest\n" {
		t.Error("zellij config not switched")
	}
	if _, err := Apply(m, "ayu-dark", nil, nil); err != nil {
		t.Fatal(err)
	}
	for p, s := range snap() {
		if base[p] != s {
			t.Errorf("%s differs after A->B->A round-trip", p)
		}
	}
}

func TestApplySkipsMissingLiveFiles(t *testing.T) {
	m := testManifest(t)
	os.Remove(filepath.Join(m.Home, ".config/btop/btop.conf"))
	res, err := Apply(m, "nord", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range res.Notices {
		if strings.Contains(n, "btop.conf") && strings.Contains(n, "skip") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected skip notice for btop.conf, got %v", res.Notices)
	}
}

func TestNormalizeRepo(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "zellij"), 0o755)
	os.WriteFile(filepath.Join(root, "zellij/config.kdl"), []byte("theme \"nord\"\n"), 0o644)
	if err := NormalizeRepo(root); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "zellij/config.kdl"))
	if string(b) != "theme \"ayu-dark\"\n" {
		t.Errorf("not normalized: %q", b)
	}
}
