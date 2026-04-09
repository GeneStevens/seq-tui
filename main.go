package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// refreshInterval is the cadence for periodic backend read refresh.
const refreshInterval = 500 * time.Millisecond

// refreshTickMsg signals that a periodic refresh should occur.
type refreshTickMsg time.Time

// scheduleRefresh returns a Bubble Tea command that fires a refreshTickMsg.
func scheduleRefresh() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return refreshTickMsg(t)
	})
}

// renderMap overlays landmarks and the player marker onto the static map
// without mutating the original constant.
func renderMap() string {
	lines := strings.Split(staticMap, "\n")

	// Overlay landmarks first
	for _, lm := range landmarks {
		if lm.y >= 0 && lm.y < len(lines) {
			row := []rune(lines[lm.y])
			if lm.x >= 0 && lm.x < len(row) {
				row[lm.x] = lm.glyph
				lines[lm.y] = string(row)
			}
		}
	}

	// Overlay threat markers
	for _, tm := range threatMarkers {
		if tm.y >= 0 && tm.y < len(lines) {
			row := []rune(lines[tm.y])
			if tm.x >= 0 && tm.x < len(row) {
				row[tm.x] = tm.glyph
				lines[tm.y] = string(row)
			}
		}
	}

	// Overlay player marker last so it is always visible
	if playerY >= 0 && playerY < len(lines) {
		line := []rune(lines[playerY])
		if playerX >= 0 && playerX < len(line) {
			line[playerX] = playerMarker
			lines[playerY] = string(line)
		}
	}

	return strings.Join(lines, "\n")
}

var dimStyle = lipgloss.NewStyle().Faint(true)

// tileDistance returns the Euclidean distance from the player to (x, y).
func tileDistance(x, y int) float64 {
	dx := float64(x - playerX)
	dy := float64(y - playerY)
	return math.Sqrt(dx*dx + dy*dy)
}

// renderStyledMap applies awareness-radius dimming to the plain map output.
// Tiles outside the radius are rendered faint. Player marker is never dimmed.
func renderStyledMap() string {
	plain := renderMap()
	lines := strings.Split(plain, "\n")
	var result strings.Builder

	for y, line := range lines {
		if y > 0 {
			result.WriteByte('\n')
		}
		for x, ch := range line {
			if x == playerX && y == playerY {
				// Player marker: always fully visible
				result.WriteRune(ch)
			} else if tileDistance(x, y) <= float64(awarenessRadius) {
				// Inside radius: normal
				result.WriteRune(ch)
			} else {
				// Outside radius: dimmed
				result.WriteString(dimStyle.Render(string(ch)))
			}
		}
	}

	return result.String()
}

// directionFromKey maps a movement key to a direction label.
// Returns empty string if the key is not a movement key.
func directionFromKey(key string) string {
	switch key {
	case "up", "k":
		return "north"
	case "down", "j":
		return "south"
	case "left", "h":
		return "west"
	case "right", "l":
		return "east"
	default:
		return ""
	}
}

// moveState tracks whether a movement intent was preview-only or actually sent.
type moveState int

const (
	moveStateNone moveState = iota
	moveStatePreview
	moveStateSent
	moveStateFailed
)

// moveIntent represents a recognized movement intent.
type moveIntent struct {
	direction string    // "north", "south", "east", "west"
	state     moveState // preview, sent, or failed
}

// preview returns the display string for this intent.
func (i moveIntent) preview() string {
	if i.direction == "" {
		return ""
	}
	switch i.state {
	case moveStateSent:
		return "intent: move " + i.direction + " (sent)"
	case moveStateFailed:
		return "intent: move " + i.direction + " (failed)"
	default:
		return "intent: move " + i.direction + " (not sent)"
	}
}

// Movement step size is now owned by the backend (PlayerMoveStep = 5.0).
// The TUI only sends directional move requests via submitDirectionalMove.

