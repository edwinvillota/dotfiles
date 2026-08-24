// Package check guards against committing secrets.
package check

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/edwinvillota/dotfiles/internal/manifest"
)

type Finding struct {
	File, Reason string
	Line         int
}

func (f Finding) String() string {
	if f.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", f.File, f.Line, f.Reason)
	}
	return fmt.Sprintf("%s: %s", f.File, f.Reason)
}

// Run inspects the files git would commit (tracked + staged) under the repo.
// If files is non-empty it is used instead of asking git.
func Run(m *manifest.Manifest, files []string) ([]Finding, error) {
	if files == nil {
		var err error
		if files, err = gitFiles(m.Root); err != nil {
			return nil, err
		}
	}
	var out []Finding
	for _, rel := range files {
		if !inUnit(m, rel) {
			continue // tool source, docs, tests: not config
		}
		abs := filepath.Join(m.Root, rel)
		if reason := forbiddenPath(m, rel); reason != "" {
			out = append(out, Finding{File: rel, Reason: reason})
			continue
		}
		fs, err := scanFile(m, abs, rel)
		if err != nil {
			continue
		}
		out = append(out, fs...)
	}
	return out, nil
}

// forbiddenPath reports why a repo-relative path must never be committed.
func forbiddenPath(m *manifest.Manifest, rel string) string {
	for _, u := range m.Units {
		src := filepath.ToSlash(u.Src)
		if rel != src && !strings.HasPrefix(rel, src+"/") {
			continue
		}
		urel := strings.TrimPrefix(strings.TrimPrefix(rel, src), "/")
		if urel == "" {
			continue
		}
		if u.IsSecret(urel) {
			return "secret file (only its .template may be committed)"
		}
		if len(u.Only) > 0 && !manifest.Match(u.Only, urel) {
			return "not in unit allowlist"
		}
	}
	return ""
}

var maybeBinary = regexp.MustCompile(`\.(wasm|png|jpg|gif|spl|zwc|pack|idx)$`)

func scanFile(m *manifest.Manifest, abs, rel string) ([]Finding, error) {
	if maybeBinary.MatchString(rel) {
		return nil, nil
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return nil, nil
	}
	var out []Finding
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if strings.Contains(line, "# public") || strings.Contains(line, "dotfiles:allow") {
			continue
		}
		for _, re := range m.SecretPatterns() {
			if re.MatchString(line) {
				out = append(out, Finding{File: rel, Line: n, Reason: "matches secret pattern " + re.String()})
				break
			}
		}
	}
	return out, nil
}

func gitFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "-C", root, "ls-files", "--cached", "--others", "--exclude-standard")
	b, err := cmd.Output()
	if err != nil {
		// not a git checkout (e.g. inside the test container): walk instead
		return walkFiles(root)
	}
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

// InstallHook writes a pre-commit hook that runs `dotfiles check`.
func InstallHook(root string) (string, error) {
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	body := "#!/bin/sh\n# installed by `dotfiles hook`\nexec \"$(git rev-parse --show-toplevel)/bin/dotfiles\" check --quiet\n"
	if err := os.WriteFile(hook, []byte(body), 0o755); err != nil {
		return "", err
	}
	return hook, nil
}

func walkFiles(root string) ([]string, error) {
	var out []string
	skip := map[string]bool{".git": true, "bin": true, "dist": true}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if d.IsDir() {
			if skip[rel] {
				return filepath.SkipDir
			}
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

func inUnit(m *manifest.Manifest, rel string) bool {
	for _, u := range m.Units {
		src := filepath.ToSlash(u.Src)
		if rel == src || strings.HasPrefix(rel, src+"/") {
			return true
		}
	}
	return false
}
