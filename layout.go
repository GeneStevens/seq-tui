package main

import (
	"github.com/charmbracelet/lipgloss"
)

const (
	headerTitle    = "seq-tui"
	headerSubtitle = "spatial view"
	footerHelp     = "q: quit"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Faint(true).
			PaddingLeft(1)

	mapBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Faint(true).
			Padding(0, 1)
)

// renderHeader returns the header strip.
func renderHeader(width int) string {
	title := headerStyle.Render(headerTitle)
	subtitle := subtitleStyle.Render(headerSubtitle)
	line := lipgloss.JoinHorizontal(lipgloss.Top, title, subtitle)
	return lipgloss.NewStyle().Width(width).Render(line)
}

// renderMapPanel returns the map inside a bordered panel.
func renderMapPanel() string {
	mapContent := renderMap()
	return mapBorderStyle.Render(mapContent)
}

// renderFooter returns the footer help strip.
func renderFooter(width int) string {
	return footerStyle.Width(width).Render(footerHelp)
}

// renderLayout composes all sections into the full view.
func renderLayout(width, height int) string {
	header := renderHeader(width)
	footer := renderFooter(width)
	mapPanel := renderMapPanel()

	// Body height is total minus header (1 line) and footer (1 line)
	bodyHeight := height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Center the map panel horizontally and vertically within the body area
	body := lipgloss.Place(width, bodyHeight,
		lipgloss.Center, lipgloss.Center,
		mapPanel,
	)

	return header + "\n" + body + "\n" + footer
}
