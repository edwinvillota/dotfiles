package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/edwinvillota/dotfiles/internal/manifest"
)

func TestRun(t *testing.T) {
	root := t.TempDir()
	w := func(p, s string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755)
		os.WriteFile(filepath.Join(root, p), []byte(s), 0o644)
	}
	w("dotfiles.toml", `
[unit.zsh]
src = "zsh/config"
dest = "~/.config/zsh"
secret = ["databases.zsh"]
[unit.ssh]
src = "ssh"
dest = "~/.ssh"
only = ["config", "*.pub"]
[secrets]
patterns = ['(?i)(password|token|psk)\s*=\s*\S', '://[^/:@\s]+:[^@\s]+@']
`)
	w("zsh/config/ok.zsh", "export EDITOR=nvim\nexport HOST=x # public\n")
	w("zsh/config/bad.zsh", "export TOKEN=abc\nexport DB=postgres://u:p@h/d\n")
	w("zsh/config/databases.zsh", "x")
	w("zsh/config/databases.zsh.template", "export PASSWORD=\n")
	w("ssh/id_rsa", "key")
	w("ssh/dev.pub", "ssh-ed25519 AAAA")
	m, err := manifest.Load(filepath.Join(root, "dotfiles.toml"), "/h")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := Run(m, []string{"zsh/config/ok.zsh", "zsh/config/bad.zsh", "zsh/config/databases.zsh",
		"zsh/config/databases.zsh.template", "ssh/id_rsa", "ssh/dev.pub", "dotfiles.toml"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, f := range fs {
		got[f.File]++
	}
	want := map[string]int{"zsh/config/bad.zsh": 2, "zsh/config/databases.zsh": 1, "ssh/id_rsa": 1}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %d findings want %d", k, got[k], v)
		}
	}
	for _, k := range []string{"zsh/config/ok.zsh", "zsh/config/databases.zsh.template", "ssh/dev.pub", "dotfiles.toml"} {
		if got[k] != 0 {
			t.Errorf("%s: false positive", k)
		}
	}
}
