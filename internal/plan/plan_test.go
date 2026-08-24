package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edwinvillota/dotfiles/internal/manifest"
)

func write(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) *manifest.Manifest {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	write(t, filepath.Join(root, "dotfiles.toml"), `
[unit.nvim]
src = "nvim"
dest = "~/.config/nvim"
granular = ["lua/plugins/*.lua"]
ignore = ["*.bak", ".claude/"]

[unit.zsh]
src = "zsh/config"
dest = "~/.config/zsh"
granular = ["*.zsh"]
secret = ["databases.zsh"]

[unit.zshrc]
src = "zsh/.zshrc"
dest = "~/.zshrc"
mode = "backup-only"

[unit.ssh]
src = "ssh"
dest = "~/.ssh"
only = ["config", "*.pub"]

[unit.lazysql]
src = "lazysql"
dest = { darwin = "~/Library/Application Support/lazysql", linux = "~/.config/lazysql" }

[profile.work]
exclude = ["nvim/lua/plugins/obsidian.lua"]
`)
	// repo side
	write(t, filepath.Join(root, "nvim/init.lua"), "repo-init")
	write(t, filepath.Join(root, "nvim/lua/plugins/lsp.lua"), "same")
	write(t, filepath.Join(root, "nvim/lua/plugins/obsidian.lua"), "repo-obs")
	write(t, filepath.Join(root, "nvim/lua/plugins/old.lua"), "only-in-repo")
	write(t, filepath.Join(root, "zsh/config/nvm.zsh"), "same")
	write(t, filepath.Join(root, "zsh/.zshrc"), "repo-zshrc")
	// live side
	write(t, filepath.Join(home, ".config/nvim/init.lua"), "live-init")
	write(t, filepath.Join(home, ".config/nvim/lua/plugins/lsp.lua"), "same")
	write(t, filepath.Join(home, ".config/nvim/lua/plugins/obsidian.lua"), "live-obs")
	write(t, filepath.Join(home, ".config/nvim/lua/plugins/csv.lua"), "only-live")
	write(t, filepath.Join(home, ".config/nvim/lazy.bak"), "junk")
	write(t, filepath.Join(home, ".config/nvim/.claude/x"), "junk")
	write(t, filepath.Join(home, ".config/zsh/nvm.zsh"), "same")
	write(t, filepath.Join(home, ".config/zsh/databases.zsh"), "PGPASSWORD=x")
	write(t, filepath.Join(home, ".zshrc"), "live-zshrc")
	write(t, filepath.Join(home, ".ssh/config"), "Host x")
	write(t, filepath.Join(home, ".ssh/id_ed25519"), "PRIVATE")
	write(t, filepath.Join(home, ".ssh/id_ed25519.pub"), "PUB")
	write(t, filepath.Join(home, ".ssh/known_hosts"), "kh")

	m, err := manifest.Load(filepath.Join(root, "dotfiles.toml"), home)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func ops(p *Plan) map[string]Op {
	out := map[string]Op{}
	for _, a := range p.Actions {
		out[a.Unit+":"+a.Rel] = a.Op
	}
	return out
}

func TestBackup(t *testing.T) {
	m := fixture(t)
	p, err := Build(m, Options{Direction: Backup})
	if err != nil {
		t.Fatal(err)
	}
	got := ops(p)
	want := map[string]Op{
		"nvim:init.lua":                 OpUpdate,
		"nvim:lua/plugins/lsp.lua":      OpNone,
		"nvim:lua/plugins/obsidian.lua": OpUpdate,
		"nvim:lua/plugins/csv.lua":      OpCreate,
		"nvim:lua/plugins/old.lua":      OpDelete,
		"zsh:nvm.zsh":                   OpNone,
		"zsh:databases.zsh":             OpCreate, // -> template
		"zshrc:":                        OpUpdate,
		"ssh:config":                    OpCreate,
		"ssh:id_ed25519.pub":            OpCreate,
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: got %v want %v", k, got[k], w)
		}
	}
	for _, a := range p.Actions {
		if a.Rel == "databases.zsh" {
			if !a.Redact || !strings.HasSuffix(a.To, ".template") {
				t.Errorf("secret must go to a template: %+v", a)
			}
		}
	}
	if _, ok := got["nvim:lazy.bak"]; ok {
		t.Error("ignored file leaked into plan")
	}
	if _, ok := got["nvim:.claude/x"]; ok {
		t.Error("ignored dir leaked into plan")
	}
	if _, ok := got["ssh:id_ed25519"]; ok {
		t.Error("private key leaked into plan")
	}
	if _, ok := got["ssh:known_hosts"]; ok {
		t.Error("non-allowlisted ssh file leaked into plan")
	}
	if len(got) != len(want) {
		t.Errorf("unexpected actions: %v", got)
	}
}

