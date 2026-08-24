package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/edwinvillota/dotfiles/internal/theme"
)

// Styles are package-level and re-initialized from the active palette by
// initStyles (called from New/NewSetup, and again after a theme switch).
// Defaults below are Ayu Dark so tests and any pre-init render look sane.
var (
	cBg     = lipgloss.Color("#0B0E14")
	cFg     = lipgloss.Color("#BFBDB6")
	cDim    = lipgloss.Color("#565B66")
	cBlue   = lipgloss.Color("#59C2FF")
	cGold   = lipgloss.Color("#E6B450")
	cGreen  = lipgloss.Color("#AAD94C")
	cOrange = lipgloss.Color("#FF8F40")
	cRed    = lipgloss.Color("#D95757")
	cPanel  = lipgloss.Color("#0D1017")
	cLine   = lipgloss.Color("#1F2430")
	cSel    = lipgloss.Color("#131721")

	sBase, sDim, sTitle, sBlue, sGreen, sOrange, sRed, sGold, sKey, sCursor,
	sPane, sPaneAct, sStatus, sModal, sTag, sTagWarn lipgloss.Style
)

func init() { rebuildStyles() }

// initStyles loads the palette for the given theme (empty = default) and
// rebuilds every style from it. Unknown/broken themes keep the defaults.
func initStyles(name string) {
	if name == "" {
		name = theme.Default
	}
	p, err := theme.Load(name)
	if err != nil {
		return
	}
	cBg = lipgloss.Color(p.Primary.Background)
	cFg = lipgloss.Color(p.Primary.Foreground)
	cDim = lipgloss.Color(p.Roles.Dim)
	cBlue = lipgloss.Color(p.Roles.Accent2)
	cGold = lipgloss.Color(p.Roles.Accent)
	cGreen = lipgloss.Color(p.Roles.Good)
	cOrange = lipgloss.Color(p.Roles.Warn)
	cRed = lipgloss.Color(p.Roles.Error)
	cPanel = lipgloss.Color(p.Roles.Panel)
	cLine = lipgloss.Color(p.Roles.Line)
	cSel = lipgloss.Color(p.Roles.Sel)
	rebuildStyles()
}

func rebuildStyles() {
	sBase = lipgloss.NewStyle().Foreground(cFg).Background(cBg)
	sDim = lipgloss.NewStyle().Foreground(cDim)
	sTitle = lipgloss.NewStyle().Foreground(cGold).Bold(true)
	sBlue = lipgloss.NewStyle().Foreground(cBlue)
	sGreen = lipgloss.NewStyle().Foreground(cGreen)
	sOrange = lipgloss.NewStyle().Foreground(cOrange)
	sRed = lipgloss.NewStyle().Foreground(cRed)
	sGold = lipgloss.NewStyle().Foreground(cGold)
	sKey = lipgloss.NewStyle().Foreground(cGold).Bold(true)
	sCursor = lipgloss.NewStyle().Background(cSel).Foreground(cFg).Bold(true)
	sPane = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cLine).Padding(0, 1)
	sPaneAct = sPane.BorderForeground(cBlue)
	sStatus = lipgloss.NewStyle().Foreground(cFg).Background(cPanel).Padding(0, 1)
	sModal = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(cGold).Padding(1, 2).Background(cPanel)
	sTag = lipgloss.NewStyle().Foreground(cBg).Background(cBlue).Padding(0, 1).Bold(true)
	sTagWarn = lipgloss.NewStyle().Foreground(cBg).Background(cOrange).Padding(0, 1).Bold(true)
}
