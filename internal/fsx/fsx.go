// Package fsx: filesystem snapshotting and safe writes.
package fsx

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Entry describes one file inside a tree.
type Entry struct {
	Rel     string // forward-slash relative path
	Size    int64
	Mode    fs.FileMode
	Symlink string // target if symlink
	hash    string
	abs     string
}

func (e *Entry) Abs() string { return e.abs }

// Hash lazily computes the sha256 of the file contents.
func (e *Entry) Hash() string {
	if e.hash != "" || e.Symlink != "" {
		return e.hash
	}
	f, err := os.Open(e.abs)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	e.hash = hex.EncodeToString(h.Sum(nil))
	return e.hash
}

// Snapshot walks root (a file or directory). For a single file the Rel is "".
// skip decides whether a relative path (and its subtree) is excluded.
func Snapshot(root string, skip func(rel string, isDir bool) bool) (map[string]*Entry, error) {
	out := map[string]*Entry{}
	fi, err := os.Lstat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	if !fi.IsDir() {
		e := &Entry{Rel: "", Size: fi.Size(), Mode: fi.Mode(), abs: root}
		if fi.Mode()&fs.ModeSymlink != 0 {
			e.Symlink, _ = os.Readlink(root)
		}
		out[""] = e
		return out, nil
	}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if skip != nil && skip(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		e := &Entry{Rel: rel, Size: info.Size(), Mode: info.Mode(), abs: p}
		if info.Mode()&fs.ModeSymlink != 0 {
			e.Symlink, _ = os.Readlink(p)
		}
		out[rel] = e
		return nil
	})
	return out, err
}

// SortedKeys returns map keys sorted.
func SortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// CopyFile copies src to dst atomically (temp + rename), preserving mode.
func CopyFile(src, dst string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		os.Remove(dst)
		return os.Symlink(target, dst)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".dotfiles-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, fi.Mode().Perm()); err != nil {
		os.Remove(tmpName)
		return err
	}
	// rename over a symlink replaces the link itself, which is what we want
	return os.Rename(tmpName, dst)
}

// Symlink creates dst -> src, replacing any existing file/link at dst.
func Symlink(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	os.Remove(dst)
	return os.Symlink(src, dst)
}

// RemoveEmptyParents deletes empty parent dirs of p up to (not including) stop.
func RemoveEmptyParents(p, stop string) {
	for d := filepath.Dir(p); d != stop && len(d) > len(stop); d = filepath.Dir(d) {
		if err := os.Remove(d); err != nil {
			return
		}
	}
}
