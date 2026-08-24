package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/edwinvillota/dotfiles/internal/apply"
	"github.com/edwinvillota/dotfiles/internal/deps"
	"github.com/edwinvillota/dotfiles/internal/plan"
	"github.com/edwinvillota/dotfiles/internal/state"
)

// setup is the guided fresh-machine walkthrough.
func (c *common) setup() error {
	in := bufio.NewReader(os.Stdin)
	ask := func(q, def string) string {
		fmt.Printf("%s [%s]: ", q, def)
		l, _ := in.ReadString('\n')
		l = strings.TrimSpace(l)
		if l == "" {
			return def
		}
		return l
	}
	yes := func(q string, def bool) bool {
		d := "y/N"
		if def {
			d = "Y/n"
		}
		a := strings.ToLower(ask(q, d))
		if a == "y/n" || a == "" {
			return def
		}
		return a == "y" || a == "yes"
	}
	step := func(n int, title, why string) {
		fmt.Printf("\n── step %d · %s ─────────────────────────────\n%s\n\n", n, title, why)
	}

	fmt.Println("dotfiles setup — guided install. Nothing happens without your ok;")
	fmt.Println("every change is previewed first and reversible with `dotfiles uninstall`.")

	// 1. profile
	step(1, "profile", "A profile excludes machine-specific files (work vs personal)\nand maps to a git branch so each machine can keep its quirks.")
	st, err := state.Load(c.m.Home)
	if err != nil {
		return err
	}
	names := sortedProfiles(c.m)
	for _, n := range names {
		p := c.m.Profiles[n]
		fmt.Printf("  %-10s branch=%-8s excludes %v\n", n, p.Branch, p.Exclude)
	}
	def := st.Profile
	if def == "" && len(names) > 0 {
		def = names[0]
	}
	for {
		p := ask("which profile is this machine?", def)
		if _, ok := c.m.Profiles[p]; ok {
			st.Profile, c.profile = p, p
			break
		}
		fmt.Println("  unknown profile, options:", strings.Join(names, ", "))
	}
	if err := st.Save(); err != nil {
		return err
	}

	// 2. deps
	step(2, "tools", "The configs are useless without the programs that read them\n(nvim, zellij, fzf, ...). Present tools are skipped; the package\nmanager is picked per platform (brew / apt / pacman).")
	pf := deps.Detect()
	all := append(append([]string{}, c.m.Deps.Core...), c.m.Deps.Extra...)
	items := deps.Resolve(c.m, pf, all)
	missing := 0
	for _, it := range items {
		if it.Status == deps.Missing || it.Status == deps.Outdated || it.Status == deps.NeedsBrew {
			missing++
		}
	}
	if missing == 0 {
		fmt.Println("  ✓ everything already installed")
	} else {
		fmt.Printf("  %d tool(s) missing — preview:\n\n", missing)
		if err := deps.Install(c.m, pf, items, true, os.Stdout); err != nil {
			return err
		}
		if yes(fmt.Sprintf("\ninstall these %d tool(s) now?", missing), true) {
			if err := deps.Install(c.m, pf, items, false, os.Stdout); err != nil {
				fmt.Println("  warning:", err, "— you can retry later with `dotfiles deps`")
			}
		} else {
			fmt.Println("  skipped — run `dotfiles deps` whenever you like")
		}
	}

	// 3. configs
	step(3, "configs", "Now the actual dotfiles. The plan below is everything that would\nbe written. Files that already exist are preserved under\n~/.local/state/dotfiles/backups/ and restorable with `dotfiles uninstall`.\n~/.zshrc and secret files are never overwritten.")
	o, err := c.options(plan.Install)
	if err != nil {
		return err
	}
	p, err := plan.Build(c.m, o)
	if err != nil {
		return err
	}
	printPlan(p)
	if len(p.Changes()) == 0 {
		fmt.Println("\n  ✓ configs already in sync")
	} else if yes("\napply this plan?", true) {
		res, err := apply.Run(c.m, p, apply.Options{Confirm: func(m string) bool { return yes(m+". continue?", false) }, Log: os.Stdout})
		if err != nil {
			return err
		}
		fmt.Printf("\ndone: %d written, %d deleted\n", res.Written, res.Deleted)
		for _, n := range res.Notices {
			fmt.Println("NOTE:", n)
		}
	} else {
		fmt.Println("  skipped — run `dotfiles install --dry-run` when ready")
	}

	// 4. finish
	step(4, "done", "What you may still want:")
	fmt.Println("  • fill in any *.zsh created from templates (secrets are blanked)")
	fmt.Println("  • start a new shell:  exec zsh")
	fmt.Println("  • protect the repo:   dotfiles hook   (pre-commit secret scan)")
	fmt.Println("  • explore/tune:       dotfiles        (TUI, ? for help)")
	fmt.Println("  • change your mind:   dotfiles uninstall")
	return nil
}
