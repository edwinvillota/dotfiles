package deps

import (
	"path/filepath"
	"strings"
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
	if w := got["wezterm"]; w.Status != Missing || w.Manager != "deb" {
		t.Errorf("wezterm on apt should install from the official .deb, got %+v", w)
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

func TestResolveDebPackage(t *testing.T) {
	m := load(t)
	t.Setenv("PATH", t.TempDir())
	p := Platform{OS: "linux", Arch: "amd64", Distro: "ubuntu", Apt: true, Sudo: true, BrewDir: "/home/linuxbrew/.linuxbrew"}
	var wez Item
	for _, it := range Resolve(m, p, []string{"wezterm"}) {
		wez = it
	}
	if wez.Status != Missing || wez.Manager != "deb" || !strings.Contains(wez.Cmd[2], "apt-get install -y") || !strings.Contains(wez.Cmd[2], "Ubuntu22.04.deb") {
		t.Errorf("wezterm deb resolve wrong: %+v", wez)
	}
	p.Arch = "arm64"
	for _, it := range Resolve(m, p, []string{"wezterm"}) {
		wez = it
	}
	if wez.Status != Missing || !strings.Contains(wez.Cmd[2], "arm64.deb") {
		t.Errorf("arm64 should install the arm64 deb: %+v", wez)
	}
}

func TestResolveArchPacman(t *testing.T) {
	m := load(t)
	t.Setenv("PATH", t.TempDir())
	p := Platform{OS: "linux", Arch: "amd64", Distro: "arch", Pacman: true, Sudo: true, BrewDir: "/home/linuxbrew/.linuxbrew"}
	got := map[string]Item{}
	for _, it := range Resolve(m, p, append(m.Deps.Core, m.Deps.Extra...)) {
		got[it.Name] = it
	}
	want := map[string]string{
		"wezterm": "wezterm", "nvim": "neovim", "zellij": "zellij", "yazi": "yazi",
		"gh": "github-cli", "sevenzip": "7zip", "powerlevel10k": "zsh-theme-powerlevel10k",
		"dust": "dust", "atuin": "atuin", "fd": "fd", "gdu": "gdu",
	}
	for n, pkg := range want {
		g := got[n]
		if g.Status != Missing || g.Manager != "pacman" || g.Pkg != pkg {
			t.Errorf("%s on arch: got %s %s %s, want pacman %s", n, g.Status, g.Manager, g.Pkg, pkg)
		}
		if len(g.Cmd) > 0 && g.Cmd[0] != "sudo" {
			t.Errorf("%s: pacman must run under sudo: %v", n, g.Cmd)
		}
	}
	// AUR-only tools fall back to Linuxbrew
	for _, n := range []string{"zsh-autocomplete", "carapace", "lazysql"} {
		if got[n].Status != NeedsBrew {
			t.Errorf("%s on arch should need Homebrew, got %s", n, got[n].Status)
		}
	}
}
