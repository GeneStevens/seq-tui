package app

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

// logApp is the chat app with sixty lines in the Events log, drawn once.
func logApp(t *testing.T) (*App, *model.Model) {
	a, m, _ := chatApp()
	a.w, a.h = 80, 24
	a.sess.View(func(m *model.Model) {
		for i := 1; i <= 60; i++ {
			m.Notice("log line " + strconv.Itoa(i))
		}
	})
	a.render(a.w, a.h, 0, screen.Plain())
	if a.mouse.hits.Log.W == 0 {
		t.Fatal("the bottom panel was not recorded")
	}
	return a, m
}

func wheel(a *App, x, y int, b tea.MouseButton) {
	a.onMouse(tea.MouseWheelMsg{X: x, Y: y, Button: b})
}

func TestTheWheelOverTheLogScrollsItAndNeverZoomsTheMap(t *testing.T) {
	a, _ := logApp(t)
	l := a.mouse.hits.Log
	zoom := a.mouse.zoom
	wheel(a, l.X+5, l.Y+2, tea.MouseWheelUp)
	if a.mouse.zoom != zoom {
		t.Errorf("the wheel over the log zoomed the map: %d to %d", zoom, a.mouse.zoom)
	}
	if p := a.mouse.scroll[screen.PanelEvents]; p.Live() || p.Back != 3 {
		t.Errorf("the wheel up scrolled the log by 3 rows: %+v", p)
	}
	wheel(a, l.X+5, l.Y+2, tea.MouseWheelDown)
	a.render(a.w, a.h, 0, screen.Plain()) // drawing settles the position
	if !a.mouse.scroll[screen.PanelEvents].Live() {
		t.Errorf("the wheel down did not come back to live: %+v", a.mouse.scroll[screen.PanelEvents])
	}
	m := a.mouse.hits.Map
	wheel(a, m.X+2, m.Y+2, tea.MouseWheelUp)
	if a.mouse.zoom == zoom {
		t.Error("the wheel over the map no longer zooms")
	}
}

func TestPageKeysHomeEndAndTabsScrollTheLog(t *testing.T) {
	a, m := logApp(t)
	for _, tc := range []struct {
		key  string
		live bool
	}{{"pgup", false}, {"pgdown", true}, {"home", false}, {"end", true}} {
		a.key(tc.key)
		a.render(a.w, a.h, 0, screen.Plain())
		if got := a.mouse.scroll[screen.PanelEvents].Live(); got != tc.live {
			t.Errorf("after %s live is %v: %+v", tc.key, got, a.mouse.scroll[screen.PanelEvents])
		}
	}
	a.key("home")
	out := a.render(a.w, a.h, 0, screen.Plain())
	if !strings.Contains(out, "↓ ") || !strings.Contains(out, "new · End") || strings.Contains(out, "log line 60") {
		t.Errorf("the oldest page shows the mark and not the newest line:\n%s", out)
	}
	// Lines that arrive leave the view where it is, and count as new.
	a.sess.View(func(m *model.Model) { m.Notice("late news") })
	out = a.render(a.w, a.h, 0, screen.Plain())
	if strings.Contains(out, "late news") || !strings.Contains(out, "new · End") {
		t.Errorf("a line arrived and moved the view:\n%s", out)
	}
	// A tab switch is live again.
	a.key("]")
	a.key("[")
	if !a.mouse.scroll[screen.PanelEvents].Live() {
		t.Error("switching tabs did not return to live")
	}
	// Each tab keeps its own position.
	a.key("pgup")
	a.render(a.w, a.h, 0, screen.Plain())
	a.view.Panel = screen.PanelChat
	hear(a, m, "say", "Mira", "", "hello")
	a.render(a.w, a.h, 0, screen.Plain())
	a.key("pgup")
	if a.mouse.scroll[screen.PanelChat].Live() {
		t.Error("the keys scrolled the wrong tab")
	}
}
