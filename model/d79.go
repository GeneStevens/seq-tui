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
	if m.Instance != nil && m.Joined != nil && m.Joined.Players > 0 {
		return m.Joined.Players
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
		return firstText(m.joinedTier())
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

// Slot holder states, as the server tells them (D82).
const (
	HolderHeld     = "held"     // the holder is in the instance
	HolderEmpty    = "empty"    // the holder is gone: open to a replacement between encounters
	HolderReserved = "reserved" // nobody has entered it yet
)

// Slot is one slot of the instance as the server lists it (D82).
type Slot struct {
	N int
	// Holder is the character's name, "" for a slot nobody has entered.
	Holder string
	// Player is the server's handle for the player, "Player 2": the same for
	// every viewer, and nothing about the account.
	Player string
	State  string
	Me     bool // the holder is this character
	Own    bool // the holder is another character of this player's account
}

// Slots lists the instance's slots as the server's holder list has them: who holds
// each, which player that is, and whether it is held, empty (open to a
// replacement) or reserved. The list is the snapshot's, kept current by the
// instance_slots event; a server that sends none gives no slots.
func (m *Model) Slots() []Slot {
	if m.Instance == nil {
		return nil
	}
	var out []Slot
	for _, h := range m.holders {
		s := Slot{N: int(h.GetSlot()), Holder: h.GetName(), Player: h.GetPlayer(), State: h.GetState()}
		if s.N == 0 {
			s.N = len(out) + 1
		}
		if h.GetCharacter() != "" && h.GetCharacter() == m.Self.ID && m.Self.ID != "" {
			s.Me = true
		} else if s.Holder != "" && s.State == HolderHeld {
			s.Own = m.isOwn(s.Holder)
		}
		out = append(out, s)
	}
	return out
}

// noteSlots keeps the holder list an instance_slots event carries.
func (m *Model) noteSlots(e *pb.Event) {
	if o := e.GetInstanceSlots(); o != nil && m.Instance != nil {
		m.holders = slices.Clone(o.GetHolders())
	}
}

// Joined is what the server told a character that filled a slot someone else
// held (D82): the preset and tier it joined and what the group rules had made of
// the instance.
type Joined struct {
	Preset, Tier, GroupTier string
	Players, Stirred, Slots int64
	Text                    string
}

// noteJoined keeps the instance_joined event.
func (m *Model) noteJoined(e *pb.Event) {
	if o := e.GetInstanceJoined(); o != nil {
		m.Joined = &Joined{Preset: o.GetPreset(), Tier: o.GetTier(), GroupTier: o.GetGroupTier(), Players: o.GetPlayers(), Stirred: o.GetStirred(), Slots: o.GetSlots(), Text: e.GetText()}
	}
}

// Vacant says how many slots have no holder in the instance now.
func (m *Model) Vacant() int {
	v := 0
	for _, s := range m.Slots() {
		if s.State == HolderEmpty {
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
// stirring, a personal reward and a seat taken in a running instance (D79, D81,
// D82). The server's own first-person words come first; the client's fallback is
// for a server that left them out.
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
		return "reward", firstText(e.GetText(), m.entitlementWords(w)), "", true
	case *pb.Event_InstanceJoined:
		o := b.InstanceJoined
		return "instance_joined", firstText(e.GetText(), m.joinedWords(o)), "", true
	case *pb.Event_EntitlementTaken:
		o := b.EntitlementTaken
		who := firstText(m.label(o.GetTaker()), "another")
		return "entitlement_taken", firstText(e.GetText(), fmt.Sprintf("Your other character, %s, took this account's %s", who, entitlementName(o.GetKind()))), "", true
	}
	return "", "", "", false
}

// joinedWords says what a replacement took a place in, when the server left the
// words out.
func (m *Model) joinedWords(o *pb.InstanceJoined) string {
	s := "You take a place in " + firstText(m.presetName(o.GetPreset()), "the instance")
	if o.GetTier() != "" {
		s += ": " + o.GetTier() + " tier"
	}
	s += ", " + playersCount(o.GetPlayers())
	if o.GetGroupTier() != "" {
		s += ", group tier " + o.GetGroupTier()
	}
	if o.GetStirred() > 0 {
		s += ", stirred"
	}
	return s
}

func playersCount(n int64) string {
	if n == 1 {
		return "1 player"
	}
	return fmt.Sprintf("%d players", n)
}

func (m *Model) presetName(id string) string {
	if p := m.Preset(id); p != nil {
		return p.GetName()
	}
	return id
}

// entitlementWords says what became of an account's personal reward for an
// encounter, and which of its characters used it.
func (m *Model) entitlementWords(w *pb.Reward) string {
	detail := w.GetDetail()
	if detail == "" { // the reward event does not carry it: the tier the instance runs at names it
		detail = m.Current().GetGroupTier().GetBonus()
	}
	what := entitlementName(detail)
	if w.GetItem() != "" {
		what += " (" + m.ItemName(w.GetItem()) + ")"
	}
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

// joinedTier is the group tier a replacement was told it joined, "" for none.
func (m *Model) joinedTier() string {
	if m.Joined != nil {
		return m.Joined.GroupTier
	}
	return ""
}
