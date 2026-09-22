#!/usr/bin/env bash
# Non-interactive counterpart to try-gui.sh: drives kitty on the virtual
# display and saves PNGs of what is actually on screen, so a change to the
# terminal config can be reviewed without a human watching the VNC session.
# Output lands in /out, which `make capture` bind-mounts.
set -u

OUT=${OUT:-/out}
mkdir -p "$OUT"

Xvfb :0 -screen 0 1600x1000x24 >/tmp/xvfb.log 2>&1 &
sleep 2
openbox >/tmp/openbox.log 2>&1 &
sleep 1

dotfiles install --unit kitty --unit zellij --unit yazi --unit zsh --yes >"$OUT/install.log" 2>&1
echo "--- installed ---" >>"$OUT/install.log"
ls -l "$HOME/.config/kitty" >>"$OUT/install.log" 2>&1

# A picture with obvious structure, so a correct render is unmistakable and a
# fallback renderer (chafa's blocks) is equally obvious.
mkdir -p "$HOME/pics"
convert -size 640x400 gradient:'#ff6188-#78dce8' "$HOME/pics/gradient.png"
convert -size 640x400 pattern:checkerboard -scale 640x400 "$HOME/pics/checker.png"

# shot <name> <seconds-to-wait> <command...>
shot() {
	local name=$1 wait=$2; shift 2
	kitty --title "$name" "$@" >/tmp/kitty-$name.log 2>&1 &
	local pid=$!
	sleep "$wait"
	import -window root "$OUT/$name.png" 2>/dev/null || xwd -root -silent | convert xwd:- "$OUT/$name.png"
	# capture what the terminal thinks is on its screen, as text
	kill $pid 2>/dev/null
	wait $pid 2>/dev/null
	sleep 1
}

# 1. kitty alone: font, colors, prompt. No multiplexer.
shot 01-kitty-plain 6 zsh -c 'echo; echo "  kitty $(kitty --version)"; echo; echo "  ANSI: \e[31mred \e[32mgreen \e[33myellow \e[34mblue \e[35mmagenta \e[36mcyan\e[0m"; echo "  Nerd glyphs:      "; echo; sleep 600'

# 2. kitty showing an image directly through its own graphics protocol.
shot 02-kitty-icat 6 zsh -c 'kitten icat --align left ~/pics/gradient.png; echo "icat exit=$?"; sleep 600'

# 3. yazi OUTSIDE zellij — the adapter it picks and whether the preview draws.
shot 03-yazi-no-zellij 9 zsh -c 'cd ~/pics && yazi; sleep 600'

# 4. yazi INSIDE zellij — the configuration the user actually runs. zellij
# spawns $SHELL in its pane, so pointing SHELL at a wrapper reproduces the
# real kitty -> zellij -> yazi nesting rather than approximating it.
printf '#!/bin/sh\ncd "$HOME/pics" && exec yazi\n' > /tmp/yazi-shell
chmod +x /tmp/yazi-shell
shot 04-yazi-in-zellij 14 zsh -c 'SHELL=/tmp/yazi-shell zellij --session cap; sleep 600'

# 5. zellij by itself, to confirm the status bar and theme survived.
shot 05-zellij-plain 9 zsh -c 'zellij --session cap2; sleep 600'

# Text-level evidence for the image question, independent of the pixels.
{
	echo "=== ya env (outside zellij) ==="
	kitty zsh -c 'ya env > /tmp/ya-plain.txt 2>&1' >/dev/null 2>&1 &
	sleep 6; kill %1 2>/dev/null; cat /tmp/ya-plain.txt 2>/dev/null
	echo
	echo "=== ya env (inside zellij) ==="
	printf '#!/bin/sh\nya env > /tmp/ya-zellij.txt 2>&1\nsleep 600\n' > /tmp/env-shell
	chmod +x /tmp/env-shell
	kitty zsh -c 'SHELL=/tmp/env-shell zellij --session capenv' >/dev/null 2>&1 &
	sleep 12; kill %1 2>/dev/null; cat /tmp/ya-zellij.txt 2>/dev/null
} > "$OUT/ya-env.txt" 2>&1

pkill -f zellij 2>/dev/null
echo "done" > "$OUT/STATUS"
ls -l "$OUT"
