package tui

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/edwinvillota/dotfiles/internal/apply"
	"github.com/edwinvillota/dotfiles/internal/deps"
	"github.com/edwinvillota/dotfiles/internal/manifest"
	"github.com/edwinvillota/dotfiles/internal/plan"
	"github.com/edwinvillota/dotfiles/internal/state"
)

// Setup is the guided fresh-machine wizard: welcome → profile → features →
// one review → run. A single confirmation for everything.
type Setup struct {
	m  *manifest.Manifest
	st *state.State
	pf deps.Platform

	step   int // 0 welcome, 1 profile, 2 features, 3 review, 4 running, 5 done
	width  int
	height int

	profiles []string
	profIdx  int

	feats  []feature
	cursor int

	items   []deps.Item
	plan    *plan.Plan
	planErr error

	log string
	err error
}

type feature struct {
	key    string // unit name or "tool:<name>"
	label  string
	desc   string
	isTool bool
	on     bool
	soon   bool   // announced but not available yet — visible, not toggleable
	header string // section header rendered above this row
}

func NewSetup(m *manifest.Manifest, st *state.State) *Setup {
	s := &Setup{m: m, st: st, pf: deps.Detect(), profiles: sortedKeys(m.Profiles)}
	for i, n := range s.profiles {
		if n == st.Profile {
			s.profIdx = i
		}
	}
	// terminal choice first: wezterm today, ghostty on the roadmap
	if u, ok := m.Units["wezterm"]; ok {
		s.feats = append(s.feats, feature{key: "wezterm", label: "wezterm",
			desc: strings.TrimPrefix(u.Dest.For(m.GOOS), "~/"), on: !st.IsDisabled("wezterm"), header: "terminal"})
	}
	gh := feature{key: "__ghostty", label: "ghostty", desc: "coming soon", soon: true}
	if len(s.feats) == 0 {
		gh.header = "terminal"
	}
	s.feats = append(s.feats, gh)
	first := true
	for _, n := range m.UnitNames() {
		if n == "wezterm" {
			continue
		}
		u := m.Units[n]
		h := ""
		if first {
			h, first = "configs", false
		}
		s.feats = append(s.feats, feature{key: n, label: n,
			desc: strings.TrimPrefix(u.Dest.For(m.GOOS), "~/"), on: !st.IsDisabled(n), header: h})
	}
	extras := append([]string{}, m.Deps.Extra...)
	sort.Strings(extras)
	for i, n := range extras {
		h := ""
		if i == 0 {
			h = "extra tools (core tools are always installed)"
		}
		s.feats = append(s.feats, feature{key: "tool:" + n, label: n, isTool: true,
			on: !st.IsDisabled("tool:" + n), header: h})
	}
	return s
}

func (s *Setup) Init() tea.Cmd { return nil }

func (s *Setup) profile() *manifest.Profile {
	if len(s.profiles) == 0 {
		return nil
	}
	return s.m.Profiles[s.profiles[s.profIdx]]
}

// compute resolves deps and the config plan for the review screen.
func (s *Setup) compute() {
	names := append([]string{}, s.m.Deps.Core...)
	for _, f := range s.feats {
		if f.isTool && f.on {
			names = append(names, strings.TrimPrefix(f.key, "tool:"))
		}
	}
	s.items = deps.Resolve(s.m, s.pf, names)
	full, err := plan.Build(s.m, plan.Options{Direction: plan.Install, Profile: s.profile()})
	if err != nil {
		s.planErr = err
		return
	}
	s.plan, s.planErr = plan.Build(s.m, plan.Options{Direction: plan.Install, Profile: s.profile(),
		Selected: s.st.Selected(full.Keys())})
}

func (s *Setup) missing() []deps.Item {
	var out []deps.Item
	for _, it := range s.items {
		if it.Status == deps.Missing || it.Status == deps.Outdated || it.Status == deps.NeedsBrew {
			out = append(out, it)
		}
	}
	return out
}

