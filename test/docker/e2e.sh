#!/usr/bin/env bash
# End-to-end: seed a "pre-existing" config, install, verify, re-install
# (idempotent), uninstall --restore, verify originals are back byte-for-byte.
set -u
pass=0; fail=0
ok()   { echo "  ✓ $*"; pass=$((pass+1)); }
bad()  { echo "  ✗ $*"; fail=$((fail+1)); }
check(){ if eval "$1"; then ok "$2"; else bad "$2"; fi; }
step() { echo; echo "== $*"; }

H=$HOME
ORIG=$(mktemp -d)

step "seed pre-existing live config"
mkdir -p "$H/.config/nvim/lua/plugins" "$H/.config/zsh" "$H/.ssh"
echo "ORIGINAL init"      > "$H/.config/nvim/init.lua"
echo "ORIGINAL only-live" > "$H/.config/nvim/lua/plugins/only-live.lua"
echo "ORIGINAL zshrc"     > "$H/.zshrc"
echo "ORIGINAL secret"    > "$H/.config/zsh/databases.zsh"; chmod 600 "$H/.config/zsh/databases.zsh"
echo "ORIGINAL key"       > "$H/.ssh/id_test"; chmod 600 "$H/.ssh/id_test"
cp -a "$H/.config" "$H/.zshrc" "$H/.ssh" "$ORIG/"

step "dotfiles check (repo must be secret-free)"
check 'dotfiles check' "check passes on repo"

step "install --dry-run writes nothing"
before=$(find "$H" -type f | sort | xargs md5sum)
dotfiles install --dry-run --profile personal >/dev/null
check '[ "$before" = "$(find "$H" -type f | sort | xargs md5sum)" ]' "dry-run left HOME untouched"

step "install"
dotfiles install --yes --profile personal | tail -3
check '[ -f "$H/.config/nvim/lua/config/lazy.lua" ]' "nvim installed"
check '[ "$(cat "$H/.zshrc")" = "ORIGINAL zshrc" ]' ".zshrc NOT overwritten (backup-only)"
check '[ "$(cat "$H/.config/zsh/databases.zsh")" = "ORIGINAL secret" ]' "live secret NOT overwritten"
check '[ -f "$H/.config/nvim/lua/plugins/only-live.lua" ]' "live-only file NOT deleted without --prune"
check '[ -f "$H/.config/zsh/zz-completion.zsh" ] || [ ! -f "$DOTFILES/zsh/config/zz-completion.zsh" ]' "zsh modular files installed"
check '[ "$(cat "$H/.ssh/id_test")" = "ORIGINAL key" ]' "ssh private key untouched"
check '[ -f "$H/.local/state/dotfiles/ledger.json" ]' "ledger written"
check 'grep -q "ORIGINAL init" "$H"/.local/state/dotfiles/backups/*/.config/nvim/init.lua' "original init.lua preserved"
check 'zsh -n "$H/.config/zsh/"*.zsh' "installed zsh files parse"
if [ -f "$H/.config/zsh/mec.zsh" ]; then bad "mec.zsh (secret) was installed although live absent? should be created from template only"; fi

step "install again (idempotent)"
out=$(dotfiles install --yes --profile personal)
check 'echo "$out" | grep -q "nothing to do"' "second install is a no-op"

step "install --prune"
dotfiles install --yes --prune --profile personal >/dev/null
check '[ ! -f "$H/.config/nvim/lua/plugins/only-live.lua" ]' "--prune removed live-only file"
check 'grep -q "ORIGINAL only-live" "$H"/.local/state/dotfiles/backups/*/.config/nvim/lua/plugins/only-live.lua' "pruned file preserved"

step "uninstall --restore"
dotfiles uninstall | tail -1
check '[ "$(cat "$H/.config/nvim/init.lua")" = "ORIGINAL init" ]' "init.lua restored"
check '[ "$(cat "$H/.config/nvim/lua/plugins/only-live.lua")" = "ORIGINAL only-live" ]' "pruned file restored"
check '[ ! -f "$H/.config/nvim/lua/config/lazy.lua" ]' "installed files removed"
check '[ ! -d "$H/.config/wezterm" ]' "empty dirs removed"
check 'diff -r "$ORIG/.config" "$H/.config" >/dev/null' ".config identical to pre-install (byte-for-byte)"
check 'diff "$ORIG/.zshrc" "$H/.zshrc" >/dev/null' ".zshrc identical"
check '[ "$(python3 -c "import json;print(len(json.load(open(\"$H/.local/state/dotfiles/ledger.json\"))[\"entries\"]))" 2>/dev/null || echo 0)" = 0 ]' "ledger emptied"

step "symlink install + uninstall"
dotfiles install --yes --symlink --unit wezterm >/dev/null
check '[ -L "$H/.config/wezterm/wezterm.lua" ]' "wezterm.lua is a symlink"
dotfiles uninstall >/dev/null
check '[ ! -e "$H/.config/wezterm" ]' "symlinks removed"

step "backup from live into a scratch checkout"
SCRATCH=$(mktemp -d); cp -a "$DOTFILES/." "$SCRATCH/"
printf '# secrets\nexport NVIM_DB_T=postgres://u:hunter2@h/d\nexport HOST=h # public\n' > "$H/.config/zsh/databases.zsh"
echo "export DEMO=1" > "$H/.config/zsh/demo.zsh"
( cd "$SCRATCH" && dotfiles backup --yes --unit zsh >/dev/null )
check '[ -f "$SCRATCH/zsh/config/demo.zsh" ]' "normal zsh file backed up"
check '[ ! -f "$SCRATCH/zsh/config/databases.zsh" ]' "secret file NOT copied"
check 'grep -q "^export NVIM_DB_T=$" "$SCRATCH/zsh/config/databases.zsh.template"' "template has blanked value"
check '! grep -rq hunter2 "$SCRATCH/zsh" "$SCRATCH/nvim"' "secret value nowhere in config dirs"
check 'grep -q "export HOST=h # public" "$SCRATCH/zsh/config/databases.zsh.template"' "# public line kept"
( cd "$SCRATCH" && dotfiles check >/dev/null 2>&1 ); check '[ $? = 0 ]' "check passes after backup"
echo 'export API_KEY=sk-live-123' > "$SCRATCH/zsh/config/oops.zsh"
( cd "$SCRATCH" && ! dotfiles check >/dev/null 2>&1 ); check '[ $? = 0 ]' "check FAILS on leaked secret"

echo; echo "RESULT: $pass passed, $fail failed"
[ "$fail" = 0 ]
