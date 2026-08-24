package deps

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/edwinvillota/dotfiles/internal/manifest"
)

// Status of one dependency after resolution.
type Status int

const (
	Present     Status = iota // found, version ok
	Outdated                  // found, below min
	Missing                   // will install
	Unsupported               // cannot install on this platform
	NeedsBrew                 // installable only via Homebrew, which is absent
)

func (s Status) String() string {
	return [...]string{"present", "outdated", "missing", "unsupported", "needs-brew"}[s]
}

// Item is a resolved dependency with the concrete action for this platform.
type Item struct {
	Name    string
	Status  Status
	Bin     string   // what we looked for
	Found   string   // where (path) or version
	Manager string   // brew | apt | pacman | git | gh-extension | script
	Pkg     string   // package name for Manager
	Cmd     []string // exact command that will run
	Note    string
	Needs   []string
}

// Resolve computes the state of every requested dep on platform p.
func Resolve(m *manifest.Manifest, p Platform, names []string) []Item {
	var out []Item
	for _, n := range names {
		out = append(out, resolveOne(m, p, n))
	}
	return out
}

func resolveOne(m *manifest.Manifest, p Platform, name string) Item {
	spec := m.Deps.Pkg[name]
	if spec == nil {
		spec = &manifest.PkgSpec{}
	}
	it := Item{Name: name, Needs: spec.Needs, Note: spec.Note}

	if len(spec.OS) > 0 && !contains(spec.OS, p.OS) {
		it.Status, it.Note = Unsupported, "not for "+p.OS
		return it
	}

	// --- presence ---
	it.Bin = spec.Bin.For(p.OS)
	if it.Bin == "" {
		it.Bin = name
	}
	switch spec.Kind {
	case "git":
		dest := expand(spec.Dest, m.Home)
		it.Manager, it.Pkg = "git", spec.URL
		if fileExists(dest) {
			it.Status, it.Found = Present, dest
			return it
		}
		it.Status = Missing
		it.Cmd = []string{"git", "clone", "--depth=1", spec.URL, dest}
		return it
	case "gh-extension":
		it.Manager, it.Pkg = "gh-extension", spec.Name
		if out, err := exec.Command("gh", "extension", "list").Output(); err == nil && strings.Contains(string(out), spec.Name) {
			it.Status, it.Found = Present, "gh extension"
			return it
		}
		it.Status = Missing
		it.Cmd = []string{"gh", "extension", "install", spec.Name}
		return it
	case "script":
		it.Manager, it.Pkg = "script", spec.URL
		if path := lookup(it.Bin, p); path != "" {
			it.Status, it.Found = Present, path
			return it
		}
		it.Status = Missing
		it.Cmd = []string{"/bin/bash", "-c", "NONINTERACTIVE=1 /bin/bash -c \"$(curl -fsSL " + spec.URL + ")\""}
		return it
	}

	if spec.Check != "" {
		it.Manager, it.Pkg = "script", "shell install"
		out, err := exec.Command("/bin/bash", "-c", spec.Check).Output()
		v := ""
		if m := verRe.FindStringSubmatch(strings.SplitN(string(out), "\n", 2)[0]); m != nil {
			v = strings.TrimPrefix(m[0], "v")
		}
		switch {
		case err == nil && (spec.Min == "" || (v != "" && !less(v, spec.Min))):
			it.Status, it.Found = Present, strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
			return it
		case err == nil:
			it.Status, it.Note = Outdated, "have "+v+", need >= "+spec.Min
		default:
			it.Status = Missing
		}
		it.Cmd = []string{"/bin/bash", "-c", spec.Run}
		return it
	}
	if path := lookup(it.Bin, p); path != "" {
		it.Found = path
		if spec.Min != "" {
			v := versionOf(path)
			it.Found = path + " " + v
			if v != "" && less(v, spec.Min) {
				it.Status = Outdated
				it.Note = "have " + v + ", need >= " + spec.Min
			} else {
				it.Status = Present
				return it
			}
		} else {
			it.Status = Present
			return it
		}
	} else {
		it.Status = Missing
	}

	// --- how to install ---
	brewName, brewCask, brewSkip, brewNote := spec.Brew.Resolve(name)
	native := p.Native()
	nativeName, nativeSkip, nativeNote := "", false, ""
	switch native {
	case "apt":
		nativeName, nativeSkip, nativeNote = spec.Apt.Resolve2(name)
	case "pacman":
		nativeName, nativeSkip, nativeNote = spec.Pacman.Resolve2(name)
	}

	switch {
	case p.OS == "darwin":
		if brewSkip {
			it.Status, it.Note = Unsupported, brewNote
			return it
		}
		if p.Brew == "" {
			it.Status, it.Manager, it.Pkg = NeedsBrew, "brew", brewName
			return it
		}
		it.Manager, it.Pkg = "brew", brewName
		if brewCask {
			it.Cmd = []string{p.Brew, "install", "--cask", brewName}
		} else {
			it.Cmd = []string{p.Brew, "install", brewName}
		}
	case p.OS == "linux":
		// prefer brew when present (matches .zshrc); else native; else brew-needed
		switch {
		case p.Apt && len(spec.Deb) > 0:
			url, ok := spec.Deb[p.Arch]
			if !ok {
				it.Status = Unsupported
				it.Note = "no official .deb for linux/" + p.Arch
				return it
			}
			it.Manager, it.Pkg = "deb", url
			sudo := ""
			if len(sudoPrefix(p)) > 0 {
				sudo = "sudo "
			}
			it.Cmd = []string{"/bin/bash", "-c",
				"set -e; t=$(mktemp)." + name + ".deb; curl -fsSL '" + url + "' -o \"$t\"; " + sudo + "apt-get install -y \"$t\"; rm -f \"$t\""}
			return it
		case p.Brew != "" && !brewSkip && !brewCask:
			it.Manager, it.Pkg = "brew", brewName
			it.Cmd = []string{p.Brew, "install", brewName}
		case native != "" && !nativeSkip:
			it.Manager, it.Pkg = native, nativeName
			it.Cmd = nativeInstall(p, native, nativeName)
		case !brewSkip && !brewCask:
			it.Status, it.Manager, it.Pkg = NeedsBrew, "brew", brewName
			if nativeNote != "" {
				it.Note = nativeNote
			}
		default:
			it.Status = Unsupported
			it.Note = firstNonEmpty(nativeNote, brewNote, "no install method on "+p.OS)
		}
	}
	return it
}

