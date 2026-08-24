package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edwinvillota/dotfiles/internal/ledger"
	"github.com/edwinvillota/dotfiles/internal/manifest"
	"github.com/edwinvillota/dotfiles/internal/plan"
)

func write(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
func read(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return "<missing>"
	}
	return string(b)
}

func setup(t *testing.T) *manifest.Manifest {
	root, home := t.TempDir(), t.TempDir()
	write(t, filepath.Join(root, "dotfiles.toml"), `
[unit.nvim]
src = "nvim"
dest = "~/.config/nvim"
ignore = ["lua/config/theme-active.lua"]
[unit.zsh]
src = "zsh/config"
dest = "~/.config/zsh"
ignore = ["00-theme.zsh"]
secret = ["databases.zsh"]
[unit.zshrc]
src = "zsh/.zshrc"
dest = "~/.zshrc"
mode = "backup-only"
`)
	write(t, filepath.Join(root, "nvim/init.lua"), "repo-init")
	write(t, filepath.Join(root, "nvim/lua/new.lua"), "new")
	write(t, filepath.Join(root, "zsh/config/nvm.zsh"), "nvm")
	write(t, filepath.Join(root, "zsh/config/databases.zsh.template"), "export NVIM_DB_X=\n")
	write(t, filepath.Join(root, "zsh/.zshrc"), "repo-zshrc")
	write(t, filepath.Join(home, ".config/nvim/init.lua"), "ORIGINAL-init")
	write(t, filepath.Join(home, ".config/nvim/stale.lua"), "stale")
	write(t, filepath.Join(home, ".zshrc"), "ORIGINAL-zshrc")
	m, err := manifest.Load(filepath.Join(root, "dotfiles.toml"), home)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func yes(string) bool { return true }

func TestInstallIdempotentUninstallRestore(t *testing.T) {
	m := setup(t)
	h := m.Home
	p, _ := plan.Build(m, plan.Options{Direction: plan.Install, Prune: true})
	res, err := Run(m, p, Options{Confirm: yes, Now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 4 || res.Deleted != 1 { // init, new.lua, nvm, databases(from template); stale deleted
		t.Errorf("unexpected result %+v", res)
	}
	if read(t, filepath.Join(h, ".config/nvim/init.lua")) != "repo-init" {
		t.Error("init not installed")
	}
	if read(t, filepath.Join(h, ".zshrc")) != "ORIGINAL-zshrc" {
		t.Error("zshrc must not be overwritten")
	}
	db := filepath.Join(h, ".config/zsh/databases.zsh")
	if read(t, db) != "export NVIM_DB_X=\n" {
		t.Error("secret not created from template")
	}
	if fi, _ := os.Stat(db); fi.Mode().Perm() != 0o600 {
		t.Errorf("secret mode = %o, want 600", fi.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(h, ".config/nvim/stale.lua")); err == nil {
		t.Error("stale not pruned")
	}

	// idempotent: second run plans nothing
	p2, _ := plan.Build(m, plan.Options{Direction: plan.Install, Prune: true})
	if n := len(p2.Changes()); n != 0 {
		t.Errorf("second install should be a no-op, got %d changes", n)
	}

	// ledger has an original for the replaced file
	led, _ := ledger.Load(h)
	e := led.Find(filepath.Join(h, ".config/nvim/init.lua"))
	if e == nil || e.Kind != ledger.Replaced || read(t, e.Backup) != "ORIGINAL-init" {
		t.Fatalf("ledger entry wrong: %+v", e)
	}

	// modify repo, reinstall: original backup must be kept, not overwritten
	write(t, filepath.Join(m.Root, "nvim/init.lua"), "repo-init-v2")
	p3, _ := plan.Build(m, plan.Options{Direction: plan.Install})
	if _, err := Run(m, p3, Options{Confirm: yes}); err != nil {
		t.Fatal(err)
	}
	led, _ = ledger.Load(h)
	e = led.Find(filepath.Join(h, ".config/nvim/init.lua"))
	if read(t, e.Backup) != "ORIGINAL-init" {
		t.Error("original backup was clobbered by reinstall")
	}

	// uninstall restores everything
	if _, err := Uninstall(m, nil, true, false, os.Stderr); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(h, ".config/nvim/init.lua")) != "ORIGINAL-init" {
		t.Error("init not restored")
	}
	if read(t, filepath.Join(h, ".config/nvim/stale.lua")) != "stale" {
		t.Error("pruned file not restored")
	}
	if _, err := os.Stat(filepath.Join(h, ".config/nvim/lua/new.lua")); err == nil {
		t.Error("created file not removed")
	}
	if _, err := os.Stat(filepath.Join(h, ".config/nvim/lua")); err == nil {
		t.Error("empty dir not cleaned")
	}
	if _, err := os.Stat(db); err == nil {
		t.Error("created secret not removed")
	}
	led, _ = ledger.Load(h)
	if len(led.Entries) != 0 {
		t.Errorf("ledger not emptied: %+v", led.Entries)
	}
}

func TestBackupWritesTemplateNotSecret(t *testing.T) {
	m := setup(t)
	write(t, filepath.Join(m.Home, ".config/zsh/databases.zsh"), "# dbs\nexport NVIM_DB_X=postgres://u:hunter2@h/db\n")
	write(t, filepath.Join(m.Home, ".config/zsh/nvm.zsh"), "nvm-live")
	p, _ := plan.Build(m, plan.Options{Direction: plan.Backup, Units: []string{"zsh"}})
	if _, err := Run(m, p, Options{Confirm: yes}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(m.Root, "zsh/config/databases.zsh")); err == nil {
		t.Fatal("secret was copied into the repo")
	}
	tmpl := read(t, filepath.Join(m.Root, "zsh/config/databases.zsh.template"))
	if strings.Contains(tmpl, "hunter2") || !strings.Contains(tmpl, "export NVIM_DB_X=\n") || !strings.Contains(tmpl, "# dbs") {
		t.Errorf("bad template:\n%s", tmpl)
	}
	if read(t, filepath.Join(m.Root, "zsh/config/nvm.zsh")) != "nvm-live" {
		t.Error("normal file not backed up")
	}
}

func TestSymlinkInstallAndUninstall(t *testing.T) {
	m := setup(t)
	p, _ := plan.Build(m, plan.Options{Direction: plan.Install, Symlink: true, Units: []string{"nvim"}})
	if _, err := Run(m, p, Options{Confirm: yes}); err != nil {
		t.Fatal(err)
	}
	init := filepath.Join(m.Home, ".config/nvim/init.lua")
	if tgt, err := os.Readlink(init); err != nil || tgt != filepath.Join(m.Root, "nvim/init.lua") {
		t.Fatalf("not linked: %v %s", err, tgt)
	}
	if _, err := Uninstall(m, nil, true, false, os.Stderr); err != nil {
		t.Fatal(err)
	}
	if read(t, init) != "ORIGINAL-init" {
		t.Error("original not restored after unlink")
	}
}

func TestZellijPermissionSeeding(t *testing.T) {
	m := setup(t)
	write(t, filepath.Join(m.Root, "dotfiles.toml"), `
[unit.zellij]
src = "zellij"
dest = "~/.config/zellij"
`)
	write(t, filepath.Join(m.Root, "zellij/plugins/zjstatus.wasm"), "wasm")
	m2, err := manifest.Load(filepath.Join(m.Root, "dotfiles.toml"), m.Home)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := plan.Build(m2, plan.Options{Direction: plan.Install})
	if _, err := Run(m2, p, Options{Confirm: yes}); err != nil {
		t.Fatal(err)
	}
	perms := ZellijPermissionsPath(m2.Home)
	b, err := os.ReadFile(perms)
	if err != nil {
		t.Fatalf("permissions not seeded: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "zjstatus.wasm\"") || !strings.Contains(got, "RunCommands") {
		t.Errorf("bad grants:\n%s", got)
	}
	// idempotent: re-run does not duplicate
	p, _ = plan.Build(m2, plan.Options{Direction: plan.Install})
	Run(m2, p, Options{Confirm: yes})
	b2, _ := os.ReadFile(perms)
	if string(b2) != got {
		t.Error("re-install duplicated grants")
	}
	// uninstall removes the seeded file
	if _, err := Uninstall(m2, nil, true, false, os.Stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(perms); err == nil {
		t.Error("uninstall should remove seeded permissions")
	}
}
