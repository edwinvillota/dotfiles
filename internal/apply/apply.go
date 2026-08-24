// Package apply executes a plan.
package apply

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/edwinvillota/dotfiles/internal/fsx"
	"github.com/edwinvillota/dotfiles/internal/ledger"
	"github.com/edwinvillota/dotfiles/internal/manifest"
	"github.com/edwinvillota/dotfiles/internal/plan"
	"github.com/edwinvillota/dotfiles/internal/redact"
)

type Options struct {
	// Confirm is asked before destructive steps (deletes). nil = refuse.
	Confirm func(msg string) bool
	Log     io.Writer
	Now     time.Time
}

// Result summarises what happened.
type Result struct {
	Written, Deleted, Linked, Skipped int
	Notices                           []string
}

// Run applies p. Install actions are recorded in the ledger and any file
// overwritten or deleted is preserved under the state dir first.
func Run(m *manifest.Manifest, p *plan.Plan, o Options) (*Result, error) {
	if o.Log == nil {
		o.Log = io.Discard
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	res := &Result{}
	changes := p.Changes()
	if len(changes) == 0 {
		return res, nil
	}

	var dels []plan.Action
	for _, a := range changes {
		if a.Op == plan.OpDelete {
			dels = append(dels, a)
		}
	}
	if len(dels) > 0 {
		msg := fmt.Sprintf("%d file(s) will be deleted", len(dels))
		if p.Direction == plan.Install {
			msg += " from your live config (originals preserved under " + ledger.Dir(m.Home) + ")"
		} else {
			msg += " from the repo (recoverable with git)"
		}
		if o.Confirm == nil || !o.Confirm(msg) {
			return nil, fmt.Errorf("aborted: deletions not confirmed")
		}
	}

	var led *ledger.Ledger
	if p.Direction == plan.Install {
		var err error
		if led, err = ledger.Load(m.Home); err != nil {
			return nil, err
		}
	}

	touched := map[string]bool{}
	for _, a := range changes {
		touched[a.Unit] = true
		if err := applyOne(m, p.Direction, a, led, o, res); err != nil {
			if led != nil {
				led.Save()
			}
			return res, fmt.Errorf("%s: %w", a.To, err)
		}
	}
	if p.Direction == plan.Install {
		if err := postInstall(m, touched, led, o.Log); err != nil {
			res.Notices = append(res.Notices, "post-install: "+err.Error())
		}
	}
	if led != nil {
		if err := led.Save(); err != nil {
			return res, err
		}
	}
	return res, nil
}

func applyOne(m *manifest.Manifest, dir plan.Direction, a plan.Action, led *ledger.Ledger, o Options, res *Result) error {
	preserve := func() (string, error) {
		if dir != plan.Install {
			return "", nil
		}
		// Already managed by us: the true original (if any) is in the ledger.
		if led != nil {
			if e := led.Find(a.To); e != nil {
				return e.Backup, nil
			}
		}
		if _, err := os.Lstat(a.To); err != nil {
			return "", nil
		}
		bp := ledger.BackupPath(m.Home, o.Now, a.To)
		if err := fsx.CopyFile(a.To, bp); err != nil {
			return "", fmt.Errorf("preserve original: %w", err)
		}
		return bp, nil
	}
	record := func(kind ledger.Kind, backup string) {
		if led != nil {
			led.Add(ledger.Entry{Path: a.To, Kind: kind, Backup: backup, Unit: a.Unit, Time: o.Now})
		}
	}

	switch a.Op {
	case plan.OpCreate, plan.OpUpdate:
		bp, err := preserve()
		if err != nil {
			return err
		}
		if a.Redact {
			if dir == plan.Backup {
				raw, err := os.ReadFile(a.From)
				if err != nil {
					return err
				}
				if err := writeAtomic(a.To, redact.Template(raw), 0o644); err != nil {
					return err
				}
				fmt.Fprintf(o.Log, "  template %s\n", a.To)
			} else {
				raw, err := os.ReadFile(a.From)
				if err != nil {
					return err
				}
				if err := writeAtomic(a.To, raw, 0o600); err != nil {
					return err
				}
				res.Notices = append(res.Notices, "created "+a.To+" from template — fill in your secrets (chmod 600 already set)")
				fmt.Fprintf(o.Log, "  create   %s (from template)\n", a.To)
			}
		} else {
			if err := fsx.CopyFile(a.From, a.To); err != nil {
				return err
			}
			fmt.Fprintf(o.Log, "  %-8s %s\n", map[plan.Op]string{plan.OpCreate: "create", plan.OpUpdate: "update"}[a.Op], a.To)
		}
		kind := ledger.Created
		if bp != "" {
			kind = ledger.Replaced
		}
		record(kind, bp)
		res.Written++
	case plan.OpLink:
		bp, err := preserve()
		if err != nil {
			return err
		}
		if err := fsx.Symlink(a.From, a.To); err != nil {
			return err
		}
		fmt.Fprintf(o.Log, "  link     %s -> %s\n", a.To, a.From)
		record(ledger.Linked, bp)
		res.Linked++
	case plan.OpDelete:
		bp, err := preserve()
		if err != nil {
			return err
		}
		if err := os.Remove(a.To); err != nil && !os.IsNotExist(err) {
			return err
		}
		root := m.DestPath(m.Units[a.Unit])
		if dir == plan.Backup {
			root = m.SrcPath(m.Units[a.Unit])
		}
		fsx.RemoveEmptyParents(a.To, root)
		fmt.Fprintf(o.Log, "  delete   %s\n", a.To)
		if bp != "" {
			record(ledger.Deleted, bp)
		}
		res.Deleted++
	}
	return nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// Uninstall reverses everything in the ledger (optionally only for units).
// Files we created are removed; files we replaced or deleted are restored
// from their preserved originals when restore is true.
func Uninstall(m *manifest.Manifest, units map[string]bool, restore bool, dryRun bool, log io.Writer) (*Result, error) {
	led, err := ledger.Load(m.Home)
	if err != nil {
		return nil, err
	}
	res := &Result{}
	var keep []ledger.Entry
	// newest first so nested state unwinds in order
	for i := len(led.Entries) - 1; i >= 0; i-- {
		e := led.Entries[i]
		if units != nil && !units[e.Unit] {
			keep = append(keep, e)
			continue
		}
		switch e.Kind {
		case ledger.Created, ledger.Linked, ledger.Replaced:
			fmt.Fprintf(log, "  remove   %s\n", e.Path)
			if !dryRun {
				if err := os.Remove(e.Path); err != nil && !os.IsNotExist(err) {
					return res, err
				}
				res.Deleted++
			}
		}
		if e.Backup != "" && restore {
			fmt.Fprintf(log, "  restore  %s\n", e.Path)
			if !dryRun {
				if err := fsx.CopyFile(e.Backup, e.Path); err != nil {
					return res, fmt.Errorf("restore %s: %w", e.Path, err)
				}
				res.Written++
			}
		} else if e.Backup != "" {
			res.Notices = append(res.Notices, "original kept at "+e.Backup)
		}
		if !dryRun {
			u := m.Units[e.Unit]
			if u != nil {
				fsx.RemoveEmptyParents(e.Path, m.DestPath(u))
			}
		}
	}
	if !dryRun {
		// remove unit roots we emptied (os.Remove refuses non-empty dirs)
		for _, e := range led.Entries {
			if u := m.Units[e.Unit]; u != nil && (units == nil || units[e.Unit]) {
				os.Remove(m.DestPath(u))
			}
		}
		// keep is reversed; order doesn't matter for correctness
		led.Entries = keep
		if err := led.Save(); err != nil {
			return res, err
		}
	}
	return res, nil
}
