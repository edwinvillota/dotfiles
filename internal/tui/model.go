// Package tui is the interactive Bubble Tea front end.
package tui

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/edwinvillota/dotfiles/internal/apply"
	"github.com/edwinvillota/dotfiles/internal/check"
	"github.com/edwinvillota/dotfiles/internal/diff"
	"github.com/edwinvillota/dotfiles/internal/manifest"
	"github.com/edwinvillota/dotfiles/internal/plan"
	"github.com/edwinvillota/dotfiles/internal/state"
)

type pane int

const (
	paneTree pane = iota
	panePreview
)

type mode int

const (
	modeNormal mode = iota
	modeFilter
	modeHelp
	modeConfirm
	modeResult
)

// row is one line of the tree.
type row struct {
	key      string // selection key: "unit" or "unit/rel"
	unit     string
	label    string
	isUnit   bool
	depth    int
	create   int
	update   int
	delete   int
	skipped  int
	secret   bool
	children int
}

type Model struct {
	m     *manifest.Manifest
	st    *state.State
	dir   plan.Direction
	plan  *plan.Plan
	prof  string
	err   error
	width int
	heigh int

	rows     []row // visible rows (after filter/collapse)
	all      []row // all rows
	cursor   int
	offset   int
	expanded map[string]bool
	filter   textinput.Model
	pane     pane
	mode     mode
	showDiff bool
	preview  viewport.Model
	pendingG bool

	result   string
	resultOK bool
	confirm  string
}

func New(m *manifest.Manifest, st *state.State) *Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.PromptStyle = sGold
	ti.TextStyle = lipgloss.NewStyle().Foreground(cFg)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(cGold)
	md := &Model{m: m, st: st, dir: plan.Backup, prof: st.Profile, expanded: map[string]bool{}, filter: ti}
	md.preview = viewport.New(0, 0)
	md.rebuild()
	return md
}

func (md *Model) profile() *manifest.Profile {
	if md.prof == "" {
		return nil
	}
	return md.m.Profiles[md.prof]
}

// rebuild recomputes the plan and the tree from current settings.
func (md *Model) rebuild() {
	full, err := plan.Build(md.m, plan.Options{Direction: md.dir, Profile: md.profile(), Symlink: md.st.Symlink && md.dir == plan.Install})
	if err != nil {
		md.err = err
		return
	}
	sel := md.st.Selected(full.Keys())
	p, err := plan.Build(md.m, plan.Options{Direction: md.dir, Profile: md.profile(), Symlink: md.st.Symlink && md.dir == plan.Install, Selected: sel})
	if err != nil {
		md.err = err
		return
	}
	md.plan = p
	md.err = nil

	units := map[string]*row{}
	granules := map[string]*row{}
	var order []string
	for _, a := range p.Actions {
		u, ok := units[a.Unit]
		if !ok {
			u = &row{key: a.Unit, unit: a.Unit, label: a.Unit, isUnit: true}
			units[a.Unit] = u
			order = append(order, a.Unit)
		}
		target := u
		if a.Granule != "" {
			g, ok := granules[a.Granule]
			if !ok {
				g = &row{key: a.Granule, unit: a.Unit, label: a.Rel, depth: 1}
				granules[a.Granule] = g
				u.children++
			}
			target = g
		}
		bump := func(r *row) {
			switch a.Op {
			case plan.OpCreate, plan.OpLink:
				r.create++
			case plan.OpUpdate:
				r.update++
			case plan.OpDelete:
				r.delete++
			case plan.OpSkip:
				r.skipped++
			}
			if a.Redact {
				r.secret = true
			}
		}
		bump(target)
		if target != u {
			bump(u)
		}
	}
	// units with no actions at all (e.g. everything ignored) still appear
	for _, n := range md.m.UnitNames() {
		if _, ok := units[n]; !ok {
			units[n] = &row{key: n, unit: n, label: n, isUnit: true}
			order = append(order, n)
		}
	}
	sort.Strings(order)
	md.all = md.all[:0]
	for _, n := range order {
		md.all = append(md.all, *units[n])
		var kids []row
		for k, g := range granules {
			if g.unit == n {
				_ = k
				kids = append(kids, *g)
			}
		}
		sort.Slice(kids, func(i, j int) bool { return kids[i].label < kids[j].label })
		md.all = append(md.all, kids...)
	}
	md.applyFilter()
}

