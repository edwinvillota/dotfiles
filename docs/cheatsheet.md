---
tags: [cheatsheet, dotfiles, themes, cli, tui]
---

# dotfiles — sync tool + theme system

Repo: `~/Documents/dev/dotfiles` · binary reads `dotfiles.toml` · full tour: `dotfiles guide`

## 1. Everyday commands

| Command | Action |
| --- | --- |
| `dotfiles` | TUI (vim keys, `?` for help) |
| `dotfiles backup` | live → repo (then git commit/push). Secrets become `*.template`, values blanked |
| `dotfiles install` | repo → live. Never touches `~/.zshrc` or existing secrets |
| `dotfiles diff` | file-by-file differences |
| `dotfiles theme` | **theme picker** — switches every tool at once |
| `dotfiles uninstall` | undo install, restore originals byte-for-byte |
| `dotfiles check` | scan repo for secret-looking content (also pre-commit via `dotfiles hook`) |
| `dotfiles profile work` | switch machine profile (personal ↔ main, work ↔ office) |

`-n` / `--dry-run` on any of these = show the plan, write nothing.

## 2. TUI keys

| Key | Action |
| --- | --- |
| `j/k` `gg/G` `ctrl+d/u` | move |
| `l` / `h` | expand / collapse unit |
| `Space` / `a` | toggle unit or file / toggle all |
| `/` | filter (Esc clears) |
| `Tab` | switch pane — preview scrolls with `j/k` |
| `d` | show diffs in preview |
| `m` | flip backup ⇄ install |
| `p` | cycle profile |
| `s` | symlink mode for install |
| `t` | **theme picker** |
| `b` / `i` / `x` | run backup / install / current direction (confirm first) |

## 3. Themes

One shared palette themes wezterm, zellij (+ zjstatus bar), nvim, fzf, bat,
yazi, btop, gh-dash, visidata and the dotfiles TUI itself.

```
dotfiles theme              # interactive picker (TTY); plain list when piped
dotfiles theme nord         # direct, for scripts
dotfiles theme render       # dev: regenerate committed assets from palettes
```

Available: `ayu-dark` (default) · `iceberg` · `jellybeans` · `kanagawa-wave` ·
`kanagawa-dragon` · `github-dark` · `github-dark-colorblind` · `nord` · `tokyo-night`

**What reloads when:**

| Tool | Pickup |
| --- | --- |
| wezterm | **live** (watches `theme.lua`) |
| zellij | restart the session |
| nvim | new instances only |
| fzf / bat | new shell, or `source ~/.config/zsh/00-theme.zsh` |
| yazi / btop / gh-dash / visidata | restart the app |

**How it's wired (for future me):**

- Palettes: `themes/<name>/palette.toml`, verbatim from terminalcolors.com —
  the single source of truth, embedded in the binary.
- Active theme is per-machine state in `~/.config/dotfiles/state.toml`, never
  a commit: `install` re-applies it (post-install hook), `backup` normalizes
  theme lines back to ayu-dark before they reach the repo.
- Machine-local generated files (ignored by sync): `wezterm/theme.lua`,
  `nvim/lua/config/theme-active.lua`, `zsh/00-theme.zsh`.
- Committed generated assets: `zellij/themes/`, `btop/themes/`,
  `yazi/flavors/` — edit the palette, run `dotfiles theme render`; a unit
  test fails if they drift.
- Not themed: powerlevel10k + plain-ANSI tools (they follow the terminal
  palette, which the theme switches anyway; bat runs `--theme=ansi`).

## 4. Safety net

- Anything replaced/deleted lands in `~/.local/state/dotfiles/backups/<ts>/`,
  recorded in `~/.local/state/dotfiles/ledger.json`; `uninstall` walks it back.
- Install deletes live-only files **only** with `--prune`.
- Re-running anything is a no-op.

## 5. Development

```
make unit        # go vet + tests (includes theme renderer drift check)
make test        # Docker e2e (~72 checks, incl. theme round-trip)
make try         # throwaway Ubuntu container — rehearse `dotfiles setup`
make try-gui     # same with wezterm on http://localhost:6080/vnc.html
                 # theme check: `dotfiles theme nord` must visibly recolor it
```

Rule: develop against the sandbox, never against live config; sync live only
after `make test` is green.
