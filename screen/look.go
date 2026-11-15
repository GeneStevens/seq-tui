package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// What a figure wears, in the server's words (D197.10). The phrase is agentkit's
// own; this file only decides which one a figure gets and where it is said.

// LookPhrase is the full look of a figure: a mob's appearance line when the
// server sends one, else the worn and held phrase. "" when the server told
// nothing (an older server sends no visible_gear).
func LookPhrase(e *pb.EntityState) string {
	if e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB && e.GetAppearance() != "" {
		return e.GetAppearance()
	}
	return agentkit.EntityLook(e, true)
}

// WearsLine is the "Wears: …" line of a figure, "" when it has no look to say.
func WearsLine(e *pb.EntityState) string {
	if p := LookPhrase(e); p != "" {
		return "Wears: " + p
	}
	return ""
}

// classWord is a player's class as the menu names it, else the id with a
// capital; "" when the server did not say.
func classWord(m *model.Model, e *pb.EntityState) string {
	id := e.GetClass()
	if id == "" {
		return ""
	}
	if c := m.Class(id); c.GetName() != "" {
		return c.GetName()
	}
	return strings.ToUpper(id[:1]) + id[1:]
}

// playerLabel is what hovering a player says: "Aria  Lv 3 Ranger", as a mob's
// says its level and band.
func playerLabel(m *model.Model, e *pb.EntityState) string {
	label := e.GetName()
	if e.GetLevel() > 0 {
		label += fmt.Sprintf("  Lv %d", e.GetLevel())
	}
	if c := classWord(m, e); c != "" {
		label += " " + c
	}
	return label
}
