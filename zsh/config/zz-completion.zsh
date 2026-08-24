# Tab completion behaviour.
#
# Note: `^I` in bindkey output IS the Tab key -- Tab sends ASCII 9, which is
# Ctrl-I. There is no separate chord to remember.
#
# Design:
#   Tab        -> normal completion: fill in the longest shared prefix and list
#                 the candidates (classic shell behaviour).
#   Tab Tab    -> fzf fuzzy picker over the candidates, so you choose instead of
#                 deleting and retyping.
#
# The `**` trigger is disabled -- double-Tab replaces it, so there is only one
# way to reach fzf.
#
# Three things fight over Tab here, all loading AFTER ~/.config/zsh/*.zsh:
#   zsh-autocomplete (.zshrc:38), oh-my-zsh/compinit (.zshrc:71), fzf (.zshrc:77)
# and the `menuselect` keymap doesn't exist until compinit runs. So the bindings
# are applied on the first prompt, once everything is loaded.

# Drop `_expand` from the completer chain.
#
# zsh-autocomplete defaults to:
#   _expand _complete _complete:-fuzzy _correct _approximate _ignored
# `_expand` runs FIRST and offers an "expansion" candidate with UNESCAPED
# spaces, e.g. `/Users/.../SABANA ZONA NORTE (1).csv`. Inserting that breaks the
# line into four words, so the next Tab completes only the trailing fragment and
# appends garbage. The real `_complete` result is escaped correctly.
#
# `_correct`/`_approximate` are also dropped: they guess at typos, which on a
# directory of similarly-named files produces more noise than help.
zstyle ':completion:*' completer _complete _complete:-fuzzy _ignored

# Colour the completion list with LS_COLORS.
zstyle ':completion:*' list-colors "${(s.:.)LS_COLORS}"

# NOTE: deliberately NOT setting `insert-unambiguous` or overriding
# `matcher-list`. oh-my-zsh sets substring matching ('l:|=* r:|=*'); combined
# with insert-unambiguous, the inserted text is not a prefix of what you typed,
# so it gets APPENDED instead of replacing it -- e.g.
# "SABANA.csv" + "SABANA\ ZONA\ NORTE\ \(1\).csv" on one line.

# Empty trigger => fzf-completion always runs the fuzzy finder when we invoke
# it, rather than only on a `**` suffix. We only ever invoke it on double-Tab.
export FZF_COMPLETION_TRIGGER=''

# Tab: first press completes normally, second consecutive press opens fzf.
#
# Detection is by BUFFER comparison, not $LASTWIDGET: when a user widget calls
# `zle <other-widget>`, LASTWIDGET becomes that other widget, so the check never
# matched and double-Tab never fired.
#
# `.complete-word` (leading dot) is zsh's BUILT-IN widget. Plain `complete-word`
# is replaced by zsh-autocomplete with a version that inserts the first match
# outright -- which is the "it picked the wrong file and I have to delete it"
# behaviour. The builtin inserts only the unambiguous prefix and lists the rest.
typeset -g _tab_prev_buffer=
typeset -g _tab_prev_cursor=

_tab_then_fzf() {
  if [[ -n $_tab_prev_buffer && $BUFFER == $_tab_prev_buffer && $CURSOR == $_tab_prev_cursor ]]; then
    # second Tab with nothing typed in between -> fuzzy picker
    _tab_prev_buffer=
    _tab_prev_cursor=
    if (( $+widgets[fzf-completion] )); then
      zle fzf-completion
    else
      zle .complete-word
    fi
  else
    zle .complete-word
    _tab_prev_buffer=$BUFFER
    _tab_prev_cursor=$CURSOR
  fi
}
zle -N _tab_then_fzf

_edwin_completion_keys() {
  zmodload -i zsh/terminfo

  bindkey '^I' _tab_then_fzf
  [[ -n $terminfo[kcbt] ]] && bindkey "$terminfo[kcbt]" reverse-menu-complete

  # If a menu ever opens, make it behave sanely.
  bindkey -M menuselect '^M' .accept-line   # Enter submits
  bindkey -M menuselect '^[' send-break     # Esc cancels, line preserved

  add-zsh-hook -d precmd _edwin_completion_keys
  unfunction _edwin_completion_keys
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd _edwin_completion_keys
