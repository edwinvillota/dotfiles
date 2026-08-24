#!/usr/bin/env bash
# Interactive "fresh machine" playground (started by `make try`).
cat <<'MSG'
┌─────────────────────────────────────────────────────────────────┐
│  FRESH-MACHINE PLAYGROUND (throwaway Ubuntu container)          │
│                                                                 │
│  Nothing here touches your real machine. The repo is mounted    │
│  read-only at $DOTFILES; the `dotfiles` binary is on PATH.      │
│                                                                 │
│  Suggested tour:                                                │
│    dotfiles guide            what everything does               │
│    dotfiles setup            the guided install (say yes!)      │
│    ls ~/.config              see what appeared                  │
│    dotfiles uninstall        watch it restore the machine       │
│    dotfiles                  the TUI also works in here         │
│                                                                 │
│  Type `exit` to leave — the container evaporates.               │
└─────────────────────────────────────────────────────────────────┘
MSG
exec bash
