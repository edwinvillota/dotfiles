// dotfiles: sync & install tool for edwinvillota/dotfiles.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edwinvillota/dotfiles/internal/manifest"
	"github.com/edwinvillota/dotfiles/internal/plan"
)

var version = "dev"

func usage() {
	fmt.Fprint(os.Stderr, `dotfiles `+version+` — sync live config <-> repo

Usage:
  dotfiles backup   [--dry-run] [--profile P] [--unit U ...]   live -> repo
  dotfiles install  [--dry-run] [--profile P] [--unit U ...] [--symlink] [--prune] repo -> live
  dotfiles diff     [--unit U ...]                              show differences
  dotfiles version

Global flags:
  --manifest PATH   dotfiles.toml (default: auto-detect from cwd or $DOTFILES)
  --home PATH       override $HOME (used by tests)
`)
}

type common struct {
	m       *manifest.Manifest
	dryRun  bool
	profile string
	units   multi
	symlink bool
	prune   bool
}

type multi []string

func (m *multi) String() string     { return fmt.Sprint([]string(*m)) }
func (m *multi) Set(s string) error { *m = append(*m, s); return nil }

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	if cmd == "version" {
		fmt.Println(version)
		return
	}
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.Usage = usage
	var c common
	var manifestPath, home string
	fs.StringVar(&manifestPath, "manifest", "", "")
	fs.StringVar(&home, "home", "", "")
	fs.BoolVar(&c.dryRun, "dry-run", false, "")
	fs.BoolVar(&c.dryRun, "n", false, "")
	fs.StringVar(&c.profile, "profile", "", "")
	fs.Var(&c.units, "unit", "")
	fs.BoolVar(&c.symlink, "symlink", false, "")
	fs.BoolVar(&c.prune, "prune", false, "")
	fs.Parse(args)

	if home == "" {
		home, _ = os.UserHomeDir()
	}
	mp, err := findManifest(manifestPath)
	if err != nil {
		die(err)
	}
	c.m, err = manifest.Load(mp, home)
	if err != nil {
		die(err)
	}

	switch cmd {
	case "backup":
		err = c.run(plan.Backup)
	case "install":
		err = c.run(plan.Install)
	case "diff":
		err = c.diff()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		die(err)
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func findManifest(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if d := os.Getenv("DOTFILES"); d != "" {
		return filepath.Join(d, "dotfiles.toml"), nil
	}
	dir, _ := os.Getwd()
	for {
		p := filepath.Join(dir, "dotfiles.toml")
		if manifest.FileExists(p) {
			return p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("dotfiles.toml not found (set $DOTFILES or --manifest)")
}

func (c *common) options(dir plan.Direction) (plan.Options, error) {
	o := plan.Options{Direction: dir, Symlink: c.symlink, Prune: c.prune}
	if c.profile != "" {
		p, ok := c.m.Profiles[c.profile]
		if !ok {
			return o, fmt.Errorf("unknown profile %q", c.profile)
		}
		o.Profile = p
	}
	if len(c.units) > 0 {
		o.Units = c.units
	}
	return o, nil
}

func (c *common) run(dir plan.Direction) error {
	o, err := c.options(dir)
	if err != nil {
		return err
	}
	p, err := plan.Build(c.m, o)
	if err != nil {
		return err
	}
	printPlan(p)
	if c.dryRun {
		fmt.Println("\n(dry run — nothing written)")
		return nil
	}
	return fmt.Errorf("applying plans is not implemented yet; use --dry-run")
}

func printPlan(p *plan.Plan) {
	unit := ""
	for _, a := range p.Actions {
		if a.Op == plan.OpNone {
			continue
		}
		if a.Unit != unit {
			unit = a.Unit
			fmt.Printf("\n[%s]\n", unit)
		}
		rel := a.Rel
		if rel == "" {
			rel = filepath.Base(a.To)
		}
		line := fmt.Sprintf("  %-4s %s", a.Op, rel)
		if a.Op == plan.OpSkip {
			line += "  (" + a.Reason + ")"
		} else if a.Backup {
			line += "  [backup]"
		}
		if a.Redact && a.Op != plan.OpSkip {
			line += "  [secret → template]"
		}
		fmt.Println(line)
	}
	cr, up, de, li, sk := p.Counts()
	fmt.Printf("\n%s: %d create, %d update, %d delete, %d link, %d skipped\n", p.Direction, cr, up, de, li, sk)
}

func (c *common) diff() error {
	return fmt.Errorf("diff not implemented yet")
}