func sudoPrefix(p Platform) []string {
	if os.Geteuid() != 0 && p.Sudo {
		return []string{"sudo"}
	}
	return nil
}

func nativeInstall(p Platform, mgr, pkg string) []string {
	cmd := sudoPrefix(p)
	switch mgr {
	case "apt":
		return append(cmd, "apt-get", "install", "-y", "--no-install-recommends", pkg)
	case "pacman":
		return append(cmd, "pacman", "-S", "--noconfirm", "--needed", pkg)
	}
	return nil
}

// lookup finds bin on PATH, in the brew prefix, or as an absolute path.
func lookup(bin string, p Platform) string {
	if filepath.IsAbs(bin) {
		if fileExists(bin) {
			return bin
		}
		return ""
	}
	if path, err := exec.LookPath(bin); err == nil {
		return path
	}
	if p.BrewDir != "" && fileExists(filepath.Join(p.BrewDir, "bin", bin)) {
		return filepath.Join(p.BrewDir, "bin", bin)
	}
	return ""
}

var verRe = regexp.MustCompile(`v?(\d+)\.(\d+)(?:\.(\d+))?`)

func versionOf(bin string) string {
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return ""
	}
	first := strings.SplitN(string(out), "\n", 2)[0]
	if m := verRe.FindStringSubmatch(first); m != nil {
		return strings.TrimPrefix(m[0], "v")
	}
	return ""
}

