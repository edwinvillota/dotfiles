package theme

import (
	"fmt"
	"regexp"
	"testing"
)

var optRe = regexp.MustCompile(`vd\.options\.(\w+) = "([^"]*)"`)

func vdOpts(t *testing.T, name string) map[string]string {
	t.Helper()
	p, err := Load(name)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	out := map[string]string{}
	for _, m := range optRe.FindAllStringSubmatch(VisiData(p), -1) {
		out[m[1]] = m[2]
	}
	return out
}

// bgIdx pulls the "on N" background out of a VisiData color spec.
func bgIdx(spec string) (int, bool) {
	m := regexp.MustCompile(`on (\d+)`).FindStringSubmatch(spec)
	if m == nil {
		return 0, false
	}
	var n int
	fmt.Sscanf(m[1], "%d", &n)
	return n, true
}

// The cursor row and cursor column have to be findable at a glance against the
// sheet background, and the cell where they cross has to be distinguishable
// from both -- otherwise you cannot tell which cell you are on.
func TestVisiDataCursorIsVisible(t *testing.T) {
	const (
		minRowVsBody = 1.70 // cursor row vs the sheet body
		minColVsBody = 1.55 // cursor column vs the sheet body
		minCellVsRow = 1.45 // the cursor cell vs the rest of its row
	)
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			o := vdOpts(t, name)
			body, ok := bgIdx(o["color_default"])
			if !ok {
				t.Fatalf("color_default has no background: %q", o["color_default"])
			}

			row, ok := bgIdx(o["color_current_row"])
			if !ok {
				t.Fatalf("color_current_row has no background: %q", o["color_current_row"])
			}
			col, ok := bgIdx(o["color_current_col"])
			if !ok {
				t.Fatalf("color_current_col has no background: %q", o["color_current_col"])
			}
			cell, ok := bgIdx(o["color_current_cell"])
			if !ok {
				t.Fatalf("color_current_cell has no background: %q", o["color_current_cell"])
			}

			if c := contrastIdx(body, row); c < minRowVsBody {
				t.Errorf("cursor row (%d) vs body (%d): contrast %.2f < %.2f", row, body, c, minRowVsBody)
			}
			if c := contrastIdx(body, col); c < minColVsBody {
				t.Errorf("cursor col (%d) vs body (%d): contrast %.2f < %.2f", col, body, c, minColVsBody)
			}
			if c := contrastIdx(row, cell); c < minCellVsRow {
				t.Errorf("cursor cell (%d) vs cursor row (%d): contrast %.2f < %.2f", cell, row, c, minCellVsRow)
			}
			if cell == col {
				t.Errorf("cursor cell and cursor column share background %d", cell)
			}
		})
	}
}

// Whatever background the cursor paints, the text on it still has to be
// readable -- a cursor row you cannot read is as bad as one you cannot find.
func TestVisiDataCursorTextIsReadable(t *testing.T) {
	const minText = 4.0
	fgRe := regexp.MustCompile(`(?:^|\s)(\d+)(?:\s+on\s+\d+)?$`)
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			o := vdOpts(t, name)
			for _, key := range []string{"color_current_row", "color_current_col", "color_current_cell"} {
				spec := o[key]
				bg, ok := bgIdx(spec)
				if !ok {
					t.Fatalf("%s has no background: %q", key, spec)
				}
				m := fgRe.FindStringSubmatch(regexp.MustCompile(`\s*on \d+$`).ReplaceAllString(spec, ""))
				if m == nil {
					t.Fatalf("%s has no explicit foreground: %q", key, spec)
				}
				var fg int
				fmt.Sscanf(m[1], "%d", &fg)
				if c := contrastIdx(fg, bg); c < minText {
					t.Errorf("%s: text %d on %d has contrast %.2f < %.2f", key, fg, bg, c, minText)
				}
			}
		})
	}
}

// De-emphasized text is meant to be quiet, not invisible. Without a floor,
// palettes whose dim role sits near their panel colour render inactive status
// and unfocused edits as near-blank (Nord measured 1.37 before this).
func TestVisiDataDimIsLegible(t *testing.T) {
	const minDim = 3.0
	dimmed := []string{
		"color_inactive_status", "color_edit_unfocused", "color_guide_unwritten",
		"color_hidden_col", "color_longname_status", "color_longname_guide",
	}
	fgRe := regexp.MustCompile(`(\d+)\s*$`)
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			o := vdOpts(t, name)
			body, _ := bgIdx(o["color_default"])
			for _, key := range dimmed {
				spec := o[key]
				bg, ok := bgIdx(spec)
				if !ok {
					bg = body // no explicit background: drawn on the sheet body
				}
				m := fgRe.FindStringSubmatch(regexp.MustCompile(`\s*on \d+$`).ReplaceAllString(spec, ""))
				if m == nil {
					t.Fatalf("%s has no foreground: %q", key, spec)
				}
				var fg int
				fmt.Sscanf(m[1], "%d", &fg)
				if c := contrastIdx(fg, bg); c < minDim {
					t.Errorf("%s: %d on %d has contrast %.2f < %.2f", key, fg, bg, c, minDim)
				}
			}
		})
	}
}

// Marks that set only a foreground -- key columns, selected rows, hidden-column
// arrows, row notes -- inherit whatever background they land on, including the
// cursor row. Palettes with a coloured selection rather than a neutral step
// (GitHub Dark's is a deep blue, Iceberg's a slate) drop these out of
// legibility there even when they read fine on the sheet body.
func TestVisiDataMarksReadableOnCursorRow(t *testing.T) {
	const minOnCursor = 3.0
	marks := []string{"color_key_col", "color_selected_row", "color_hidden_col", "color_note_row"}
	fgRe := regexp.MustCompile(`(\d+)\s*$`)
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			o := vdOpts(t, name)
			rowBg, ok := bgIdx(o["color_current_row"])
			if !ok {
				t.Fatalf("color_current_row has no background: %q", o["color_current_row"])
			}
			for _, key := range marks {
				m := fgRe.FindStringSubmatch(o[key])
				if m == nil {
					t.Fatalf("%s has no foreground: %q", key, o[key])
				}
				var fg int
				fmt.Sscanf(m[1], "%d", &fg)
				if c := contrastIdx(fg, rowBg); c < minOnCursor {
					t.Errorf("%s: %d on cursor row %d has contrast %.2f < %.2f", key, fg, rowBg, c, minOnCursor)
				}
			}
		})
	}
}
