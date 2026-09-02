// Package theme renders one shared palette into every themable tool's
// native config format and switches the active theme on a machine.
package theme

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"

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
	Normal ANSI  `toml:"normal"`
	Bright ANSI  `toml:"bright"`
	Roles  Roles `toml:"roles"`
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
	def(&p.Roles.Dim, p.Bright.Black)
	def(&p.Roles.Panel, Mix(p.Primary.Background, p.Primary.Foreground, 0.06))
	def(&p.Roles.Line, Mix(p.Primary.Background, p.Primary.Foreground, 0.12))
	def(&p.Roles.Sel, p.Selection.Background)
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
		"roles.sel": p.Roles.Sel,
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

func lumIdx(i int) float64 {
	r, g, b := idxRGB(i)
	f := func(c int) float64 {
		s := float64(c) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*f(r) + 0.7152*f(g) + 0.0722*f(b)
}

// contrastIdx is the WCAG contrast ratio between two xterm-256 indices.
// Colors are compared after quantization because that is what the terminal
// actually paints -- two distinct hex values can land on the same index.
func contrastIdx(a, b int) float64 {
	l1, l2 := lumIdx(a), lumIdx(b)
	if l2 > l1 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
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
