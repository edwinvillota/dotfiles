package main

import (
	"fmt"
	"os"

	"github.com/edwinvillota/dotfiles/internal/state"
	"github.com/edwinvillota/dotfiles/internal/theme"
	"github.com/edwinvillota/dotfiles/internal/tui"
)

// themeCmd: `dotfiles theme` lists themes, `dotfiles theme NAME` applies one.
// `dotfiles theme render` regenerates the committed per-theme assets
// (zellij themes, btop themes, yazi flavors) from themes/*/palette.toml.
func (c *common) themeCmd(args []string) error {
	st, err := state.Load(c.m.Home)
	if err != nil {
		return err
	}
	active := st.Theme
	if active == "" {
		active = theme.Default
	}

	if len(args) == 0 {
		// interactive selector when on a terminal — names are hard to
		// remember; plain list otherwise (scripts, pipes)
		if isTerminal() {
			summary, err := tui.RunThemePicker(c.m, st)
			if err != nil {
				return err
			}
			if summary == "" {
				fmt.Println("theme unchanged")
			} else {
				fmt.Print(summary)
			}
			return nil
		}
		fmt.Println("themes (● = active):")
		for _, n := range theme.Names() {
			p, err := theme.Load(n)
			if err != nil {
				return err
			}
			mark := " "
			if n == active {
				mark = "●"
			}
			fmt.Printf("  %s %-24s %s\n", mark, n, p.Label)
		}
		fmt.Println("\nswitch with: dotfiles theme <name>")
		return nil
	}

	name := args[0]
	if name == "render" {
		if err := theme.RenderAssets(c.m.Root); err != nil {
			return err
		}
		if err := theme.NormalizeRepo(c.m.Root); err != nil {
			return err
		}
		fmt.Println("rendered zellij/themes, btop/themes and yazi/flavors; repo configs normalized to " + theme.Default)
		return nil
	}
	p, err := theme.Load(name)
	if err != nil {
		return err
	}
	if c.dryRun {
		fmt.Printf("would apply theme %s (%s)\n", name, p.Label)
		return nil
	}
	fmt.Printf("applying theme %s (%s)\n\n", name, p.Label)
	res, err := theme.Apply(c.m, name, nil, os.Stdout)
	if err != nil {
		return err
	}
	st.Theme = name
	if err := st.Save(); err != nil {
		return err
	}
	fmt.Printf("\ndone: %d file(s) written, active theme saved to state.toml\n", res.Written)
	for _, n := range res.Notices {
		fmt.Println("NOTE:", n)
	}
	fmt.Println("\nwhat picks it up when:")
	for _, r := range res.Reload {
		fmt.Println("  -", r)
	}
	return nil
}
