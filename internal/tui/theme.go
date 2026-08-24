package tui

import "github.com/charmbracelet/lipgloss"

// Ayu Dark.
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

	sBase    = lipgloss.NewStyle().Foreground(cFg).Background(cBg)
	sDim     = lipgloss.NewStyle().Foreground(cDim)
	sTitle   = lipgloss.NewStyle().Foreground(cGold).Bold(true)
	sBlue    = lipgloss.NewStyle().Foreground(cBlue)
	sGreen   = lipgloss.NewStyle().Foreground(cGreen)
	sOrange  = lipgloss.NewStyle().Foreground(cOrange)
	sRed     = lipgloss.NewStyle().Foreground(cRed)
	sGold    = lipgloss.NewStyle().Foreground(cGold)
	sKey     = lipgloss.NewStyle().Foreground(cGold).Bold(true)
	sCursor  = lipgloss.NewStyle().Background(cSel).Foreground(cFg).Bold(true)
	sPane    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cLine).Padding(0, 1)
	sPaneAct = sPane.BorderForeground(cBlue)
	sStatus  = lipgloss.NewStyle().Foreground(cFg).Background(cPanel).Padding(0, 1)
	sModal   = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(cGold).Padding(1, 2).Background(cPanel)
	sTag     = lipgloss.NewStyle().Foreground(cBg).Background(cBlue).Padding(0, 1).Bold(true)
	sTagWarn = lipgloss.NewStyle().Foreground(cBg).Background(cOrange).Padding(0, 1).Bold(true)
)
