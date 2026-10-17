#!/usr/bin/env bash
# One-off catch-up for a second machine (the work laptop) after the
# October 2026 update round (PRs #21-#24). Run it AFTER:
#
#   git pull && dotfiles diff && dotfiles install && dotfiles deps --core
#
# Those bring over everything that lives in the repo. This script does the
# parts that don't: package upgrades, the zsh-autocomplete pin, and removing
# lazysql. It only touches what is installed, backs up before changing
# anything, and keeps old brew versions around for rollback.
#
#   scripts/catchup-2026-10.sh --dry-run   # show what would run
#   scripts/catchup-2026-10.sh             # do it (asks before each step)
#   scripts/catchup-2026-10.sh -y          # do it without asking
#
# Safe to delete once the laptop is caught up.
set -euo pipefail

DRY=0 YES=0
for arg in "$@"; do
  case "$arg" in
    -n|--dry-run) DRY=1 ;;
    -y|--yes) YES=1 ;;
    -h|--help) sed -n '2,17p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $arg (try --help)" >&2; exit 2 ;;
  esac
done

# zsh-autocomplete 26.08.x breaks two-line p10k prompts with a transient prompt
# (upstream marlonrichert/zsh-autocomplete#891); this is the last good release.
ZAC_GOOD=25.03.19
# kitty 0.49.2 makes the cursor jump while zsh-autocomplete draws its list
# (kitty PR #10560, no option to turn it off). 0.49.0 is fine.
KITTY_BAD=0.49.2

BACKUPS="$HOME/.local/state/dotfiles/backups"
TS=$(date +%Y%m%d-%H%M%S)

bold() { printf '\n\033[1m%s\033[0m\n' "$*"; }
info() { printf '  %s\n' "$*"; }
warn() { printf '  \033[33m! %s\033[0m\n' "$*"; }

run() {
  printf '  \033[2m$ %s\033[0m\n' "$*"
  (( DRY )) || "$@"
}

confirm() {
  (( DRY || YES )) && return 0
  read -r -p "  $1 [y/N] " reply
  [[ $reply == [yY]* ]]
}

if (( EUID == 0 )); then
  echo "run this as your user, not root (Homebrew refuses to run as root)" >&2
  exit 1
fi

BREW=$(command -v brew || true)
for candidate in /home/linuxbrew/.linuxbrew/bin/brew /opt/homebrew/bin/brew /usr/local/bin/brew; do
  [[ -n $BREW ]] && break
  [[ -x $candidate ]] && BREW=$candidate
done
if [[ -z $BREW ]]; then
  echo "Homebrew not found; run 'dotfiles deps --core' first" >&2
  exit 1
fi
brew() { "$BREW" "$@"; }

installed() { brew list --formula --versions "$1" >/dev/null 2>&1; }

(( DRY )) && bold "Dry run: nothing will be changed"

# --- 1. zsh-autocomplete: pin the last good version -------------------------
bold "1. zsh-autocomplete"
if ! installed zsh-autocomplete; then
  info "not installed via Homebrew, skipping"
else
  versions=$(brew list --formula --versions zsh-autocomplete | cut -d' ' -f2-)
  info "installed: $versions"
  if [[ $versions == "$ZAC_GOOD" ]]; then
    if brew list --pinned | grep -qx zsh-autocomplete; then
      info "already pinned at $ZAC_GOOD"
    elif confirm "pin zsh-autocomplete at $ZAC_GOOD so brew upgrade skips it?"; then
      run brew pin zsh-autocomplete
    fi
  else
    warn "expected only $ZAC_GOOD. 26.08.x breaks the two-line p10k prompt (upstream #891)."
    warn "Not pinning a broken version. Leaving it alone; ask Claude to roll it back."
  fi
fi

# --- 2. atuin: back up before its one-way database migration ----------------
bold "2. atuin backup"
if [[ -d $HOME/.local/share/atuin ]] && installed atuin; then
  archive="$BACKUPS/atuin-data-before-upgrade-$TS.tar.gz"
  run mkdir -p "$BACKUPS"
  run tar czf "$archive" -C "$HOME/.local/share" atuin
  run chmod 600 "$archive"
