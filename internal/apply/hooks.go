package apply

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/edwinvillota/dotfiles/internal/ledger"
	"github.com/edwinvillota/dotfiles/internal/manifest"
)

// postInstall runs unit-specific fixups after files are in place.
// Currently: seed zellij's plugin-permission grants, because status-bar
// plugins (zjstatus) can never take focus, so their y/n permission prompt
// is unanswerable on a fresh machine.
func postInstall(m *manifest.Manifest, unitsTouched map[string]bool, led *ledger.Ledger, log io.Writer) error {
	if unitsTouched["zellij"] {
		if err := seedZellijPermissions(m, led, log); err != nil {
			return fmt.Errorf("zellij permissions: %w", err)
		}
	}
	return nil
}

// ZellijPermissionsPath is where zellij caches granted plugin permissions.
func ZellijPermissionsPath(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library/Caches/org.Zellij-Contributors.Zellij/permissions.kdl")
	}
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" || home != os.Getenv("HOME") {
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "zellij/permissions.kdl")
}

var zellijGrants = []string{"ReadApplicationState", "ChangeApplicationState", "RunCommands"}

func seedZellijPermissions(m *manifest.Manifest, led *ledger.Ledger, log io.Writer) error {
	u, ok := m.Units["zellij"]
	if !ok {
		return nil
	}
	plugDir := filepath.Join(m.DestPath(u), "plugins")
	wasms, _ := filepath.Glob(filepath.Join(plugDir, "*.wasm"))
	if len(wasms) == 0 {
		return nil
	}
	sort.Strings(wasms)
	dest := ZellijPermissionsPath(m.Home)
	existing, _ := os.ReadFile(dest)
	var sb strings.Builder
	sb.Write(existing)
	added := 0
	for _, w := range wasms {
		if strings.Contains(string(existing), `"`+w+`"`) {
			continue
		}
		fmt.Fprintf(&sb, "%q {\n", w)
		for _, g := range zellijGrants {
			fmt.Fprintf(&sb, "    %s\n", g)
		}
		sb.WriteString("}\n")
		added++
	}
	if added == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dest, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(log, "  grant    zellij plugin permissions for %d plugin(s) (%s)\n", added, dest)
	if led != nil && len(existing) == 0 {
		led.Add(ledger.Entry{Path: dest, Kind: ledger.Created, Unit: "zellij"})
	}
	return nil
}
