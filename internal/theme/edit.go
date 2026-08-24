package theme

import (
	"fmt"
	"regexp"
	"strings"
)

// The switchers below flip exactly one theme-selecting line (or block) in a
// unit-managed config, leaving everything else byte-identical.

var (
	zellijThemeRe = regexp.MustCompile(`(?m)^theme "[^"]*"$`)
	yaziFlavorRe  = regexp.MustCompile(`(?m)^dark = "[^"]*"$`)
	btopThemeRe   = regexp.MustCompile(`(?m)^color_theme = "[^"]*"$`)
	ghDashBlockRe = regexp.MustCompile(`(?m)^theme:\n(?:[ \t][^\n]*\n|\n)*`)
)

func SetZellijTheme(content []byte, name string) ([]byte, error) {
	return replaceOne(zellijThemeRe, content, fmt.Sprintf("theme %q", name), "zellij config.kdl: no `theme \"...\"` line")
}

func SetYaziFlavor(content []byte, name string) ([]byte, error) {
	return replaceOne(yaziFlavorRe, content, fmt.Sprintf("dark = %q", name), "yazi theme.toml: no `dark = \"...\"` line under [flavor]")
}

func SetBtopTheme(content []byte, name string) ([]byte, error) {
	return replaceOne(btopThemeRe, content, fmt.Sprintf("color_theme = %q", name), "btop.conf: no `color_theme = \"...\"` line")
}

// SetGhDashTheme replaces the whole top-level `theme:` block.
func SetGhDashTheme(content []byte, p *Palette) ([]byte, error) {
	block := GhDashThemeBlock(p)
	if !ghDashBlockRe.Match(content) {
		// no block yet: append one
		s := string(content)
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		return []byte(s + "\n" + block), nil
	}
	return replaceOne(ghDashBlockRe, content, block, "")
}

func replaceOne(re *regexp.Regexp, content []byte, repl, missing string) ([]byte, error) {
	loc := re.FindIndex(content)
	if loc == nil {
		return nil, fmt.Errorf("%s", missing)
	}
	out := string(content[:loc[0]]) + repl + string(content[loc[1]:])
	return []byte(out), nil
}
