package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
)

// The card of a character (D101 §9): what a click on a name in the chat opens. It
// shows what the server says of the character in answer to a whois (name, class,
// level, zone or "anonymous", whether it is online), and the four things to do:
// Tell, Invite, Ignore, Report. The screen only draws it; the program asks the
// server and carries the actions out.

// The states of a card.
const (
	CardAsking  = iota // asked, no answer yet
	CardFound          // the server knows the character
	CardUnknown        // the server knows no character by that name
	CardFailed         // the question was not answered (Err says why)
)

// CharCard is the card of one character. Class, Level, Zone, Online and Status are
// the server's whois reply; Zone is empty for an offline character and one that is
// anonymous.
type CharCard struct {
	Name   string
	State  int
	Kind   string
	Class  string
	Level  uint32
	Zone   string
	Online bool
	Status string
	Err    string
}

// The actions of a card, in the order they are drawn.
const (
	ActTell = iota
	ActInvite
	ActIgnore
	ActReport
	CardActions
)

// CardKeys are the keys of the card's actions, by action.
var CardKeys = [CardActions]string{"t", "i", "n", "r"}

var cardActLabels = [CardActions]string{"Tell", "Invite", "Ignore", "Report"}

// Acts says the card has actions: the character is known, or being asked about.
func (c *CharCard) Acts() bool { return c.State == CardAsking || c.State == CardFound }

// Where says where the character is, in words.
func (c *CharCard) Where() string {
	switch {
	case !c.Online:
		return "offline"
	case c.Zone == "":
		return "online, anonymous: the zone is hidden"
	}
	return "online, in " + c.Zone
}

// Info are the plain lines of the card above its actions: who, class and level,
// where, status.
func (c *CharCard) Info() []string {
	switch c.State {
	case CardAsking:
		return []string{c.Name, "asking the server…"}
	case CardUnknown:
		return []string{c.Name, "no character by that name"}
	case CardFailed:
		return []string{c.Name, "could not ask: " + c.Err}
	}
	who := c.Name
	switch c.Kind {
	case model.KindGM:
		who += "  (game master)"
	case model.KindNPC:
		who += "  (NPC)"
	}
	level := "level unknown"
	if c.Level > 0 {
		level = fmt.Sprintf("level %d", c.Level)
	}
	lines := []string{who, strings.TrimSpace(strings.ToUpper(level[:1]) + level[1:] + " " + c.Class), c.Where()}
	if c.Status != "" {
		lines = append(lines, "status: "+model.StatusWords(c.Status))
	}
	return lines
}

// actionsRow is the row of actions as keyed markup, and the column each begins at
// within it (the markup's braces are not drawn).
func actionsRow() (markup string, cols [CardActions]int) {
	var b strings.Builder
	col := 1
	b.WriteString(" ")
	for i := range cardActLabels {
		cols[i] = col
		item := "{" + strings.ToUpper(CardKeys[i]) + "} " + cardActLabels[i]
		b.WriteString(item)
		col += width(item) - 2
		if i < CardActions-1 {
			b.WriteString("   ")
			col += 3
		}
	}
	return b.String(), cols
}

// personHeight is how many rows the card takes, its frame included.
func personHeight(c *CharCard) int {
	n := len(c.Info()) + 2 // a blank and the hint, or the actions
	if c.Acts() {
		n += 2
	}
	return n + 2
}

// withPerson puts the card of a character over the top of the map panel mp, which
// is w cells wide and starts on the screen row top. Everything beside the map
// stays, with the vitals and the target: the card hides the map and nothing that
// is danger.
func withPerson(mp []string, m *model.Model, v View, w, top int, st Styler) []string {
	c := v.Person
	if c == nil {
		return mp
	}
	h := personHeight(c)
	if h > len(mp)-2 {
		return mp
	}
	var inner []string
	info := c.Info()
	for i, l := range info {
		l = fit(" "+l, w-2)
		if i == 0 {
			l = st.Bold(l)
		}
		inner = append(inner, l)
	}
	inner = append(inner, fit("", w-2))
	if c.Acts() {
		markup, cols := actionsRow()
		inner = append(inner, keyed(markup, w-2, ident, st))
		if v.Hits != nil {
			y := top + 1 + len(inner) - 1
			for i, col := range cols {
				v.Hits.CardActs = append(v.Hits.CardActs, Tab{Rect{1 + col, y, width(cardActLabels[i]) + 2, 1}, i})
			}
		}
		inner = append(inner, fit("", w-2))
	}
	inner = append(inner, st.Dim(fit(" Esc closes", w-2)))
	if v.Hits != nil {
		v.Hits.Card = Rect{0, top, w, h}
	}
	return append(box("Character: "+clean(c.Name), w, h, inner, st), mp[h:]...)
}
