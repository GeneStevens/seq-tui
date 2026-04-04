package main

import (
	"fmt"
	"math"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

// moveResultMsg carries the result of a movement submission + readback.
type moveResultMsg struct {
	result    moveResult
	direction string
}

type model struct {
	width      int
	height     int
	lastIntent moveIntent       // most recent inert movement intent preview
	target     backendTarget    // backend target config
	zoneRead   zoneReadResult   // result of zone status read
	mapRead    mapReadResult    // result of map geometry read
	mobRead    mobReadResult    // result of mob-position read
	playerRead playerReadResult // result of player join + state read
}

func (m model) Init() tea.Cmd {
	// Perform zone status, map geometry, and mob position reads at startup
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
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
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
	return renderLayout(m.width, m.height, m.lastIntent.preview(), m.target, m.zoneRead, m.mapRead, m.mobRead, m.playerRead)
}

func main() {
	p := tea.NewProgram(model{target: defaultTarget()}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
