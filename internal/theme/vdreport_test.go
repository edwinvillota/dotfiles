package theme

import (
	"regexp"
	"strconv"
	"testing"
)

func atoiTest(s string) int { n, _ := strconv.Atoi(s); return n }

// TestVisiDataCursorReport is a diagnostic, not an assertion: run with -v to
// see what each theme actually paints for the cursor.
func TestVisiDataCursorReport(t *testing.T) {
	bgOf := regexp.MustCompile(`on (\d+)`)
	t.Logf("%-24s %-6s %-22s %-22s %-22s", "theme", "body", "current_row", "current_col", "current_cell")
	for _, name := range Names() {
		o := vdOpts(t, name)
		var body int
		if m := bgOf.FindStringSubmatch(o["color_default"]); m != nil {
			body = atoiTest(m[1])
		}
		row, _ := bgIdx(o["color_current_row"])
		col, _ := bgIdx(o["color_current_col"])
		cell, _ := bgIdx(o["color_current_cell"])
		t.Logf("%-24s %-6d %-22s %-22s %-22s  | row/body %.2f  col/body %.2f  cell/row %.2f",
			name, body, o["color_current_row"], o["color_current_col"], o["color_current_cell"],
			contrastIdx(body, row), contrastIdx(body, col), contrastIdx(row, cell))
	}
}
