package deps

import (
	"path/filepath"
	"testing"

	"github.com/edwinvillota/dotfiles/internal/manifest"
)

func load(t *testing.T) *manifest.Manifest {
	t.Helper()
	// use the real manifest so the table itself is under test
	root, _ := filepath.Abs("../..")
	m, err := manifest.Load(filepath.Join(root, "dotfiles.toml"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestResolveLinuxApt(t *testing.T) {
	m := load(t)
	p := Platform{OS: "linux", Arch: "amd64", Distro: "ubuntu", Apt: true, BrewDir: "/home/linuxbrew/.linuxbrew"}
	t.Setenv("PATH", t.TempDir()) // nothing installed
	got := map[string]Item{}
	for _, it := range Resolve(m, p, append(m.Deps.Core, m.Deps.Extra...)) {
		got[it.Name] = it
	}
	want := map[string]struct {
		st  Status
		mgr string
		pkg string
	}{
		"fd":        {Missing, "apt", "fd-find"},
		"ripgrep":   {Missing, "apt", "ripgrep"},
		"nvim":      {NeedsBrew, "brew", "neovim"},
		"zellij":    {NeedsBrew, "brew", "zellij"},
		"gdu":       {Missing, "apt", "gdu"},
		"sevenzip":  {Missing, "apt", "7zip"},
		"poppler":   {Missing, "apt", "poppler-utils"},
		"docker":    {Missing, "apt", "docker.io"},
		"lazysql":   {NeedsBrew, "brew", "lazysql"},
		"oh-my-zsh": {Missing, "git", "https://github.com/ohmyzsh/ohmyzsh.git"},
		"gh-dash":   {Missing, "gh-extension", "dlvhdr/gh-dash"},
	}
	for n, w := range want {
		g := got[n]
		if g.Status != w.st || g.Manager != w.mgr || g.Pkg != w.pkg {
			t.Errorf("%s: got %s %s %s, want %s %s %s", n, g.Status, g.Manager, g.Pkg, w.st, w.mgr, w.pkg)
		}
	}
	if got["colima"].Status != Unsupported {
		t.Error("colima should be darwin-only")
	}
	if got["wezterm"].Status != Unsupported {
		t.Errorf("wezterm on apt should be unsupported with a note, got %s", got["wezterm"].Status)
	}
	if got["gdu"].Bin != "gdu" {
		t.Errorf("gdu bin on linux = %q", got["gdu"].Bin)
	}
}

func TestResolveDarwin(t *testing.T) {
	m := load(t)
	fake := t.TempDir()
	p := Platform{OS: "darwin", Arch: "arm64", BrewDir: fake, Brew: fake + "/bin/brew"}
	t.Setenv("PATH", t.TempDir())
	got := map[string]Item{}
	for _, it := range Resolve(m, p, []string{"gdu", "wezterm", "fd", "colima"}) {
		got[it.Name] = it
	}
	if got["gdu"].Bin != "gdu-go" {
		t.Errorf("gdu bin on darwin = %q, want gdu-go", got["gdu"].Bin)
	}
	// WezTerm.app may exist on the host; only assert the install shape when missing
	if w := got["wezterm"]; w.Status == Missing && (len(w.Cmd) != 4 || w.Cmd[2] != "--cask") {
		t.Errorf("wezterm should be a cask install: %v", w.Cmd)
	}
	if got["fd"].Pkg != "fd" || got["colima"].Status != Missing {
		t.Errorf("unexpected: fd=%+v colima=%+v", got["fd"], got["colima"])
	}
}

func TestVersionLess(t *testing.T) {
	if !less("0.9.5", "0.12") || less("0.12.4", "0.12") || less("1.0", "0.99") {
		t.Error("version compare wrong")
	}
}