func TestInstallBackupOnlyAndProfile(t *testing.T) {
	m := fixture(t)
	p, err := Build(m, Options{Direction: Install, Profile: m.Profiles["work"]})
	if err != nil {
		t.Fatal(err)
	}
	got := ops(p)
	if got["zshrc:"] != OpSkip {
		t.Errorf("zshrc must never be overwritten on install, got %v", got["zshrc:"])
	}
	if got["nvim:lua/plugins/obsidian.lua"] != OpSkip {
		t.Error("profile exclude not honored")
	}
	if got["nvim:lua/plugins/csv.lua"] != OpSkip {
		t.Error("install must not delete live-only files without --prune")
	}
	pp, _ := Build(m, Options{Direction: Install, Prune: true})
	if ops(pp)["nvim:lua/plugins/csv.lua"] != OpDelete {
		t.Error("--prune should delete live-only files")
	}
	if got["nvim:init.lua"] != OpUpdate {
		t.Error("install should update differing file")
	}
	// backup-only unit installs when absent
	os.Remove(filepath.Join(m.Home, ".zshrc"))
	p, _ = Build(m, Options{Direction: Install})
	if ops(p)["zshrc:"] != OpCreate {
		t.Error("backup-only unit should install when live absent")
	}
}

func TestSelection(t *testing.T) {
	m := fixture(t)
	sel := map[string]bool{"nvim/lua/plugins/csv.lua": true}
	p, _ := Build(m, Options{Direction: Backup, Selected: sel})
	got := ops(p)
	if got["nvim:lua/plugins/csv.lua"] != OpCreate {
		t.Error("selected granule should act")
	}
	if got["nvim:init.lua"] != OpSkip || got["nvim:lua/plugins/obsidian.lua"] != OpSkip {
		t.Error("unselected items should skip")
	}
	c, _, _, _, _ := p.Counts()
	if c != 1 {
		t.Errorf("want 1 create, got %d", c)
	}
}

func TestSymlinkInstall(t *testing.T) {
	m := fixture(t)
	p, _ := Build(m, Options{Direction: Install, Symlink: true, Units: []string{"nvim"}})
	got := ops(p)
	if got["nvim:init.lua"] != OpLink || got["nvim:lua/plugins/lsp.lua"] != OpLink {
		t.Errorf("symlink mode should link files: %v", got)
	}
	// once linked, plan is a no-op
	u := m.Units["nvim"]
	for _, a := range p.Changes() {
		if a.Op == OpLink {
			os.Remove(a.To)
			os.MkdirAll(filepath.Dir(a.To), 0o755)
			os.Symlink(a.From, a.To)
		}
	}
	_ = u
	p, _ = Build(m, Options{Direction: Install, Symlink: true, Units: []string{"nvim"}})
	if got := ops(p); got["nvim:init.lua"] != OpNone {
		t.Errorf("linked file should be OpNone, got %v", got["nvim:init.lua"])
	}
}

func TestPerOSDest(t *testing.T) {
	m := fixture(t)
	u := m.Units["lazysql"]
	m.GOOS = "darwin"
	if got := m.DestPath(u); got != filepath.Join(m.Home, "Library/Application Support/lazysql") {
		t.Errorf("darwin dest = %q", got)
	}
	m.GOOS = "linux"
	if got := m.DestPath(u); got != filepath.Join(m.Home, ".config/lazysql") {
		t.Errorf("linux dest = %q", got)
	}
}

func TestSecretInstall(t *testing.T) {
	m := fixture(t)
	tmpl := filepath.Join(m.Root, "zsh/config/databases.zsh.template")
	write(t, tmpl, "export X=\n")
	// live exists -> never overwritten
	p, _ := Build(m, Options{Direction: Install, Units: []string{"zsh"}})
	if ops(p)["zsh:databases.zsh"] != OpSkip {
		t.Error("must not overwrite live secret")
	}
	if _, ok := ops(p)["zsh:databases.zsh.template"]; ok {
		t.Error("template must not be installed as a regular file")
	}
	os.Remove(filepath.Join(m.Home, ".config/zsh/databases.zsh"))
	p, _ = Build(m, Options{Direction: Install, Units: []string{"zsh"}})
	var a Action
	for _, x := range p.Actions {
		if x.Rel == "databases.zsh" {
			a = x
		}
	}
	if a.Op != OpCreate || !a.Redact || a.From != tmpl {
		t.Errorf("expected create-from-template, got %+v", a)
	}
}
