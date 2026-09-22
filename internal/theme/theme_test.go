package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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
		"wezterm": Wezterm(p), "kitty": Kitty(p), "zellij": ZellijTheme(p), "zjstatus": ZjstatusLayout(p),
		"zsh": ZshEnv(p), "yazi": YaziFlavor(p), "btop": BtopTheme(p), "ghdash": GhDashThemeBlock(p), "visidata": VisiData(p),
	} {
		// visidata speaks xterm-256 indices, never hex (see VisiData)
		if name != "zjstatus" && name != "visidata" && !strings.Contains(out, p.Primary.Background) {
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
[unit.visidata]
src = "visidata/.visidatarc"
dest = "~/.visidatarc"
`)
	m, err := manifest.Load(filepath.Join(root, "dotfiles.toml"), home)
	if err != nil {
		t.Fatal(err)
	}
	// "live" files for the mutating switchers
	for p, s := range map[string]string{
		".config/zellij/config.kdl": "theme \"ayu-dark\"\nrest\n",
		".config/btop/btop.conf":    "color_theme = \"ayu-dark\"\n",
		".visidatarc":               "# rc\n",
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
		for _, dir := range []string{".config", ".visidata"} {
			filepath.Walk(filepath.Join(m.Home, dir), func(p string, fi os.FileInfo, err error) error {
				if err == nil && !fi.IsDir() {
					b, _ := os.ReadFile(p)
					out[p] = string(b)
				}
				return nil
			})
		}
		return out
	}
	if _, err := Apply(m, "ayu-dark", nil, nil); err != nil {
		t.Fatal(err)
	}
	base := snap()
	if _, ok := base[filepath.Join(m.Home, ".visidata/theme.py")]; !ok {
		t.Error("visidata theme not written")
	}
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
	os.Remove(filepath.Join(m.Home, ".visidatarc"))
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
	if _, err := os.Stat(filepath.Join(m.Home, ".visidata/theme.py")); err == nil {
		t.Error("visidata theme written without an installed .visidatarc")
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

// VisiData's defaults are written as "white on black", and the terminal remaps
// ANSI black to the theme's palette (jellybeans: #929292, a mid grey), so the
// renderer must emit fixed cube indices only — never a low ANSI number or name.
// VisiData is given ANSI slots 0-15 deliberately, which is the opposite of
// what its stock theme does: the stock `white on black` names slots blindly
// and paints whatever the active terminal palette happens to hold there,
// which is how unthemed VisiData ends up grey. A slot is safe only when the
// palette that defines the remapping puts our exact color in it -- then the
// terminal hands back the hex we asked for instead of the nearest cube cell,
// which across these palettes is off by as much as deltaE 24.
//
// So the rule this test enforces is not "never use 0-15" but "use a slot only
// where the hex matches the palette exactly".
func TestVisiDataSlotsAreExact(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		// Paint may only return a slot for a color the palette holds in that
		// slot, and may never miss one it does.
		probes := []string{
			p.Primary.Foreground, p.Primary.Background, p.Roles.Accent, p.Roles.Accent2,
			p.Roles.Good, p.Roles.Warn, p.Roles.Error, p.Roles.Dim, p.Roles.Panel,
			p.Roles.Line, p.Roles.Sel, p.Selection.Text, p.Cursor.Cursor,
		}
		sl := p.ansiSlots()
		probes = append(probes, sl[:]...)
		for t2 := 0.0; t2 <= 1.0; t2 += 0.01 {
			probes = append(probes, Mix(p.Primary.Background, p.Primary.Foreground, t2))
			probes = append(probes, Mix(p.Roles.Accent, p.Primary.Foreground, t2))
		}
		for _, hex := range probes {
			i := p.Paint(hex)
			switch {
			case i < 0 || i > 255:
				t.Errorf("%s: Paint(%s) = %d, out of range", name, hex, i)
			case i < 16:
				if got := sl[i]; !strings.EqualFold(got, hex) {
					t.Errorf("%s: Paint(%s) = slot %d, which paints %s -- a slot is only allowed on an exact match",
						name, hex, i, got)
				}
			default:
				if s := p.slot(hex); s >= 0 {
					t.Errorf("%s: Paint(%s) = cube %d but the palette holds that exact color in slot %d",
						name, hex, i, s)
				}
			}
		}

		// and the rendered file may only carry indices produced that way
		out := VisiData(p)
		if !strings.Contains(out, "vd.options.color_default = ") {
			t.Fatal("visidata renderer missing color_default")
		}
		re := regexp.MustCompile(`vd\.options\.(color_\w+) = "([^"]*)"`)
		seen, slots := 0, 0
		for _, m := range re.FindAllStringSubmatch(out, -1) {
			seen++
			for _, tok := range strings.Fields(m[2]) {
				switch tok {
				case "on", "bold", "underline", "italic", "reverse":
					continue
				}
				n, err := strconv.Atoi(tok)
				if err != nil {
					t.Errorf("%s: %s: %q is not a color index", name, m[1], tok)
					continue
				}
				if n < 0 || n > 255 {
					t.Errorf("%s: %s: index %d out of range", name, m[1], n)
					continue
				}
				if n < 16 {
					slots++
					if p.slot(sl[n]) < 0 {
						t.Errorf("%s: %s: slot %d is not a palette color", name, m[1], n)
					}
				}
			}
		}
		if seen < 20 {
			t.Errorf("%s: only %d color options rendered", name, seen)
		}
		// ayu-dark pins an accent that is not one of its ANSI colors, but every
		// palette here has several roles that are -- if nothing maps, the
		// exact-match path has silently stopped working.
		if slots == 0 {
			t.Errorf("%s: no color resolved to an ANSI slot, so every color is quantized again", name)
		}
	}
}

