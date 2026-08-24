// dotfiles: sync & install tool for edwinvillota/dotfiles.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edwinvillota/dotfiles/internal/apply"
	"github.com/edwinvillota/dotfiles/internal/check"
	"github.com/edwinvillota/dotfiles/internal/deps"
	"github.com/edwinvillota/dotfiles/internal/ledger"
	"github.com/edwinvillota/dotfiles/internal/manifest"
	"github.com/edwinvillota/dotfiles/internal/plan"
)

var version = "dev"

func usage() {
	fmt.Fprint(os.Stderr, `dotfiles `+version+` — sync live config <-> repo

Usage:
  dotfiles backup   [--dry-run] [--profile P] [--unit U ...]   live -> repo
  dotfiles install  [--dry-run] [--profile P] [--unit U ...] [--symlink] [--prune] repo -> live
  dotfiles deps     [--dry-run] [--core|--extra] [--only NAME ...]  install missing tools
  dotfiles uninstall [--dry-run] [--unit U ...] [--no-restore]  undo install, restore originals
  dotfiles check    [--quiet]                                   refuse secrets in the repo
  dotfiles hook                                                  install pre-commit hook
  dotfiles diff     [--unit U ...]                              show differences
  dotfiles version

  --yes / -y        skip confirmation prompts (deletions)

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
	yes     bool
	quiet   bool
	noRest  bool
	core    bool
	extra   bool
	only    multi
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
	fs.BoolVar(&c.yes, "yes", false, "")
	fs.BoolVar(&c.yes, "y", false, "")
	fs.BoolVar(&c.quiet, "quiet", false, "")
	fs.BoolVar(&c.noRest, "no-restore", false, "")
	fs.BoolVar(&c.core, "core", false, "")
	fs.BoolVar(&c.extra, "extra", false, "")
	fs.Var(&c.only, "only", "")
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
	case "deps":
		err = c.deps()
	case "uninstall":
		err = c.uninstall()
	case "check":
		err = c.check()
	case "hook":
		var h string
		if h, err = check.InstallHook(c.m.Root); err == nil {
			fmt.Println("installed", h)
		}
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
	if d := os.Getenv("DOTFILES"); d != "" {
		return filepath.Join(d, "dotfiles.toml"), nil
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
	if len(p.Changes()) == 0 {
		fmt.Println("\nnothing to do")
		return nil
	}
	if dir == plan.Backup {
		if fs, _ := check.Run(c.m, nil); len(fs) > 0 {
			// pre-existing problems must be fixed before adding more files
			fmt.Fprintln(os.Stderr, "\nrefusing to back up: repo already contains secret-looking content:")
			for _, f := range fs {
				fmt.Fprintln(os.Stderr, "  ", f)
			}
			return fmt.Errorf("run `dotfiles check` and fix the findings first")
		}
	}
	fmt.Println()
	res, err := apply.Run(c.m, p, apply.Options{Confirm: c.confirm, Log: os.Stdout})
	if err != nil {
		return err
	}
	fmt.Printf("\ndone: %d written, %d linked, %d deleted\n", res.Written, res.Linked, res.Deleted)
	for _, n := range res.Notices {
		fmt.Println("NOTE:", n)
	}
	if dir == plan.Install {
		fmt.Println("ledger:", filepath.Join(ledger.Dir(c.m.Home), "ledger.json"))
	}
	if dir == plan.Backup {
		if fs, _ := check.Run(c.m, nil); len(fs) > 0 {
			fmt.Fprintln(os.Stderr, "\nWARNING: the backup introduced secret-looking content — do NOT commit until fixed:")
			for _, f := range fs {
				fmt.Fprintln(os.Stderr, "  ", f)
			}
			return fmt.Errorf("secret check failed")
		}
	}
	return nil
}

func (c *common) confirm(msg string) bool {
	if c.yes {
		return true
	}
	fmt.Printf("%s. Continue? [y/N] ", msg)
	r := bufio.NewReader(os.Stdin)
	l, _ := r.ReadString('\n')
	l = strings.ToLower(strings.TrimSpace(l))
	return l == "y" || l == "yes"
}

func (c *common) uninstall() error {
	var units map[string]bool
	if len(c.units) > 0 {
		units = map[string]bool{}
		for _, u := range c.units {
			units[u] = true
		}
	}
	res, err := apply.Uninstall(c.m, units, !c.noRest, c.dryRun, os.Stdout)
	if err != nil {
		return err
	}
	if c.dryRun {
		fmt.Println("\n(dry run — nothing changed)")
		return nil
	}
	fmt.Printf("\ndone: %d removed, %d restored\n", res.Deleted, res.Written)
	for _, n := range res.Notices {
		fmt.Println("NOTE:", n)
	}
	return nil
}

func (c *common) check() error {
	fs, err := check.Run(c.m, nil)
	if err != nil {
		return err
	}
	if len(fs) == 0 {
		if !c.quiet {
			fmt.Println("ok: no secrets found in committed/unignored files")
		}
		return nil
	}
	fmt.Fprintln(os.Stderr, "secret check FAILED:")
	for _, f := range fs {
		fmt.Fprintln(os.Stderr, "  ", f)
	}
	fmt.Fprintln(os.Stderr, "\nmark a line `# public` if the value is safe, or add the file to `secret` in dotfiles.toml")
	os.Exit(1)
	return nil
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

func (c *common) deps() error {
	p := deps.Detect()
	names := c.only
	if len(names) == 0 {
		switch {
		case c.core:
			names = c.m.Deps.Core
		case c.extra:
			names = c.m.Deps.Extra
		default:
			names = append(append([]string{}, c.m.Deps.Core...), c.m.Deps.Extra...)
		}
	}
	brew := p.Brew
	if brew == "" {
		brew = "absent (" + p.BrewDir + ")"
	}
	fmt.Printf("platform: %s/%s %s  brew: %s  native: %s\n\n", p.OS, p.Arch, p.Distro, brew, orNone(p.Native()))
	items := deps.Resolve(c.m, p, names)
	var todo int
	for _, it := range items {
		mark := map[deps.Status]string{deps.Present: "✓", deps.Outdated: "↑", deps.Missing: "→", deps.Unsupported: "⊘", deps.NeedsBrew: "→"}[it.Status]
		detail := it.Found
		switch it.Status {
		case deps.Missing, deps.Outdated:
			detail = it.Manager + " " + it.Pkg
			todo++
		case deps.NeedsBrew:
			detail = "brew " + it.Pkg + " (Homebrew will be installed first)"
			todo++
		case deps.Unsupported:
			detail = it.Note
		}
		if it.Status == deps.Outdated {
			detail += "  (" + it.Note + ")"
		}
		fmt.Printf("  %s %-24s %s\n", mark, it.Name, detail)
	}
	fmt.Printf("\n%d to install\n", todo)
	if todo == 0 {
		return nil
	}
	fmt.Println()
	if err := deps.Install(c.m, p, items, c.dryRun, os.Stdout); err != nil {
		return err
	}
	if c.dryRun {
		fmt.Println("\n(dry run — nothing installed)")
	}
	return nil
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