type setupDone struct {
	log string
	err error
}

func (s *Setup) run() tea.Cmd {
	m, st, pf, items, p := s.m, s.st, s.pf, s.items, s.plan
	prof := ""
	if len(s.profiles) > 0 {
		prof = s.profiles[s.profIdx]
	}
	return func() tea.Msg {
		var buf bytes.Buffer
		st.Profile = prof
		if err := st.Save(); err != nil {
			return setupDone{buf.String(), err}
		}
		if err := deps.Install(m, pf, items, false, &buf); err != nil {
			fmt.Fprintf(&buf, "\ntool install issue (retry later with `dotfiles deps`): %v\n", err)
		}
		res, err := apply.Run(m, p, apply.Options{Confirm: func(string) bool { return true }, Log: &buf})
		if err != nil {
			return setupDone{buf.String(), err}
		}
		fmt.Fprintf(&buf, "\nconfigs: %d written, %d deleted\n", res.Written, res.Deleted)
		for _, n := range res.Notices {
			fmt.Fprintln(&buf, "NOTE:", n)
		}
		return setupDone{buf.String(), nil}
	}
}

func (s *Setup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		return s, nil
	case setupDone:
		s.log, s.err, s.step = msg.log, msg.err, 5
		return s, nil
	case tea.KeyMsg:
		k := msg.String()
		if k == "ctrl+c" || (k == "q" && s.step != 4) {
			return s, tea.Quit
		}
		switch s.step {
		case 0:
			if k == "enter" || k == " " {
				s.step = 1
			}
		case 1:
			switch k {
			case "j", "down":
				if s.profIdx < len(s.profiles)-1 {
					s.profIdx++
				}
			case "k", "up":
				if s.profIdx > 0 {
					s.profIdx--
				}
			case "enter":
				s.step = 2
			}
		case 2:
			switch k {
			case "j", "down":
				if s.cursor < len(s.feats)-1 {
					s.cursor++
				}
			case "k", "up":
				if s.cursor > 0 {
					s.cursor--
				}
			case "g":
				s.cursor = 0
			case "G":
				s.cursor = len(s.feats) - 1
			case " ":
				f := &s.feats[s.cursor]
				if !f.soon {
					f.on = !f.on
					s.st.Set(f.key, f.on)
				}
			case "enter":
				s.st.Save()
				s.compute()
				s.step = 3
			case "esc", "h":
				s.step = 1
			}
		case 3:
			switch k {
			case "enter", "y":
				s.step = 4
				return s, s.run()
			case "esc", "h", "n":
				s.step = 2
			}
		case 5:
			if k == "enter" {
				return s, tea.Quit
			}
		}
	}
	return s, nil
}

