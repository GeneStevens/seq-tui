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

// moveStep is the distance in world units for one movement step.
const moveStep = 20.0

// directionOffset returns world-coordinate deltas for a direction.
// x = east/west, y = north/south (in backend ground-plane convention).
func directionOffset(dir string) (dx, dy float64) {
	switch dir {
	case "north":
		return 0, moveStep
	case "south":
		return 0, -moveStep
	case "east":
		return moveStep, 0
	case "west":
		return -moveStep, 0
	default:
		return 0, 0
	}
}

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
		// Periodic refresh: read mobs, player state, zone status, encounters; schedule next tick
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
		return m, nil
	case encounterReadResultMsg:
		m.encounterRead = msg.result
		// Reconcile local roster focus against new backend data
		var enc *encounterSummary
		if m.playerRead.HasActiveEncounter {
			enc = findPlayerEncounter(msg.result.Encounters, m.playerRead.ActiveEncounterID)
		}
		newEntries := buildRosterEntries(enc)
		m.rosterFocus = reconcileFocus(m.rosterFocus, m.rosterEntries, newEntries)
		m.rosterEntries = newEntries
		return m, nil
	case targetConfirmMsg:
		m.targetConfirm = msg.result
		return m, nil
	case moveResultMsg:
		if msg.result.OK {
			m.lastIntent = moveIntent{direction: msg.direction, state: moveStateSent}
			if msg.result.HasPos {
				m.playerRead.Position = msg.result.Position
				m.playerRead.HasPos = true
			}
		} else {
			m.lastIntent = moveIntent{direction: msg.direction, state: moveStateFailed}
		}
		// Re-query proximity if active and position changed
		return m, maybeRefreshProximity(&m)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.rosterFocus = moveFocusDown(m.rosterFocus, len(m.rosterEntries))
			return m, maybeRefreshProximity(&m)
		case "shift+tab":
			m.rosterFocus = moveFocusUp(m.rosterFocus, len(m.rosterEntries))
			return m, maybeRefreshProximity(&m)
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
				// If player is joined with a known position, submit real move
				if m.playerRead.State == playerReadOK && m.playerRead.HasPos {
					m.lastIntent = moveIntent{direction: dir, state: moveStatePreview}
					target := m.target
					currentPos := m.playerRead.Position
					dx, dy := directionOffset(dir)
					return m, func() tea.Msg {
						return moveResultMsg{
							result:    submitMoveAndReadback(target, currentPos, dx, dy),
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
	return renderLayout(m.width, m.height, m.lastIntent.preview(), m.target, m.zoneRead, m.mapRead, m.mobRead, m.playerRead, m.encounterRead, m.rosterFocus, m.rosterEntries, m.targetConfirm)
}

func main() {
	p := tea.NewProgram(model{target: defaultTarget()}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