func (md *Model) applyFilter() {
	q := strings.ToLower(md.filter.Value())
	md.rows = md.rows[:0]
	for _, r := range md.all {
		if r.depth > 0 && !md.expanded[r.unit] && q == "" {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(r.key), q) {
			continue
		}
		md.rows = append(md.rows, r)
	}
	if md.cursor >= len(md.rows) {
		md.cursor = max(0, len(md.rows)-1)
	}
	md.updatePreview()
}

func (md *Model) enabled(r row) bool {
	if md.st.IsDisabled(r.key) {
		return false
	}
	if !r.isUnit && md.st.IsDisabled(r.unit) {
		return false
	}
	return true
}

func (md *Model) current() *row {
	if len(md.rows) == 0 {
		return nil
	}
	return &md.rows[md.cursor]
}

// ---- preview ----------------------------------------------------------------

func (md *Model) updatePreview() {
	r := md.current()
	if r == nil || md.plan == nil {
		md.preview.SetContent(sDim.Render("nothing selected"))
		return
	}
	var sb strings.Builder
	u := md.m.Units[r.unit]
	fmt.Fprintf(&sb, "%s  %s\n", sTitle.Render(r.key), sDim.Render(arrow(md.dir, md.m, u)))
	if !md.enabled(*r) {
		sb.WriteString(sOrange.Render("disabled — press Space to enable") + "\n")
	}
	sb.WriteString("\n")
	n := 0
	for _, a := range md.plan.Actions {
		if a.Unit != r.unit {
			continue
		}
		if !r.isUnit && a.Granule != r.key {
			continue
		}
		if r.isUnit && a.Granule != "" && md.expanded[r.unit] {
			continue // shown under the granule rows
		}
		if a.Op == plan.OpNone {
			continue
		}
		n++
		rel := a.Rel
		if rel == "" {
			rel = filepath.Base(a.To)
		}
		line := opStyle(a.Op).Render(fmt.Sprintf("%-4s", a.Op.String())) + " " + rel
		if a.Op == plan.OpSkip {
			line += sDim.Render("  " + a.Reason)
		}
		if a.Redact && a.Op != plan.OpSkip {
			line += sGold.Render("  [secret → template]")
		}
		if a.Backup && a.Op != plan.OpSkip {
			line += sDim.Render("  [original preserved]")
		}
		sb.WriteString(line + "\n")
		if md.showDiff && a.Op == plan.OpUpdate && !a.Redact {
			d := diff.Unified(a.To, a.From, "target", "source", true)
			if md.dir == plan.Backup {
				d = diff.Unified(a.To, a.From, "repo", "live", true)
			} else {
				d = diff.Unified(a.To, a.From, "live", "repo", true)
			}
			sb.WriteString(indent(d, "    ") + "\n")
		}
	}
	if n == 0 {
		sb.WriteString(sGreen.Render("✓ in sync") + "\n")
	}
	md.preview.SetContent(sb.String())
	md.preview.GotoTop()
}

func arrow(d plan.Direction, m *manifest.Manifest, u *manifest.Unit) string {
	if u == nil {
		return ""
	}
	src, dst := tilde(m.DestPath(u), m.Home), u.Src
	if d == plan.Install {
		src, dst = u.Src, tilde(m.DestPath(u), m.Home)
	}
	return src + " → " + dst
}