func (s *Setup) View() string {
	if s.width == 0 {
		return "…"
	}
	var body string
	switch s.step {
	case 0:
		body = lipgloss.JoinVertical(lipgloss.Left,
			"This will set up this machine from the dotfiles repo:",
			"",
			"  1. pick a "+sBlue.Render("profile")+" (which machine this is)",
			"  2. choose the "+sBlue.Render("features")+" you want — configs and tools",
			"  3. review one summary, confirm "+sGold.Render("once")+", done",
			"",
			sDim.Render("Nothing is written before the final confirmation. Anything replaced is")+"",
			sDim.Render("preserved and restorable with `dotfiles uninstall`. ~/.zshrc and secret")+"",
			sDim.Render("files are never overwritten."),
			"",
			s.hint("Enter continue · q quit"))
	case 1:
		lines := []string{sTitle.Render("Which machine is this?"), ""}
		for i, n := range s.profiles {
			p := s.m.Profiles[n]
			cur := "  "
			st := lipgloss.NewStyle().Foreground(cFg)
			if i == s.profIdx {
				cur, st = sBlue.Render("▍ "), sCursor
			}
			lines = append(lines, cur+st.Render(fmt.Sprintf("%-10s", n))+sDim.Render("  branch "+p.Branch+"  excludes "+fmt.Sprint(p.Exclude)))
		}
		lines = append(lines, "", s.hint("j/k choose · Enter continue · q quit"))
		body = strings.Join(lines, "\n")
	case 2:
		lines := []string{sTitle.Render("What do you want on this machine?"), ""}
		vis := s.height - 16
		start := 0
		if s.cursor >= vis {
			start = s.cursor - vis + 1
		}
		shown := 0
		for i, f := range s.feats {
			if i < start {
				continue
			}
			if shown >= vis {
				break
			}
			if f.header != "" {
				lines = append(lines, sGold.Render("─ "+f.header))
				shown++
			}
			box := sGreen.Render("[x]")
			if f.soon {
				box = sDim.Render("[·]")
			} else if !f.on {
				box = sDim.Render("[ ]")
			}
			cur := "  "
			name := lipgloss.NewStyle().Foreground(cFg).Render(fmt.Sprintf("%-16s", f.label))
			if i == s.cursor {
				cur = sBlue.Render("▍ ")
				name = sCursor.Render(fmt.Sprintf("%-16s", f.label))
			}
			desc := sDim.Render(f.desc)
			if f.soon {
				desc = sOrange.Render("coming soon")
				name = sDim.Render(fmt.Sprintf("%-16s", f.label))
				if i == s.cursor {
					name = sCursor.Render(fmt.Sprintf("%-16s", f.label))
				}
			}
			lines = append(lines, cur+box+" "+name+desc)
			shown++
		}
		lines = append(lines, "", s.hint("Space toggle · j/k move · Enter continue · Esc back"))
		body = strings.Join(lines, "\n")
	case 3:
		if s.planErr != nil {
			body = sRed.Render("error: "+s.planErr.Error()) + "\n\n" + s.hint("Esc back")
			break
		}
		miss := s.missing()
		cr, up, de, li, _ := s.plan.Counts()
		lines := []string{sTitle.Render("Review — this is everything that will happen"), ""}
		if len(miss) == 0 {
			lines = append(lines, sGreen.Render("✓")+" tools: all installed")
		} else {
			lines = append(lines, fmt.Sprintf("%s tools to install (%d):", sOrange.Render("→"), len(miss)))
			for i, it := range miss {
				if i >= 8 {
					lines = append(lines, sDim.Render(fmt.Sprintf("    … and %d more", len(miss)-8)))
					break
				}
				lines = append(lines, "    "+it.Name+sDim.Render("  via "+it.Manager))
			}
		}
		lines = append(lines, "",
			fmt.Sprintf("%s configs: %s create  %s update  %s delete  %s link",
				sOrange.Render("→"), sGreen.Render(fmt.Sprint(cr)), sBlue.Render(fmt.Sprint(up)),
				sRed.Render(fmt.Sprint(de)), sGreen.Render(fmt.Sprint(li))),
			sDim.Render("    replaced files are preserved; `dotfiles uninstall` restores them"), "",
			s.hint("Enter run everything · Esc back"))
		body = strings.Join(lines, "\n")
	case 4:
		body = sGold.Render("working… ") + sDim.Render("(tools may compile; this can take a while)")
	case 5:
		head := sGreen.Render("✓ all done")
		if s.err != nil {
			head = sRed.Render("✗ finished with an error: " + s.err.Error())
		}
		tail := s.log
		if lines := strings.Split(tail, "\n"); len(lines) > s.height-14 {
			tail = strings.Join(lines[len(lines)-(s.height-14):], "\n")
		}
		body = head + "\n\n" + tail + "\n" + sDim.Render("next: exec zsh · dotfiles (TUI) · dotfiles uninstall to undo") + "\n\n" + s.hint("Enter to finish")
	}
	out := lipgloss.JoinVertical(lipgloss.Left, Banner(s.width), "",
		lipgloss.NewStyle().PaddingLeft(2).Render(body))
	return sBase.Render(out)
}

func (s *Setup) hint(t string) string { return sDim.Render(t) }
