package model

import (
	"fmt"
	"slices"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The group rules (D79, D81): how many players the instance holds and the tier
// that sets, what it stirred, who holds which slot, when a replacement may
// join, the player's status and the personal reward of an encounter. The wire
// carries counts and names, never an account (the D17 test bans the word), so
// a slot's account is told only as far as the client knows its own characters.

// Refusal is the server's last refusal to open or enter an expedition, in the
// server's words: one expedition per account, a join in the middle of an
// encounter, a slot that is not for you. It stays until an instance opens or
// the character goes in.
type Refusal struct {
	Reason string
	Text   string
}

// Status values a character may set on itself (D79).
const (
	StatusAway      = "away"
	StatusAutopilot = "autopilot"
)

// StatusWords is a status as a player reads it ("" for none).
func StatusWords(s string) string {
	switch s {
	case StatusAway:
		return "away"
	case StatusAutopilot:
		return "on autopilot"
	}
	return s
}

// SetOwn tells the model the names of the account's other characters this
// client plays (the second character of a duo), so a slot or a reward that
// belongs to one of them reads as the player's own.
func (m *Model) SetOwn(names ...string) {
	m.own = nil
	for _, n := range names {
		if n != "" {
			m.own = append(m.own, Key(n))
		}
	}
}

func (m *Model) isOwn(name string) bool { return slices.Contains(m.own, Key(name)) }

// refusalReason says whether reason is one of the refusals the expedition rules make (D79).
func refusalReason(reason string) bool {
	if strings.HasPrefix(reason, "expedition_active") {
		return true
	}
	switch reason {
	case "instance_exists", "encounter_open", "slot_account", "no_slot", "party_full", "not_in_party", "instance_closing", "instance_opening":
		return true
	}
	return false
}

// refusalWords is the refusal in plain words when the server sent none.
func refusalWords(reason, name string) string {
	if n, ok := strings.CutPrefix(reason, "expedition_active:"); ok {
		name = n
		reason = "expedition_active"
	}
	switch reason {
	case "expedition_active", "instance_exists":
		if name != "" {
			return "You are already on an expedition: " + name + ". Finish it or leave it first"
		}
		return "You are already on an expedition. Finish it or leave it first"
	case "encounter_open":
		return "An encounter is under way: a replacement can join between encounters"
	case "slot_account":
		return "That slot was held by another player: a second character of an account already in cannot take it"
	case "no_slot":
		return "Every slot is held: a replacement never adds one"
	case "party_full":
		return "The instance is full"
	case "not_in_party":
		return "That instance belongs to another group"
	}
	return Words(reason)
}

// noteRefusal keeps the refusal an event carries, when it is one of the
// expedition rules'.
func (m *Model) noteRefusal(e *pb.Event) {
	switch b := e.GetBody().(type) {
	case *pb.Event_InstanceOpenFailed:
		o := b.InstanceOpenFailed
		m.Refusal = &Refusal{Reason: o.GetReason(), Text: firstText(e.GetText(), refusalWords(o.GetReason(), o.GetName()))}
	case *pb.Event_Denied:
		d := b.Denied
		if d.GetActor() != m.Self.ID || !refusalReason(d.GetReason()) {
			return
		}
		m.Refusal = &Refusal{Reason: d.GetReason(), Text: firstText(d.GetText(), e.GetText(), refusalWords(d.GetReason(), ""))}
	}
}

func firstText(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// noteStirred keeps what the instance told this character it did (D79).
func (m *Model) noteStirred(e *pb.Event) {
	o := e.GetInstanceStirred()
	if o == nil {
		return
	}
	text := firstText(o.GetText(), e.GetText())
	if text != "" && !slices.Contains(m.stirs, text) {
		m.stirs = append(m.stirs, text)
	}
}

// Players is how many different players (accounts) the instance holds, 0 when
// the server did not say.
func (m *Model) Players() int64 {
	if m.Instance != nil && m.Instance.GetPlayers() > 0 {
		return m.Instance.GetPlayers()
	}
	if m.Opened != nil {
		return m.Opened.Players
	}
	return 0
}

// GroupTierName is the group tier the instance runs at, "" for none.
func (m *Model) GroupTierName() string {
	if m.Instance != nil && m.Instance.GetGroupTier() != "" {
		return m.Instance.GetGroupTier()
	}
	if m.Instance != nil {
		return ""
	}
	if m.Opened != nil {
		return m.Opened.GroupTier
	}
	return ""
}

// Stirred is what the instance said it did in answer to the group that opened
// it (D79): the snapshot's list, then any told since that it does not hold.
func (m *Model) Stirred() []string {
	var out []string
	if m.Instance != nil {
		out = append(out, m.Instance.GetStirred()...)
	}
	for _, s := range m.stirs {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// SlotCount is how many slots the instance was locked with (0: unknown).
func (m *Model) SlotCount() int64 {
	if m.Instance != nil && m.Instance.GetSlots() > 0 {
		return m.Instance.GetSlots()
	}
	if m.Instance == nil && m.Opened != nil {
		return m.Opened.Slots
	}
	return 0
}

// Slot is one slot of the instance as the client can tell it.
type Slot struct {
	N      int
	Holder string // the character's name; "" for an empty slot
	Me     bool   // the holder is this character
	Own    bool   // the holder is another character of this player's account
}

// Slots lists the instance's slots: this character, the other characters of
// its group in the instance, and the rest empty. The wire does not say which
// slot a character was made for, so the order is the order they are seen in.
func (m *Model) Slots() []Slot {
	n := m.SlotCount()
	if n == 0 || m.Instance == nil {
		return nil
	}
	var out []Slot
	add := func(name string, me, own bool) {
		out = append(out, Slot{N: len(out) + 1, Holder: name, Me: me, Own: own})
	}
	add(m.Self.Name, true, true)
	for _, id := range m.GroupIDs() {
		en := m.Group[id].Entity
		if en.GetKind() != pb.EntityKind_ENTITY_KIND_CHARACTER || en.GetOwner() != "" {
			continue
		}
		add(en.GetName(), false, m.isOwn(en.GetName()))
	}
	for int64(len(out)) < n {
		add("", false, false)
	}
	return out
}

// Vacant says how many slots have no holder in the instance now.
func (m *Model) Vacant() int {
	v := 0
	for _, s := range m.Slots() {
		if s.Holder == "" {
			v++
		}
	}
	return v
}

// EncounterOpen says an encounter of this character or its group is open: a
// replacement waits until it ends (D79).
func (m *Model) EncounterOpen() bool { return len(m.Encounters) > 0 }

// ReplacementWords says when a replacement may join.
func (m *Model) ReplacementWords() string {
	switch {
	case m.EncounterOpen():
		return "replacement waits for the encounter"
	case m.Vacant() == 0:
		return "replacement open; no slot is empty"
	}
	return "replacement open between encounters"
}

// describeGroup is the log line of an event of the group rules: the instance
// stirring and a personal reward (D79, D81).
func (m *Model) describeGroup(e *pb.Event) (kind, text, reason string, ok bool) {
	switch b := e.GetBody().(type) {
	case *pb.Event_InstanceStirred:
		o := b.InstanceStirred
		return "instance_stirred", firstText(o.GetText(), e.GetText(), "The instance stirs"), "", true
	case *pb.Event_Reward:
		w := b.Reward
		if w.GetKind() != "personal" {
			return "", "", "", false
		}
		return "reward", m.entitlementWords(w), "", true
	}
	return "", "", "", false
}

// entitlementWords says what became of an account's personal reward for an
// encounter, and which of its characters used it.
func (m *Model) entitlementWords(w *pb.Reward) string {
	detail := w.GetDetail()
	if detail == "" { // the reward event does not carry it: the tier the instance runs at names it
		detail = m.Current().GetGroupTier().GetBonus()
	}
	what := entitlementName(detail)
	who := m.label(w.GetRecipient())
	switch {
	case w.GetRecipient() == m.Self.ID && m.Self.ID != "":
		return fmt.Sprintf("Your account's %s for this encounter went to you", what)
	case m.isOwn(who):
		return fmt.Sprintf("Your account's %s for this encounter went to your other character, %s", what, who)
	}
	return fmt.Sprintf("%s used their own %s for this encounter", who, what)
}

// entitlementName is a reward's id as words: "bonus-roll" is "bonus roll".
func entitlementName(id string) string {
	if id == "" {
		return "personal reward"
	}
	return strings.NewReplacer("-", " ", "_", " ").Replace(id)
}
