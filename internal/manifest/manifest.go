// Package manifest loads and validates dotfiles.toml.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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
	Dest     Dest     `toml:"dest"`
	Mode     Mode     `toml:"mode"`
	Granular []string `toml:"granular"`
	Ignore   []string `toml:"ignore"`
	Secret   []string `toml:"secret"`
	// Only, if set, is an allowlist: paths not matching are ignored.
	Only []string `toml:"only"`
}

// Dest is either a single path or a per-OS table {darwin = "...", linux = "..."}.
type Dest struct {
	All   string
	PerOS map[string]string
}

func (d *Dest) UnmarshalTOML(v any) error {
	switch x := v.(type) {
	case string:
		d.All = x
	case map[string]any:
		d.PerOS = map[string]string{}
		for k, val := range x {
			s, ok := val.(string)
			if !ok {
				return fmt.Errorf("dest.%s must be a string", k)
			}
			d.PerOS[k] = s
		}
	default:
		return fmt.Errorf("dest must be a string or table")
	}
	return nil
}

// For returns the path for the given GOOS ("" if unsupported there).
func (d Dest) For(goos string) string {
	if d.All != "" {
		return d.All
	}
	if p, ok := d.PerOS[goos]; ok {
		return p
	}
	return d.PerOS["default"]
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

// PkgRef is a package-manager entry: a name string, or a table
// { cask = "x" } / { skip = true, note = "..." }.
type PkgRef struct {
	Name string
	Cask bool
	Skip bool
	Note string
	set  bool
}

func (r *PkgRef) UnmarshalTOML(v any) error {
	r.set = true
	switch x := v.(type) {
	case string:
		r.Name = x
	case map[string]any:
		if c, ok := x["cask"].(string); ok {
			r.Name, r.Cask = c, true
		}
		if s, ok := x["skip"].(bool); ok {
			r.Skip = s
		}
		if n, ok := x["note"].(string); ok {
			r.Note = n
		}
	default:
		return fmt.Errorf("package ref must be a string or table")
	}
	return nil
}

// Resolve returns (name, cask, skip, note) defaulting name to def.
func (r PkgRef) Resolve(def string) (string, bool, bool, string) {
	if r.Name == "" {
		return def, r.Cask, r.Skip, r.Note
	}
	return r.Name, r.Cask, r.Skip, r.Note
}

// PkgRef2 is Resolve without the cask flag (apt/pacman).
func (r PkgRef) Resolve2(def string) (string, bool, string) {
	n, _, s, note := r.Resolve(def)
	return n, s, note
}

type PkgSpec struct {
	Bin           Dest     `toml:"bin"`
	Brew          PkgRef   `toml:"brew"`
	Apt           PkgRef   `toml:"apt"`
	Pacman        PkgRef   `toml:"pacman"`
	Min           string   `toml:"min"`
	Kind          string   `toml:"kind"`
	URL           string   `toml:"url"`
	Dest          string   `toml:"dest"`
	Name          string   `toml:"name"`
	Needs         []string `toml:"needs"`
	OS            []string `toml:"os"`
	Note          string   `toml:"note"`
	AptPrereqs    []string `toml:"apt_prereqs"`
	PacmanPrereqs []string `toml:"pacman_prereqs"`
}

type Deps struct {
	Core  []string            `toml:"core"`
	Extra []string            `toml:"extra"`
	Pkg   map[string]*PkgSpec `toml:"pkg"`
}

type Manifest struct {
	Deps     Deps                `toml:"deps"`
	Global   Global              `toml:"global"`
	Units    map[string]*Unit    `toml:"unit"`
	Profiles map[string]*Profile `toml:"profile"`
	Secrets  Secrets             `toml:"secrets"`

	Root    string `toml:"-"` // directory containing the manifest
	Home    string `toml:"-"`
	GOOS    string `toml:"-"`
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
	m.GOOS = runtime.GOOS
	for name, u := range m.Units {
		u.Name = name
		if u.Mode == "" {
			u.Mode = ModeSync
		}
		if u.Mode != ModeSync && u.Mode != ModeBackupOnly {
			return nil, fmt.Errorf("unit %s: unknown mode %q", name, u.Mode)
		}
		if u.Src == "" || u.Dest.For(runtime.GOOS) == "" {
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
func (m *Manifest) DestPath(u *Unit) string { return expand(u.Dest.For(m.GOOS), m.Home) }

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
	if Match(m.Global.Ignore, rel) || u.IsIgnored(rel) {
		return true
	}
	// Allowlist: a file is kept only if it matches; directories are always
	// descended so nested matches can be found.
	if len(u.Only) > 0 && !Match(u.Only, rel) && !u.dirOnPathToOnly(rel) {
		return true
	}
	return false
}

// dirOnPathToOnly reports whether rel could be a directory prefix of an
// allowlisted pattern (so Snapshot keeps walking into it).
func (u *Unit) dirOnPathToOnly(rel string) bool {
	for _, p := range u.Only {
		if strings.HasPrefix(p, rel+"/") {
			return true
		}
	}
	return false
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
