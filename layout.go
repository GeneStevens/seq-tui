package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	headerTitle    = "seq-tui"
	headerSubtitle = "spatial view"
	footerHelp = "hjkl/arrows: move  tab/S-tab: roster  t: confirm  a: attack  p: pickup  q: quit"

	// Minimum terminal width to show side panels alongside the map.
	sidePanelMinWidth = 70
	// Width allocated to the side column.
	sidePanelWidth = 24

	nearbyTitle    = "Nearby"
	statusTitle    = "Status"
	encounterTitle = "Encounter"
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

// renderMapPanel returns the map inside a bordered panel.
// Uses backend-sourced map when available, falls back to placeholder with overlays.
// Focus projection overlays the focused roster entry's map position purely visually.
func renderMapPanel(mr mapReadResult, mobr mobReadResult, pr playerReadResult, focus rosterFocus, entries []rosterEntry) string {
	var mapContent string
	if mr.State == mapReadOK && mr.MapText != "" {
		mapContent = mr.MapText
		// Overlay mob positions if available
		if mobr.State == mobReadOK && len(mobr.Mobs) > 0 {
			mapContent = overlayMobs(mapContent, mobr.Mobs, mr.Bounds, mr.MapWidth, mr.MapHeight)
		}
		// Overlay player marker from backend position (last, so always visible)
		if pr.State == playerReadOK && pr.HasPos {
			mapContent = overlayPlayer(mapContent, pr.Position, mr.Bounds, mr.MapWidth, mr.MapHeight)
		}
		// Focus projection: highlight the focused roster entry on the map
		if fe := focusedEntry(focus, entries); fe != nil {
			switch fe.kind {
			case "mb":
				if mobr.State == mobReadOK {
					mapContent = overlayFocusedMob(mapContent, mobr.Mobs, fe.id, mr.Bounds, mr.MapWidth, mr.MapHeight)
				}
			case "pc":
				if pr.State == playerReadOK && pr.HasPos {
					mapContent = overlayFocusedPlayer(mapContent, pr.Position, mr.Bounds, mr.MapWidth, mr.MapHeight)
				}
			}
		}
	} else {
		mapContent = renderStyledMap()
	}
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
func renderStatusPanel(width int, target backendTarget, zr zoneReadResult, mr mapReadResult, mobr mobReadResult, pr playerReadResult) string {
	title := panelTitleStyle.Render(statusTitle)
	vis := strings.ToLower(target.Visibility)
	items := []string{
		panelItemStyle.Render("  target: local"),
		panelItemStyle.Render("  zone: " + target.Zone),
		panelItemStyle.Render("  mode: " + strings.ToLower(target.Mode)),
		panelItemStyle.Render("  visibility: " + vis),
		panelItemStyle.Render("  " + zr.statusLabel()),
		panelItemStyle.Render("  " + mr.mapStatusLabel()),
		panelItemStyle.Render("  " + mobr.mobStatusLabel()),
		panelItemStyle.Render("  " + pr.playerStatusLabel()),
		panelItemStyle.Render("  client: thin"),
	}
	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderEncounterPanel returns the encounter status panel based on backend-owned facts.
func renderEncounterPanel(width int, pr playerReadResult, er encounterReadResult, focus rosterFocus) string {
	title := panelTitleStyle.Render(encounterTitle)

	var items []string

	if pr.State != playerReadOK {
		items = append(items, panelItemStyle.Render("  no player"))
	} else if er.State == encounterReadFailed {
		items = append(items, panelItemStyle.Render("  unavailable"))
	} else if er.State == encounterReadNotAttempted {
		items = append(items, panelItemStyle.Render("  pending"))
	} else {
		// encounterReadOK — show zone encounter count
		items = append(items, panelItemStyle.Render(fmt.Sprintf("  zone: %d enc", er.Count)))

		if pr.HasActiveEncounter {
			items = append(items, panelItemStyle.Render("  active: yes"))
			// Find matching encounter summary for detail
			if enc := findPlayerEncounter(er.Encounters, pr.ActiveEncounterID); enc != nil {
				items = append(items, panelItemStyle.Render("  "+enc.State))
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  pcs:%d mobs:%d", enc.PlayerCount, enc.MobCount)))
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  alive:%d dead:%d", enc.MobsAlive, enc.MobsDead)))
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  act:%d", enc.ActionIndex)))
				if enc.CompletedReason != "" {
					items = append(items, panelItemStyle.Render("  "+enc.CompletedReason))
				}
				// Roster: backend-owned participant lists with local focus
				items = append(items, renderRosterSection(enc, width, focus)...)
			} else {
				items = append(items, panelItemStyle.Render("  no details"))
			}
		} else {
			items = append(items, panelItemStyle.Render("  active: no"))
		}
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderRosterSection returns roster lines for the encounter panel.
// Shows backend-owned player and mob IDs, truncated for panel width.
// Local focus indicator (>) shown on the focused entry. Read-only,
// no selection with gameplay meaning.
func renderRosterSection(enc *encounterSummary, panelWidth int, focus rosterFocus) []string {
	var lines []string
	// Available width inside the panel: panelWidth - border(2) - padding(2) - indent(2)
	maxIDWidth := panelWidth - 6
	if maxIDWidth < 4 {
		maxIDWidth = 4
	}

	lines = append(lines, panelItemStyle.Render("  ---roster---"))

	if len(enc.PlayerIDs) == 0 && len(enc.MobIDs) == 0 {
		lines = append(lines, panelItemStyle.Render("  no roster data"))
		return lines
	}

	entryIdx := 0
	for _, pid := range enc.PlayerIDs {
		prefix := "  "
		if focus.index == entryIdx {
			prefix = "> "
		}
		label := prefix + "pc:" + truncateID(pid, maxIDWidth-4)
		lines = append(lines, panelItemStyle.Render(label))
		entryIdx++
	}
	for _, mid := range enc.MobIDs {
		prefix := "  "
		if focus.index == entryIdx {
			prefix = "> "
		}
		label := prefix + "mb:" + truncateID(mid, maxIDWidth-4)
		lines = append(lines, panelItemStyle.Render(label))
		entryIdx++
	}

	return lines
}

// truncateID shortens an ID string to fit within maxLen characters.
func truncateID(id string, maxLen int) string {
	if maxLen < 1 {
		maxLen = 1
	}
	if len(id) <= maxLen {
		return id
	}
	if maxLen <= 2 {
		return id[:maxLen]
	}
	return id[:maxLen-2] + ".."
}

// renderProximityPanel returns a compact panel showing backend-owned proximity data.
// Read-only, no targeting authority, no gameplay semantics.
func renderProximityPanel(width int, tc targetConfirmResult) string {
	title := panelTitleStyle.Render("Proximity")

	var items []string

	switch tc.State {
	case targetConfirmNone:
		items = append(items, panelItemStyle.Render("  none"))
	case targetConfirmFailed:
		items = append(items, panelItemStyle.Render("  unavailable"))
	case targetConfirmOK:
		if tc.MobName != "" {
			items = append(items, panelItemStyle.Render("  "+truncateID(tc.MobName, width-6)))
		} else {
			items = append(items, panelItemStyle.Render("  "+tc.TargetKind+":"+truncateID(tc.TargetID, width-9)))
		}
		if tc.Found {
			items = append(items, panelItemStyle.Render("  found: yes"))
		} else {
			items = append(items, panelItemStyle.Render("  found: no"))
		}
		if tc.Found {
			if tc.WithinProximity {
				items = append(items, panelItemStyle.Render("  within: yes"))
			} else {
				items = append(items, panelItemStyle.Render("  within: no"))
			}
			items = append(items, panelItemStyle.Render(fmt.Sprintf("  dist: %.1f", tc.Distance)))
		}
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderCombatPanel returns a compact panel showing backend-owned combat readback.
// Only populated after an attack has been submitted. Shows encounter state changes
// from backend truth without any client-side combat logic or interpretation.
func renderCombatPanel(width int, ar attackResult, pr playerReadResult, er encounterReadResult) string {
	title := panelTitleStyle.Render("Combat")

	var items []string

	if ar.State == attackStateNone {
		items = append(items, panelItemStyle.Render("  none"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	// Show submission result
	if ar.State == attackStateSent {
		items = append(items, panelItemStyle.Render("  intent: accepted"))
	} else {
		items = append(items, panelItemStyle.Render("  intent: failed"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	// Show backend-owned encounter readback
	if !pr.HasActiveEncounter {
		items = append(items, panelItemStyle.Render("  enc: none"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	if er.State != encounterReadOK {
		items = append(items, panelItemStyle.Render("  enc: unavailable"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	enc := findPlayerEncounter(er.Encounters, pr.ActiveEncounterID)
	if enc == nil {
		items = append(items, panelItemStyle.Render("  enc: no details"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	// Backend-owned encounter facts
	items = append(items, panelItemStyle.Render("  "+enc.State))
	items = append(items, panelItemStyle.Render(fmt.Sprintf("  act:%d", enc.ActionIndex)))
	items = append(items, panelItemStyle.Render(fmt.Sprintf("  alive:%d dead:%d", enc.MobsAlive, enc.MobsDead)))

	if enc.CompletedReason != "" {
		items = append(items, panelItemStyle.Render("  "+enc.CompletedReason))
	}

	// Check if the attacked mob is still in the encounter roster
	if ar.TargetID != "" {
		mobPresent := false
		for _, mid := range enc.MobIDs {
			if mid == ar.TargetID {
				mobPresent = true
				break
			}
		}
		if mobPresent {
			items = append(items, panelItemStyle.Render("  mob: in roster"))
		} else {
			items = append(items, panelItemStyle.Render("  mob: gone"))
		}
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderLootPanel returns a compact panel showing backend-owned loot readback.
// Read-only display of drop state from encounter summary, plus pickup submission result.
// No loot logic, no inventory simulation, no reward interpretation.
func renderLootPanel(width int, pr playerReadResult, er encounterReadResult, pk pickupResult) string {
	title := panelTitleStyle.Render("Loot")

	var items []string

	// Show pickup submission result if any
	if pk.State == pickupStateSent {
		items = append(items, panelItemStyle.Render("  pickup: accepted"))
	} else if pk.State == pickupStateFailed {
		items = append(items, panelItemStyle.Render("  pickup: failed"))
	}

	// Find active encounter for loot readback
	if !pr.HasActiveEncounter {
		if len(items) == 0 {
			items = append(items, panelItemStyle.Render("  none"))
		}
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	if er.State != encounterReadOK {
		items = append(items, panelItemStyle.Render("  enc: unavailable"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	enc := findPlayerEncounter(er.Encounters, pr.ActiveEncounterID)
	if enc == nil {
		items = append(items, panelItemStyle.Render("  enc: no details"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	// Show encounter completion state
	if enc.State != "Completed" {
		items = append(items, panelItemStyle.Render("  enc: "+enc.State))
		if len(items) == 0 {
			items = append(items, panelItemStyle.Render("  none"))
		}
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	// Encounter is completed — show loot truth
	if enc.LootExpired {
		items = append(items, panelItemStyle.Render("  loot: expired"))
	} else if !enc.DropsGenerated {
		items = append(items, panelItemStyle.Render("  drops: none"))
	} else if len(enc.Drops) == 0 {
		items = append(items, panelItemStyle.Render("  drops: 0 remaining"))
	} else {
		items = append(items, panelItemStyle.Render(fmt.Sprintf("  drops: %d", len(enc.Drops))))
		// Show up to 3 item IDs compactly
		maxShow := 3
		if len(enc.Drops) < maxShow {
			maxShow = len(enc.Drops)
		}
		for i := 0; i < maxShow; i++ {
			items = append(items, panelItemStyle.Render("  "+truncateID(enc.Drops[i], width-6)))
		}
		if len(enc.Drops) > 3 {
			items = append(items, panelItemStyle.Render(fmt.Sprintf("  +%d more", len(enc.Drops)-3)))
		}
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderSideColumn stacks the side panels vertically.
func renderSideColumn(width int, target backendTarget, zr zoneReadResult, mr mapReadResult, mobr mobReadResult, pr playerReadResult, er encounterReadResult, focus rosterFocus, tc targetConfirmResult, ar attackResult, pk pickupResult) string {
	nearby := renderNearbyPanel(width)
	encounter := renderEncounterPanel(width, pr, er, focus)
	proximity := renderProximityPanel(width, tc)
	combat := renderCombatPanel(width, ar, pr, er)
	loot := renderLootPanel(width, pr, er, pk)
	status := renderStatusPanel(width, target, zr, mr, mobr, pr)
	return lipgloss.JoinVertical(lipgloss.Left, nearby, "", encounter, "", proximity, "", combat, "", loot, "", status)
}

// renderFooter returns the footer help strip with status labels.
func renderFooter(width int, intentPreview string, focusLabel string, targetLabel string, attackLabel string, pickupLabel string) string {
	help := footerHelp
	if intentPreview != "" {
		help = intentPreview + "  " + help
	}
	if pickupLabel != "" {
		help = pickupLabel + "  " + help
	}
	if attackLabel != "" {
		help = attackLabel + "  " + help
	}
	if targetLabel != "" {
		help = targetLabel + "  " + help
	}
	if focusLabel != "" {
		help = focusLabel + "  " + help
	}
	return footerStyle.Width(width).Render(help)
}

// renderLayout composes all sections into the full view.
func renderLayout(width, height int, lastInput string, target backendTarget, zr zoneReadResult, mr mapReadResult, mobr mobReadResult, pr playerReadResult, er encounterReadResult, focus rosterFocus, entries []rosterEntry, tc targetConfirmResult, ar attackResult, pk pickupResult) string {
	header := renderHeader(width)
	focusLabel := focusPreviewLabel(focus, entries)
	targetLabel := tc.targetStatusLabel()
	attackLabel := ar.attackStatusLabel()
	pickupLabel := pk.pickupStatusLabel()
	footer := renderFooter(width, lastInput, focusLabel, targetLabel, attackLabel, pickupLabel)
	mapPanel := renderMapPanel(mr, mobr, pr, focus, entries)

	// Body height is total minus header (1 line) and footer (1 line)
	bodyHeight := height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	if width >= sidePanelMinWidth {
		// Side-by-side: map on left, info panels on right
		sideCol := renderSideColumn(sidePanelWidth, target, zr, mr, mobr, pr, er, focus, tc, ar, pk)
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
