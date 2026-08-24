# Point the `dotfiles` tool at the repo so it works from any directory.
# Checks the common clone locations; first match wins.
for _dotfiles_dir in "$HOME/Documents/dev/dotfiles" "$HOME/.dotfiles" "$HOME/dotfiles"; do
  if [[ -f "$_dotfiles_dir/dotfiles.toml" ]]; then
    export DOTFILES="$_dotfiles_dir"
    break
  fi
done
unset _dotfiles_dir
