// Package theme renders one shared palette into every themable tool's
// native config format and switches the active theme on a machine.
package theme

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/edwinvillota/dotfiles/themes"
)

type ANSI struct {
	Black   string `toml:"black"`
	Red     string `toml:"red"`
	Green   string `toml:"green"`
	Yellow  string `toml:"yellow"`
	Blue    string `toml:"blue"`
	Magenta string `toml:"magenta"`
	Cyan    string `toml:"cyan"`
	White   string `toml:"white"`
}

// Roles are the semantic colors the renderers consume. Every role has a
// derivation rule from the raw palette; a palette.toml may override any of
// them under [roles].
type Roles struct {
	Accent  string `toml:"accent"`  // highlights, titles, active tab
	Accent2 string `toml:"accent2"` // borders, links, secondary emphasis
	Good    string `toml:"good"`    // success
	Warn    string `toml:"warn"`    // warnings
	Error   string `toml:"error"`   // errors
	Dim     string `toml:"dim"`     // de-emphasized text
	Panel   string `toml:"panel"`   // status bars, modals
	Line    string `toml:"line"`    // borders, separators
	Sel     string `toml:"sel"`     // cursor-row background
}

// Picker are the Snacks-picker colors nvim/lua/config/highlights.lua paints.
// The picker draws file rows itself instead of asking the colorscheme, so
// without these every theme inherits whatever was hard-coded -- the reason
// only ayu-dark ever looked right. Each key derives from the palette; a
// palette.toml may override any of them under [picker].
type Picker struct {
	File         string `toml:"file"`          // matched file name
	Folder       string `toml:"folder"`        // a directory row
	Hidden       string `toml:"hidden"`        // dotfile path / hidden git status
	Ignored      string `toml:"ignored"`       // git-ignored path (node_modules, dist)
	Match        string `toml:"match"`         // the typed substring inside a row
	Selection    string `toml:"selection"`     // multi-select marker
	Prompt       string `toml:"prompt"`        // the "> " input prompt
	GitAdded     string `toml:"git_added"`     // A / staged
	GitModified  string `toml:"git_modified"`  // M
	GitDeleted   string `toml:"git_deleted"`   // D
	GitRenamed   string `toml:"git_renamed"`   // R
	GitUntracked string `toml:"git_untracked"` // ??
}

// Tag are the JSX/HTML markup colors nvim/lua/config/highlights.lua fills in
// when the colorscheme leaves them unstyled. Several colorschemes define no
// treesitter tag captures at all, so the element name and the attribute name
// fall through to plain text and the whole of JSX reads as one color.
type Tag struct {
	Tag       string `toml:"tag"`       // the element name: <div
	Attribute string `toml:"attribute"` // the attribute name: data-slot=
}

type Palette struct {
	Name   string `toml:"name"`
	Label  string `toml:"label"`
	Source string `toml:"source"`
	Nvim   string `toml:"nvim"` // nvim colorscheme name

	Primary struct {
		Foreground string `toml:"foreground"`
		Background string `toml:"background"`
	} `toml:"primary"`
	Cursor struct {
		Text   string `toml:"text"`
		Cursor string `toml:"cursor"`
	} `toml:"cursor"`
	Selection struct {
		Text       string `toml:"text"`
		Background string `toml:"background"`
	} `toml:"selection"`
	Normal ANSI   `toml:"normal"`
	Bright ANSI   `toml:"bright"`
	Roles  Roles  `toml:"roles"`
	Picker Picker `toml:"picker"`
	Tag    Tag    `toml:"tag"`
}

var hexRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Load parses and validates one embedded palette.
func Load(name string) (*Palette, error) {
	b, err := themes.FS.ReadFile(name + "/palette.toml")
	if err != nil {
		return nil, fmt.Errorf("unknown theme %q", name)
	}
	p := &Palette{}
	if err := toml.Unmarshal(b, p); err != nil {
		return nil, fmt.Errorf("theme %s: %w", name, err)
	}
	if p.Name != name {
		return nil, fmt.Errorf("theme %s: palette declares name %q", name, p.Name)
	}
	p.deriveRoles()
	p.derivePicker()
	p.deriveTag()
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("theme %s: %w", name, err)
	}
	return p, nil
}