func TestXterm256(t *testing.T) {
	for hex, want := range map[string]int{
		"#000000": 16, "#ffffff": 231, "#121212": 233, "#ff0000": 196,
	} {
		if got := Xterm256(hex); got != want {
			t.Errorf("Xterm256(%s) = %d, want %d", hex, got, want)
		}
	}
	if n := Xterm256("#929292"); n < 16 {
		t.Errorf("Xterm256 returned remapped index %d", n)
	}
}

// Every theme must resolve a full set of picker colors, and NvimActive must
// carry them: highlights.lua falls back to ayu-dark for any key it misses, so
// a gap would silently paint one theme's picker in another theme's colors.
func TestPickerColorsComplete(t *testing.T) {
	keys := []string{
		"file", "folder", "hidden", "ignored", "match", "selection", "prompt",
		"git_added", "git_modified", "git_deleted", "git_renamed", "git_untracked",
	}
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out := NvimActive(p)
		for _, k := range keys {
			if !strings.Contains(out, k+" = \"#") {
				t.Errorf("%s: NvimActive is missing picker.%s", name, k)
			}
		}
	}
}

// ayu-dark's picker colors were hand-tuned before the theme system existed.
// They are pinned in its palette.toml so switching themes and coming back
// leaves the picker exactly as it was.
func TestAyuPickerPinned(t *testing.T) {
	p, err := Load("ayu-dark")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"file": "#a3b7cc", "match": "#cc7e7e", "git_modified": "#a3b7cc",
		"git_untracked": "#d4d4d4", "ignored": "#5c6370",
	}
	got := map[string]string{
		"file": p.Picker.File, "match": p.Picker.Match, "git_modified": p.Picker.GitModified,
		"git_untracked": p.Picker.GitUntracked, "ignored": p.Picker.Ignored,
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("ayu-dark picker.%s = %s, want %s", k, got[k], w)
		}
	}
}

// Git status colors must be visually distinct, or a modified file reads the
// same as an untracked one -- the bug that made github-dark unusable.
func TestGitStatusColorsDistinct(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]string{}
		for role, c := range map[string]string{
			"added": p.Picker.GitAdded, "modified": p.Picker.GitModified,
			"deleted": p.Picker.GitDeleted, "untracked": p.Picker.GitUntracked,
		} {
			if role == "added" || role == "untracked" {
				continue // both are green by convention
			}
			if prev, dup := seen[c]; dup {
				t.Errorf("%s: git %s and %s are both %s", name, prev, role, c)
			}
			seen[c] = role
		}
	}
}

