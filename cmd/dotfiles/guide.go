package main

import "fmt"

func printGuide() {
	fmt.Print(`WHAT THIS TOOL DOES
  Your configs live in two places: the LIVE config your programs read
  (~/.config/nvim, ~/.zshrc, ...) and this git REPO. ` + "`dotfiles`" + ` moves
  changes between them, safely, in either direction.

      live config  --- backup --->  repo   (then: git commit / push)
      live config  <-- install ---  repo   (fresh machine, or pull changes)

EVERYDAY COMMANDS
  dotfiles                  Open the TUI. Left: what is synced (toggle with
                            Space). Right: what would change. 'b' backs up,
                            'i' installs — both show a confirmation first.
  dotfiles backup           Copy live config into the repo. Secrets are never
                            copied: they become *.template files with values
                            blanked. Run this after you tweak your config,
                            then commit and push.
  dotfiles install          Copy repo onto this machine. Never overwrites
                            ~/.zshrc or an existing secret file; anything it
                            replaces is preserved and restorable.
  dotfiles diff             Show file-by-file differences before deciding.

  Add --dry-run (-n) to ANY of these to see the plan without writing.

FRESH MACHINE
  dotfiles setup            Interactive walkthrough: pick a profile, install
                            missing tools, then install the configs. Explains
                            each step and asks before doing anything.
  dotfiles deps             Just the tools. Supported platforms: macOS (brew),
                            Ubuntu (apt + official .deb), Arch (pacman);
                            Linuxbrew is bootstrapped when a tool exists
                            nowhere else. wezterm included on all three.
  dotfiles uninstall        Undo an install: removes what was placed and
                            restores what was there before, byte for byte.

SAFETY NET
  dotfiles check            Scan for secret-looking content in the repo.
  dotfiles hook             Install check as a git pre-commit hook.
  Overwritten/deleted files land in ~/.local/state/dotfiles/backups/<time>/.
  Install deletes nothing that only exists live unless you pass --prune.

MACHINE DIFFERENCES
  dotfiles profile personal   This laptop  (branch main)
  dotfiles profile work       Work laptop  (branch office)
  A profile excludes machine-specific files (see [profile.*] in
  dotfiles.toml) and is remembered in ~/.config/dotfiles/state.toml,
  together with anything you toggle off in the TUI.

TRY IT WITHOUT RISK
  make try                  Drop into a throwaway Ubuntu container that has
                            nothing installed, and run 'dotfiles setup' there.
  make test                 Automated version of the same (44+ checks).

Full key reference inside the TUI: press ?
`)
}
