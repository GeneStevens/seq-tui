package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The mouse (D77) and the character sidebar (D78). A click is turned into the
// same Actions a key makes; the game is fully playable without a mouse, and each
// mouse action has its key: the arrows walk (on the large map a cursor and Enter),
// Tab and Esc select, F and G act, L loots all, + and - zoom, Ctrl+A names everything,
// C holds the sidebar. The server decides whether an intent is allowed.

// mouseState is what the program keeps for the pointer and the sidebar.
type mouseState struct {
	hits    screen.Hits
	hover   *screen.Pt
	alt     bool // a mouse event came with Alt held
	labels  bool // the toggle that stands in for Alt
	zoom    int  // the map's zoom
	bigZoom int  // the large map's zoom
	card    string
	cursor  *pb.Pos // the large map's keyboard cursor
	side    screen.Sidebar
}

// mouseView puts the mouse state in the view the screen draws.
func (a *App) mouseView(v *screen.View) {
	ms := &a.mouse
	v.Hits, v.Hover, v.Labels = &ms.hits, ms.hover, ms.alt || ms.labels
	v.Zoom, v.BigZoom, v.Card, v.Cursor, v.Side = ms.zoom, ms.bigZoom, ms.card, ms.cursor, &ms.side
}

// refresh draws the screen once, for the geometry of what is on it: a scenario
// has no renderer, and a click must be read against the screen as it is now.
func (a *App) refresh() {
	_ = a.render(a.w, a.h, 0, screen.Plain())
}

// zoomMap zooms whichever map is shown by steps.
func (a *App) zoomMap(by int) {
	if a.overview {
		a.mouse.bigZoom = min(max(a.mouse.bigZoom+by, 0), screen.MaxZoom)
		return
	}
	a.mouse.zoom = min(max(a.mouse.zoom+by, screen.MinZoom), screen.MaxZoom)
}

// moveCursor moves the large map's cursor a cell with a key.
func (a *App) moveCursor(k string, m *model.Model) {
	a.refresh()
	h := &a.mouse.hits
	at := a.mouse.cursor
	if at == nil {
		at = m.Self.Pos
	}
	pt, ok := h.CellOf(at)
	if !ok {
		pt = screen.Pt{X: h.Map.X + h.Map.W/2, Y: h.Map.Y + h.Map.H/2}
	}
	switch k {
	case "up", "w", "k":
		pt.Y--
	case "down", "s", "j":
		pt.Y++
	case "left", "a", "h":
		pt.X--
	case "right", "d", "l":
		pt.X++
	}
	if p := h.WorldAt(pt.X, pt.Y); p != nil {
		a.mouse.cursor = p
	}
}

// sideShown says a character sidebar is beside the screen that is open.
func (a *App) sideShown() bool {
	return (a.traders != nil || a.talk != nil || a.board != nil) && a.inv == nil && a.trade == nil && a.cons == nil &&
		!a.overview && a.slots == nil && a.role == nil && a.quest == nil && a.menu == nil && a.soc == nil && a.ses == nil && !a.view.Help
}

// sideKey handles the keys of the sidebar (D78): C turns to the next tab (and, in
// a window too narrow for the sidebar beside the screen, holds the drawer open or
// shut); in the drawer Tab, the arrows and Esc work on the drawer. It says whether
// it took the key.
func (a *App) sideKey(k string, act *Action) bool {
	if !a.sideShown() {
		return false
	}
	sb := &a.mouse.side
	narrow := a.narrow()
	switch {
	case k == "c":
		switch {
		case narrow && !sb.Drawer:
			sb.Drawer = true
		case narrow:
			sb.Drawer = false
		default:
			*act = Action{Kind: SelectSide, Count: (sb.Tab + 1) % screen.Tabs}
		}
		return true
	case k == "pgdown":
		*act = Action{Kind: ScrollSide, Count: 4}
		return true
	case k == "pgup":
		*act = Action{Kind: ScrollSide, Count: -4}
		return true
	case sb.Drawer && narrow:
		switch k {
		case "esc", "q":
			sb.Drawer = false
		case "tab", "right", "d":
			*act = Action{Kind: SelectSide, Count: (sb.Tab + 1) % screen.Tabs}
		case "left", "a":
			*act = Action{Kind: SelectSide, Count: (sb.Tab + screen.Tabs - 1) % screen.Tabs}
		case "up", "w", "k":
			*act = Action{Kind: ScrollSide, Count: -1}
		case "down", "s", "j":
			*act = Action{Kind: ScrollSide, Count: 1}
		case "ctrl+c":
			*act = Action{Kind: Quit}
		}
		return true // the drawer holds the keys until it is closed
	}
	return false
}

