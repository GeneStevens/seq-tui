package app

import (
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
	sel     *selection     // the mob, person or drop last clicked: W walks to it (D109)
	spot    *pb.Pos        // the ground last clicked: W walks there when nothing is selected
	hinted  bool           // the "Option-click to walk" hint has been shown this session
	cursor  *pb.Pos        // the large map's keyboard cursor
	side    screen.Sidebar // the sidebar beside a service
	charTab screen.Sidebar // the character view's own tab, scroll and cursors
}

// activeSide is the tab state the keys and clicks turn: the character view's
// when it is open, else the sidebar's.
func (a *App) activeSide() *screen.Sidebar {
	if a.char {
		return &a.mouse.charTab
	}
	return &a.mouse.side
}

// mouseView puts the mouse state in the view the screen draws.
func (a *App) mouseView(v *screen.View) {
	ms := &a.mouse
	v.Hits, v.Hover, v.Labels = &ms.hits, ms.hover, ms.alt || ms.labels
	v.Zoom, v.BigZoom, v.Card, v.Cursor, v.Side = ms.zoom, ms.bigZoom, ms.card, ms.cursor, a.activeSide()
}

// refresh makes sure the geometry of what is on the screen is as the screen is
// now, for a click to be read against it. The last drawing recorded it, so it
// draws again only when something has changed since (or the last drawing was
// for another size): a click, a wheel turn and a cursor key do not each cost a
// full render (T4).
func (a *App) refresh() {
	if p := &a.paint; p.hitsStale || p.hitsW != a.w || p.hitsH != a.h {
		_ = a.render(a.w, a.h, 0, screen.Plain())
	}
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
func (a *App) moveCursor(k string) {
	a.refresh()
	h := &a.mouse.hits
	at := a.mouse.cursor
	if at == nil {
		a.focusSess().View(func(m *model.Model) { at = m.Self.Pos })
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
	return (a.traders != nil || a.talk != nil || a.board != nil) && !a.char && a.trade == nil && a.cons == nil &&
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
	if a.replay != nil {
		return a, nil // a recording is drawn: nothing on it is the game
	}
	ms.alt = mo.Mod.Contains(tea.ModAlt)
	switch msg.(type) {
	case tea.MouseMotionMsg:
		ms.hover = &screen.Pt{X: mo.X, Y: mo.Y}
		return a, nil
	case tea.MouseWheelMsg:
		ms.hover = &screen.Pt{X: mo.X, Y: mo.Y}
		a.refresh()
		switch {
		case a.chat.drawer != nil && ms.hits.Drawer.W > 0 && ms.hits.Drawer.In(mo.X, mo.Y):
			// The wheel over the drawer scrolls the chat back and forward.
			a.chat.drawer.Scroll = max(a.chat.drawer.Scroll+map[tea.MouseButton]int{tea.MouseWheelUp: 3, tea.MouseWheelDown: -3}[mo.Button], 0)
			return a, nil
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
	for _, t := range h.Panel {
		if t.Rect.In(x, y) {
			return a.do(Action{Kind: SelectPanel, Count: t.Tab})
		}
	}
	if h.Card.W > 0 {
		// The character's card holds the clicks: an action, or nothing.
		for _, t := range h.CardActs {
			if t.Rect.In(x, y) {
				return a.do(Action{Kind: PersonDo, Count: t.Tab})
			}
		}
		if !h.Card.In(x, y) {
			return a.do(Action{Kind: PersonClose})
		}
		return a, nil
	}
	for _, n := range h.Names {
		if n.Rect.In(x, y) {
			return a.do(Action{Kind: OpenPerson, Note: n.Name})
		}
	}
	if a.chat.drawer != nil && h.Drawer.W > 0 && h.Drawer.In(x, y) {
		return a, nil // the drawer is over the map: a click on it walks nowhere
	}
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
		act = a.clickAction(m, pick, a.mouse.alt)
	})
	return a.do(act)
}

// clickAction is what a click on a pick means (D77, D109). A plain click never
// moves the character: the first click on a mob or an NPC selects it and opens
// its card, a second click on it acts (a first click never attacks), a click on
// loot takes everything within reach, a click on the ground lets go. Option-click
// walks: to the spot, or up to the thing and on to act on it.
func (a *App) clickAction(m *model.Model, p screen.Pick, alt bool) Action {
	act := a.pickAction(m, p, alt)
	if !m.TurnBased() {
		return act
	}
	// In a turn-based instance a bare intent is refused (D30): the click adds a step
	// to the plan, as the key would.
	switch act.Kind {
	case Approach:
		act = Action{Kind: Move, X: act.X, Y: act.Y} // a plan walks to the place: it cannot follow
		step := act
		return Action{Kind: PlanAdd, Step: &step}
	case Loot:
		act.ID = p.ID // a plan's loot step names its drop
		fallthrough
	case Move, Target, Attack:
		step := act
		return Action{Kind: PlanAdd, Step: &step}
	}
	return act
}

// pickAction is what a click on p means: alt is Option (Alt) held. A plain click
// never moves the character (D109); Option-click is the only mouse action that
// does.
func (a *App) pickAction(m *model.Model, p screen.Pick, alt bool) Action {
	ms := &a.mouse
	approachIt := func() Action {
		return Action{Kind: Approach, ID: p.ID, Count: int(p.Kind), Note: p.Name, X: p.Pos.GetX(), Y: p.Pos.GetY()}
	}
	switch p.Kind {
	case screen.PickSelf:
		return Action{} // you are already here
	case screen.PickMob:
		ms.sel = &selection{id: p.ID, kind: p.Kind, name: p.Name}
		selected := ms.card == p.ID
		ms.card = p.ID
		if alt {
			return approachIt()
		}
		switch {
		case selected && attackingIt(m, p.ID):
			if d, far := screen.AttackTooFar(m); far {
				return Action{Note: screen.FarWords(d)}
			}
			return Action{Note: "Already attacking " + p.Name + " (F stops)"}
		case selected:
			return Action{Kind: Attack, On: true, Note: "attack " + p.Name}
		}
		if m.Self.Target == p.ID {
			return Action{}
		}
		return Action{Kind: Target, ID: p.ID}
	case screen.PickNPC:
		ms.sel = &selection{id: p.ID, kind: p.Kind, name: p.Name}
		n := m.NPC(p.ID)
		inReach := n != nil && m.InReach(n)
		switch {
		case alt && !inReach:
			ms.card = p.ID
			return approachIt()
		case ms.card != p.ID && !alt:
			ms.card = p.ID
			if !inReach {
				return Action{Note: tooFar(m, p.Pos)}
			}
			return Action{}
		case !inReach:
			return Action{Note: tooFar(m, p.Pos)}
		}
		ms.card = "" // the conversation takes the card's place
		return Action{Kind: Talk, NPC: p.ID, Note: p.Name}
	case screen.PickDrop:
		ms.card = ""
		ms.sel = &selection{id: p.ID, kind: p.Kind, name: p.Name}
		d := m.Drops[p.ID]
		switch {
		case !d.GetCanLoot():
			return Action{Note: p.Name + " is not yours to take"}
		case d2(d.GetPos(), m.Self.Pos) > dropReachMm*dropReachMm:
			if alt {
				return approachIt()
			}
			return Action{Note: tooFar(m, d.GetPos())}
		}
		return Action{Kind: Loot}
	}
	ms.card, ms.cursor, ms.sel = "", nil, nil
	if alt {
		ms.spot = nil
		return Action{Kind: Move, X: p.Pos.GetX(), Y: p.Pos.GetY()}
	}
	// A plain click on the ground, a door or a marker only lets go of the selection.
	a.approach = nil
	ms.spot = p.Pos
	if !ms.hinted {
		ms.hinted = true
		return Action{Note: "Option-click to walk here (or click, then W)"}
	}
	return Action{}
}
