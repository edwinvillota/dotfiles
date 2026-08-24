# dotfiles

Personal configuration for nvim, zsh (oh-my-zsh + powerlevel10k), zellij, wezterm,
yazi, btop, atuin, gh/gh-dash, lazydocker, lazysql, colima, visidata, ssh (public
parts only) — plus `dotfiles`, a small Go tool that keeps the live config and this
repo in sync in **both directions**, on macOS and Linux, with dry-runs, backups and
rollback.

```
dotfiles guide           # plain-English tour of the tool — start here
dotfiles setup           # guided fresh-machine walkthrough (profile → tools → configs)
dotfiles                 # TUI (vim keys). Toggle units/files, preview, diff, run.
dotfiles backup  [-n]    # live  -> repo
dotfiles install [-n]    # repo  -> live   (never overwrites ~/.zshrc or secrets)
dotfiles deps    [-n]    # install missing tools (brew / apt / pacman)
dotfiles diff            # what differs between repo and live
dotfiles uninstall       # undo install, restore the originals
dotfiles check           # refuse to commit anything secret-looking
dotfiles profile work    # personal (branch main) | work (branch office)
```
`-n` / `--dry-run` shows exactly what would happen and writes nothing.

## Fresh machine

```sh
curl -fsSL https://raw.githubusercontent.com/edwinvillota/dotfiles/main/bootstrap.sh | bash
export DOTFILES=~/.dotfiles
dotfiles setup              # guided: profile → tools → configs, all confirmed first
```
Only `git` and `curl` are required up front; the tool is a single static binary.
To rehearse a fresh install first: `make try` opens a throwaway Ubuntu container
with nothing installed where you can run `dotfiles setup` end to end.

## How it works

Everything is declared in [`dotfiles.toml`](dotfiles.toml):

- **units** map a repo path to a live path (`~/.config/nvim`, `~/.zshrc`, …).
  `granular` globs become individually toggleable sub-units (each nvim plugin
  spec, each `~/.config/zsh/*.zsh` file). `ignore` is never synced; `only` is an
  allowlist (used for `~/.ssh`: `config` + `*.pub`, never keys).
- **`mode = "backup-only"`** (`~/.zshrc`, `~/.gitconfig`): backed up, but on
  install only written when absent — the modular `~/.config/zsh/*.zsh` layout and
  its load order are never touched.
- **secrets** (`zsh/config/databases.zsh`, `mec.zsh`): never copied. Backup writes
  `<name>.template` with every `VAR=value` blanked (add `# public` to a line to keep
  its value). Install creates the live file from the template (`chmod 600`) only if
  it does not exist. `.gitignore` hard-blocks the real files and `dotfiles check`
  (also a pre-commit hook via `dotfiles hook`) scans for secret patterns.
- **profiles** exclude machine-specific files (`personal` ↔ `main`,
  `work` ↔ `office`). `dotfiles profile work` creates the branch if missing.
- **deps** is a per-platform package table: brew names, apt/pacman names, binary
  names that differ (`gdu` → `gdu-go` on macOS, `fd` → `fd-find` on apt, …),
  minimum versions (nvim ≥ 0.12), git/gh-extension installs (oh-my-zsh, gh-dash).

Safety: install preserves anything it overwrites or deletes under
`~/.local/state/dotfiles/backups/<ts>/` and records it in a ledger; `uninstall`
restores it byte-for-byte. Install never deletes live-only files unless `--prune`.
Re-running is a no-op. Per-machine choices (profile, toggles, symlink mode) live in
`~/.config/dotfiles/state.toml`, never in the repo.

## TUI keys

`j/k` move · `gg`/`G` · `l`/`h` expand/collapse · `Space` toggle · `a` all ·
`/` filter · `Tab` pane · `d` diff · `m` backup⇄install · `p` profile ·
`s` symlink · `b` backup · `i` install · `?` help · `q` quit

## Development

```sh
make unit               # go vet + tests (planner, redactor, deps table, TUI, …)
make try                # interactive fresh-machine container (rehearse `dotfiles setup`)
make install-bin        # install the binary to ~/.local/bin
make test               # full install/uninstall cycle in a throwaway Ubuntu container
E2E_NET=1 make test     # + real apt installs and git clones
E2E_BREW=1 make test    # + Linuxbrew bootstrap (slow)
make release            # darwin-arm64, linux-amd64, linux-arm64 in dist/
```
