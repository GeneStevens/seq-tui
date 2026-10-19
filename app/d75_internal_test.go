package app

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// vendorWorld is a character two metres from a vendor, with a guard far off.
func vendorWorld() (*App, *model.Model) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self:  &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 10000}, Hp: 50, MaxHp: 50}, Crowns: 30},
		Items: []*pb.ItemInfo{{Id: "worn-cap", Name: "Worn Cap", Tier: "common", Slot: "head", SellPrice: 2}},
		Npcs: []*pb.NpcInfo{
			{Id: "vendor", Name: "Hilda Brasswick", Title: "Vendor", Role: "vendor", ReachMm: 5000, Pos: &pb.Pos{X: 12000}, Stock: []*pb.StockItem{{Item: "worn-cap", Price: 8}}},
			{Id: "guard", Name: "Town Guard", Title: "Guard", Role: "guard", ReachMm: 5000, Pos: &pb.Pos{X: 60000}},
		},
	}}})
	return New(client.NewSession(client.Config{Model: m})), m
}

func screenNow(a *App) string {
	return sgr.ReplaceAllString(a.render(140, 45, 0, screen.Plain()), "")
}

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// The trade window stays with the vendor it was opened on, whoever walks by
// (D78): it held an index into a list sorted by distance, and a passing guard
// switched it to the guard and back.
func TestAWindowStaysWithItsNPCWhenAGuardWalksBy(t *testing.T) {
	a, m := vendorWorld()
	a.key("n")
	a.key("enter") // the nearest: the vendor
	if s := screenNow(a); !strings.Contains(s, "For sale (Enter buys one)") || !strings.Contains(s, "Hilda Brasswick <Vendor>") {
		t.Fatalf("no vendor window:\n%s", s)
	}
	// The guard steps up beside the character: nearer than the vendor.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 2, GameTimeMs: 2000,
		Body: &pb.Event_NpcMoved{NpcMoved: &pb.NpcMoved{Id: "guard", Pos: &pb.Pos{X: 10500}}}}}})
	if list := screen.TradersInView(m); list[0].GetId() != "guard" {
		t.Fatalf("the guard is not first in the list now: %v", list)
	}
	s := screenNow(a)
	if !strings.Contains(s, "For sale (Enter buys one)") || !strings.Contains(s, "Hilda Brasswick <Vendor>") || strings.Contains(s, "Nothing to buy or sell here") {
		t.Fatalf("the window followed the guard:\n%s", s)
	}
	var got Action
	a.sess.View(func(m *model.Model) { got = a.tradersKey("enter", m) })
	if got.Kind != Buy || got.NPC != "vendor" {
		t.Fatalf("Enter acts on %+v, not on the vendor", got)
	}
	// And back: the guard walks off again; the window has not moved.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 3, GameTimeMs: 3000,
		Body: &pb.Event_NpcMoved{NpcMoved: &pb.NpcMoved{Id: "guard", Pos: &pb.Pos{X: 60000}}}}}})
	if s := screenNow(a); !strings.Contains(s, "Hilda Brasswick <Vendor>") {
		t.Fatalf("the window lost the vendor:\n%s", s)
	}
}

// A window closes only when the character is out of the NPC's reach, with a
// line that says so.
func TestAWindowClosesWhenTheCharacterIsOutOfReach(t *testing.T) {
	for _, tc := range []struct {
		name string
		open func(a *App)
		shut func(a *App) bool
		what string
	}{
		{"trade", func(a *App) { a.traders = &screen.Traders{Open: true, NPC: "vendor"} }, func(a *App) bool { return a.traders == nil }, "trade"},
		{"talk", func(a *App) { a.talk = &screen.Talk{NPC: "vendor"} }, func(a *App) bool { return a.talk == nil }, "conversation"},
		{"board", func(a *App) { a.board = &screen.Board{NPC: "vendor"} }, func(a *App) bool { return a.board == nil }, "board"},
	} {
		a, m := vendorWorld()
		tc.open(a)
		a.walkedAway(m)
		if tc.shut(a) {
			t.Fatalf("%s: closed while in reach", tc.name)
		}
		m.Self.Pos = &pb.Pos{X: 30000} // 18 m away
		a.walkedAway(m)
		if !tc.shut(a) || a.view.Notice != "You walked away from Hilda Brasswick, so the "+tc.what+" closed" {
			t.Fatalf("%s: open %v, notice %q", tc.name, !tc.shut(a), a.view.Notice)
		}
	}
}

// The bell is off unless asked for, and rings once for each mob that begins an
// attack (D75).
func TestTheBellRingsOnceForEachAttack(t *testing.T) {
	attack := func(m *model.Model, seq uint64, mob string) {
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: seq, GameTimeMs: 2000,
			Body: &pb.Event_AttackBegan{AttackBegan: &pb.AttackBegan{Actor: "me", Mob: mob, Name: "Hedge Brigand", Level: 4, Con: "hard", Danger: true}}}}})
	}
	for _, on := range []bool{false, true} {
		a, m := vendorWorld()
		a.Bell = on
		if a.ring(m) {
			t.Fatalf("bell %v: rang with no attack", on)
		}
		attack(m, 2, "mob-1")
		if got := a.ring(m); got != on {
			t.Fatalf("bell %v: rang %v at the first attack", on, got)
		}
		if a.ring(m) {
			t.Fatalf("bell %v: rang twice for one attack", on)
		}
		attack(m, 3, "mob-2")
		if got := a.ring(m); got != on {
			t.Fatalf("bell %v: rang %v at the second attack", on, got)
		}
	}
}
