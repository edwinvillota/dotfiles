package theme

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDumpVisiDataThemes writes every rendered theme to $VD_DUMP_DIR when set,
// so external tooling (VisiData itself) can validate the output. No-op
// otherwise, so it stays out of the way in normal runs.
func TestDumpVisiDataThemes(t *testing.T) {
	dir := os.Getenv("VD_DUMP_DIR")
	if dir == "" {
		t.Skip("VD_DUMP_DIR not set")
	}
	for _, name := range Names() {
		p, err := Load(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".py"), []byte(VisiData(p)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
