// Package ledger records what install wrote so uninstall can reverse it.
package ledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Kind string

const (
	Created  Kind = "created"  // file did not exist before
	Replaced Kind = "replaced" // existing file backed up then overwritten
	Linked   Kind = "linked"   // symlink placed (Backup set if something was there)
	Deleted  Kind = "deleted"  // live file removed (--prune); Backup holds it
)

type Entry struct {
	Path   string    `json:"path"` // absolute live path
	Kind   Kind      `json:"kind"`
	Backup string    `json:"backup,omitempty"` // absolute path of preserved original
	Unit   string    `json:"unit"`
	Time   time.Time `json:"time"`
}

type Ledger struct {
	Entries []Entry `json:"entries"`
	path    string
}

// Dir returns the state directory for a given home.
func Dir(home string) string {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" && home == os.Getenv("HOME") {
		return filepath.Join(x, "dotfiles")
	}
	return filepath.Join(home, ".local", "state", "dotfiles")
}

func Load(home string) (*Ledger, error) {
	l := &Ledger{path: filepath.Join(Dir(home), "ledger.json")}
	b, err := os.ReadFile(l.path)
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	return l, json.Unmarshal(b, l)
}

func (l *Ledger) Save() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(l, "", "  ")
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}

// BackupPath returns where the original of `live` should be preserved for a
// run started at ts.
func BackupPath(home string, ts time.Time, live string) string {
	rel := strings.TrimPrefix(live, home)
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	return filepath.Join(Dir(home), "backups", ts.Format("2006-01-02T15-04-05"), rel)
}

// Add records an entry, replacing any older entry for the same path so the
// ledger stays idempotent across repeated installs.
func (l *Ledger) Add(e Entry) {
	for i, old := range l.Entries {
		if old.Path == e.Path {
			// keep the ORIGINAL backup: that's what the user had before we
			// ever touched the file
			if old.Backup != "" && e.Backup == "" {
				e.Backup = old.Backup
			}
			if old.Kind == Created {
				e.Kind = Created // we created it; a later overwrite is still ours
				e.Backup = ""
			} else if old.Kind == Replaced && e.Kind != Linked {
				e.Kind = Replaced
			}
			l.Entries[i] = e
			return
		}
	}
	l.Entries = append(l.Entries, e)
}

func (l *Ledger) Remove(path string) {
	out := l.Entries[:0]
	for _, e := range l.Entries {
		if e.Path != path {
			out = append(out, e)
		}
	}
	l.Entries = out
}

func (l *Ledger) Find(path string) *Entry {
	for i := range l.Entries {
		if l.Entries[i].Path == path {
			return &l.Entries[i]
		}
	}
	return nil
}