func vdIndices(s string) []int {
	var out []int
	for _, f := range strings.Fields(s) {
		if n, err := strconv.Atoi(f); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// VisiData composites color_bottom_hdr at precedence 5 over the last line of
// the column header, and the header is one line for an ordinary sheet, so
// color_default_hdr never reaches it. Setting bottom_hdr to the plain
// foreground is what drew every theme's column names in white.
func TestVisiDataHeaderCarriesAccent(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		got := vdIndices(vdOpts(t, name)["color_bottom_hdr"])
		if len(got) != 2 {
			t.Errorf("%s: color_bottom_hdr wants a foreground and a background, got %v", name, got)
			continue
		}
		if want := p.Paint(p.Roles.Accent); got[0] != want {
			t.Errorf("%s: color_bottom_hdr foreground is %d, want the accent %d", name, got[0], want)
		}
		if got[0] == p.Paint(p.Primary.Foreground) {
			t.Errorf("%s: color_bottom_hdr foreground is the plain foreground, so the header reads as unthemed", name)
		}
	}
}

// The 256-color cube has almost no dark saturated cells, so any chrome derived
// by mixing a dark background toward its foreground quantizes onto the
// grayscale ramp. A theme built only that way renders VisiData near-monochrome
// whatever its palette -- gray rules, white column names, white chrome. These
// six surfaces are where the palette has to land for VisiData to read as
// themed at all, so every one of them must resolve to a colored index.
func TestVisiDataIsNotMonochrome(t *testing.T) {
	required := []string{
		"color_bottom_hdr",    // column names
		"color_column_sep",    // the grid
		"color_key_col",       // key columns
		"color_selected_row",  // selection
		"color_current_cell",  // where the cursor is
		"color_active_status", // the status bar
	}
	for _, name := range Names() {
		o := vdOpts(t, name)
		for _, k := range required {
			hued := false
			for _, i := range vdIndices(o[k]) {
				if palOf(t, name).hue(i) {
					hued = true
				}
			}
			if !hued {
				t.Errorf("%s: %s = %q resolves to no colored index, so it renders gray under every theme", name, k, o[k])
			}
		}
	}
}

// Column separators draw the grid. `line` is a 12% mix off the background,
// which quantizes to a single index step and renders the rules invisible, so
// the renderer walks toward accent2 instead. Check they end up readable.
func TestVisiDataSeparatorsAreVisible(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		got := vdIndices(vdOpts(t, name)["color_column_sep"])
		if len(got) != 1 {
			t.Fatalf("%s: color_column_sep should be a bare foreground, got %v", name, got)
		}
		if c := p.contrastOnBody(got[0]); c < 2.2 {
			t.Errorf("%s: column separators contrast %.2f against the sheet, want >= 2.20", name, c)
		}
	}
}

// A fuzzy match has to be findable inside the file name it is drawn in.
// Contrast ratio alone does not catch this: on kanagawa-wave the accent and
// the foreground scored 1.16 and were both warm and light, so the match read
// as more text. deltaE is what separates "different color" from "same color".
func TestPickerMatchStandsOut(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		if d := deltaE(p.Picker.Match, p.Picker.File); d < minMatchDelta {
			t.Errorf("%s: match %s is only deltaE %.1f from the file name %s, want >= %d",
				name, p.Picker.Match, d, p.Picker.File, minMatchDelta)
		}
	}
}

