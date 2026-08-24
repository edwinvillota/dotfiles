package tui

import (
	"bytes"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/edwinvillota/dotfiles/internal/manifest"
	"github.com/edwinvillota/dotfiles/internal/state"
	"github.com/edwinvillota/dotfiles/internal/theme"
)

// picker is the standalone selector behind `dotfiles theme` (no arguments):
// theme names are hard to remember, so choosing is arrow keys + Enter.
type picker struct {
	m       *manifest.Manifest
	st      *state.State
	names   []string
	idx     int
	width   int
	height  int
	summary string
	err     error
}

// RunThemePicker opens the interactive selector and applies the choice.
// It returns what should be printed after the terminal is restored.
func RunThemePicker(m *manifest.Manifest, st *state.State) (string, error) {
	initStyles(st.Theme)
	active := st.Theme
	if active == "" {
		active = theme.Default
	}
	p := &picker{m: m, st: st, names: theme.Names()}
	for i, n := range p.names {
		if n == active {
			p.idx = i
		}
	}
	out, err := tea.NewProgram(p, tea.WithAltScreen()).Run()
	if err != nil {
		return "", err
	}
	res := out.(*picker)
	return res.summary, res.err
}

func (p *picker) Init() tea.Cmd { return nil }

func (p *picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return p, tea.Quit
		case "j", "down":
			if p.idx < len(p.names)-1 {
				p.idx++
			}
		case "k", "up":
			if p.idx > 0 {
				p.idx--
			}
		case "enter":
			name := p.names[p.idx]
			var buf bytes.Buffer
			res, err := theme.Apply(p.m, name, nil, &buf)
			if err != nil {
				p.err = fmt.Errorf("%s%w", buf.String(), err)
				return p, tea.Quit
			}
			p.st.Theme = name
			if err := p.st.Save(); err != nil {
				p.err = err
				return p, tea.Quit
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "theme switched to %s (%d file(s) written)\n", name, res.Written)
			for _, n := range res.Notices {
				sb.WriteString("note: " + n + "\n")
			}
			sb.WriteString("\nwhat picks it up when:\n")
			for _, r := range res.Reload {
				sb.WriteString("  - " + r + "\n")
			}
			p.summary = sb.String()
			return p, tea.Quit
		}
	}
	return p, nil
}

func (p *picker) View() string {
	if p.width == 0 {
		return "…"
	}
	active := p.st.Theme
	if active == "" {
		active = theme.Default
	}
	var sb strings.Builder
	sb.WriteString(sTitle.Render("theme — one look for every tool") + "\n\n")
	for i, n := range p.names {
		label := n
		if pal, err := theme.Load(n); err == nil {
			label = pal.Label
		}
		cur := "  "
		if i == p.idx {
			cur = sBlue.Render("▍ ")
		}
		mark := "  "
		if n == active {
			mark = sGreen.Render("● ")
		}
		line := fmt.Sprintf("%-26s", label)
		if i == p.idx {
			line = sCursor.Render(line)
		}
		sb.WriteString(cur + mark + line + "\n")
	}
	sb.WriteString("\n" + sDim.Render("j/k choose · Enter apply · Esc cancel"))
	box := sModal.Render(sb.String())
	return sBase.Render(lipgloss.Place(p.width, p.height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(cBg)))
}
