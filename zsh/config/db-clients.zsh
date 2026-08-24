# Launch lazysql against the connections defined in databases.zsh.
#
#   db              -> pick a connection interactively (fzf)
#   db dev          -> connect to the first name matching "dev"
#   db -l           -> list configured connection names
#
# Reads the same NVIM_DB_<NAME> variables Neovim uses, so shell and editor
# never drift apart.

# lazysql shells out to this for "open in external editor" (it execs the binary
# directly, so it must be a bare command name -- no flags). Scoped to lazysql
# via SQL_EDITOR so it does not change EDITOR for git and everything else.
export SQL_EDITOR=nvim

db() {
  local -a names
  names=(${(f)"$(env | grep '^NVIM_DB_' | cut -d= -f1 | sed 's/^NVIM_DB_//' | tr 'A-Z' 'a-z' | sort)"})

  if (( ${#names} == 0 )); then
    print -u2 "db: no NVIM_DB_* connections defined (see ~/.config/zsh/databases.zsh)"
    return 1
  fi

  if [[ "$1" == "-l" || "$1" == "--list" ]]; then
    print -l $names
    return 0
  fi

  local choice
  if (( $# )); then
    choice=${names[(r)*$1*]}
    if [[ -z "$choice" ]]; then
      print -u2 "db: no connection matching '$1'. Available: ${names[*]}"
      return 1
    fi
  elif (( ${#names} == 1 )); then
    choice=$names[1]
  else
    choice=$(print -l $names | fzf --height=40% --reverse --prompt='database> ') || return 1
  fi

  local var="NVIM_DB_${choice:u}"
  # Prefer the locally patched build (autocomplete fixes) when present.
  local bin=lazysql
  [[ -x "$HOME/.local/bin/lazysql-patched" ]] && bin="$HOME/.local/bin/lazysql-patched"
  "$bin" "${(P)var}"
}

# Diagnostic: print the raw bytes a key combo sends. Press keys, then Ctrl-C.
# Ctrl-_ should show as ^_ ; if nothing prints, the terminal never sent it.
keytest() { print "Press keys (Ctrl-C to exit):"; cat -v }