// Accent colors are tuned against the sheet background, but VisiData also
// paints them on the panel that menus, popups and the sidebar sit on. On
// palettes whose panel is a mid-dark gray a muted accent drops under the
// legibility floor there, which is how the aggregator summary ended up at 3.9
// on nord. Every foreground VisiData draws on the panel has to clear it.
func TestVisiDataPanelTextIsLegible(t *testing.T) {
	floors := map[string]float64{
		"color_bottom_hdr": 4.5, "color_default_hdr": 4.5, "color_menu": 4.5,
		"color_sidebar": 4.5, "color_sidebar_title": 4.5, "color_top_status": 4.5,
		"color_active_status": 4.5, "color_cmdpalette": 4.5, "color_aggregator": 4.5,
		"color_code": 4.5, "color_keystrokes": 4.5, "color_edit_cell": 4.5,
		"color_inactive_status": 3.0, "color_menu_help": 3.0, "color_guide_unwritten": 3.0,
	}
	for _, name := range Names() {
		p := palOf(t, name)
		for opt, floor := range floors {
			spec := vdOpts(t, name)[opt]
			parts := strings.SplitN(spec, " on ", 2)
			if len(parts) != 2 {
				continue // foreground-only options are composited elsewhere
			}
			fg, bg := vdIndices(parts[0]), vdIndices(parts[1])
			if len(fg) != 1 || len(bg) != 1 {
				continue
			}
			if r := p.contrast(fg[0], bg[0]); r < floor {
				t.Errorf("%s: %s = %q has contrast %.2f, want >= %.1f", name, opt, spec, r, floor)
			}
		}
	}
}

// The git status colors share one narrow column in the picker, so they have to
// be told apart from each other and not just from the background. Palettes
// collapse pairs in ways contrast ratios do not catch: github-dark-colorblind
// moves red to orange, landing it beside its yellow at deltaE 17.
func TestGitStatusColorsSeparate(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		if name == "ayu-dark" {
			continue // pinned to hand-tuned values; see its palette.toml
		}
		cols := map[string]string{
			"added": p.Picker.GitAdded, "modified": p.Picker.GitModified,
			"deleted": p.Picker.GitDeleted, "renamed": p.Picker.GitRenamed,
		}
		names := []string{"added", "modified", "deleted", "renamed"}
		for i, a := range names {
			for _, b := range names[i+1:] {
				if d := deltaE(cols[a], cols[b]); d < minGitDelta {
					t.Errorf("%s: git %s (%s) and %s (%s) are only deltaE %.1f apart, want >= %d",
						name, a, cols[a], b, cols[b], d, minGitDelta)
				}
			}
		}
	}
}

// The markup colors only reach the screen on a colorscheme that left the JSX
// captures unstyled, so they have to be clearly not the plain foreground (or
// they change nothing) and clearly not each other (or the attribute name reads
// as part of the element name).
func TestTagColorsAreDistinct(t *testing.T) {
	const minTagDelta = 20
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		fg := p.Primary.Foreground
		for label, c := range map[string]string{"tag": p.Tag.Tag, "attribute": p.Tag.Attribute} {
			if d := deltaE(c, fg); d < minTagDelta {
				t.Errorf("%s: tag.%s %s is only deltaE %.1f from the foreground %s, so filling it would change nothing",
					name, label, c, d, fg)
			}
		}
		if d := deltaE(p.Tag.Tag, p.Tag.Attribute); d < minTagDelta {
			t.Errorf("%s: tag %s and attribute %s are only deltaE %.1f apart",
				name, p.Tag.Tag, p.Tag.Attribute, d)
		}
	}
}

// A picker row's loudness has to match how much it matters: a source file
// first, a dotfile behind it, a node_modules hit quietest. The derived ladder
// is monotonic by construction (file = fg, hidden = 55% of the way there,
// ignored = 28%), but a palette that pins these by hand can invert it --
// ayu-dark's hidden paths used to outshine its file names, 11.2 against 9.4.
func TestPickerLadderDescends(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		bg := p.Primary.Background
		file, hidden, ignored := contrastHex(p.Picker.File, bg), contrastHex(p.Picker.Hidden, bg), contrastHex(p.Picker.Ignored, bg)
		if hidden > file {
			t.Errorf("%s: hidden paths (%s, %.2f) are louder than file names (%s, %.2f)",
				name, p.Picker.Hidden, hidden, p.Picker.File, file)
		}
		if ignored > hidden {
			t.Errorf("%s: ignored paths (%s, %.2f) are louder than hidden ones (%s, %.2f)",
				name, p.Picker.Ignored, ignored, p.Picker.Hidden, hidden)
		}
	}
}