// narrow says the window is too narrow to have the sidebar beside a screen.
func (a *App) narrow() bool {
	rw := min(max(a.w*3/10, 28), 44)
	return a.w-rw-2 < 60
}

// onMouse handles the pointer: hover, the wheel and clicks.
func (a *App) onMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	mo := msg.Mouse()
	ms := &a.mouse
	ms.alt = mo.Mod.Contains(tea.ModAlt)
	switch msg.(type) {
	case tea.MouseMotionMsg:
		ms.hover = &screen.Pt{X: mo.X, Y: mo.Y}
		return a, nil
	case tea.MouseWheelMsg:
		ms.hover = &screen.Pt{X: mo.X, Y: mo.Y}
		a.refresh()
		switch {
		case ms.hits.Side.In(mo.X, mo.Y):
			return a.do(Action{Kind: ScrollSide, Count: map[tea.MouseButton]int{tea.MouseWheelUp: -1, tea.MouseWheelDown: 1}[mo.Button]})
		case mo.Button == tea.MouseWheelUp:
			return a.do(Action{Kind: ZoomMap, Count: 1})
		case mo.Button == tea.MouseWheelDown:
			return a.do(Action{Kind: ZoomMap, Count: -1})
		}
		return a, nil
	case tea.MouseClickMsg:
		ms.hover = &screen.Pt{X: mo.X, Y: mo.Y}
		switch mo.Button {
		case tea.MouseLeft:
			return a.click(mo.X, mo.Y)
		case tea.MouseRight:
			// The right button lets go of the selection, as Esc does.
			ms.card = ""
		}
	}
	return a, nil
}

// click is a left click at the screen cell (x, y).
func (a *App) click(x, y int) (tea.Model, tea.Cmd) {
	a.refresh()
	h := &a.mouse.hits
	for _, t := range h.Tabs {
		if t.Rect.In(x, y) {
			return a.do(Action{Kind: SelectSide, Count: t.Tab})
		}
	}
	var act Action
	a.focusSess().View(func(m *model.Model) {
		pick, ok := h.PickAt(m, x, y)
		if !ok {
			return
		}
		act = a.clickAction(m, pick)
	})
	return a.do(act)
}

// clickAction is what a click on a pick means (D77). Clicking the ground walks
// there; the first click on a mob or an NPC selects it and opens its card; a
// second click on it acts (a first click never attacks); a click on loot takes
// everything within reach.
func (a *App) clickAction(m *model.Model, p screen.Pick) Action {
	ms := &a.mouse
	switch p.Kind {
	case screen.PickMob:
		if ms.card == p.ID {
			return Action{Kind: Attack, On: true, Note: "attack " + p.Name}
		}
		ms.card = p.ID
		if m.Self.Target == p.ID {
			return Action{}
		}
		return Action{Kind: Target, ID: p.ID}
	case screen.PickNPC:
		if ms.card != p.ID {
			ms.card = p.ID
			return Action{}
		}
		if n := m.NPC(p.ID); n != nil && !m.InReach(n) {
			return Action{Kind: Move, X: p.Pos.GetX(), Y: p.Pos.GetY(), Note: "walk up to " + p.Name}
		}
		return Action{Kind: Talk, NPC: p.ID, Note: p.Name}
	case screen.PickDrop:
		ms.card = ""
		d := m.Drops[p.ID]
		switch {
		case !d.GetCanLoot():
			return Action{Note: p.Name + " is not yours to take"}
		case d2(d.GetPos(), m.Self.Pos) > 5000*5000:
			return Action{Kind: Move, X: p.Pos.GetX(), Y: p.Pos.GetY(), Note: fmt.Sprintf("walk to %s; click again to loot", p.Name)}
		}
		return Action{Kind: Loot}
	}
	ms.card = ""
	ms.cursor = nil
	return Action{Kind: Move, X: p.Pos.GetX(), Y: p.Pos.GetY()}
}
