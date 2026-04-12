package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	headerTitle    = "seq-tui"
	headerSubtitle = "spatial view"
	footerHelp = "hjkl:mv g:step tab:ros []:lt t:prx a:atk p:pk r:res d:diag q"

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

	// Event stream emphasis styles — presentational only, no gameplay semantics.
	// Strong: combat results (hits, victory, defeat) — the "what just happened" signal.
	// Medium: actions and denials — player-initiated events.
	// Dim: movement and focus — routine navigational events.
	eventStrongStyle = lipgloss.NewStyle().Bold(true)
	eventMediumStyle = lipgloss.NewStyle()
	eventDimStyle    = lipgloss.NewStyle().Faint(true)
	// Latest event marker — visually strongest, uses bold + ">" prefix.
	eventLatestStyle = lipgloss.NewStyle().Bold(true)
	// Resolved header style — state anchor, slightly stronger than dim.
	resolvedHeaderStyle = lipgloss.NewStyle().Bold(true).Faint(true)
)

// Spatial entity glyph styles — presentational only, no gameplay semantics.
// Player green does not imply health/safety. Mob red does not imply threat/aggro.
// Uses lipgloss styles with the default renderer. Tests must force a stable color
// profile via lipgloss.SetColorProfile() to get deterministic ANSI output.
var (
	playerGlyphStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))            // green
	focusedPlayerGlyphStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)  // green bold
	mobGlyphStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))             // red
	focusedMobGlyphStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)  // red bold
	wallGlyphStyle          = lipgloss.NewStyle().Faint(true)                                // dim
	attackTargetGlyphStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)  // yellow bold
	hitTargetGlyphStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true) // bright white bold — hit impact
	denyTargetGlyphStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true) // red dim — denied
	playerCombatGlyphStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)  // cyan bold — in encounter
)

