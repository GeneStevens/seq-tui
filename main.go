package main

import (
	"flag"
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

// eventTextForResult returns a human-readable event text for a backend combat result.
// Maps internal result kinds to compact readable forms for the event stream.
// This is purely presentational — no gameplay inference.
func eventTextForResult(kind string, value int) string {
	var text string
	switch kind {
	case "damage_applied":
		text = fmt.Sprintf("hit %d", value)
	case "attack_miss":
		text = "miss"
	case "heal_applied":
		text = fmt.Sprintf("heal %d", value)
	case "out_of_range":
		text = "deny range"
	case "los_blocked":
		text = "deny los"
	case "invalid_target":
		text = "deny target"
	case "cooldown_set":
		text = "cooldown"
	case "cooldown_blocked":
		text = "deny cooldown"
	case "cast_start":
		text = "casting"
	case "cast_complete":
		text = "cast complete"
	case "cast_denied":
		text = "deny cast"
	case "cast_cancelled":
		text = "cast cancelled"
	case "interrupt_applied":
		text = "interrupted"
	case "interrupt_denied":
		text = "deny interrupt"
	case "interrupt_lockout_started":
		text = "lockout"
	case "interrupt_lockout_blocked":
		text = "lockout blocked"
	case "blocked_wounded":
		text = "blocked"
	case "gcd_started":
		text = "gcd"
	case "gcd_blocked":
		text = "deny gcd"
	case "ability_queued":
		text = "queued"
	case "ability_executed_from_queue":
		text = "from queue"
	case "resource_blocked":
		text = "deny mana"
	case "resource_spent":
		text = fmt.Sprintf("mana -%d", value)
	case "resource_regen":
		text = fmt.Sprintf("mana +%d", value)
	default:
		text = kind
		if value > 0 {
			text += fmt.Sprintf(" %d", value)
		}
	}
	return text
}

// eventTextForCombatResult returns enriched event text with actor>target prefix
// when actor and target are available. Falls back to eventTextForResult when
// actor/target are empty (e.g. legacy events without engagement data).
func eventTextForCombatResult(kind string, value int, actor, target string) string {
	if actor == "" && target == "" {
		return eventTextForResult(kind, value)
	}
	actorLabel := truncateID(actor, 8)
	targetLabel := truncateID(target, 8)
	prefix := actorLabel + ">" + targetLabel
	switch kind {
	case "damage_applied":
		return fmt.Sprintf("%s hit %d", prefix, value)
	case "attack_miss":
		return prefix + " miss"
	case "heal_applied":
		return fmt.Sprintf("%s heal %d", prefix, value)
	default:
		return eventTextForResult(kind, value)
	}
}

// eventEntry is one line in the event stream panel.
// Kind classifies the source; text is the compact display string.
type eventEntry struct {
	kind string // "combat", "move", "atk", "focus", "enc"
	text string
}

// eventLogMax is the maximum number of entries kept in the event stream.
const eventLogMax = 12

// eventLog is a bounded FIFO of recent event entries.
// Newest entries are appended at the end. When full, the oldest is dropped.
type eventLog struct {
	entries []eventEntry
}

// append adds an entry, dropping the oldest if the log is full.
func (l *eventLog) append(e eventEntry) {
	l.entries = append(l.entries, e)
	if len(l.entries) > eventLogMax {
		l.entries = l.entries[len(l.entries)-eventLogMax:]
	}
}

// recent returns the most recent entries, newest last.
func (l *eventLog) recent() []eventEntry {
	return l.entries
}

// len returns the number of entries in the log.
func (l *eventLog) len() int {
	return len(l.entries)
}

// directionTowardMob returns the dominant cardinal direction from player to mob
// using backend-owned positions. Returns the axis with the larger absolute delta.
// Ties break to the horizontal axis (east/west). Returns empty string if
// positions are identical. Purely presentation logic — no pathfinding.
// directionEpsilon is the minimum distance threshold for directionTowardMob.
// If the player is within this distance of the mob, they are treated as co-located
// and no movement direction is returned. Prevents floating-point near-zero deltas
// from causing spurious movement when positions render to the same screen cell.
const directionEpsilon = 0.5

func directionTowardMob(playerPos playerPosResult, mobPos mobPosVec3) string {
	dx := mobPos.X - playerPos.X
	dy := mobPos.Y - playerPos.Y
	absDX := math.Abs(dx)
	absDY := math.Abs(dy)
	if absDX < directionEpsilon && absDY < directionEpsilon {
		return ""
	}
	if absDX >= absDY {
		if dx > 0 {
			return "east"
		}
		return "west"
	}
	// dy > 0 means mob is at higher Y → move "north" (backend "up" = +Y)
	if dy > 0 {
		return "north"
	}
	return "south"
}

// findMobPosition returns the position of a mob by its live identity.
// Matches LiveMobID first (authoritative per live instance), then ProcessID,
// then MobID as fallback. Returns nil if not found.
func findMobPosition(mobs []mobPosition, id string) *mobPosVec3 {
	for i := range mobs {
		if mobs[i].LiveMobID != "" && mobs[i].LiveMobID == id {
			return &mobs[i].Position
		}
	}
	for i := range mobs {
		if mobs[i].ProcessID == id {
			return &mobs[i].Position
		}
	}
	for i := range mobs {
		if mobs[i].MobID == id {
			return &mobs[i].Position
		}
	}
	return nil
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
	source    string // "move" (default) or "step" (g key step-toward)
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
	events           eventLog            // bounded event stream for right-rail panel
	lastSeenActIdx      uint64 // last seen encounter ActionIndex for event dedup
	lastSeenTimelineLen int    // last seen encounter timeline length for denial detection
	lastSeenEncID       string // last seen encounter ID for lifecycle events
	lastSeenEncState    string // last seen encounter state for completion detection
	lastSeenMobsDead    int    // last seen dead mob count for defeat event detection
	lastDenialKind      string // last emitted denial kind for dedup
	showDiag         bool                // diagnostics section visible (d key toggle)
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
				return playerReadResultMsg{result: joinAndReadPlayer(target)}
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
		// Invalidate sticky target if the selected ProcessID no longer exists
		if fe := focusedEntry(m.rosterFocus, m.rosterEntries); fe != nil && fe.kind == "mb" {
			if msg.result.State == mobReadOK && findMobPosition(msg.result.Mobs, fe.id) == nil {
				m.rosterFocus = rosterFocus{index: -1}
				m.rosterEntries = nil
			}
		}
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
		if newEntries != nil {
			// Encounter roster exists — merge with existing entries to preserve
			// sticky ProcessID identity. Only replace entries when the user's
			// selected target identity would be preserved.
			newEntries = mergeRosterIdentity(m.rosterEntries, newEntries)
			m.rosterFocus = reconcileFocus(m.rosterFocus, m.rosterEntries, newEntries)
			m.rosterEntries = newEntries
		}
		// When newEntries is nil (no encounter), preserve any pre-encounter mob entries
		// that tab built from mob_positions. This prevents the 500ms refresh from
		// wiping focus state between tab and attack.
		// Reconcile loot focus against new backend drops
		newDrops := currentDrops(&m)
		m.lootFocus = reconcileLootFocus(m.lootFocus, oldDrops, newDrops)
		// Event stream: detect encounter lifecycle and combat event progression
		if enc != nil {
			// Encounter lifecycle: new encounter or state change
			if enc.EncounterID != m.lastSeenEncID {
				m.events.append(eventEntry{kind: "enc", text: "enc:opened"})
				m.lastSeenEncID = enc.EncounterID
				m.lastSeenEncState = enc.State
				m.lastSeenActIdx = enc.ActionIndex
				m.lastSeenTimelineLen = enc.TimelineLength
				m.lastSeenMobsDead = enc.MobsDead
				m.lastDenialKind = ""
			} else if enc.State == "Completed" && m.lastSeenEncState != "Completed" {
				// Outcome label
				outcomeLabel := "enc:ended"
				if enc.CompletedReason == "all_mobs_dead" {
					outcomeLabel = "VICTORY"
				} else if enc.CompletedReason == "all_players_dead" {
					outcomeLabel = "DEFEAT"
				}
				m.events.append(eventEntry{kind: "enc", text: outcomeLabel})
				// XP payoff — backend-owned, no client computation
				if enc.XPAwarded > 0 {
					m.events.append(eventEntry{kind: "enc", text: fmt.Sprintf("xp +%d", enc.XPAwarded)})
				}
				// Loot availability
				if enc.DropsGenerated && len(enc.Drops) > 0 && !enc.LootExpired {
					m.events.append(eventEntry{kind: "enc", text: fmt.Sprintf("loot %d items", len(enc.Drops))})
				}
				m.lastSeenEncState = enc.State
			}
			// Mob defeat detection: MobsDead increased
			if enc.MobsDead > m.lastSeenMobsDead {
				m.events.append(eventEntry{kind: "combat", text: "mob defeated"})
				m.lastSeenMobsDead = enc.MobsDead
			}
			// Combat event: ActionIndex advanced (damage/miss/heal)
			if enc.ActionIndex > m.lastSeenActIdx {
				text := eventTextForCombatResult(enc.LatestResultKind, enc.LatestResultValue, enc.LatestResultActor, enc.LatestResultTarget)
				m.events.append(eventEntry{kind: "combat", text: text})
				m.lastSeenActIdx = enc.ActionIndex
				m.lastSeenTimelineLen = enc.TimelineLength
				m.lastDenialKind = "" // reset denial dedup on new combat event
			}
			// Denial event: timeline grew without ActionIndex advancing (dedup repeated denials)
			if enc.TimelineLength > m.lastSeenTimelineLen && enc.ActionIndex == m.lastSeenActIdx {
				if enc.LatestDenialKind != "" && enc.LatestDenialKind != m.lastDenialKind {
					text := eventTextForResult(enc.LatestDenialKind, 0)
					m.events.append(eventEntry{kind: "combat", text: text})
					m.lastDenialKind = enc.LatestDenialKind
				}
				m.lastSeenTimelineLen = enc.TimelineLength
			}
		}
		return m, nil
	case targetConfirmMsg:
		m.targetConfirm = msg.result
		return m, nil
	case attackResultMsg:
		m.lastAttack = msg.result
		m.slog.LogPoll("attack_result", msg.result.State == attackStateSent)
		// Event stream: attack submission result
		if msg.result.State == attackStateSent {
			m.events.append(eventEntry{kind: "atk", text: "attack"})
		} else if msg.result.State == attackStateFailed {
			errText := "atk fail"
			if msg.result.Error != "" {
				// Show the first 18 chars of the error for compact display
				reason := msg.result.Error
				if len(reason) > 18 {
					reason = reason[:18]
				}
				errText = "atk:" + reason
			}
			m.events.append(eventEntry{kind: "atk", text: errText})
		}
		if msg.result.State == attackStateSent {
			// Immediate refresh: reduce enc:opening dwell by reading
			// encounter and player state right after successful attack submit.
			// Purely transport — no gameplay inference.
			target := m.target
			return m, tea.Batch(
				func() tea.Msg {
					return encounterReadResultMsg{result: fetchZoneEncounters(target)}
				},
				func() tea.Msg {
					return playerReadResultMsg{result: readPlayerState(target)}
				},
			)
		}
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
			if msg.source == "step" {
				m.events.append(eventEntry{kind: "move", text: "step " + msg.direction})
			} else {
				m.events.append(eventEntry{kind: "move", text: "move " + msg.direction})
			}
		} else {
			m.lastIntent = moveIntent{direction: msg.direction, state: moveStateFailed}
			if msg.source == "step" {
				m.events.append(eventEntry{kind: "move", text: "step blocked " + msg.direction})
			} else {
				m.events.append(eventEntry{kind: "move", text: "move blocked " + msg.direction})
			}
		}
		// Render-observable state snapshot after move processing
		m.slog.LogState(map[string]any{
			"name":       "render_player",
			"move_ok":    msg.result.OK,
			"move_dir":   msg.direction,
			"player_pos": []float64{m.playerRead.Position.X, m.playerRead.Position.Y},
			"has_pos":    m.playerRead.HasPos,
		})
		if msg.result.OK {
			// Immediate refresh: reduce spatial context staleness by reading
			// mob positions, player state, and encounters right after successful
			// move submit. Purely transport — no movement prediction.
			target := m.target
			proxCmd := maybeRefreshProximity(&m)
			return m, tea.Batch(
				func() tea.Msg {
					return mobReadResultMsg{result: fetchMobPositions(target)}
				},
				func() tea.Msg {
					return playerReadResultMsg{result: readPlayerState(target)}
				},
				func() tea.Msg {
					return encounterReadResultMsg{result: fetchZoneEncounters(target)}
				},
				proxCmd,
			)
		}
		// Re-query proximity if active and position changed (failed moves only)
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
			// Pre-encounter: always rebuild mob list with fresh distances so Tab
			// reflects current spatial truth. During encounter, use backend roster.
			if !m.playerRead.HasActiveEncounter && m.playerRead.State == playerReadOK && m.playerRead.HasPos && m.mobRead.State == mobReadOK {
				prevFocusID := ""
				if fe := focusedEntry(m.rosterFocus, m.rosterEntries); fe != nil {
					prevFocusID = fe.id
				}
				m.rosterEntries = buildMobEntriesByDistance(m.playerRead.Position, m.mobRead.Mobs)
				// Preserve focus on the same mob if it still exists in the new list
				m.rosterFocus = rosterFocus{index: -1}
				if prevFocusID != "" {
					for i, e := range m.rosterEntries {
						if e.id == prevFocusID {
							m.rosterFocus = rosterFocus{index: i}
							break
						}
					}
				}
			}
			m.rosterFocus = moveFocusDown(m.rosterFocus, len(m.rosterEntries))
			fe := focusedEntry(m.rosterFocus, m.rosterEntries)
			if fe != nil {
				source := "encounter_roster"
				if !m.playerRead.HasActiveEncounter {
					source = "nearest_mob"
				}
				m.slog.LogState(map[string]any{"name": "tab_focus", "source": source, "focus_kind": fe.kind, "focus_id": fe.id, "roster_len": len(m.rosterEntries)})
				if fe.kind == "mb" {
					m.slog.LogState(map[string]any{"name": "canonical_target_set", "mob_id": fe.id, "canonical": fe.canonicalMobID, "source": source})
				}
				focusLabel := "focus"
				if fe.kind == "mb" {
					focusLabel = "focus mob"
				} else {
					focusLabel = "focus " + fe.kind
				}
				m.events.append(eventEntry{kind: "focus", text: focusLabel})
			} else {
				m.slog.LogState(map[string]any{"name": "tab_focus", "focus_kind": "none", "roster_len": len(m.rosterEntries), "has_encounter": m.playerRead.HasActiveEncounter, "mob_count": m.mobRead.Count})
			}
			// Auto-query proximity for focused mob
			fe = focusedEntry(m.rosterFocus, m.rosterEntries)
			if fe != nil && fe.kind == "mb" {
				entry := *fe
				bt := m.target
				m.lastProximityPos = m.playerRead.Position
				m.lastProximityID = entry.id
				return m, func() tea.Msg {
					return targetConfirmMsg{result: queryTargetProximity(bt, entry)}
				}
			}
			return m, nil
		case "shift+tab":
			// Pre-encounter: rebuild with fresh distances, same as tab
			if !m.playerRead.HasActiveEncounter && m.playerRead.State == playerReadOK && m.playerRead.HasPos && m.mobRead.State == mobReadOK {
				prevFocusID := ""
				if fe := focusedEntry(m.rosterFocus, m.rosterEntries); fe != nil {
					prevFocusID = fe.id
				}
				m.rosterEntries = buildMobEntriesByDistance(m.playerRead.Position, m.mobRead.Mobs)
				m.rosterFocus = rosterFocus{index: -1}
				if prevFocusID != "" {
					for i, e := range m.rosterEntries {
						if e.id == prevFocusID {
							m.rosterFocus = rosterFocus{index: i}
							break
						}
					}
				}
			}
			m.rosterFocus = moveFocusUp(m.rosterFocus, len(m.rosterEntries))
			// Auto-query proximity for focused mob
			fe := focusedEntry(m.rosterFocus, m.rosterEntries)
			if fe != nil && fe.kind == "mb" {
				entry := *fe
				bt := m.target
				m.lastProximityPos = m.playerRead.Position
				m.lastProximityID = entry.id
				return m, func() tea.Msg {
					return targetConfirmMsg{result: queryTargetProximity(bt, entry)}
				}
			}
			return m, nil
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
		case "d":
			// Toggle diagnostics section visibility
			m.showDiag = !m.showDiag
			return m, nil
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
			// Fail closed: refuse to submit attack if focused mob lacks live identity
			if fe.id == "" {
				m.slog.LogState(map[string]any{"name": "attack_skip", "reason": "no_live_mob_id", "focus_id": fe.id})
				m.lastAttack = attackResult{State: attackStateFailed, Error: "no live target"}
				return m, nil
			}
			m.slog.LogIntent("attack", map[string]any{"target_live_id": fe.id, "mob_id": fe.mobID, "canonical": true})
			m.slog.LogState(map[string]any{"name": "attack_submit", "live_mob_id": fe.id, "mob_id": fe.mobID, "has_encounter": m.playerRead.HasActiveEncounter})
			entry := *fe
			bt := m.target
			return m, func() tea.Msg {
				return attackResultMsg{result: submitBasicAttack(bt, entry.id)}
			}
		case "g":
			// Step toward focused mob — compute dominant cardinal direction from
			// backend-owned player and mob positions, then submit via existing
			// directional move. One press = one step. No pathfinding.
			if m.playerRead.State != playerReadOK || !m.playerRead.HasPos {
				m.slog.LogState(map[string]any{"name": "goto_skip", "reason": "no_player_pos"})
				return m, nil
			}
			fe := focusedEntry(m.rosterFocus, m.rosterEntries)
			if fe == nil || fe.kind != "mb" {
				m.slog.LogState(map[string]any{"name": "goto_skip", "reason": "no_mob_focused"})
				return m, nil
			}
			if m.mobRead.State != mobReadOK {
				m.slog.LogState(map[string]any{"name": "goto_skip", "reason": "no_mob_data"})
				return m, nil
			}
			mobPos := findMobPosition(m.mobRead.Mobs, fe.id)
			if mobPos == nil {
				m.slog.LogState(map[string]any{"name": "goto_skip", "reason": "mob_not_found", "mob_id": fe.id})
				return m, nil
			}
			dir := directionTowardMob(m.playerRead.Position, *mobPos)
			m.slog.LogState(map[string]any{
				"name":       "goto_trace",
				"player_x":  m.playerRead.Position.X,
				"player_y":  m.playerRead.Position.Y,
				"mob_x":     mobPos.X,
				"mob_y":     mobPos.Y,
				"dx":        mobPos.X - m.playerRead.Position.X,
				"dy":        mobPos.Y - m.playerRead.Position.Y,
				"direction":  dir,
				"live_id":   fe.id,
				"mob_id":    fe.mobID,
			})
			if dir == "" {
				m.slog.LogState(map[string]any{"name": "goto_skip", "reason": "already_at_mob"})
				return m, nil
			}
			m.slog.LogIntent("goto", map[string]any{"mob_id": fe.id, "dir": dir})
			m.lastIntent = moveIntent{direction: dir, state: moveStatePreview}
			target := m.target
			return m, func() tea.Msg {
				return moveResultMsg{
					result:    submitDirectionalMove(target, dir),
					direction: dir,
					source:    "step",
				}
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
	return renderLayout(m.width, m.height, m.lastIntent.preview(), m.target, m.zoneRead, m.mapRead, m.mobRead, m.playerRead, m.encounterRead, m.rosterFocus, m.rosterEntries, m.targetConfirm, m.lastAttack, m.lastPickup, m.inventoryRead, m.invCountAtPickup, m.lootFocus, m.lastRespawn, m.events, m.showDiag)
}

func main() {
	smokeFlag := flag.Bool("smoke", false, "run startup smoke check and exit")
	flag.Parse()

	if *smokeFlag {
		os.Exit(runSmokeCmd(defaultTarget(), os.Stdout))
	}

	globalSessionID = generateSessionID()
	slog := newSessionLogger()
	globalSessionLog = slog
	defer slog.Close()
	p := tea.NewProgram(model{target: defaultTarget(), showDiag: true, slog: slog}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