func less(a, b string) bool {
	pa, pb := nums(a), nums(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

func nums(v string) [3]int {
	var out [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(strings.TrimSpace(s))
	}
	return out
}

// Install runs the commands for Missing/Outdated items in dependency order.
// Homebrew is bootstrapped first if any item needs it.
func Install(m *manifest.Manifest, p Platform, items []Item, dryRun bool, log io.Writer) error {
	needBrew := false
	for _, it := range items {
		if it.Status == NeedsBrew {
			needBrew = true
		}
	}
	if needBrew {
		if err := bootstrapBrew(m, p, dryRun, log); err != nil {
			return err
		}
		if !dryRun {
			p = Detect()
			names := make([]string, len(items))
			for i, it := range items {
				names[i] = it.Name
			}
			items = Resolve(m, p, names)
		}
	}
	done := map[string]bool{}
	for _, it := range items {
		if it.Status == Present {
			done[it.Name] = true
		}
	}
	var order []Item
	var visit func(it Item)
	byName := map[string]Item{}
	for _, it := range items {
		byName[it.Name] = it
	}
	seen := map[string]bool{}
	visit = func(it Item) {
		if seen[it.Name] {
			return
		}
		seen[it.Name] = true
		for _, n := range it.Needs {
			if d, ok := byName[n]; ok {
				visit(d)
			}
		}
		order = append(order, it)
	}
	for _, it := range items {
		visit(it)
	}
	// apt needs its package lists refreshed once on a fresh machine
	aptUpdated := false
	var failed []string
	for _, it := range order {
		if it.Status != Missing && it.Status != Outdated {
			continue
		}
		if (it.Manager == "apt" || it.Manager == "deb") && !aptUpdated {
			upd := sudoPrefix(p)
			upd = append(upd, "apt-get", "update")
			fmt.Fprintf(log, "→ apt: %s\n", strings.Join(upd, " "))
			if !dryRun {
				if err := run(upd, log); err != nil {
					return fmt.Errorf("apt-get update: %w", err)
				}
			}
			aptUpdated = true
		}
		for _, n := range it.Needs {
			if !done[n] {
				fmt.Fprintf(log, "  skip %s: needs %s\n", it.Name, n)
				continue
			}
		}
		fmt.Fprintf(log, "→ %s: %s\n", it.Name, strings.Join(it.Cmd, " "))
		if dryRun {
			done[it.Name] = true
			continue
		}
		cmd := exec.Command(it.Cmd[0], it.Cmd[1:]...)
		cmd.Stdout, cmd.Stderr, cmd.Stdin = log, log, os.Stdin
		cmd.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_ENV_HINTS=1")
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(log, "  FAILED %s: %v\n", it.Name, err)
			failed = append(failed, it.Name)
			continue
		}
		done[it.Name] = true
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to install: %s", strings.Join(failed, ", "))
	}
	return nil
}

func bootstrapBrew(m *manifest.Manifest, p Platform, dryRun bool, log io.Writer) error {
	spec := m.Deps.Pkg["homebrew"]
	if spec == nil {
		return fmt.Errorf("deps.pkg.homebrew missing from manifest")
	}
	var pre []string
	switch p.Native() {
	case "apt":
		pre = spec.AptPrereqs
	case "pacman":
		pre = spec.PacmanPrereqs
	}
	if len(pre) > 0 {
		cmd := nativeInstall(p, p.Native(), strings.Join(pre, " "))
		// nativeInstall takes one pkg; splice the list in
		cmd = append(cmd[:len(cmd)-1], pre...)
		if p.Native() == "apt" {
			upd := append(sudoPrefix(p), "apt-get", "update")
			fmt.Fprintf(log, "→ homebrew prerequisites: %s\n", strings.Join(upd, " "))
			if !dryRun {
				if err := run(upd, log); err != nil {
					return err
				}
			}
		}
		fmt.Fprintf(log, "→ homebrew prerequisites: %s\n", strings.Join(cmd, " "))
		if !dryRun {
			if err := run(cmd, log); err != nil {
				return err
			}
		}
	}
	it := resolveOne(m, p, "homebrew")
	if it.Status == Present {
		return nil
	}
	fmt.Fprintf(log, "→ homebrew: %s\n", strings.Join(it.Cmd, " "))
	if dryRun {
		return nil
	}
	return run(it.Cmd, log)
}

func run(argv []string, log io.Writer) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = log, log, os.Stdin
	return cmd.Run()
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

func expand(p, home string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}