// colorizeMapContent applies presentational color to entity glyphs in the viewport.
// Walks each character and wraps recognized entity markers with lipgloss styles.
// ANSI escape sequences are zero-width in terminals, so cell width is preserved.
// This is purely visual — colors carry no gameplay meaning.
func colorizeMapContent(mapContent string) string {
	var sb strings.Builder
	for _, ch := range mapContent {
		switch ch {
		case '@':
			sb.WriteString(playerGlyphStyle.Render("@"))
		case '*':
			sb.WriteString(playerCombatGlyphStyle.Render("*"))
		case '&':
			sb.WriteString(focusedPlayerGlyphStyle.Render("&"))
		case 'm':
			sb.WriteString(mobGlyphStyle.Render("m"))
		case 'M':
			sb.WriteString(focusedMobGlyphStyle.Render("M"))
		case 'X':
			sb.WriteString(attackTargetGlyphStyle.Render("X"))
		case '!':
			sb.WriteString(hitTargetGlyphStyle.Render("!"))
		case '?':
			sb.WriteString(denyTargetGlyphStyle.Render("?"))
		case '2', '3', '4', '5', '6', '7', '8', '9', '+':
			// Multi-mob count glyphs — same color as regular mobs
			sb.WriteString(mobGlyphStyle.Render(string(ch)))
		case '#':
			sb.WriteString(wallGlyphStyle.Render("#"))
		case '\n':
			sb.WriteByte('\n')
		default:
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}

// renderHeader returns the header strip.
func renderHeader(width int) string {
	title := headerStyle.Render(headerTitle)
	subtitle := subtitleStyle.Render(headerSubtitle)
	line := lipgloss.JoinHorizontal(lipgloss.Top, title, subtitle)
	return lipgloss.NewStyle().Width(width).Render(line)
}

// attackTargetGlyph returns the glyph to use for the attack target based on
// the latest combat result. Uses backend encounter truth to differentiate
// hit impact from attack submission from denial. Purely presentational.
func attackTargetGlyph(latestResultKind string) rune {
	switch {
	case latestResultKind == "damage_applied" || latestResultKind == "heal_applied":
		return '!' // hit/heal impact
	case isDenialResult(latestResultKind):
		return '?' // denied
	default:
		return 'X' // attack submitted or other
	}
}

// renderMapPanel returns the map inside a bordered panel.
// Uses backend-sourced map when available, falls back to placeholder with overlays.
// Focus projection overlays the focused roster entry's map position purely visually.
// panelWidth and panelHeight are the available space for the bordered panel.
// When a backend map with valid dimensions is available, a viewport is extracted
// centered on the player position (or map center if no player).
func renderMapPanel(mr mapReadResult, mobr mobReadResult, pr playerReadResult, focus rosterFocus, entries []rosterEntry, panelWidth, panelHeight int, ar attackResult, latestResultKind string) string {
	var mapContent string
	if mr.State == mapReadOK && mr.MapText != "" {
		// Viewport content dimensions (inside border + padding)
		vpWidth := panelWidth - 4 // 2 border + 2 padding
		vpHeight := panelHeight - 2 // 2 border
		if vpWidth < 1 {
			vpWidth = 1
		}
		if vpHeight < 1 {
			vpHeight = 1
		}

		// Determine center point for viewport
		var centerX, centerZ float64
		hasCenter := false
		if pr.State == playerReadOK && pr.HasPos {
			centerX = pr.Position.X
			centerZ = pr.Position.Y // player Y maps to world Z
			hasCenter = true
		}

		if len(mr.Lines) > 0 && vpWidth > 0 && vpHeight > 0 {
			// Adaptive path: re-rasterize at viewport resolution with adaptive world bounds.
			// Smaller viewports get a tighter local world window with native-resolution detail.
			if !hasCenter {
				centerX = (mr.Bounds.MinX + mr.Bounds.MaxX) / 2
				centerZ = (mr.Bounds.MinZ + mr.Bounds.MaxZ) / 2
			}
			ascii, vpBounds := rasterizeAdaptiveViewport(mr.Lines, mr.Bounds, centerX, centerZ, vpWidth, vpHeight)
			mapContent = ascii

			// Overlays use viewport-local bounds and dimensions.
			// Ordering: mobs → focused mob → attack target → player (last, always visible)
			if mobr.State == mobReadOK && len(mobr.Mobs) > 0 {
				mapContent = overlayMobs(mapContent, mobr.Mobs, vpBounds, vpWidth, vpHeight)
			}
			if fe := focusedEntry(focus, entries); fe != nil {
				if fe.kind == "mb" && mobr.State == mobReadOK {
					mapContent = overlayFocusedMob(mapContent, mobr.Mobs, fe.id, vpBounds, vpWidth, vpHeight)
				}
			}
			if ar.State == attackStateSent && ar.TargetID != "" && mobr.State == mobReadOK {
				mapContent = overlayAttackTarget(mapContent, mobr.Mobs, ar.TargetID, vpBounds, vpWidth, vpHeight, attackTargetGlyph(latestResultKind))
			}
			// Player marker last — always visible, never hidden by mob overlays
			if pr.State == playerReadOK && pr.HasPos {
				mapContent = overlayPlayer(mapContent, pr.Position, vpBounds, vpWidth, vpHeight, pr.HasActiveEncounter)
			}
			if fe := focusedEntry(focus, entries); fe != nil {
				if fe.kind == "pc" && pr.State == playerReadOK && pr.HasPos {
					mapContent = overlayFocusedPlayer(mapContent, pr.Position, vpBounds, vpWidth, vpHeight)
				}
			}
		} else {
			// Legacy path: overlay on pre-rasterized canvas, then extract viewport
			// Ordering: mobs → focused mob → attack target → player (last, always visible)
			mapContent = mr.MapText
			if mobr.State == mobReadOK && len(mobr.Mobs) > 0 {
				mapContent = overlayMobs(mapContent, mobr.Mobs, mr.Bounds, mr.MapWidth, mr.MapHeight)
			}
			if fe := focusedEntry(focus, entries); fe != nil {
				if fe.kind == "mb" && mobr.State == mobReadOK {
					mapContent = overlayFocusedMob(mapContent, mobr.Mobs, fe.id, mr.Bounds, mr.MapWidth, mr.MapHeight)
				}
			}
			if ar.State == attackStateSent && ar.TargetID != "" && mobr.State == mobReadOK {
				mapContent = overlayAttackTarget(mapContent, mobr.Mobs, ar.TargetID, mr.Bounds, mr.MapWidth, mr.MapHeight, attackTargetGlyph(latestResultKind))
			}
			// Player marker last — always visible
			if pr.State == playerReadOK && pr.HasPos {
				mapContent = overlayPlayer(mapContent, pr.Position, mr.Bounds, mr.MapWidth, mr.MapHeight, pr.HasActiveEncounter)
			}
			if fe := focusedEntry(focus, entries); fe != nil {
				if fe.kind == "pc" && pr.State == playerReadOK && pr.HasPos {
					mapContent = overlayFocusedPlayer(mapContent, pr.Position, mr.Bounds, mr.MapWidth, mr.MapHeight)
				}
			}
			if mr.MapWidth > 0 && mr.MapHeight > 0 {
				var centerCol, centerRow int
				if hasCenter {
					centerCol, centerRow = mr.Bounds.projectToCell(centerX, centerZ, mr.MapWidth, mr.MapHeight)
				} else {
					centerCol = mr.MapWidth / 2
					centerRow = mr.MapHeight / 2
				}
				mapContent = extractViewport(mapContent, mr.MapWidth, mr.MapHeight, centerCol, centerRow, vpWidth, vpHeight)
			}
		}
		// Apply presentational color after all overlays.
		// Must happen after rasterization/extraction since ANSI codes would break rune-based slicing.
		mapContent = colorizeMapContent(mapContent)
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
		panelItemStyle.Render("  " + target.Zone),
		panelItemStyle.Render("  " + strings.ToLower(target.Mode) + "/" + vis),
		panelItemStyle.Render("  " + zr.statusLabel()),
		panelItemStyle.Render("  " + mr.mapStatusLabel()),
		panelItemStyle.Render("  " + mobr.mobStatusLabel()),
		panelItemStyle.Render("  " + pr.playerStatusLabel()),
	}
	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderEncounterPanel returns the encounter status panel based on backend-owned facts.
// playerID is used to mark self in the roster with `*` suffix.
func renderEncounterPanel(width int, pr playerReadResult, er encounterReadResult, focus rosterFocus, playerID string) string {
	title := panelTitleStyle.Render(encounterTitle)

	var items []string

	if pr.State != playerReadOK {
		items = append(items, panelItemStyle.Render("  no player"))
	} else if er.State == encounterReadFailed {
		items = append(items, panelItemStyle.Render("  unavailable"))
	} else if er.State == encounterReadNotAttempted {
		items = append(items, panelItemStyle.Render("  pending"))
	} else {
		if pr.HasActiveEncounter {
			// Find matching encounter summary for detail
			if enc := findPlayerEncounter(er.Encounters, pr.ActiveEncounterID); enc != nil {
				if enc.State == "Completed" {
					// Compact one-line completion summary
					reason := enc.CompletedReason
					if reason == "" {
						reason = "completed"
					}
					items = append(items, panelItemStyle.Render(fmt.Sprintf("  done:%s", truncateID(reason, width-9))))
					items = append(items, panelItemStyle.Render(fmt.Sprintf("  %dp/%dm %da/%dd", enc.PlayerCount, enc.MobCount, enc.MobsAlive, enc.MobsDead)))
				} else {
					// Active encounter: state + counts on one line
					items = append(items, panelItemStyle.Render(fmt.Sprintf("  %s %dp/%dm", enc.State, enc.PlayerCount, enc.MobCount)))
					// Compact: alive/dead + action index on one line
					items = append(items, panelItemStyle.Render(fmt.Sprintf("  %da/%dd act:%d", enc.MobsAlive, enc.MobsDead, enc.ActionIndex)))
				}
				// Roster: backend-owned participant lists with local focus
				items = append(items, renderRosterSection(enc, width, focus, playerID)...)
			} else {
				items = append(items, panelItemStyle.Render("  no enc details"))
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
// Local focus indicator (>) shown on the focused entry. Self marked with `*`.
// Read-only, no selection with gameplay meaning.
func renderRosterSection(enc *encounterSummary, panelWidth int, focus rosterFocus, playerID string) []string {
	var lines []string
	// Available width inside the panel: panelWidth - border(2) - padding(2) - indent(2) - self marker(1)
	maxIDWidth := panelWidth - 7
	if maxIDWidth < 4 {
		maxIDWidth = 4
	}

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
		suffix := ""
		if pid == playerID {
			suffix = "*"
		}
		label := prefix + "pc:" + truncateID(pid, maxIDWidth-4) + suffix
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

// renderProximityPanel returns a compact panel showing target/proximity data.
// When a mob is focused, shows locally-computed distance and range status.
// When backend proximity data is available, enriches with authoritative range.
// Falls back to nearest mob when no target is focused.
func renderProximityPanel(width int, tc targetConfirmResult, ar attackResult, mobr mobReadResult, pr playerReadResult, focus rosterFocus, entries []rosterEntry) string {
	title := panelTitleStyle.Render("Target")

	var items []string

	fe := focusedEntry(focus, entries)

	// If a mob is focused and we have position data, show target info
	if fe != nil && fe.kind == "mb" && pr.State == playerReadOK && pr.HasPos && mobr.State == mobReadOK {
		mobPos := findMobPosition(mobr.Mobs, fe.id)
		if mobPos != nil {
			dx := mobPos.X - pr.Position.X
			dy := mobPos.Y - pr.Position.Y
			dist := math.Sqrt(dx*dx + dy*dy)

			// Target identity — show canonical mob_id for readability, fall back to live ID
			displayID := fe.mobID
			if displayID == "" {
				displayID = fe.id
			}
			nameLabel := truncateID(displayID, width-6)
			items = append(items, panelItemStyle.Render("  "+nameLabel))

			// Distance and range status
			// Use backend proximity truth if available for this target
			if tc.State == targetConfirmOK && tc.TargetID == fe.id && tc.Found {
				if tc.WithinProximity {
					items = append(items, panelItemStyle.Render(fmt.Sprintf("  IN RANGE dist:%.0f", tc.Distance)))
				} else {
					items = append(items, panelItemStyle.Render(fmt.Sprintf("  OOR dist:%.0f", tc.Distance)))
				}
			} else {
				// Local distance estimate (no authoritative range check)
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  dist:%.0f", dist)))
			}
		} else {
			displayID := fe.mobID
			if displayID == "" {
				displayID = fe.id
			}
			items = append(items, panelItemStyle.Render("  "+truncateID(displayID, width-6)))
			items = append(items, panelItemStyle.Render("  pos unknown"))
		}
	} else if tc.State == targetConfirmOK {
		// Backend proximity data for a target (fallback path)
		nameLabel := ""
		if tc.MobName != "" {
			nameLabel = truncateID(tc.MobName, width-11)
		} else {
			nameLabel = tc.TargetKind + ":" + truncateID(tc.TargetID, width-14)
		}
		items = append(items, panelItemStyle.Render("  "+nameLabel))
		if !tc.Found {
			items = append(items, panelItemStyle.Render("  not found"))
		} else if tc.WithinProximity {
			items = append(items, panelItemStyle.Render(fmt.Sprintf("  IN RANGE dist:%.0f", tc.Distance)))
		} else {
			items = append(items, panelItemStyle.Render(fmt.Sprintf("  OOR dist:%.0f", tc.Distance)))
		}
	} else {
		// No target — show nearest mob as orientation aid
		if pr.State == playerReadOK && pr.HasPos && mobr.State == mobReadOK && len(mobr.Mobs) > 0 {
			nearID, nearDist := nearestMob(pr.Position, mobr.Mobs)
			if nearID != "" {
				items = append(items, panelItemStyle.Render("  nearest:"))
				items = append(items, panelItemStyle.Render("  "+truncateID(nearID, width-6)))
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  dist:%.0f", nearDist)))
			} else {
				items = append(items, panelItemStyle.Render("  no target"))
			}
		} else {
			items = append(items, panelItemStyle.Render("  no target"))
		}
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// nearestMob returns the ID and distance of the nearest mob to the player position.
// Pure read of mob positions — no gameplay inference, no targeting authority.
func nearestMob(playerPos playerPosResult, mobs []mobPosition) (string, float64) {
	if len(mobs) == 0 {
		return "", 0
	}
	bestID := ""
	bestDist := math.MaxFloat64
	for _, mob := range mobs {
		dx := mob.Position.X - playerPos.X
		dy := mob.Position.Y - playerPos.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < bestDist {
			bestDist = dist
			bestID = mob.MobID
			if bestID == "" {
				bestID = mob.ProcessID
			}
		}
	}
	return bestID, bestDist
}

// mobsEngagingPlayer returns the IDs of mobs whose selected target is the given player.
// Purely a read of backend-owned mob_threat data. No threat inference.
func mobsEngagingPlayer(enc *encounterSummary, playerID string) []string {
	if enc == nil || playerID == "" {
		return nil
	}
	var engaged []string
	for _, mt := range enc.MobThreat {
		if mt.SelectedTargetPlayerID == playerID {
			engaged = append(engaged, mt.MobID)
		}
	}
	return engaged
}

// isMobEngagingPlayer returns true if the given mob is targeting the given player
// according to backend mob_threat data. No threat inference.
func isMobEngagingPlayer(enc *encounterSummary, mobID, playerID string) bool {
	if enc == nil || playerID == "" {
		return false
	}
	for _, mt := range enc.MobThreat {
		if mt.MobID == mobID && mt.SelectedTargetPlayerID == playerID {
			return true
		}
	}
	return false
}

// compactResultKind maps backend result kind strings to compact labels.
// Unknown kinds are returned as-is (preserves backend truth).
func compactResultKind(kind string) string {
	switch kind {
	case "damage_applied":
		return "dmg"
	case "attack_miss":
		return "miss"
	case "heal_applied":
		return "heal"
	case "out_of_range":
		return "oor"
	case "los_blocked":
		return "los"
	case "cooldown_set":
		return "cd"
	case "blocked_wounded":
		return "blocked"
	case "invalid_target":
		return "bad_tgt"
	default:
		return kind
	}
}

// isDenialResult returns true if the backend result kind represents an action
// denial (target not reachable / not valid) rather than combat resolution.
// Used to distinguish "encounter active, action denied" from "combat resolving".
func isDenialResult(kind string) bool {
	switch kind {
	case "out_of_range", "los_blocked", "invalid_target":
		return true
	default:
		return false
	}
}

// renderCombatMobRoster returns lines showing per-mob status in the encounter.
// Uses backend MobIDs order (deterministic). Indicators:
//   - `>` prefix = your current attack target (from ar.TargetID)
//   - `<-` suffix = mob is engaging/targeting you (from mob_threat)
//   - `~` suffix = tab-focused mob (from focusedMobID)
//
// Read-only from backend truth. No threat calculation.
func renderCombatMobRoster(enc *encounterSummary, ar attackResult, playerID string, maxWidth int, focusedMobID string) []string {
	if enc == nil {
		return nil
	}

	// Determine the label for a target no longer in the roster.
	// Uses backend encounter state — no inference.
	goneLabel := "(gone)"
	if enc.CompletedReason == "all_mobs_dead" {
		goneLabel = "(dead)"
	}

	// Hint for retargeting when target is gone during active combat
	retargetHint := ""
	if enc.State != "Completed" {
		retargetHint = " tab>a"
	}

	// If no mobs in roster but attack target exists, show it
	if len(enc.MobIDs) == 0 {
		if ar.State == attackStateSent && ar.TargetID != "" {
			idWidth := maxWidth - 12
			if idWidth < 4 {
				idWidth = 4
			}
			return []string{
				panelItemStyle.Render("> " + truncateID(ar.TargetID, idWidth) + " " + goneLabel + retargetHint),
			}
		}
		return nil
	}

	var lines []string

	// Max ID width: maxWidth - prefix(2) - suffix(3) - padding(2) = maxWidth-7
	idWidth := maxWidth - 7
	if idWidth < 4 {
		idWidth = 4
	}

	for _, mid := range enc.MobIDs {
		prefix := "  "
		if ar.State == attackStateSent && ar.TargetID == mid {
			prefix = "> "
		}
		suffix := ""
		if focusedMobID == mid {
			suffix = "~"
		}
		if isMobEngagingPlayer(enc, mid, playerID) {
			suffix += "<-"
		}
		if suffix != "" {
			suffix = " " + suffix
		}
		label := prefix + truncateID(mid, idWidth) + suffix
		lines = append(lines, panelItemStyle.Render(label))
	}

	// If attack target is no longer in the roster, show it explicitly
	if ar.State == attackStateSent && ar.TargetID != "" {
		found := false
		for _, mid := range enc.MobIDs {
			if mid == ar.TargetID {
				found = true
				break
			}
		}
		if !found {
			lines = append(lines, panelItemStyle.Render("> "+truncateID(ar.TargetID, idWidth)+" "+goneLabel+retargetHint))
		}
	}

	return lines
}

// findMobThreatHP returns hp_current, hp_max for a mob from the encounter's mob_threat
// entries. Returns 0, 0 if not found. Backend truth only.
func findMobThreatHP(enc *encounterSummary, mobID string) (int, int) {
	if enc == nil || mobID == "" {
		return 0, 0
	}
	for _, mt := range enc.MobThreat {
		if mt.MobID == mobID {
			return mt.HPCurrent, mt.HPMax
		}
	}
	return 0, 0
}

// renderHPBar returns a compact ASCII bar representation of hp/max.
// Example: [████████  ] 80/100
// Width is the total available characters including the bar and numbers.
func renderHPBar(current, max, barWidth int) string {
	if max <= 0 || barWidth < 4 {
		return fmt.Sprintf("%d/%d", current, max)
	}
	filled := 0
	if max > 0 && current > 0 {
		filled = (current * barWidth) / max
		if filled < 1 && current > 0 {
			filled = 1
		}
	}
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("·", barWidth-filled)
	return fmt.Sprintf("%s %d/%d", bar, current, max)
}

// renderCombatPanel returns a compact panel showing backend-owned combat readback.
// Shows encounter state, attack resolution, target status, and mob engagement
// from backend truth without any client-side combat logic or interpretation.
func renderCombatPanel(width int, ar attackResult, pr playerReadResult, er encounterReadResult, target backendTarget, inv inventoryReadResult, focusedMobID string) string {
	title := panelTitleStyle.Render("Combat")

	var items []string

	// Show encounter-level combat readback if player is in an active encounter,
	// regardless of whether an attack has been submitted
	if pr.State == playerReadOK && pr.HasActiveEncounter && er.State == encounterReadOK {
		enc := findPlayerEncounter(er.Encounters, pr.ActiveEncounterID)
		if enc != nil {
			// Compact state line with alive/dead counts merged
			if enc.State == "Completed" {
				// Completion: compact one-line with phase suffix
				reason := enc.CompletedReason
				if reason == "" {
					reason = "done"
				}
				if enc.DropsGenerated && len(enc.Drops) > 0 && !enc.LootExpired {
					items = append(items, panelItemStyle.Render("  "+truncateID(reason, width-8)+"/L"))
				} else {
					items = append(items, panelItemStyle.Render("  "+truncateID(reason, width-6)))
				}
			} else {
				// Active encounter: compact HP combat frame
				barW := width - 14 // space for label + numbers
				if barW < 3 {
					barW = 3
				}

				// Player HP line
				if inv.State == inventoryReadOK && inv.HasLifecycle && inv.HPMax > 0 {
					items = append(items, eventStrongStyle.Render("  You "+renderHPBar(inv.HPCurrent, inv.HPMax, barW)))
				}

				// Target HP line — use attack target or focused mob
				targetMobID := ""
				if ar.State == attackStateSent && ar.TargetID != "" {
					targetMobID = ar.TargetID
				} else if focusedMobID != "" {
					targetMobID = focusedMobID
				}
				if targetMobID != "" {
					hpC, hpM := findMobThreatHP(enc, targetMobID)
					if hpM > 0 {
						mobLabel := truncateID(targetMobID, 6)
						items = append(items, panelItemStyle.Render("  "+mobLabel+" "+renderHPBar(hpC, hpM, barW)))
					}
				}

				// Compact state line
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  %da/%dd act:%d", enc.MobsAlive, enc.MobsDead, enc.ActionIndex)))
			}

			// Active-only sections: attack status, result, roster, summary
			// Suppressed when encounter is Completed (combat over — these are stale)
			if enc.State != "Completed" {

				// Attack submission status
				if ar.State == attackStateSent {
					label := "  atk:" + truncateID(ar.TargetID, width-8)
					items = append(items, panelItemStyle.Render(label))
				} else if ar.State == attackStateFailed {
					label := "  atk:fail"
					if ar.Error != "" {
						label += " " + truncateID(ar.Error, width-14)
					}
					items = append(items, panelItemStyle.Render(label))
				}

				// Backend-owned latest attack result — compact wording.
				// Denial results (oor, los, bad_tgt) use "deny:" prefix to
				// distinguish "action blocked" from "combat resolving" (res:).
				if enc.LatestResultKind != "" {
					kind := compactResultKind(enc.LatestResultKind)
					prefix := "res"
					if isDenialResult(enc.LatestResultKind) {
						prefix = "deny"
					}
					resultLabel := fmt.Sprintf("  %s:%s", prefix, kind)
					if enc.LatestResultValue > 0 {
						resultLabel += fmt.Sprintf(" %d", enc.LatestResultValue)
					}
					items = append(items, panelItemStyle.Render(resultLabel))
				}

				// Per-mob roster
				rosterLines := renderCombatMobRoster(enc, ar, target.Player, width-4, focusedMobID)
				items = append(items, rosterLines...)

				// Backend-owned text summary
				if enc.TextSummaryLatest != "" {
					items = append(items, panelItemStyle.Render("  "+truncateID(enc.TextSummaryLatest, width-6)))
				}
			}

			content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
			return panelBorderStyle.Width(width - 4).Render(content)
		}
	}

	// No active encounter — show joined status or attack submission state
	if pr.State != playerReadOK {
		items = append(items, panelItemStyle.Render("  not joined"))
	} else if ar.State == attackStateNone {
		items = append(items, panelItemStyle.Render("  none"))
	} else if ar.State == attackStateSent {
		// Attack accepted but encounter read hasn't arrived yet.
		// "opening" signals that an encounter is expected imminently.
		items = append(items, panelItemStyle.Render("  atk:"+truncateID(ar.TargetID, width-8)))
		items = append(items, panelItemStyle.Render("  enc: opening"))
	} else {
		items = append(items, panelItemStyle.Render("  atk:fail"))
		if ar.Error != "" {
			items = append(items, panelItemStyle.Render("  "+truncateID(ar.Error, width-6)))
		}
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderLootPanel returns a compact panel showing backend-owned loot readback.
// Read-only display of drop state from encounter summary, plus pickup submission result.
// No loot logic, no inventory simulation, no reward interpretation.
func renderLootPanel(width int, pr playerReadResult, er encounterReadResult, pk pickupResult, inv inventoryReadResult, invAtPickup int, lootFocus int) string {
	title := panelTitleStyle.Render("Loot")

	var items []string

	// Show pickup submission result — suppress when inventory delta already confirmed
	pickupConfirmed := pk.State == pickupStateSent && inv.State == inventoryReadOK && invAtPickup >= 0 && inv.Count > invAtPickup
	if pk.State == pickupStateSent && !pickupConfirmed {
		label := "  pk:" + truncateID(pk.ItemID, width-7)
		items = append(items, panelItemStyle.Render(label))
	} else if pk.State == pickupStateFailed {
		label := "  pk:fail"
		if pk.Error != "" {
			label += " " + truncateID(pk.Error, width-13)
		}
		items = append(items, panelItemStyle.Render(label))
	}

	// Find active encounter for loot readback
	if pr.State != playerReadOK {
		items = append(items, panelItemStyle.Render("  not joined"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}
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
		items = append(items, panelItemStyle.Render("  loot: none"))
	} else if len(enc.Drops) == 0 {
		items = append(items, panelItemStyle.Render("  loot: collected"))
	} else {
		items = append(items, panelItemStyle.Render("  loot: ready"))
		items = append(items, panelItemStyle.Render(fmt.Sprintf("  drops: %d", len(enc.Drops))))
		// Show drop rows with selection indicator
		maxShow := 3
		if len(enc.Drops) < maxShow {
			maxShow = len(enc.Drops)
		}
		for i := 0; i < maxShow; i++ {
			prefix := "  "
			if lootFocus == i {
				prefix = "> "
			}
			dropLabel := truncateID(enc.Drops[i], width-10)
			// Mark the item that was submitted for pickup (suppress once confirmed)
			if pk.State == pickupStateSent && !pickupConfirmed && pk.ItemID == enc.Drops[i] {
				dropLabel += " [pk]"
			}
			items = append(items, panelItemStyle.Render(prefix+dropLabel))
		}
		if len(enc.Drops) > 3 {
			items = append(items, panelItemStyle.Render(fmt.Sprintf("  +%d more", len(enc.Drops)-3)))
		}
	}

	// Show backend-owned inventory confirmation — compact form
	if inv.State == inventoryReadOK {
		if pk.State == pickupStateSent && invAtPickup >= 0 {
			delta := inv.Count - invAtPickup
			if delta > 0 {
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  inv:+%d", delta)))
			} else {
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  inv:%d pending", inv.Count)))
			}
		} else {
			items = append(items, panelItemStyle.Render(fmt.Sprintf("  inv: %d", inv.Count)))
		}
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderPlayerPanel returns a compact panel showing backend-owned player lifecycle state.
// Displays HP, can-act status, blocked reason, and encounter membership.
// All values are read-only from backend truth — no client-side inference.
func renderPlayerPanel(width int, pr playerReadResult, inv inventoryReadResult, rs respawnResult) string {
	title := panelTitleStyle.Render("Player")

	var items []string

	if pr.State != playerReadOK {
		items = append(items, panelItemStyle.Render("  not joined"))
		content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	// Lifecycle fields from gameplay_status — most important state first
	if inv.State == inventoryReadOK && inv.HasLifecycle {
		// State precedence: dead → blocked → alive
		if inv.HPMax > 0 && inv.HPCurrent <= 0 {
			// Dead: highest priority — show prominently with HP
			items = append(items, panelItemStyle.Render(fmt.Sprintf("  DEAD %d/%d", inv.HPCurrent, inv.HPMax)))
			if rs.State == respawnStateSent {
				items = append(items, panelItemStyle.Render("  respawn: sent"))
			} else if rs.State == respawnStateFailed {
				items = append(items, panelItemStyle.Render("  respawn: failed"))
			}
		} else if !inv.CanAct {
			// Blocked but alive: show reason prominently
			label := "  blocked"
			if inv.BlockedReason != "" {
				label += ":" + truncateID(inv.BlockedReason, width-12)
			}
			items = append(items, panelItemStyle.Render(label))
			if inv.HPMax > 0 {
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  hp: %d/%d", inv.HPCurrent, inv.HPMax)))
			}
		} else {
			// Alive and can act: normal display
			if inv.HPMax > 0 {
				items = append(items, panelItemStyle.Render(fmt.Sprintf("  hp: %d/%d", inv.HPCurrent, inv.HPMax)))
			}
		}

		// Encounter membership
		if pr.HasActiveEncounter {
			items = append(items, panelItemStyle.Render("  enc: active"))
		} else {
			items = append(items, panelItemStyle.Render("  enc: none"))
		}
	} else if inv.State == inventoryReadFailed {
		items = append(items, panelItemStyle.Render("  status: unavailable"))
	} else {
		// No lifecycle yet — show enc membership from player read
		if pr.HasActiveEncounter {
			items = append(items, panelItemStyle.Render("  enc: active"))
		} else {
			items = append(items, panelItemStyle.Render("  enc: none"))
		}
		items = append(items, panelItemStyle.Render("  status: pending"))
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderOutcomePanel returns a compact post-combat outcome summary.
// Only rendered when the player's encounter is completed. Shows victory/defeat,
// XP awarded, and loot status from backend truth only. Returns empty string
// when there is no completed encounter to summarize.
func renderOutcomePanel(width int, pr playerReadResult, er encounterReadResult) string {
	if pr.State != playerReadOK || !pr.HasActiveEncounter || er.State != encounterReadOK {
		return ""
	}
	enc := findPlayerEncounter(er.Encounters, pr.ActiveEncounterID)
	if enc == nil || enc.State != "Completed" {
		return ""
	}

	title := panelTitleStyle.Render("Outcome")
	var items []string

	// Victory / defeat label from backend completion reason
	switch enc.CompletedReason {
	case "all_mobs_dead":
		items = append(items, panelItemStyle.Render("  victory"))
	case "all_players_dead":
		items = append(items, panelItemStyle.Render("  defeat"))
	default:
		reason := enc.CompletedReason
		if reason == "" {
			reason = "complete"
		}
		items = append(items, panelItemStyle.Render("  "+truncateID(reason, width-6)))
	}

	// XP awarded — backend-owned, no client computation
	if enc.XPAwarded > 0 {
		items = append(items, panelItemStyle.Render(fmt.Sprintf("  xp +%d", enc.XPAwarded)))
	}

	// Loot status from backend truth
	if enc.DropsGenerated {
		if enc.LootExpired {
			items = append(items, panelItemStyle.Render("  loot expired"))
		} else if len(enc.Drops) > 0 {
			items = append(items, panelItemStyle.Render(fmt.Sprintf("  loot %d items", len(enc.Drops))))
		} else {
			items = append(items, panelItemStyle.Render("  loot collected"))
		}
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// eventEmphasis classifies an event text into strong/medium/dim based on
// simple string prefix checks. Purely presentational — no gameplay semantics.
func eventEmphasis(text string) string {
	switch {
	case strings.HasPrefix(text, "hit "),
		strings.HasPrefix(text, "heal "),
		text == "victory",
		text == "defeat",
		text == "encounter opened",
		text == "encounter complete",
		text == "miss":
		return "strong"
	case strings.HasPrefix(text, "attack"),
		strings.HasPrefix(text, "deny "),
		text == "cooldown",
		text == "blocked":
		return "medium"
	default:
		// move, focus, and anything else
		return "dim"
	}
}

// renderEventLine renders a single event entry with emphasis-based styling.
// The latest event (isLatest=true) gets a ">" prefix and bold treatment.
func renderEventLine(text string, maxWidth int, isLatest bool) string {
	if isLatest {
		return eventLatestStyle.Render("> " + truncateID(text, maxWidth-2))
	}
	emphasis := eventEmphasis(text)
	switch emphasis {
	case "strong":
		return eventStrongStyle.Render("  " + truncateID(text, maxWidth))
	case "medium":
		return eventMediumStyle.Render("  " + truncateID(text, maxWidth))
	default:
		return eventDimStyle.Render("  " + truncateID(text, maxWidth))
	}
}

// resolvedHeader returns a compact one-line outcome summary for display at
// the top of the Events panel when an encounter is completed. Returns empty
// string when no completed encounter exists. Backend truth only.
func resolvedHeader(width int, pr playerReadResult, er encounterReadResult) string {
	if pr.State != playerReadOK || !pr.HasActiveEncounter || er.State != encounterReadOK {
		return ""
	}
	enc := findPlayerEncounter(er.Encounters, pr.ActiveEncounterID)
	if enc == nil || enc.State != "Completed" {
		return ""
	}

	// Build compact summary using tight separators for narrow panels
	var parts []string
	switch enc.CompletedReason {
	case "all_mobs_dead":
		parts = append(parts, "victory")
	case "all_players_dead":
		parts = append(parts, "defeat")
	default:
		parts = append(parts, "done")
	}
	if enc.XPAwarded > 0 {
		parts = append(parts, fmt.Sprintf("+%dxp", enc.XPAwarded))
	}
	if enc.DropsGenerated {
		if enc.LootExpired {
			parts = append(parts, "expired")
		} else if len(enc.Drops) > 0 {
			parts = append(parts, fmt.Sprintf("%dloot", len(enc.Drops)))
		}
	}

	summary := strings.Join(parts, " ")
	// Available content width: panel width - border(2) - padding(2) - "== " prefix(3) - " ==" suffix(3) = width - 10
	contentW := width - 10
	if contentW < 6 {
		contentW = 6
	}
	return resolvedHeaderStyle.Render("== " + truncateID(summary, contentW) + " ==")
}

// renderEventPanel returns a compact panel showing recent gameplay events.
// When an encounter is completed, a resolved header line appears at the top
// for at-a-glance payoff visibility. Backend truth only, no gameplay inference.
func renderEventPanel(width int, events eventLog, pr playerReadResult, er encounterReadResult) string {
	title := panelTitleStyle.Render("Events")

	var items []string

	// Resolved header — compact outcome line at top when encounter completed
	if header := resolvedHeader(width, pr, er); header != "" {
		items = append(items, header)
	}

	entries := events.recent()
	if len(entries) == 0 && len(items) == 0 {
		content := title + "\n" + panelItemStyle.Render("  (none)")
		return panelBorderStyle.Width(width - 4).Render(content)
	}

	maxTextWidth := width - 6 // border(2) + padding(2) + indent(2)
	if maxTextWidth < 4 {
		maxTextWidth = 4
	}
	for i, e := range entries {
		isLatest := i == len(entries)-1 // newest entry is last
		items = append(items, renderEventLine(e.text, maxTextWidth, isLatest))
	}

	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderDiagnosticsSection returns a subordinated diagnostics area combining
// zone status and ambient cues. Uses a single border and dimmed title to
// visually subordinate below gameplay-facing panels.
func renderDiagnosticsSection(width int, target backendTarget, zr zoneReadResult, mr mapReadResult, mobr mobReadResult, pr playerReadResult, showDiag bool) string {
	if !showDiag {
		return ""
	}
	title := dimStyle.Render("Diagnostics")
	vis := strings.ToLower(target.Visibility)
	var items []string
	items = append(items, panelItemStyle.Render("  "+target.Zone+" "+strings.ToLower(target.Mode)+"/"+vis))
	items = append(items, panelItemStyle.Render("  "+zr.statusLabel()))
	items = append(items, panelItemStyle.Render("  "+mobr.mobStatusLabel()))
	items = append(items, panelItemStyle.Render("  "+pr.playerStatusLabel()))
	content := title + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
	return panelBorderStyle.Width(width - 4).Render(content)
}

// renderSideColumn stacks the side panels vertically.
// Hierarchy: tactical (player+combat) → events → details (loot+encounter+proximity) → diagnostics.
func renderSideColumn(width int, target backendTarget, zr zoneReadResult, mr mapReadResult, mobr mobReadResult, pr playerReadResult, er encounterReadResult, focus rosterFocus, entries []rosterEntry, tc targetConfirmResult, ar attackResult, pk pickupResult, inv inventoryReadResult, invAtPickup int, lootFocus int, rs respawnResult, events eventLog, showDiag bool) string {
	// === Tactical section (glanceable at top) ===
	player := renderPlayerPanel(width, pr, inv, rs)
	fmID := ""
	if fe := focusedEntry(focus, entries); fe != nil && fe.kind == "mb" {
		fmID = fe.id
	}
	combat := renderCombatPanel(width, ar, pr, er, target, inv, fmID)

	// === Events section (center of gravity) ===
	eventsPanel := renderEventPanel(width, events, pr, er)

	// === Detail section (contextual) ===
	loot := renderLootPanel(width, pr, er, pk, inv, invAtPickup, lootFocus)
	encounter := renderEncounterPanel(width, pr, er, focus, target.Player)
	proximity := renderProximityPanel(width, tc, ar, mobr, pr, focus, entries)

	// === Diagnostics section (subordinated) ===
	diag := renderDiagnosticsSection(width, target, zr, mr, mobr, pr, showDiag)

	panels := []string{player, combat, eventsPanel, loot, encounter, proximity}
	if diag != "" {
		panels = append(panels, diag)
	}
	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

// renderFooter returns the footer help strip with status labels.
// When playerJoined is false, shows a reduced hint set since most actions require joining.
func renderFooter(width int, intentPreview string, focusLabel string, targetLabel string, attackLabel string, pickupLabel string, playerJoined bool, inv inventoryReadResult) string {
	help := footerHelp
	if !playerJoined {
		help = "hjkl:mv q  (joining...)"
	} else if inv.State == inventoryReadOK && inv.HasLifecycle && inv.HPMax > 0 && inv.HPCurrent <= 0 {
		help = "[r:respawn]  " + help
	} else if inv.State == inventoryReadOK && inv.HasLifecycle && !inv.CanAct {
		help = "[blocked]  " + help
	}
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
func renderLayout(width, height int, lastInput string, target backendTarget, zr zoneReadResult, mr mapReadResult, mobr mobReadResult, pr playerReadResult, er encounterReadResult, focus rosterFocus, entries []rosterEntry, tc targetConfirmResult, ar attackResult, pk pickupResult, inv inventoryReadResult, invAtPickup int, lootFocus int, rs respawnResult, events eventLog, showDiag bool) string {
	header := renderHeader(width)
	focusLabel := focusPreviewLabel(focus, entries)
	targetLabel := tc.targetStatusLabel()
	attackLabel := ar.attackStatusLabel()
	pickupLabel := pk.pickupStatusLabel()
	footer := renderFooter(width, lastInput, focusLabel, targetLabel, attackLabel, pickupLabel, pr.State == playerReadOK, inv)

	// Body height is total minus header (1 line) and footer (1 line)
	bodyHeight := height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Compute available map panel dimensions
	var mapPanelW, mapPanelH int
	if width >= sidePanelMinWidth {
		mapPanelW = width - sidePanelWidth - 2 // 2 = gap between map and side column
	} else {
		mapPanelW = width
	}
	mapPanelH = bodyHeight

	// Extract latest combat result kind for spatial interaction emphasis
	latestResultKind := ""
	if pr.HasActiveEncounter && er.State == encounterReadOK {
		if enc := findPlayerEncounter(er.Encounters, pr.ActiveEncounterID); enc != nil {
			latestResultKind = enc.LatestResultKind
		}
	}
	mapPanel := renderMapPanel(mr, mobr, pr, focus, entries, mapPanelW, mapPanelH, ar, latestResultKind)

	var body string
	if width >= sidePanelMinWidth {
		// Side-by-side: map on left, info panels on right
		sideCol := renderSideColumn(sidePanelWidth, target, zr, mr, mobr, pr, er, focus, entries, tc, ar, pk, inv, invAtPickup, lootFocus, rs, events, showDiag)
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
