// Package theme renders one shared palette into every themable tool's
// native config format and switches the active theme on a machine.
package theme

import (
	"fmt"
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