// Names lists all embedded themes, sorted.
func Names() []string {
	ents, _ := themes.FS.ReadDir(".")
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// maxDimShare is the most of the foreground's contrast that de-emphasized text
// may carry before it stops reading as de-emphasized. targetDimShare is what a
// re-derived dim aims for -- the middle of what the palettes here already do.
const (
	maxDimShare    = 0.45
	targetDimShare = 0.33
	minDimOnBody   = 3.0
)

func (p *Palette) deriveRoles() {
	def := func(dst *string, v string) {
		if *dst == "" {
			*dst = v
		}
	}
	def(&p.Roles.Accent, p.Bright.Yellow)
	def(&p.Roles.Accent2, p.Bright.Blue)
	def(&p.Roles.Good, p.Bright.Green)
	def(&p.Roles.Warn, p.Normal.Yellow)
	def(&p.Roles.Error, p.Normal.Red)
	pinnedDim := p.Roles.Dim != ""
	def(&p.Roles.Dim, p.Bright.Black)
	if !pinnedDim {
		p.Roles.Dim = p.dimmed(p.Roles.Dim)
	}
	def(&p.Roles.Panel, Mix(p.Primary.Background, p.Primary.Foreground, 0.06))
	def(&p.Roles.Line, Mix(p.Primary.Background, p.Primary.Foreground, 0.12))
	def(&p.Roles.Sel, p.Selection.Background)
}

// dimmed keeps `dim` actually de-emphasized. It defaults to the palette's
// bright black, which on most themes is a mid grey -- but jellybeans' is
// #bdbdbd and kanagawa-dragon's #a6a69c, near enough to the foreground that
// an inactive zellij tab read as loud as the active one (9.97 against 13.92 on
// jellybeans, where every other theme sits around a third of the foreground's
// contrast). When that happens, re-derive it as a mix off the background at
// the share the rest of the palettes already use.
func (p *Palette) dimmed(dim string) string {
	bg, fg := p.Primary.Background, p.Primary.Foreground
	fgc := contrastHex(fg, bg)
	if contrastHex(dim, bg) <= maxDimShare*fgc {
		return dim
	}
	want := targetDimShare * fgc
	if want < minDimOnBody {
		want = minDimOnBody
	}
	best, bestErr := dim, math.Inf(1)
	for t := 0.10; t <= 0.95; t += 0.01 {
		x := Mix(bg, fg, t)
		c := contrastHex(x, bg)
		if c < minDimOnBody {
			continue
		}
		if e := math.Abs(c - want); e < bestErr {
			best, bestErr = x, e
		}
	}
	return best
}

// derivePicker fills every unset [picker] key from the palette. Hidden and
// ignored rows step down in three stages towards the background so a
// node_modules hit never reads as loud as a source file.
// minMatchDelta is the perceptual distance a match span needs from the file
// name around it before it reads as highlighted rather than as more text.
const minMatchDelta = 25

// minGitDelta is the perceptual distance two git status colors need from each
// other. They sit in one narrow column, so telling them apart is the whole job.
const minGitDelta = 18

func (p *Palette) derivePicker() {
	// A palette may pin any of these; only derived values get adjusted below,
	// so a hand-tuned theme stays exactly as its author wrote it.
	pinned := p.Picker
	def := func(dst *string, v string) {
		if *dst == "" {
			*dst = v
		}
	}
	bg, fg := p.Primary.Background, p.Primary.Foreground
	def(&p.Picker.File, fg)
	def(&p.Picker.Folder, p.Roles.Accent2)
	def(&p.Picker.Hidden, Mix(bg, fg, 0.55))
	def(&p.Picker.Ignored, Mix(bg, fg, 0.28))
	// The match span has to be findable inside the file name it is drawn in.
	// The accent is the right color for it, but on a warm palette like
	// kanagawa-wave the accent and the foreground are nearly the same color
	// and bold alone does not rescue it -- so fall back to whichever role
	// reads furthest from the file name.
	if p.Picker.Match == "" {
		p.Picker.Match = p.Roles.Accent
		if deltaE(p.Roles.Accent, p.Picker.File) < minMatchDelta {
			for _, c := range []string{p.Roles.Accent2, p.Roles.Warn, p.Roles.Error, p.Roles.Good} {
				if deltaE(c, p.Picker.File) > deltaE(p.Picker.Match, p.Picker.File) {
					p.Picker.Match = c
				}
			}
		}
	}
	def(&p.Picker.Selection, p.Roles.Good)
	def(&p.Picker.Prompt, p.Roles.Accent2)
	def(&p.Picker.GitAdded, p.Normal.Green)
	def(&p.Picker.GitModified, p.Normal.Yellow)
	def(&p.Picker.GitDeleted, p.Normal.Red)
	def(&p.Picker.GitRenamed, p.Normal.Cyan)
	def(&p.Picker.GitUntracked, p.Bright.Green)

	// The git status colors share one narrow column, so they have to be told
	// apart from each other, not just from the background. Some palettes
	// collapse a pair: github-dark-colorblind moves red to orange, landing it
	// next to its yellow, and both kanagawa palettes put added and renamed in
	// nearly the same green. Nudge the less load-bearing one of the pair until
	// it separates -- lightness first, which is also the axis that survives
	// color vision deficiency.
	separate := func(dst *string, was string, from string, alts ...string) {
		if was != "" || deltaE(*dst, from) >= minGitDelta {
			return // pinned by the palette, or already distinct
		}
		for _, a := range alts {
			if deltaE(a, from) >= minGitDelta {
				*dst = a
				return
			}
		}
		for t := 0.10; t <= 1.0; t += 0.05 {
			if x := Mix(*dst, p.Primary.Foreground, t); deltaE(x, from) >= minGitDelta {
				*dst = x
				return
			}
		}
	}
	separate(&p.Picker.GitModified, pinned.GitModified, p.Picker.GitDeleted, p.Bright.Yellow)
	separate(&p.Picker.GitRenamed, pinned.GitRenamed, p.Picker.GitAdded, p.Bright.Cyan, p.Normal.Magenta)
}

// deriveTag fills the markup colors. They only ever reach the screen on a
// colorscheme that left the capture unstyled, so the job is simply to be
// clearly not the plain foreground, and clearly not each other.
func (p *Palette) deriveTag() {
	if p.Tag.Tag == "" {
		p.Tag.Tag = p.Roles.Accent2
	}
	if p.Tag.Attribute == "" {
		p.Tag.Attribute = p.Roles.Accent
	}
}

func (p *Palette) validate() error {
	if p.Label == "" {
		return fmt.Errorf("missing label")
	}
	if p.Nvim == "" {
		return fmt.Errorf("missing nvim colorscheme")
	}
	all := map[string]string{
		"primary.foreground": p.Primary.Foreground, "primary.background": p.Primary.Background,
		"cursor.text": p.Cursor.Text, "cursor.cursor": p.Cursor.Cursor,
		"selection.text": p.Selection.Text, "selection.background": p.Selection.Background,
		"roles.accent": p.Roles.Accent, "roles.accent2": p.Roles.Accent2,
		"roles.good": p.Roles.Good, "roles.warn": p.Roles.Warn, "roles.error": p.Roles.Error,
		"roles.dim": p.Roles.Dim, "roles.panel": p.Roles.Panel, "roles.line": p.Roles.Line,
		"roles.sel":   p.Roles.Sel,
		"picker.file": p.Picker.File, "picker.folder": p.Picker.Folder,
		"picker.hidden":  p.Picker.Hidden,
		"picker.ignored": p.Picker.Ignored, "picker.match": p.Picker.Match,
		"picker.selection": p.Picker.Selection, "picker.prompt": p.Picker.Prompt,
		"picker.git_added": p.Picker.GitAdded, "picker.git_modified": p.Picker.GitModified,
		"picker.git_deleted": p.Picker.GitDeleted, "picker.git_renamed": p.Picker.GitRenamed,
		"picker.git_untracked": p.Picker.GitUntracked,
		"tag.tag":              p.Tag.Tag,
		"tag.attribute":        p.Tag.Attribute,
	}
	for pre, a := range map[string]*ANSI{"normal": &p.Normal, "bright": &p.Bright} {
		for k, v := range map[string]string{
			"black": a.Black, "red": a.Red, "green": a.Green, "yellow": a.Yellow,
			"blue": a.Blue, "magenta": a.Magenta, "cyan": a.Cyan, "white": a.White,
		} {
			all[pre+"."+k] = v
		}
	}
	for k, v := range all {
		if !hexRe.MatchString(v) {
			return fmt.Errorf("%s: %q is not a #rrggbb color", k, v)
		}
	}
	return nil
}

// Mix blends a towards b by t (0..1) in RGB space.
// idxRGB maps an xterm-256 index back to rgb, mirroring the 6x6x6 cube and
// grayscale ramp Xterm256 searches. Indices 0-15 are never produced by
// Xterm256, so they are not handled.
func idxRGB(i int) (int, int, int) {
	levels := []int{0, 95, 135, 175, 215, 255}
	if i >= 16 && i <= 231 {
		n := i - 16
		return levels[n/36], levels[n/6%6], levels[n%6]
	}
	v := 8 + 10*(i-232)
	return v, v, v
}

// labOf converts a hex color to CIE L*a*b*. Contrast ratios only see
// luminance, which is blind to two colors that differ in hue but not
// brightness -- gold on warm cream scores 1.00 and is still legible, while
// gold on pale gold scores the same and is not. deltaE separates those.
func labOf(hex string) (l, a, b float64) {
	rr, gg, bb := rgb(hex)
	lin := func(c int) float64 {
		s := float64(c) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	r, g, bl := lin(rr), lin(gg), lin(bb)
	x := (0.4124*r + 0.3576*g + 0.1805*bl) / 0.95047
	y := 0.2126*r + 0.7152*g + 0.0722*bl
	z := (0.0193*r + 0.1192*g + 0.9505*bl) / 1.08883
	f := func(t float64) float64 {
		if t > 0.008856 {
			return math.Cbrt(t)
		}
		return 7.787*t + 16.0/116.0
	}
	fx, fy, fz := f(x), f(y), f(z)
	return 116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)
}

// deltaE is the CIE76 perceptual distance between two colors. Roughly: under
// 15 the two read as the same color at text size, over 25 they read apart.
func deltaE(a, b string) float64 {
	l1, a1, b1 := labOf(a)
	l2, a2, b2 := labOf(b)
	return math.Sqrt((l1-l2)*(l1-l2) + (a1-a2)*(a1-a2) + (b1-b2)*(b1-b2))
}

// idxHex is the color an xterm-256 index actually paints, as hex. Indices
// 0-15 are the terminal's palette slots and are never produced by Xterm256,
// so they are not handled here.
func idxHex(i int) string {
	r, g, b := idxRGB(i)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// lumHex is the WCAG relative luminance of a hex color.
func lumHex(hex string) float64 {
	r, g, b := rgb(hex)
	f := func(c int) float64 {
		s := float64(c) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*f(r) + 0.7152*f(g) + 0.0722*f(b)
}

// contrastHex is the WCAG contrast ratio between two hex colors. VisiData's
// sheet body has no background index -- it inherits the terminal's, which is
// the palette's own background at full fidelity -- so anything drawn on the
// body has to be measured against that hex, not against a quantized index.
func contrastHex(a, b string) float64 {
	l1, l2 := lumHex(a), lumHex(b)
	if l2 > l1 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// contrastOnBody is the contrast of a color index against the true terminal
// background -- VisiData's sheet body carries no background index, so it
// inherits the terminal's, which is this palette's own background at full
// fidelity.
func (p *Palette) contrastOnBody(i int) float64 {
	return contrastHex(p.PaintedHex(i), p.Primary.Background)
}

func Mix(a, b string, t float64) string {
	ar, ag, ab := rgb(a)
	br, bg, bb := rgb(b)
	c := func(x, y int) int { return x + int(float64(y-x)*t+0.5) }
	return fmt.Sprintf("#%02x%02x%02x", c(ar, br), c(ag, bg), c(ab, bb))
}

func rgb(hex string) (int, int, int) {
	if !hexRe.MatchString(hex) {
		return 0, 0, 0
	}
	v, _ := strconv.ParseInt(hex[1:], 16, 32)
	return int(v >> 16), int(v >> 8 & 0xff), int(v & 0xff)
}

// Xterm256 returns the nearest xterm-256 color index for a hex color,
// searching only the fixed 6x6x6 cube (16-231) and the grayscale ramp
// (232-255). Indices 0-15 are deliberately excluded: the terminal remaps
// those to the active theme's ANSI slots, so "black" is #929292 under
// jellybeans — exactly the trap that makes unthemed VisiData unreadable.
func Xterm256(hex string) int {
	r, g, b := rgb(hex)
	best, bestDist := 16, 1<<30
	try := func(idx, cr, cg, cb int) {
		d := (r-cr)*(r-cr) + (g-cg)*(g-cg) + (b-cb)*(b-cb)
		if d < bestDist {
			best, bestDist = idx, d
		}
	}
	levels := []int{0, 95, 135, 175, 215, 255}
	for i, cr := range levels {
		for j, cg := range levels {
			for k, cb := range levels {
				try(16+36*i+6*j+k, cr, cg, cb)
			}
		}
	}
	for i := 0; i < 24; i++ {
		v := 8 + 10*i
		try(232+i, v, v, v)
	}
	return best
}

// ansiSlots is the palette's 16 terminal color slots, in the order every
// terminal numbers them: 0-7 normal black..white, 8-15 the brights. Wezterm
// writes exactly this list, so an index below 16 paints the hex found here.
func (p *Palette) ansiSlots() [16]string {
	return [16]string{
		p.Normal.Black, p.Normal.Red, p.Normal.Green, p.Normal.Yellow,
		p.Normal.Blue, p.Normal.Magenta, p.Normal.Cyan, p.Normal.White,
		p.Bright.Black, p.Bright.Red, p.Bright.Green, p.Bright.Yellow,
		p.Bright.Blue, p.Bright.Magenta, p.Bright.Cyan, p.Bright.White,
	}
}

// slot is the ANSI index whose palette hex is exactly this color, or -1.
// Exactness is the whole safety argument: a slot is only ever used where the
// terminal will remap it back to the very color we asked for, so the
// substitution is lossless rather than approximate.
func (p *Palette) slot(hex string) int {
	for i, s := range p.ansiSlots() {
		if strings.EqualFold(s, hex) {
			return i
		}
	}
	return -1
}

// Paint is the color index VisiData should be given for a hex. Slots 0-15
// carry the palette color exactly; everything else falls back to the fixed
// cube, where the nearest cell can be perceptually far away (up to deltaE 24
// across these palettes, and seven of nine accents collapsed onto one of two
// golds before this).
func (p *Palette) Paint(hex string) int {
	if i := p.slot(hex); i >= 0 {
		return i
	}
	return Xterm256(hex)
}

// PaintedHex is the color a terminal carrying this palette shows for an
// index: the palette's own hex for a slot, the fixed cube cell otherwise.
// Every contrast and distance check below measures against this, so the
// floors are checked on what actually reaches the screen.
func (p *Palette) PaintedHex(i int) string {
	if i >= 0 && i < 16 {
		return p.ansiSlots()[i]
	}
	return idxHex(i)
}

// contrast is the WCAG ratio between two indices as painted.
func (p *Palette) contrast(a, b int) float64 {
	return contrastHex(p.PaintedHex(a), p.PaintedHex(b))
}

// hue reports whether an index paints a colored pixel rather than a neutral.
func (p *Palette) hue(i int) bool {
	r, g, b := rgb(p.PaintedHex(i))
	return r != g || g != b
}
