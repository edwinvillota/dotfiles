# yazi — terminal file manager
#
# `y` wraps yazi so the shell follows you: on quit, the directory you were
# browsing becomes your cwd. Use plain `yazi` when you don't want that.
function y() {
	local tmp="$(mktemp -t "yazi-cwd.XXXXXX")" cwd
	yazi "$@" --cwd-file="$tmp"
	if cwd="$(command cat -- "$tmp")" && [ -n "$cwd" ] && [ "$cwd" != "$PWD" ]; then
		builtin cd -- "$cwd"
	fi
	rm -f -- "$tmp"
}
