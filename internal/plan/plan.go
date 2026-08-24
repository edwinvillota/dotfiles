// Package plan computes what backup/install would do without doing it.
package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/edwinvillota/dotfiles/internal/fsx"
	"github.com/edwinvillota/dotfiles/internal/manifest"
)

type Direction int

const (
	Backup  Direction = iota // live -> repo
	Install                  // repo -> live
)

func (d Direction) String() string {
	if d == Backup {
		return "backup"
	}
	return "install"
}

type Op int

const (
	OpNone   Op = iota // identical, nothing to do
	OpCreate           // target missing
	OpUpdate           // target differs
	OpDelete           // source missing, target present
	OpLink             // symlink mode: create/replace link
	OpSkip             // explicitly skipped (secret, backup-only, excluded, unselected)
)

func (o Op) String() string {
	return [...]string{"=", "+", "~", "-", "@", "skip"}[o]
}

// Action is one planned file operation.
type Action struct {
	Unit    string
	Granule string // "unit/rel" if part of a granular sub-unit, else ""
	Rel     string
	Op      Op
	From    string // absolute source path (may be "" for delete)
	To      string // absolute target path
	Reason  string // for OpSkip
	Backup  bool   // target exists and will be preserved before overwrite
}

func (a Action) Key() string {
	if a.Granule != "" {
		return a.Granule
	}
	return a.Unit
}

// Options controls planning.
type Options struct {
	Direction Direction
	Profile   *manifest.Profile
	Symlink   bool
	// Prune allows Install to delete live files absent from the repo.
	Prune bool
	// Selected, if non-nil, is the set of selection keys ("unit" or "unit/rel")
	// enabled; anything else becomes OpSkip. nil = everything.
	Selected map[string]bool
	// Units restricts planning to these unit names (nil = all).
	Units []string
}

type Plan struct {
	Direction Direction
	Actions   []Action
}

func (p *Plan) Counts() (create, update, del, link, skip int) {
	for _, a := range p.Actions {
		switch a.Op {
		case OpCreate:
			create++
		case OpUpdate:
			update++
		case OpDelete:
			del++
		case OpLink:
			link++
		case OpSkip:
			skip++
		}
	}
	return
}

// Changes returns actions that actually modify something.
func (p *Plan) Changes() []Action {
	var out []Action
	for _, a := range p.Actions {
		if a.Op != OpNone && a.Op != OpSkip {
			out = append(out, a)
		}
	}
	return out
}

// Keys returns every selection key present in the plan (units and granules), sorted.
func (p *Plan) Keys() []string {
	set := map[string]bool{}
	for _, a := range p.Actions {
		set[a.Unit] = true
		if a.Granule != "" {
			set[a.Granule] = true
		}
	}
	return fsx.SortedKeys(set)
}

// Build computes the plan for the manifest under opts.
func Build(m *manifest.Manifest, opts Options) (*Plan, error) {
	names := opts.Units
	if names == nil {
		names = m.UnitNames()
	}
	p := &Plan{Direction: opts.Direction}
	for _, name := range names {
		u, ok := m.Units[name]
		if !ok {
			return nil, fmt.Errorf("unknown unit %q", name)
		}
		acts, err := planUnit(m, u, opts)
		if err != nil {
			return nil, fmt.Errorf("unit %s: %w", name, err)
		}
		p.Actions = append(p.Actions, acts...)
	}
	return p, nil
}

func planUnit(m *manifest.Manifest, u *manifest.Unit, opts Options) ([]Action, error) {
	repo, live := m.SrcPath(u), m.DestPath(u)
	skip := func(rel string, isDir bool) bool {
		if isDir && len(u.Only) > 0 {
			return manifest.Match(m.Global.Ignore, rel) || u.IsIgnored(rel)
		}
		return m.IsIgnored(u, rel)
	}
	repoSnap, err := fsx.Snapshot(repo, skip)
	if err != nil {
		return nil, err
	}
	liveSnap, err := fsx.Snapshot(live, skip)
	if err != nil {
		return nil, err
	}

	var src, dst map[string]*fsx.Entry
	var srcRoot, dstRoot string
	if opts.Direction == Backup {
		src, dst, srcRoot, dstRoot = liveSnap, repoSnap, live, repo
	} else {
		src, dst, srcRoot, dstRoot = repoSnap, liveSnap, repo, live
	}

	// In symlink-install mode a whole non-granular unit is one link; if the
	// live side is already that link, every file is "identical".
	liveIsLink := false
	if fi, err := os.Lstat(live); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if t, _ := os.Readlink(live); t == repo {
			liveIsLink = true
		}
	}

	all := map[string]bool{}
	for k := range src {
		all[k] = true
	}
	for k := range dst {
		all[k] = true
	}
	rels := fsx.SortedKeys(all)

	var out []Action
	for _, rel := range rels {
		a := Action{Unit: u.Name, Granule: u.GranuleOf(rel), Rel: rel,
			From: join(srcRoot, rel), To: join(dstRoot, rel)}
		s, d := src[rel], dst[rel]
		switch {
		case u.IsSecret(rel):
			a.Op, a.Reason = OpSkip, "secret"
		case opts.Profile.Excluded(u.Name, rel):
			a.Op, a.Reason = OpSkip, "excluded by profile"
		case opts.Selected != nil && !opts.Selected[a.Key()]:
			a.Op, a.Reason = OpSkip, "not selected"
		case opts.Direction == Install && u.Mode == manifest.ModeBackupOnly && d != nil:
			a.Op, a.Reason = OpSkip, "backup-only: live file exists"
		case opts.Direction == Install && opts.Symlink:
			if liveIsLink || (d != nil && d.Symlink == a.From) {
				a.Op = OpNone
			} else if s == nil && !opts.Prune {
				a.Op, a.Reason = OpSkip, "live-only (use --prune to delete)"
			} else if s == nil {
				a.Op = OpDelete
			} else {
				a.Op, a.Backup = OpLink, d != nil && d.Symlink == ""
			}
		case s == nil && opts.Direction == Install && !opts.Prune:
			a.Op, a.From, a.Reason = OpSkip, "", "live-only (use --prune to delete)"
		case s == nil:
			a.Op, a.From = OpDelete, ""
		case d == nil:
			a.Op = OpCreate
		case same(s, d):
			a.Op = OpNone
		default:
			a.Op, a.Backup = OpUpdate, true
		}
		out = append(out, a)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

func same(a, b *fsx.Entry) bool {
	if a.Symlink != "" || b.Symlink != "" {
		return a.Symlink == b.Symlink
	}
	if a.Size != b.Size {
		return false
	}
	return a.Hash() == b.Hash()
}

func join(root, rel string) string {
	if rel == "" {
		return root
	}
	return filepath.Join(root, filepath.FromSlash(rel))
}
