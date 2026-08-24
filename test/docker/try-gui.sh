#!/usr/bin/env bash
# Started by `make try-gui`: virtual display + browser access, then a shell.
set -e
Xvfb :0 -screen 0 1600x1000x24 >/tmp/xvfb.log 2>&1 &
sleep 1
openbox >/tmp/openbox.log 2>&1 &
x11vnc -display :0 -nopw -forever -shared -quiet >/tmp/x11vnc.log 2>&1 &
websockify --web=/usr/share/novnc 6080 localhost:5900 >/tmp/novnc.log 2>&1 &
cat <<'MSG'
┌────────────────────────────────────────────────────────────────────┐
│  FRESH-MACHINE PLAYGROUND — WITH DISPLAY                           │
│                                                                    │
│  Open in your browser:  http://localhost:6080/vnc.html             │
│  (click Connect — you'll see an empty desktop)                     │
│                                                                    │
│  In THIS shell, walk through a real install:                       │
│    dotfiles setup            profile → tools (incl. wezterm .deb)  │
│    wezterm &                 opens in the browser window, with     │
│                              your wezterm.lua, MesloLGS NF, zsh,   │
│                              p10k, zellij — test everything there  │
│                                                                    │
│  Notes: amd64 emulation on Apple Silicon → not fast; Homebrew      │
│  compiles can take a while. `dotfiles uninstall` reverts configs.  │
│  Exit this shell and the machine evaporates.                       │
└────────────────────────────────────────────────────────────────────┘
MSG
exec bash
