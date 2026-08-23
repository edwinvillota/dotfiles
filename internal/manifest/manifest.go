// Package manifest loads and validates dotfiles.toml.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type Mode string

const (
	ModeSync       Mode = "sync"        // both directions (default)
	ModeBackupOnly Mode = "backup-only" // install only if dest absent
)

type Unit struct {
	Name     string   `toml:"-"`
	Src      string   `toml:"src"`
	Dest     string   `toml:"dest"`
	Mode     Mode     `toml:"mode"`
	Granular []string `toml:"granular"`
	Ignore   []string `toml:"ignore"`
	Secret   []string `toml:"secret"`
}

type Profile struct {
	Branch  string   `toml:"branch"`
	Exclude []string `toml:"exclude"` // "unit/relative/path" or "unit"
}

type Secrets struct {
	Patterns []string `toml:"patterns"`
}

type Global struct {
	Ignore []string `toml:"ignore"`
}

type Manifest struct {
	Global   Global              `toml:"global"`
	Units    map[string]*Unit    `toml:"unit"`
	Profiles map[string]*Profile `toml:"profile"`
	Secrets  Secrets             `toml:"secrets"`

	Root    string `toml:"-"` // directory containing the manifest
	Home    string `toml:"-"`
	secrets []*regexp.Regexp
}

func Load(path, home string) (*Manifest, error) {
	var m Manifest
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	m.Root = filepath.Dir(abs)
	m.Home = home
	for name, u := range m.Units {
		u.Name = name
		if u.Mode == "" {
			u.Mode = ModeSync
		}
		if u.Mode != ModeSync && u.Mode != ModeBackupOnly {
			return nil, fmt.Errorf("unit %s: unknown mode %q", name, u.Mode)
		}
		if u.Src == "" || u.Dest == "" {
			return nil, fmt.Errorf("unit %s: src and dest are required", name)
		}
	}
	for _, p := range m.Secrets.Patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("secrets pattern %q: %w", p, err)
		}
		m.secrets = append(m.secrets, re)
	}
	return &m, nil
}

func (m *Manifest) SecretPatterns() []*regexp.Regexp { return m.secrets }

// UnitNames returns unit names sorted for stable output.
func (m *Manifest) UnitNames() []string {
	names := make([]string, 0, len(m.Units))
	for n := range m.Units {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// SrcPath is the absolute repo-side path of the unit.
func (m *Manifest) SrcPath(u *Unit) string { return filepath.Join(m.Root, u.Src) }

// DestPath is the absolute live path of the unit, with ~ expanded.
func (m *Manifest) DestPath(u *Unit) string { return expand(u.Dest, m.Home) }

func expand(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// Match reports whether rel (a unit-relative path, forward slashes) matches
// any of the patterns. A pattern ending in "/" matches a directory prefix.
// Other patterns match against the basename and the full relative path.
func Match(patterns []string, rel string) bool {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	for _, p := range patterns {
		if strings.HasSuffix(p, "/") {
			d := strings.TrimSuffix(p, "/")
			// match the pattern against rel and each of its ancestor dirs
			for a := rel; a != "." && a != ""; a = pathDir(a) {
				if ok, _ := filepath.Match(d, a); ok {
					return true
				}
				if !strings.Contains(d, "/") {
					if ok, _ := filepath.Match(d, filepath.Base(a)); ok {
						return true
					}
				}
			}
			continue
		}
		if ok, _ := filepath.Match(p, rel); ok {
			return true
		}
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
	}
	return false
}

// IsIgnored / IsSecret classify a unit-relative path.
func (u *Unit) IsIgnored(rel string) bool { return Match(u.Ignore, rel) }

// IsIgnored applies global + unit ignores.
func (m *Manifest) IsIgnored(u *Unit, rel string) bool {
	return Match(m.Global.Ignore, rel) || u.IsIgnored(rel)
}
func (u *Unit) IsSecret(rel string) bool { return Match(u.Secret, rel) }

// GranuleOf returns the granular key ("unit/rel") if rel matches a granular
// glob, else "" (meaning the file belongs to the unit as a whole).
func (u *Unit) GranuleOf(rel string) string {
	if Match(u.Granular, rel) {
		return u.Name + "/" + filepath.ToSlash(rel)
	}
	return ""
}

// Excluded reports whether the profile excludes the unit or the given
// unit-relative path.
func (p *Profile) Excluded(unit, rel string) bool {
	if p == nil {
		return false
	}
	key := unit + "/" + filepath.ToSlash(rel)
	for _, e := range p.Exclude {
		if e == unit || e == key {
			return true
		}
	}
	return false
}

// FileExists is a tiny helper used by callers to check src presence.
func FileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func pathDir(p string) string {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return ""
	}
	return p[:i]
}