// De-emphasized text has to read as de-emphasized. `dim` defaults to the
// palette's bright black, which is a mid grey on most themes but #bdbdbd on
// jellybeans and #a6a69c on kanagawa-dragon -- close enough to the foreground
// that an inactive zellij tab was as loud as the active one (9.97 against
// 13.92, where every other palette sits near a third of the foreground).
func TestDimReadsDimmer(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		fgc := contrastHex(p.Primary.Foreground, p.Primary.Background)
		dimc := contrastHex(p.Roles.Dim, p.Primary.Background)
		if dimc > maxDimShare*fgc {
			t.Errorf("%s: dim %s carries %.2f of the %.2f the foreground does (%.0f%%), want at most %.0f%%",
				name, p.Roles.Dim, dimc, fgc, 100*dimc/fgc, 100*maxDimShare)
		}
	}
}

// zellij draws its bars from the theme's eight colors, not from fg/bg: the
// status bar is `black` text on `white` and `white` text on `black`. A palette
// whose ANSI black is not dark therefore has no dark end -- jellybeans' black
// #929292 against its white #dedede put every segment of the bottom bar
// between 1.86 and 2.31 contrast, measured in a real zellij session.
func TestZellijUIHasDarkAnchor(t *testing.T) {
	re := regexp.MustCompile(`(?m)^\s+(fg|black|white)\s+"(#[0-9a-fA-F]{6})"`)
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, m := range re.FindAllStringSubmatch(ZellijTheme(p), -1) {
			got[m[1]] = m[2]
		}
		if len(got) != 3 {
			t.Fatalf("%s: zellij theme is missing fg/black/white: %v", name, got)
		}
		if c := contrastHex(got["black"], got["fg"]); c < minUIAnchor {
			t.Errorf("%s: zellij black %s has contrast %.2f against fg %s, want >= %.1f",
				name, got["black"], c, got["fg"], minUIAnchor)
		}
		if c := contrastHex(got["white"], got["black"]); c < minUIPair {
			t.Errorf("%s: zellij white %s on black %s has contrast %.2f, want >= %.1f",
				name, got["white"], got["black"], c, minUIPair)
		}
	}
}

// yazi paints the glyph in front of every row from its own [icon] table, whose
// colors are hard-coded hexes -- the generic folder icon is #03a9f4 under every
// theme, so folders read blue whatever the palette says. The flavor restates
// yazi's fallback rules with palette colors; these are the ones that reach the
// screen for an ordinary directory or file.
func TestYaziIconsComeFromPalette(t *testing.T) {
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		out := YaziFlavor(p)
		if !strings.Contains(out, "prepend_conds = [") {
			t.Errorf("%s: flavor has no [icon] rules, so yazi paints its own icon colors", name)
			continue
		}
		for _, want := range []struct{ rule, color string }{
			{`{ if = "dir",           text = "\ue5ff", fg = "%s" }`, p.Roles.Accent2},
			{`{ if = "dir & hovered", text = "\ue5fe", fg = "%s" }`, p.Roles.Accent2},
			{`{ if = "exec",          text = "\uf489", fg = "%s" }`, p.Roles.Good},
			{`{ if = "!dir",          text = "\uf15b", fg = "%s" }`, p.Primary.Foreground},
		} {
			if line := fmt.Sprintf(want.rule, want.color); !strings.Contains(out, line) {
				t.Errorf("%s: flavor is missing %s", name, line)
			}
		}
		for _, stock := range []string{"#03a9f4", "#8bc34a", "#9e9e9e", "#cddc39", "#f44336"} {
			if strings.Contains(out, `fg = "`+stock+`"`) {
				t.Errorf("%s: flavor still carries yazi's stock icon color %s", name, stock)
			}
		}
	}
}