// zoneReadResultMsg carries the result of a zone status read back to the model.
type zoneReadResultMsg struct {
	result zoneReadResult
}

// mapReadResultMsg carries the result of a map geometry read back to the model.
type mapReadResultMsg struct {
	result mapReadResult
}

// mobReadResultMsg carries the result of a mob-position read back to the model.
type mobReadResultMsg struct {
	result mobReadResult
}

// playerReadResultMsg carries the result of a player join/read back to the model.
type playerReadResultMsg struct {
	result playerReadResult
}

// encounterReadResultMsg carries the result of an encounter read back to the model.
type encounterReadResultMsg struct {
	result encounterReadResult
}

// moveResultMsg carries the result of a movement submission + readback.
type moveResultMsg struct {
	result    moveResult
	direction string
}

// targetConfirmMsg carries the result of a backend target proximity query.
type targetConfirmMsg struct {
	result targetConfirmResult
}

// attackResultMsg carries the result of a BasicAttack intent submission.
type attackResultMsg struct {
	result attackResult
}

// pickupResultMsg carries the result of a pickup_item intent submission.
type pickupResultMsg struct {
	result pickupResult
}

// inventoryReadResultMsg carries the result of an inventory read.
type inventoryReadResultMsg struct {
	result inventoryReadResult
}

// respawnResultMsg carries the result of a Respawn intent submission.
type respawnResultMsg struct {
	result respawnResult
}

// proximityNeedsRefresh returns true if there is an active proximity confirmation
// and either the player position or the focused entry has changed since it was queried.
// Returns false if no proximity query has been made yet (State == targetConfirmNone).
func proximityNeedsRefresh(tc targetConfirmResult, lastPos playerPosResult, lastID string, currentPos playerPosResult, currentEntry *rosterEntry) bool {
	if tc.State == targetConfirmNone {
		return false
	}
	// Check if focused entry changed
	if currentEntry == nil {
		return false // nothing to refresh for
	}
	if currentEntry.id != lastID {
		return true
	}
	// Check if player position changed
	if currentPos.X != lastPos.X || currentPos.Y != lastPos.Y {
		return true
	}
	return false
}

type model struct {
	width            int
	height           int
	lastIntent       moveIntent          // most recent inert movement intent preview
	target           backendTarget       // backend target config
	zoneRead         zoneReadResult      // result of zone status read
	mapRead          mapReadResult       // result of map geometry read
	mobRead          mobReadResult       // result of mob-position read
	playerRead       playerReadResult    // result of player join + state read
	encounterRead    encounterReadResult // result of zone encounter read
	rosterFocus      rosterFocus         // purely local, non-authoritative roster focus
	rosterEntries    []rosterEntry       // current flat roster for focus navigation
	targetConfirm    targetConfirmResult // backend-authoritative target confirmation
	lastProximityPos playerPosResult     // player position at last proximity query
	lastProximityID  string              // roster entry ID at last proximity query
	lastAttack       attackResult        // result of most recent BasicAttack submission
	lastPickup       pickupResult        // result of most recent pickup_item submission
	inventoryRead    inventoryReadResult // backend-owned player inventory
	invCountAtPickup int                 // inventory count when last pickup was submitted (-1 = no pickup yet)
	lootFocus        int                 // local selection index into encounter Drops; -1 = none
	lastRespawn      respawnResult       // result of most recent Respawn intent submission
	slog             *sessionLogger      // developer-facing session event log (nil-safe)
}

// currentDrops returns the current drop list from the active encounter, or nil.
func currentDrops(m *model) []string {
	if !m.playerRead.HasActiveEncounter || m.encounterRead.State != encounterReadOK {
		return nil
	}
	enc := findPlayerEncounter(m.encounterRead.Encounters, m.playerRead.ActiveEncounterID)
	if enc == nil {
		return nil
	}
	return enc.Drops
}

