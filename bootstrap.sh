#!/usr/bin/env bash
# One-line bootstrap for a fresh machine:
#   curl -fsSL https://raw.githubusercontent.com/edwinvillota/dotfiles/main/bootstrap.sh | bash
#
# What it does (nothing else):
#   1. clones the repo to ~/.dotfiles (or uses $DOTFILES if already set)
#   2. downloads the `dotfiles` binary for this OS/arch from GitHub Releases
#      (falls back to `go build` if Go is present, otherwise tells you what to do)
#   3. prints the next commands — it does NOT install anything or touch your config.
set -euo pipefail

REPO="${DOTFILES_REPO:-edwinvillota/dotfiles}"
DIR="${DOTFILES:-$HOME/.dotfiles}"
BIN_DIR="${DOTFILES_BIN:-$HOME/.local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac
case "$os" in darwin|linux) ;; *) echo "unsupported OS: $os" >&2; exit 1 ;; esac

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1 — install it first (e.g. apt-get install -y $1 / xcode-select --install)" >&2; exit 1; }; }
need git; need curl

if [ -d "$DIR/.git" ]; then
  echo "repo already at $DIR"
else
  echo "cloning $REPO -> $DIR"
  git clone "https://github.com/$REPO.git" "$DIR"
fi

mkdir -p "$BIN_DIR"
asset="dotfiles-$os-$arch"
url="https://github.com/$REPO/releases/latest/download/$asset"
echo "fetching $url"
if curl -fsSL "$url" -o "$BIN_DIR/dotfiles.tmp"; then
  chmod +x "$BIN_DIR/dotfiles.tmp" && mv "$BIN_DIR/dotfiles.tmp" "$BIN_DIR/dotfiles"
elif command -v go >/dev/null 2>&1; then
  echo "no release asset; building from source with $(go version)"
  (cd "$DIR" && go build -o "$BIN_DIR/dotfiles" ./cmd/dotfiles)
else
  rm -f "$BIN_DIR/dotfiles.tmp"
  echo "could not download a release binary and Go is not installed." >&2
  echo "either publish a release (see .github/workflows/release.yml) or install Go and re-run." >&2
  exit 1
fi

case ":$PATH:" in *":$BIN_DIR:"*) ;; *) echo "note: add $BIN_DIR to PATH (the zsh config in this repo does)";; esac

cat <<MSG

ready: $BIN_DIR/dotfiles ($("$BIN_DIR/dotfiles" version))

next steps (nothing has been installed or changed yet):
  export DOTFILES=$DIR
  $BIN_DIR/dotfiles profile personal      # or: work
  $BIN_DIR/dotfiles deps --dry-run        # review, then drop --dry-run
  $BIN_DIR/dotfiles install --dry-run     # review, then drop --dry-run
  $BIN_DIR/dotfiles                       # or use the TUI
MSG
