package launcher

import "github.com/charmbracelet/lipgloss"

// Layout constants (character cells).
const (
	artW      = 15 // banner art inner width
	artH      = 7  // banner art inner height
	cardW     = artW + 2 // card outer width incl. left/right border
	gap       = 2  // horizontal gap between cards
	sidebarW  = 12 // sidebar inner width
	rowStride = artH + 2 + 1 + 1 // card border rows + label + row gap
)

// Palette.
var (
	accent   = lipgloss.Color("#7C5CFF")
	fgBright = lipgloss.Color("#FFFFFF")
	fgDim    = lipgloss.Color("#8A8A99")
)

// Styles.
var (
	wordmarkStyle  = lipgloss.NewStyle().Foreground(fgBright).Bold(true)
	navItemStyle   = lipgloss.NewStyle().Foreground(fgDim)
	navActiveStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	sidebarStyle   = lipgloss.NewStyle().Width(sidebarW).Padding(1, 1)

	cardStyle       = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(fgDim)
	cardFocusStyle  = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(accent)
	labelStyle      = lipgloss.NewStyle().Foreground(fgDim).Width(cardW).Align(lipgloss.Center)
	labelFocusStyle = lipgloss.NewStyle().Foreground(fgBright).Bold(true).Width(cardW).Align(lipgloss.Center)

	defaultBannerStyle = lipgloss.NewStyle().
				Width(artW).Height(artH).
				Align(lipgloss.Center, lipgloss.Center).
				Foreground(fgBright).Background(accent).Bold(true)

	headerStyle     = lipgloss.NewStyle().Foreground(fgBright).Bold(true)
	countStyle      = lipgloss.NewStyle().Foreground(fgDim)
	comingSoonStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	subtleStyle     = lipgloss.NewStyle().Foreground(fgDim)
	footerStyle     = lipgloss.NewStyle().Foreground(fgDim)
)