// reconcileLootFocus adjusts the loot selection index against the current drop list.
// If the previously selected item ID is still present, focus follows it.
// If it disappeared, clamp to last item or -1 if empty.
func reconcileLootFocus(oldFocus int, oldDrops, newDrops []string) int {
	if len(newDrops) == 0 {
		return -1
	}
	if oldFocus < 0 {
		return -1 // stay unfocused
	}
	// Try to find the previously selected item by ID
	if oldFocus < len(oldDrops) {
		prevID := oldDrops[oldFocus]
		for i, id := range newDrops {
			if id == prevID {
				return i
			}
		}
	}
	// Previously selected item disappeared — clamp
	if oldFocus >= len(newDrops) {
		return len(newDrops) - 1
	}
	return oldFocus
}

// maybeRefreshProximity checks if a proximity re-query is needed and, if so,
// updates the model's tracking state and returns a Cmd. Returns nil Cmd if no refresh needed.
func maybeRefreshProximity(m *model) tea.Cmd {
	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if !proximityNeedsRefresh(m.targetConfirm, m.lastProximityPos, m.lastProximityID, m.playerRead.Position, fe) {
		return nil
	}
	entry := *fe
	bt := m.target
	m.lastProximityPos = m.playerRead.Position
	m.lastProximityID = entry.id
	return func() tea.Msg {
		return targetConfirmMsg{result: queryTargetProximity(bt, entry)}
	}
}

