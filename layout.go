package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	headerTitle    = "seq-tui"
	headerSubtitle = "spatial view"
	footerHelp = "hjkl/arrows: move  q: quit"

	// Minimum terminal width to show side panels alongside the map.
	sidePanelMinWidth = 70
	// Width allocated to the side column.
	sidePanelWidth = 24

	nearbyTitle = "Nearby"
	statusTitle = "Status"
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

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true)

	panelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Padding(0, 1)

	panelItemStyle = lipgloss.NewStyle().
			Faint(true)
)

// renderHeader returns the header strip.
func renderHeader(width int) string {
	title := headerStyle.Render(headerTitle)
	subtitle := subtitleStyle.Render(headerSubtitle)
	line := lipgloss.JoinHorizontal(lipgloss.Top, title, subtitle)
	return lipgloss.NewStyle().Width(width).Render(line)
}

// renderMapPanel returns the map inside a bordered panel with awareness dimming.
func renderMapPanel() string {
	mapContent := renderStyledMap()
	return mapBorderStyle.Render(mapContent)
}

// renderNearbyPanel returns the static nearby-awareness panel.
func renderNearbyPanel(width int) string {
	title := panelTitleStyle.Render(nearbyTitle)
	items := []string{
		panelItemStyle.Render("  shadowed arch"),
		panelItemStyle.Render("  faint movement?"),
		panelItemStyle.Render("  uneasy presence"),
		panelItemStyle.Render("  cold draft north"),
		panelItemStyle.Render("  echo from east"),
		panelItemStyle.Render("  stone dampness"),
		panelItemStyle.Render("  dust in torchlight"),
		panelItemStyle.Render("  deep silence"),
	}
	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content) // -4 for border+padding
}

// renderStatusPanel returns the status panel with target and read info.
func renderStatusPanel(width int, target backendTarget, zr zoneReadResult) string {
	title := panelTitleStyle.Render(statusTitle)
	vis := strings.ToLower(target.Visibility)
	items := []string{
		panelItemStyle.Render("  target: local"),
		panelItemStyle.Render("  zone: " + target.Zone),
		panelItemStyle.Render("  mode: " + strings.ToLower(target.Mode)),
		panelItemStyle.Render("  visibility: " + vis),
		panelItemStyle.Render("  " + zr.statusLabel()),
		panelItemStyle.Render("  client: thin"),
	}
	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderSideColumn stacks the nearby and status panels vertically.
func renderSideColumn(width int, target backendTarget, zr zoneReadResult) string {
	nearby := renderNearbyPanel(width)
	status := renderStatusPanel(width, target, zr)
	return lipgloss.JoinVertical(lipgloss.Left, nearby, "", status)
}

// renderFooter returns the footer help strip with optional intent preview.
func renderFooter(width int, intentPreview string) string {
	help := footerHelp
	if intentPreview != "" {
		help = intentPreview + "  " + footerHelp
	}
	return footerStyle.Width(width).Render(help)
}

// renderLayout composes all sections into the full view.
func renderLayout(width, height int, lastInput string, target backendTarget, zr zoneReadResult) string {
	header := renderHeader(width)
	footer := renderFooter(width, lastInput)
	mapPanel := renderMapPanel()

	// Body height is total minus header (1 line) and footer (1 line)
	bodyHeight := height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	if width >= sidePanelMinWidth {
		// Side-by-side: map on left, info panels on right
		sideCol := renderSideColumn(sidePanelWidth, target, zr)
		combined := lipgloss.JoinHorizontal(lipgloss.Top, mapPanel, "  ", sideCol)
		body = lipgloss.Place(width, bodyHeight,
			lipgloss.Center, lipgloss.Center,
			combined,
		)
	} else {
		// Narrow terminal: map only, centered
		body = lipgloss.Place(width, bodyHeight,
			lipgloss.Center, lipgloss.Center,
			mapPanel,
		)
	}

	return header + "\n" + body + "\n" + footer
}
