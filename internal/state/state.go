// Package state persists per-machine choices (never committed).
package state

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

type State struct {
	Profile  string   `toml:"profile"`
	Disabled []string `toml:"disabled"` // selection keys ("unit" or "unit/rel") turned off
	Symlink  bool     `toml:"symlink"`
	path     string
}

func Path(home string) string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" && home == os.Getenv("HOME") {
		return filepath.Join(x, "dotfiles", "state.toml")
	}
	return filepath.Join(home, ".config", "dotfiles", "state.toml")
}

func Load(home string) (*State, error) {
	s := &State{path: Path(home)}
	if _, err := toml.DecodeFile(s.path, s); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *State) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	sort.Strings(s.Disabled)
	f, err := os.Create(s.path + ".tmp")
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(s); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(s.path+".tmp", s.path)
}

func (s *State) IsDisabled(key string) bool {
	for _, d := range s.Disabled {
		if d == key {
			return true
		}
	}
	return false
}

func (s *State) Set(key string, enabled bool) {
	out := s.Disabled[:0]
	for _, d := range s.Disabled {
		if d != key {
			out = append(out, d)
		}
	}
	s.Disabled = out
	if !enabled {
		s.Disabled = append(s.Disabled, key)
	}
}

// Selected builds the planner's selection set from all known keys.
// A granule is selected only if both it and its unit are enabled.
func (s *State) Selected(keys []string) map[string]bool {
	sel := map[string]bool{}
	for _, k := range keys {
		if s.IsDisabled(k) {
			continue
		}
		if i := indexSlash(k); i > 0 && s.IsDisabled(k[:i]) {
			continue
		}
		sel[k] = true
	}
	return sel
}

func indexSlash(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}
