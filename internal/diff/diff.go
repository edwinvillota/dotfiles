// Package diff renders unified diffs between repo and live files.
package diff

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Unified returns a unified diff of a -> b (labels shown in the header).
// It prefers `git diff --no-index` (available wherever git is) and falls
// back to a built-in line diff. color enables ANSI output from git.
func Unified(aPath, bPath, aLabel, bLabel string, color bool) string {
	a, aErr := os.ReadFile(aPath)
	b, bErr := os.ReadFile(bPath)
	if aErr != nil && bErr != nil {
		return ""
	}
	if bytes.IndexByte(a, 0) >= 0 || bytes.IndexByte(b, 0) >= 0 {
		return fmt.Sprintf("Binary files differ (%d vs %d bytes)\n", len(a), len(b))
	}
	if git, err := exec.LookPath("git"); err == nil {
		args := []string{"diff", "--no-index", "--no-ext-diff"}
		if color {
			args = append(args, "--color=always")
		} else {
			args = append(args, "--color=never")
		}
		args = append(args, "--src-prefix="+aLabel+"/", "--dst-prefix="+bLabel+"/", "--", devnullIf(aErr, aPath), devnullIf(bErr, bPath))
		out, _ := exec.Command(git, args...).Output() // exit 1 == differences
		if len(out) > 0 {
			return clean(string(out), aLabel, bLabel)
		}
	}
	return simple(string(a), string(b), aLabel, bLabel)
}

func devnullIf(err error, p string) string {
	if err != nil {
		return os.DevNull
	}
	return p
}

// simple is a minimal LCS-based unified diff, no context trimming.
func simple(a, b, al, bl string) string {
	x, y := strings.Split(a, "\n"), strings.Split(b, "\n")
	n, m := len(x), len(y)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", al, bl)
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && x[i] == y[j]:
			sb.WriteString(" " + x[i] + "\n")
			i++
			j++
		case j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]):
			sb.WriteString("+" + y[j] + "\n")
			j++
		default:
			sb.WriteString("-" + x[i] + "\n")
			i++
		}
	}
	return sb.String()
}

// clean drops git's "diff --git"/"index" header lines and replaces the
// absolute-path ---/+++ lines with the given labels.
func clean(out, al, bl string) string {
	var keep []string
	for _, l := range strings.Split(out, "\n") {
		plain := stripSGR(l)
		switch {
		case strings.HasPrefix(plain, "diff --git"), strings.HasPrefix(plain, "index "),
			strings.HasPrefix(plain, "new file mode"), strings.HasPrefix(plain, "deleted file mode"):
			continue
		case strings.HasPrefix(plain, "--- "):
			keep = append(keep, "--- "+al)
		case strings.HasPrefix(plain, "+++ "):
			keep = append(keep, "+++ "+bl)
		default:
			keep = append(keep, l)
		}
	}
	return strings.Join(keep, "\n")
}

func stripSGR(s string) string {
	var sb strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			in = true
		case in && r == 'm':
			in = false
		case !in:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