func tilde(p, home string) string {
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func indent(s, pad string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

func opStyle(o plan.Op) lipgloss.Style {
	switch o {
	case plan.OpCreate, plan.OpLink:
		return sGreen
	case plan.OpUpdate:
		return sBlue
	case plan.OpDelete:
		return sRed
	case plan.OpSkip:
		return sDim
	}
	return sDim
}

// ---- tea.Model ---------------------------------------------------------------

func (md *Model) Init() tea.Cmd { return nil }

type applyDone struct {
	out string
	err error
}

func (md *Model) runApply() tea.Cmd {
	m, p, dir := md.m, md.plan, md.dir
	return func() tea.Msg {
		var buf bytes.Buffer
		if dir == plan.Backup {
			if fs, _ := check.Run(m, nil); len(fs) > 0 {
				for _, f := range fs {
					fmt.Fprintln(&buf, "  ", f)
				}
				return applyDone{buf.String(), fmt.Errorf("repo already contains secret-looking content; fix before backing up")}
			}
		}
		res, err := apply.Run(m, p, apply.Options{Confirm: func(string) bool { return true }, Log: &buf})
		if err != nil {
			return applyDone{buf.String(), err}
		}
		fmt.Fprintf(&buf, "\ndone: %d written, %d linked, %d deleted\n", res.Written, res.Linked, res.Deleted)
		for _, n := range res.Notices {
			fmt.Fprintln(&buf, "NOTE:", n)
		}
		if dir == plan.Backup {
			if fs, _ := check.Run(m, nil); len(fs) > 0 {
				fmt.Fprintln(&buf, "\nWARNING: secret-looking content found — do NOT commit until fixed:")
				for _, f := range fs {
					fmt.Fprintln(&buf, "  ", f)
				}
				return applyDone{buf.String(), fmt.Errorf("secret check failed")}
			}
			fmt.Fprintln(&buf, "\nsecret check: ok — review with `git diff`, then commit")
		}
		return applyDone{buf.String(), nil}
	}
}

func (md *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		md.width, md.heigh = msg.Width, msg.Height
		md.layout()
		return md, nil
	case applyDone:
		md.result, md.resultOK = msg.out, msg.err == nil
		if msg.err != nil {
			md.result += "\n" + sRed.Render("error: "+msg.err.Error())
		}
		md.mode = modeResult
		md.rebuild()
		return md, nil
	case tea.KeyMsg:
		return md.key(msg)
	}
	return md, nil
}

func (md *Model) layout() {
	md.preview.Width = md.previewWidth() - 4
	md.preview.Height = md.heigh - 5
}

func (md *Model) treeWidth() int    { return max(30, md.width*2/5) }
func (md *Model) previewWidth() int { return md.width - md.treeWidth() }

func (md *Model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := k.String()
	switch md.mode {
	case modeFilter:
		switch s {
		case "esc":
			md.filter.SetValue("")
			md.filter.Blur()
			md.mode = modeNormal
			md.applyFilter()
		case "enter":
			md.filter.Blur()
			md.mode = modeNormal
		default:
			var cmd tea.Cmd
			md.filter, cmd = md.filter.Update(k)
			md.cursor = 0
			md.applyFilter()
			return md, cmd
		}
		return md, nil
	case modeHelp, modeResult:
		if s == "q" || s == "esc" || s == "?" || s == "enter" {
			md.mode = modeNormal
		} else {
			var cmd tea.Cmd
			md.preview, cmd = md.preview.Update(k)
			return md, cmd
		}
		return md, nil
	case modeConfirm:
		switch s {
		case "y", "Y", "enter":
			md.mode = modeNormal
			md.result = "working…"
			return md, md.runApply()
		default:
			md.mode = modeNormal
		}
		return md, nil
	}

	// normal mode
	if md.pendingG {
		md.pendingG = false
		if s == "g" {
			md.cursor = 0
			md.updatePreview()
			return md, nil
		}
	}
	if md.pane == panePreview {
		switch s {
		case "j", "down":
			md.preview.LineDown(1)
			return md, nil
		case "k", "up":
			md.preview.LineUp(1)
			return md, nil
		case "ctrl+d", "d":
			if s == "ctrl+d" {
				md.preview.HalfViewDown()
				return md, nil
			}
		case "ctrl+u":
			md.preview.HalfViewUp()
			return md, nil
		case "g":
			md.preview.GotoTop()
			return md, nil
		case "G":
			md.preview.GotoBottom()
			return md, nil
		}
	}
	switch s {
	case "q", "ctrl+c":
		return md, tea.Quit
	case "?":
		md.mode = modeHelp
	case "tab":
		md.pane = (md.pane + 1) % 2
	case "/":
		md.mode = modeFilter
		md.filter.Focus()
		return md, textinput.Blink
	case "esc":
		if md.filter.Value() != "" {
			md.filter.SetValue("")
			md.applyFilter()
		}
	case "j", "down":
		if md.cursor < len(md.rows)-1 {
			md.cursor++
		}
		md.updatePreview()
	case "k", "up":
		if md.cursor > 0 {
			md.cursor--
		}
		md.updatePreview()
	case "g":
		md.pendingG = true
	case "G":
		md.cursor = max(0, len(md.rows)-1)
		md.updatePreview()
	case "ctrl+d":
		md.cursor = min(len(md.rows)-1, md.cursor+md.pageSize()/2)
		md.updatePreview()
	case "ctrl+u":
		md.cursor = max(0, md.cursor-md.pageSize()/2)
		md.updatePreview()
	case "l", "right", "enter":
		if r := md.current(); r != nil && r.isUnit && r.children > 0 {
			md.expanded[r.unit] = true
			md.applyFilter()
		}
	case "h", "left":
		if r := md.current(); r != nil {
			if !r.isUnit {
				md.expanded[r.unit] = false
				md.applyFilter()
				for i, x := range md.rows {
					if x.key == r.unit {
						md.cursor = i
					}
				}
				md.updatePreview()
			} else if md.expanded[r.unit] {
				md.expanded[r.unit] = false
				md.applyFilter()
			}
		}
	case " ":
		if r := md.current(); r != nil {
			md.st.Set(r.key, !md.enabled(*r) || (!r.isUnit && md.st.IsDisabled(r.unit) && !md.st.IsDisabled(r.key)))
			if !r.isUnit && md.st.IsDisabled(r.unit) {
				// enabling a child of a disabled unit enables the unit
				md.st.Set(r.unit, true)
			}
			md.st.Save()
			md.rebuild()
		}
	case "a":
		anyDisabled := len(md.st.Disabled) > 0
		md.st.Disabled = nil
		if !anyDisabled {
			for _, r := range md.all {
				if r.isUnit {
					md.st.Set(r.key, false)
				}
			}
		}
		md.st.Save()
		md.rebuild()
	case "m":
		if md.dir == plan.Backup {
			md.dir = plan.Install
		} else {
			md.dir = plan.Backup
		}
		md.rebuild()
	case "p":
		names := append([]string{""}, sortedKeys(md.m.Profiles)...)
		for i, n := range names {
			if n == md.prof {
				md.prof = names[(i+1)%len(names)]
				break
			}
		}
		md.st.Profile = md.prof
		md.st.Save()
		md.rebuild()
	case "s":
		md.st.Symlink = !md.st.Symlink
		md.st.Save()
		md.rebuild()
	case "d":
		md.showDiff = !md.showDiff
		md.updatePreview()
	case "r":
		md.rebuild()
	case "b", "i":
		if s == "b" {
			md.dir = plan.Backup
		} else {
			md.dir = plan.Install
		}
		md.rebuild()
		md.askConfirm()
	case "x":
		md.askConfirm()
	}
	return md, nil
}

func (md *Model) askConfirm() {
	if md.plan == nil {
		return
	}
	ch := md.plan.Changes()
	if len(ch) == 0 {
		md.result, md.resultOK, md.mode = sGreen.Render("nothing to do — everything is in sync"), true, modeResult
		return
	}
	cr, up, de, li, _ := md.plan.Counts()
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s\n\n", sTitle.Render(strings.ToUpper(md.dir.String())))
	fmt.Fprintf(&sb, "%s create   %s update   %s delete   %s link\n\n",
		sGreen.Render(fmt.Sprint(cr)), sBlue.Render(fmt.Sprint(up)), sRed.Render(fmt.Sprint(de)), sGreen.Render(fmt.Sprint(li)))
	if de > 0 {
		sb.WriteString(sRed.Render("Deletions:") + "\n")
		n := 0
		for _, a := range ch {
			if a.Op == plan.OpDelete {
				n++
				if n <= 12 {
					sb.WriteString("  - " + a.Unit + "/" + a.Rel + "\n")
				}
			}
		}
		if n > 12 {
			fmt.Fprintf(&sb, "  … and %d more\n", n-12)
		}
		if md.dir == plan.Install {
			sb.WriteString(sDim.Render("(originals are preserved and restorable with `dotfiles uninstall`)") + "\n")
		}
		sb.WriteString("\n")
	}
	if md.dir == plan.Install && md.st.Symlink {
		sb.WriteString(sGold.Render("symlink mode") + "\n\n")
	}
	sb.WriteString("Proceed? " + sKey.Render("y") + " / " + sKey.Render("n"))
	md.confirm = sb.String()
	md.mode = modeConfirm
}

func (md *Model) pageSize() int { return max(1, md.heigh-6) }

func sortedKeys[V any](m map[string]V) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---- view -------------------------------------------------------------------

func (md *Model) View() string {
	if md.width == 0 {
		return "loading…"
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, md.viewTree(), md.viewPreview())
	out := lipgloss.JoinVertical(lipgloss.Left, md.viewHeader(), body, md.viewStatus())
	switch md.mode {
	case modeHelp:
		return md.overlay(out, md.helpText())
	case modeConfirm:
		return md.overlay(out, md.confirm)
	case modeResult:
		return md.overlay(out, md.resultText())
	}
	return sBase.Render(out)
}

func (md *Model) viewHeader() string {
	dir := sTag.Render(strings.ToUpper(md.dir.String()))
	if md.dir == plan.Install {
		dir = sTagWarn.Render("INSTALL")
	}
	prof := md.prof
	if prof == "" {
		prof = "none"
	}
	parts := []string{sTitle.Render(" dotfiles"), dir, sDim.Render("profile:") + " " + sBlue.Render(prof)}
	if md.st.Symlink && md.dir == plan.Install {
		parts = append(parts, sGold.Render("symlink"))
	}
	if md.plan != nil {
		cr, up, de, li, sk := md.plan.Counts()
		parts = append(parts, fmt.Sprintf("%s %s %s %s %s",
			sGreen.Render(fmt.Sprintf("+%d", cr+li)), sBlue.Render(fmt.Sprintf("~%d", up)),
			sRed.Render(fmt.Sprintf("-%d", de)), sDim.Render(fmt.Sprintf("skip %d", sk)), ""))
	}
	if md.err != nil {
		parts = append(parts, sRed.Render(md.err.Error()))
	}
	return lipgloss.NewStyle().Width(md.width).Render(strings.Join(parts, "  "))
}

func (md *Model) viewTree() string {
	h := md.heigh - 5
	if md.cursor < md.offset {
		md.offset = md.cursor
	}
	if md.cursor >= md.offset+h {
		md.offset = md.cursor - h + 1
	}
	w := md.treeWidth() - 4
	var lines []string
	if md.mode == modeFilter || md.filter.Value() != "" {
		lines = append(lines, md.filter.View())
		h--
	}
	for i := md.offset; i < len(md.rows) && i < md.offset+h; i++ {
		r := md.rows[i]
		box := sGreen.Render("●")
		if !md.enabled(r) {
			box = sDim.Render("○")
		}
		ind := "  "
		if r.isUnit && r.children > 0 {
			if md.expanded[r.unit] {
				ind = sDim.Render("▾ ")
			} else {
				ind = sDim.Render("▸ ")
			}
		}
		name := r.label
		if r.depth > 0 {
			name = "    " + name
		}
		var badges []string
		if r.create > 0 {
			badges = append(badges, sGreen.Render(fmt.Sprintf("+%d", r.create)))
		}
		if r.update > 0 {
			badges = append(badges, sBlue.Render(fmt.Sprintf("~%d", r.update)))
		}
		if r.delete > 0 {
			badges = append(badges, sRed.Render(fmt.Sprintf("-%d", r.delete)))
		}
		if r.secret {
			badges = append(badges, sGold.Render("🔒"))
		}
		if len(badges) == 0 && md.enabled(r) {
			badges = append(badges, sDim.Render("✓"))
		}
		right := strings.Join(badges, " ")
		left := ind + box + " " + name
		pad := w - lipgloss.Width(left) - lipgloss.Width(right)
		if pad < 1 {
			pad = 1
		}
		line := left + strings.Repeat(" ", pad) + right
		if i == md.cursor && md.pane == paneTree {
			line = sCursor.Render(lipgloss.NewStyle().Width(w).Render(line))
		}
		lines = append(lines, line)
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	st := sPane
	if md.pane == paneTree {
		st = sPaneAct
	}
	return st.Width(md.treeWidth() - 2).Height(md.heigh - 4).Render(strings.Join(lines, "\n"))
}

func (md *Model) viewPreview() string {
	st := sPane
	if md.pane == panePreview {
		st = sPaneAct
	}
	return st.Width(md.previewWidth() - 2).Height(md.heigh - 4).Render(md.preview.View())
}

func (md *Model) viewStatus() string {
	keys := []string{"j/k", "move", "Space", "toggle", "/", "filter", "d", "diff", "m", "backup⇄install", "p", "profile", "b", "backup", "i", "install", "?", "help", "q", "quit"}
	var sb strings.Builder
	for i := 0; i+1 < len(keys); i += 2 {
		next := sKey.Render(keys[i]) + " " + sDim.Render(keys[i+1]) + "  "
		if lipgloss.Width(sb.String())+lipgloss.Width(next) > md.width-2 {
			break
		}
		sb.WriteString(next)
	}
	return sStatus.Width(md.width).MaxHeight(1).Render(sb.String())
}

func (md *Model) overlay(bg, content string) string {
	box := sModal.Width(min(md.width-6, 90)).Render(content)
	return lipgloss.Place(md.width, md.heigh, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(cBg))
}

func (md *Model) resultText() string {
	title := sGreen.Render("✓ done")
	if !md.resultOK {
		title = sRed.Render("✗ failed")
	}
	body := md.result
	lines := strings.Split(body, "\n")
	if len(lines) > md.heigh-10 {
		lines = append(lines[:md.heigh-12], sDim.Render(fmt.Sprintf("… %d more lines", len(lines)-(md.heigh-12))))
		body = strings.Join(lines, "\n")
	}
	return title + "\n\n" + body + "\n\n" + sDim.Render("press Enter to continue")
}

func (md *Model) helpText() string {
	k := func(s string) string { return sKey.Render(fmt.Sprintf("%-10s", s)) }
	rows := [][2]string{
		{"j / k", "move down / up"}, {"gg / G", "first / last"}, {"ctrl+d/u", "half page"},
		{"l / h", "expand / collapse unit"}, {"Space", "toggle unit or file"}, {"a", "toggle all"},
		{"/", "filter (Esc clears)"}, {"Tab", "switch pane (preview scrolls with j/k)"},
		{"d", "show diffs in preview"}, {"m", "switch backup ⇄ install"},
		{"p", "cycle profile (none → personal → work)"}, {"s", "symlink mode for install"},
		{"b", "run BACKUP  (live → repo)"}, {"i", "run INSTALL (repo → live)"}, {"x", "run current direction"},
		{"r", "refresh"}, {"?", "this help"}, {"q", "quit"},
	}
	var sb strings.Builder
	sb.WriteString(sTitle.Render("keys") + "\n\n")
	for _, r := range rows {
		sb.WriteString(k(r[0]) + " " + r[1] + "\n")
	}
	sb.WriteString("\n" + sTitle.Render("legend") + "\n\n")
	sb.WriteString(sGreen.Render("+n") + " create  " + sBlue.Render("~n") + " update  " + sRed.Render("-n") + " delete  " + sGold.Render("🔒") + " secret → template  " + sGreen.Render("●") + "/" + sDim.Render("○") + " enabled/disabled\n")
	sb.WriteString("\n" + sDim.Render("selection and profile are saved to "+tilde(state.Path(md.m.Home), md.m.Home)))
	return sb.String()
}
