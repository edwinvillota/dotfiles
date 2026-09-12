# Launch lazysql against the connections defined in databases.zsh, and plot
# query results with `chart`.
#
#   db              -> pick a connection interactively (fzf)
#   db dev          -> connect to the first name matching "dev"
#   db -l           -> list configured connection names
#
#   chart dev 'select day, revenue from sales'   -> plot a query
#   chart 'select ...'                           -> only one connection defined
#   psql ... --csv | chart                       -> plot CSV on stdin
#   chart -i results.csv                         -> plot a CSV file
#
# Reads the same NVIM_DB_<NAME> variables Neovim uses, so shell and editor
# never drift apart.

# lazysql shells out to this for "open in external editor" (it execs the binary
# directly, so it must be a bare command name -- no flags). Scoped to lazysql
# via SQL_EDITOR so it does not change EDITOR for git and everything else.
export SQL_EDITOR=nvim

# Connection names, lowercased, from the NVIM_DB_* environment.
_db_names() {
  print -l ${(f)"$(env | grep '^NVIM_DB_' | cut -d= -f1 | sed 's/^NVIM_DB_//' | tr 'A-Z' 'a-z' | sort)"}
}

# Resolve a connection name to its URL. $1 is an optional substring; with none,
# picks the only connection or prompts with fzf. Prints the URL on stdout.
_db_url() {
  local -a names
  names=(${(f)"$(_db_names)"})

  if (( ${#names} == 0 )); then
    print -u2 "no NVIM_DB_* connections defined (see ~/.config/zsh/databases.zsh)"
    return 1
  fi

  local choice
  if [[ -n "$1" ]]; then
    choice=${names[(r)*$1*]}
    if [[ -z "$choice" ]]; then
      print -u2 "no connection matching '$1'. Available: ${names[*]}"
      return 1
    fi
  elif (( ${#names} == 1 )); then
    choice=$names[1]
  else
    choice=$(print -l $names | fzf --height=40% --reverse --prompt='database> ') || return 1
  fi

  local var="NVIM_DB_${choice:u}"
  print -r -- "${(P)var}"
}

db() {
  if [[ "$1" == "-l" || "$1" == "--list" ]]; then
    _db_names
    return 0
  fi

  local url
  url=$(_db_url "$1") || return 1

  # Prefer the locally patched build (autocomplete fixes) when present.
  local bin=lazysql
  [[ -x "$HOME/.local/bin/lazysql-patched" ]] && bin="$HOME/.local/bin/lazysql-patched"
  "$bin" "$url"
}

# --- charting ---------------------------------------------------------------

CHART_HOME=${CHART_HOME:-$HOME/.config/chart}
CHART_VENV=${CHART_VENV:-$HOME/.local/share/chart/venv}

# plotext is pip-only, so it gets its own venv rather than polluting the system
# python. Created on first use; `chart --setup` forces a rebuild.
_chart_python() {
  if [[ ! -x "$CHART_VENV/bin/python" ]]; then
    print -u2 "chart: creating venv at $CHART_VENV (first run)"
    python3 -m venv "$CHART_VENV" || return 1
    "$CHART_VENV/bin/pip" -q install --upgrade pip >/dev/null 2>&1
    "$CHART_VENV/bin/pip" -q install plotext || return 1
  fi
  print -r -- "$CHART_VENV/bin/python"
}

chart() {
  local -a passthru
  local conn="" sql="" setup=0

  while (( $# )); do
    case "$1" in
      --setup) setup=1; shift ;;
      -h|--help)
        print "usage: chart [conn] 'SQL'        plot a query against a NVIM_DB_* connection"
        print "       chart -f query.sql        run a query from a file"
        print "       chart -i results.csv      plot a CSV file"
        print "       ... | chart               plot CSV on stdin"
        print "  options: -t line|bar|barh|scatter|hist|box  -y col1,col2  -x col"
        print "           -T title  -W width  -H height"
        return 0 ;;
      -f|--file)
        [[ -r "$2" ]] || { print -u2 "chart: cannot read $2"; return 1 }
        sql=$(<"$2"); shift 2 ;;
      -i|--input|-t|--type|-T|--title|-W|--width|-H|--height|-x|--xcol|-y|--ycols|--theme)
        passthru+=("$1" "$2"); shift 2 ;;
      *)
        # first bare word is a connection name if it matches one, else it's SQL
        if [[ -z "$sql" && "$1" != *[[:space:]]* ]] && print -l ${(f)"$(_db_names)"} | grep -qx "$1"; then
          conn="$1"
        elif [[ -z "$sql" ]]; then
          sql="$1"
        else
          passthru+=("$1")
        fi
        shift ;;
    esac
  done

  if (( setup )); then
    rm -rf "$CHART_VENV"
    _chart_python >/dev/null && print "chart: venv ready at $CHART_VENV"
    return $?
  fi

  local py
  py=$(_chart_python) || { print -u2 "chart: could not prepare the python venv"; return 1 }

  # -i FILE or stdin: no database involved
  if [[ " ${passthru[*]} " == *" -i "* || " ${passthru[*]} " == *" --input "* ]] || [[ -z "$sql" && ! -t 0 ]]; then
    "$py" "$CHART_HOME/chart.py" "${passthru[@]}"
    return $?
  fi

  if [[ -z "$sql" ]]; then
    print -u2 "chart: no query given (try: chart -h)"
    return 1
  fi

  local url
  url=$(_db_url "$conn") || return 1

  # --no-psqlrc: ~/.psqlrc switches format based on ON_ERROR_STOP and turns on
  # \timing, none of which should reach the parser. --csv is set explicitly.
  psql "$url" --no-psqlrc --csv -q -v ON_ERROR_STOP=1 -c "$sql" \
    | "$py" "$CHART_HOME/chart.py" "${passthru[@]}"
  # report whichever end of the pipe failed: psql's status alone would hide a
  # renderer error (an empty result set, a non-numeric column, ...)
  local -a st=(${pipestatus[@]})
  (( st[1] )) && return $st[1]
  return $st[2]
}
