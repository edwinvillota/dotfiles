// Package deps installs the tools the configs need.
package deps

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Platform struct {
	OS      string // darwin | linux
	Arch    string
	Distro  string // ubuntu, debian, arch, ... ("" on darwin)
	Brew    string // path to brew binary, "" if absent
	BrewDir string // expected prefix even if not installed
	Apt     bool
	Pacman  bool
	Sudo    bool
}

func Detect() Platform {
	p := Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
	switch p.OS {
	case "darwin":
		p.BrewDir = "/opt/homebrew"
		if p.Arch == "amd64" {
			p.BrewDir = "/usr/local"
		}
	case "linux":
		p.BrewDir = "/home/linuxbrew/.linuxbrew"
		p.Distro = distro()
		p.Apt = has("apt-get")
		p.Pacman = has("pacman")
		p.Sudo = has("sudo")
	}
	if b := p.BrewDir + "/bin/brew"; fileExists(b) {
		p.Brew = b
	} else if b, err := exec.LookPath("brew"); err == nil {
		p.Brew = b
	}
	return p
}

// Native is the native package manager name ("apt", "pacman", or "").
func (p Platform) Native() string {
	switch {
	case p.Apt:
		return "apt"
	case p.Pacman:
		return "pacman"
	}
	return ""
}

func distro() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "ID=") {
			return strings.Trim(strings.TrimPrefix(sc.Text(), "ID="), `"`)
		}
	}
	return ""
}

func has(bin string) bool      { _, err := exec.LookPath(bin); return err == nil }
func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
