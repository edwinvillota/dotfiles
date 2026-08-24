package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DEV — DEV. The house brand.
const devArt = `
██████╗  ███████╗ ██╗   ██╗
██╔══██╗ ██╔════╝ ██║   ██║
██║  ██║ █████╗   ██║   ██║
██║  ██║ ██╔══╝   ╚██╗ ██╔╝
██████╔╝ ███████╗  ╚████╔╝
╚═════╝  ╚══════╝   ╚═══╝`

// Banner renders the DEV header with the dotfiles subtitle in Ayu colors.
func Banner(width int) string {
	art := lipgloss.NewStyle().Foreground(cGold).Bold(true).Render(strings.TrimPrefix(devArt, "\n"))
	sub := lipgloss.NewStyle().Foreground(cBlue).Render("· d o t f i l e s ·")
	tag := sDim.Render("")
	b := lipgloss.JoinVertical(lipgloss.Center, art, "", sub, tag)
	if width > 0 {
		return lipgloss.PlaceHorizontal(width, lipgloss.Center, b)
	}
	return b
}