else
  info "no atuin data or atuin not installed via Homebrew, skipping"
fi

# --- 3. upgrades ------------------------------------------------------------
bold "3. Homebrew upgrades"
wanted=(gh neovim tree-sitter lazygit yazi ripgrep fd bat dust gdu sevenzip
        poppler imagemagick zoxide carapace atuin tmux)
to_upgrade=()
for f in "${wanted[@]}"; do installed "$f" && to_upgrade+=("$f"); done
info "installed and in scope: ${to_upgrade[*]:-none}"

if confirm "run brew update && brew upgrade on those?"; then
  run brew update
  # Keep the old versions on disk so a bad release can be rolled back.
  (( ${#to_upgrade[@]} )) && run env HOMEBREW_NO_INSTALL_CLEANUP=1 "$BREW" upgrade "${to_upgrade[@]}"
fi

# tree-sitter 0.27 split the CLI into its own formula; nvim-treesitter needs it.
if ! command -v tree-sitter >/dev/null 2>&1; then
  if confirm "tree-sitter CLI missing; brew install tree-sitter-cli?"; then
    run brew install tree-sitter-cli
  fi
fi

if command -v gh >/dev/null 2>&1 && gh extension list 2>/dev/null | grep -q 'dlvhdr/gh-dash'; then
  confirm "upgrade the gh-dash extension?" && run gh extension upgrade dash
fi

# --- 4. lazysql: removed from the dotfiles ----------------------------------
bold "4. lazysql removal"
lazysql_paths=()
for p in "$HOME/.config/lazysql" "$HOME/Library/Application Support/lazysql" "$HOME/.local/bin/lazysql-patched"; do
  [[ -e $p ]] && lazysql_paths+=("$p")
done
if ! installed lazysql && (( ${#lazysql_paths[@]} == 0 )); then
  info "nothing to remove"
elif confirm "archive lazysql config/history and remove lazysql?"; then
  if (( ${#lazysql_paths[@]} )); then
    run mkdir -p "$BACKUPS"
    run tar czf "$BACKUPS/lazysql-removed-$TS.tar.gz" "${lazysql_paths[@]}"
    run rm -rf "${lazysql_paths[@]}"
  fi
  installed lazysql && run brew uninstall lazysql
fi

# --- 5. nvim plugins: match the repo's lazy-lock.json -----------------------
bold "5. nvim plugins"
if command -v nvim >/dev/null 2>&1 && [[ -f $HOME/.config/nvim/lazy-lock.json ]]; then
  confirm "restore nvim plugins to the synced lazy-lock.json?" && run nvim --headless "+Lazy! restore" +qa
else
  info "nvim or ~/.config/nvim/lazy-lock.json missing (run 'dotfiles install' first), skipping"
fi

# --- 6. report --------------------------------------------------------------
bold "6. Check"
for f in gh nvim tmux atuin zoxide carapace lazygit yazi rg tree-sitter; do
  command -v "$f" >/dev/null 2>&1 || continue
  case $f in
    tmux) v=$(tmux -V) ;;
    yazi) v=$(yazi --version 2>/dev/null | awk '/Version:/{print $2; exit}') ;;
    lazygit) v=$(lazygit --version 2>/dev/null | tr ',' '\n' | awk -F= '/^ ?version=/{print $2; exit}') ;;
    *) v=$("$f" --version 2>/dev/null | head -1) ;;
  esac
  info "$f: $v"
done
if command -v kitty >/dev/null 2>&1; then
  kv=$(kitty --version 2>/dev/null | awk '{print $2}')
  info "kitty: $kv"
  [[ $kv == "$KITTY_BAD" ]] && warn "kitty $KITTY_BAD makes the cursor jump with zsh-autocomplete; 0.49.0 is fine"
fi

bold "Next"
info "Restart tmux (and the terminal) to switch to the new tmux."
info "Open a new shell and type a command with completions showing: the p10k prompt should stay put."
(( DRY )) && info "This was a dry run; rerun without --dry-run to apply."
exit 0
