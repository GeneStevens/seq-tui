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

type model struct {
	width     int
	height    int
	lastInput string // most recent inert movement acknowledgement
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
				m.lastInput = dir + " (not connected)"
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
	return renderLayout(m.width, m.height, m.lastInput)
}

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