func (m model) Init() tea.Cmd {
	// Perform initial reads at startup + schedule first refresh tick
	target := m.target
	return tea.Batch(
		func() tea.Msg {
			return zoneReadResultMsg{result: fetchZoneStatus(target)}
		},
		func() tea.Msg {
			return mapReadResultMsg{result: fetchZoneMap(target)}
		},
		func() tea.Msg {
			return mobReadResultMsg{result: fetchMobPositions(target)}
		},
		func() tea.Msg {
			return playerReadResultMsg{result: joinAndReadPlayer(target)}
		},
		func() tea.Msg {
			return encounterReadResultMsg{result: fetchZoneEncounters(target)}
		},
		scheduleRefresh(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshTickMsg:
		// Periodic refresh: read mobs, player state, zone status, encounters, inventory; schedule next tick
		target := m.target
		return m, tea.Batch(
			func() tea.Msg {
				return mobReadResultMsg{result: fetchMobPositions(target)}
			},
			func() tea.Msg {
				if m.playerRead.State == playerReadOK {
					return playerReadResultMsg{result: readPlayerState(target)}
				}
				return nil
			},
			func() tea.Msg {
				return zoneReadResultMsg{result: fetchZoneStatus(target)}
			},
			func() tea.Msg {
				return encounterReadResultMsg{result: fetchZoneEncounters(target)}
			},
			func() tea.Msg {
				return inventoryReadResultMsg{result: fetchPlayerInventory(target)}
			},
			scheduleRefresh(),
		)
	case zoneReadResultMsg:
		m.zoneRead = msg.result
		return m, nil
	case mapReadResultMsg:
		m.mapRead = msg.result
		return m, nil
	case mobReadResultMsg:
		m.mobRead = msg.result
		return m, nil
	case playerReadResultMsg:
		m.playerRead = msg.result
		m.slog.LogPoll("player_state", msg.result.State == playerReadOK)
		if msg.result.State == playerReadOK {
			m.slog.LogPlayerSnapshot("player_state",
				true, msg.result.HasPos,
				msg.result.Position.X, msg.result.Position.Y,
				msg.result.HasActiveEncounter)
		}
		return m, nil
	case encounterReadResultMsg:
		m.slog.LogPoll("encounters", msg.result.State == encounterReadOK)
		oldDrops := currentDrops(&m)
		m.encounterRead = msg.result
		// Reconcile local roster focus against new backend data
		var enc *encounterSummary
		if m.playerRead.HasActiveEncounter {
			enc = findPlayerEncounter(msg.result.Encounters, m.playerRead.ActiveEncounterID)
		}
		newEntries := buildRosterEntries(enc)
		m.rosterFocus = reconcileFocus(m.rosterFocus, m.rosterEntries, newEntries)
		m.rosterEntries = newEntries
		// Reconcile loot focus against new backend drops
		newDrops := currentDrops(&m)
		m.lootFocus = reconcileLootFocus(m.lootFocus, oldDrops, newDrops)
		return m, nil
	case targetConfirmMsg:
		m.targetConfirm = msg.result
		return m, nil
	case attackResultMsg:
		m.lastAttack = msg.result
		m.slog.LogPoll("attack_result", msg.result.State == attackStateSent)
		return m, nil
	case pickupResultMsg:
		m.lastPickup = msg.result
		return m, nil
	case inventoryReadResultMsg:
		m.inventoryRead = msg.result
		return m, nil
	case respawnResultMsg:
		m.lastRespawn = msg.result
		return m, nil
	case moveResultMsg:
		m.slog.LogPoll("move_result", msg.result.OK)
		if msg.result.OK {
			m.lastIntent = moveIntent{direction: msg.direction, state: moveStateSent}
			if msg.result.HasPos {
				m.playerRead.Position = msg.result.Position
				m.playerRead.HasPos = true
			}
		} else {
			m.lastIntent = moveIntent{direction: msg.direction, state: moveStateFailed}
		}
		// Render-observable state snapshot after move processing
		m.slog.LogState(map[string]any{
			"name":       "render_player",
			"move_ok":    msg.result.OK,
			"move_dir":   msg.direction,
			"player_pos": []float64{m.playerRead.Position.X, m.playerRead.Position.Y},
			"has_pos":    m.playerRead.HasPos,
		})
		// Re-query proximity if active and position changed
		return m, maybeRefreshProximity(&m)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		m.slog.LogKey(key)
		switch key {
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.rosterFocus = moveFocusDown(m.rosterFocus, len(m.rosterEntries))
			fe := focusedEntry(m.rosterFocus, m.rosterEntries)
			if fe != nil {
				m.slog.LogState(map[string]any{"name": "tab_focus", "focus_kind": fe.kind, "focus_id": fe.id, "roster_len": len(m.rosterEntries)})
			} else {
				m.slog.LogState(map[string]any{"name": "tab_focus", "focus_kind": "none", "roster_len": len(m.rosterEntries), "has_encounter": m.playerRead.HasActiveEncounter})
			}
			return m, maybeRefreshProximity(&m)
		case "shift+tab":
			m.rosterFocus = moveFocusUp(m.rosterFocus, len(m.rosterEntries))
			return m, maybeRefreshProximity(&m)
		case "[":
			// Move loot selection up
			if m.lootFocus > 0 {
				m.lootFocus--
			}
			return m, nil
		case "]":
			// Move loot selection down
			drops := currentDrops(&m)
			if m.lootFocus < len(drops)-1 {
				m.lootFocus++
			} else if m.lootFocus < 0 && len(drops) > 0 {
				m.lootFocus = 0
			}
			return m, nil
		case "p":
			m.slog.LogIntent("pickup", nil)
			// Submit pickup_item intent for the selected drop
			if m.playerRead.State != playerReadOK || !m.playerRead.HasActiveEncounter {
				m.lastPickup = pickupResult{State: pickupStateFailed, Error: "no encounter"}
				return m, nil
			}
			if m.encounterRead.State != encounterReadOK {
				m.lastPickup = pickupResult{State: pickupStateFailed, Error: "encounter unavailable"}
				return m, nil
			}
			enc := findPlayerEncounter(m.encounterRead.Encounters, m.playerRead.ActiveEncounterID)
			if enc == nil || !enc.DropsGenerated || len(enc.Drops) == 0 {
				m.lastPickup = pickupResult{State: pickupStateFailed, Error: "no drops"}
				return m, nil
			}
			if enc.LootExpired {
				m.lastPickup = pickupResult{State: pickupStateFailed, Error: "loot expired"}
				return m, nil
			}
			// Use selected drop, fall back to first if no selection
			idx := m.lootFocus
			if idx < 0 || idx >= len(enc.Drops) {
				idx = 0
			}
			encID := enc.EncounterID
			itemID := enc.Drops[idx]
			bt := m.target
			m.invCountAtPickup = m.inventoryRead.Count
			return m, func() tea.Msg {
				return pickupResultMsg{result: submitPickupItem(bt, encID, itemID)}
			}
		case "r":
			m.slog.LogIntent("respawn", nil)
			// Submit Respawn intent via existing backend dev surface
			if m.playerRead.State != playerReadOK {
				m.lastRespawn = respawnResult{State: respawnStateFailed, Error: "no player"}
				return m, nil
			}
			bt := m.target
			return m, func() tea.Msg {
				return respawnResultMsg{result: submitRespawn(bt)}
			}
		case "a":
			// Submit BasicAttack intent against focused mob
			if m.playerRead.State != playerReadOK {
				m.slog.LogState(map[string]any{"name": "attack_skip", "reason": "no_player"})
				m.lastAttack = attackResult{State: attackStateFailed, Error: "no player"}
				return m, nil
			}
			fe := focusedEntry(m.rosterFocus, m.rosterEntries)
			if fe == nil || fe.kind != "mb" {
				reason := "no_mob_focused"
				if fe == nil {
					reason = fmt.Sprintf("no_focus roster_len=%d has_enc=%v", len(m.rosterEntries), m.playerRead.HasActiveEncounter)
				} else {
					reason = fmt.Sprintf("focus_not_mob kind=%s id=%s", fe.kind, fe.id)
				}
				m.slog.LogState(map[string]any{"name": "attack_skip", "reason": reason, "roster_len": len(m.rosterEntries)})
				m.lastAttack = attackResult{State: attackStateFailed, Error: "no mob focused"}
				return m, nil
			}
			m.slog.LogIntent("attack", map[string]any{"target": fe.id})
			entry := *fe
			bt := m.target
			return m, func() tea.Msg {
				return attackResultMsg{result: submitBasicAttack(bt, entry.id)}
			}
		case "t":
			// Submit target confirmation for the focused roster entry
			if fe := focusedEntry(m.rosterFocus, m.rosterEntries); fe != nil {
				entry := *fe
				bt := m.target
				m.lastProximityPos = m.playerRead.Position
				m.lastProximityID = entry.id
				return m, func() tea.Msg {
					return targetConfirmMsg{result: queryTargetProximity(bt, entry)}
				}
			}
			// No focused entry — clear target honestly
			m.targetConfirm = targetConfirmResult{State: targetConfirmNone}
			m.lastProximityID = ""
			return m, nil
		default:
			if dir := directionFromKey(key); dir != "" {
				m.slog.LogIntent("move", map[string]any{"dir": dir})
				// If player is joined, submit backend-authoritative directional move
				if m.playerRead.State == playerReadOK {
					m.lastIntent = moveIntent{direction: dir, state: moveStatePreview}
					target := m.target
					return m, func() tea.Msg {
						return moveResultMsg{
							result:    submitDirectionalMove(target, dir),
							direction: dir,
						}
					}
				}
				// Otherwise, keep as preview only
				m.lastIntent = moveIntent{direction: dir, state: moveStatePreview}
				return m, nil
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	return renderLayout(m.width, m.height, m.lastIntent.preview(), m.target, m.zoneRead, m.mapRead, m.mobRead, m.playerRead, m.encounterRead, m.rosterFocus, m.rosterEntries, m.targetConfirm, m.lastAttack, m.lastPickup, m.inventoryRead, m.invCountAtPickup, m.lootFocus, m.lastRespawn)
}

func main() {
	slog := newSessionLogger()
	globalSessionLog = slog
	defer slog.Close()
	p := tea.NewProgram(model{target: defaultTarget(), slog: slog}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
