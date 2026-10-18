package model

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"google.golang.org/protobuf/proto"
)

// Contracts, talk, rares and world events (SEQ-044, SEQ-045). As everywhere in
// this package, the model keeps what the server said and computes no rule.

// Dialogue is the last thing an NPC said to the character in answer to a talk:
// its lines, and the actions it offers. Talks counts the dialogues the character
// has been told, so a screen can tell a new answer from an old one.
type Dialogue struct {
	NPC     string
	Lines   []string
	Options []*pb.DialogueOption
	Talks   int
}

// WorldEventState is the last announcement of a timed world event in the zone:
// kind is "started", "leak", "won" or "lost"; At is the game time it was told.
// Active is true from "started" until "won" or "lost".
type WorldEventState struct {
	Event  string
	Name   string // the event's display name, as the server sent it (D55)
	Kind   string
	Text   string
	Leaks  int64
	At     int64
	Active bool
}

// IsRare says whether the entity is a roaming rare: the server marks it (D52).
func IsRare(e *pb.EntityState) bool { return e.GetRare() }

// RaresInView lists the roaming rares in view, sorted by id.
func (m *Model) RaresInView() []*pb.EntityState {
	var out []*pb.EntityState
	for _, id := range m.EntityIDs() {
		if e := m.Entities[id]; IsRare(e) && !e.GetDead() {
			out = append(out, e)
		}
	}
	return out
}

// Contract is the held contract with that id, or nil.
func (m *Model) Contract(id string) *pb.ContractInfo {
	for _, c := range m.Self.Contracts {
		if c.GetId() == id {
			return c
		}
	}
	return nil
}

// ContractName is the name of a contract the model has seen held or offered, or
// its id.
func (m *Model) ContractName(id string) string {
	if n := m.contractNames[id]; n != "" {
		return n
	}
	return id
}

// noteContracts remembers the names of the contracts a snapshot offered or held,
// so a log line can name one that is gone by then. A held contract carries its
// own name, text and reward wherever the character is (D52).
func (m *Model) noteContracts(s *pb.Snapshot) {
	if m.contractNames == nil {
		m.contractNames = map[string]string{}
	}
	for _, c := range s.GetSelf().GetContracts() {
		m.contractNames[c.GetId()] = c.GetName()
	}
	m.noteQuestHeld(s.GetSelf().GetContracts())
	for _, n := range s.GetNpcs() {
		for _, o := range n.GetOffers() {
			m.contractNames[o.GetId()] = o.GetName()
		}
	}
}

// Boards lists the contract boards in view, nearest first.
func (m *Model) Boards() []*pb.NpcInfo { return m.npcsWith("contract_board") }

func (m *Model) npcsWith(role string) []*pb.NpcInfo {
	var out []*pb.NpcInfo
	for _, n := range m.NPCs {
		if n.GetRole() == role {
			out = append(out, n)
		}
	}
	sp := m.Self.Pos
	sort.SliceStable(out, func(i, j int) bool {
		return d2(out[i].GetPos(), sp) < d2(out[j].GetPos(), sp)
	})
	return out
}

func d2(a, b *pb.Pos) int64 {
	dx, dy := a.GetX()-b.GetX(), a.GetY()-b.GetY()
	return dx*dx + dy*dy
}

// NearestNPC is the NPC the character has seen nearest to it (any role), nil if
// none. The snapshot lists the NPCs in view when it is sent and the character
// walks on after it, so the ones remembered are the better guide to who is near.
func (m *Model) NearestNPC() *pb.NpcInfo {
	var best *pb.NpcInfo
	for _, n := range m.KnownNPCs() {
		if best == nil || d2(n.GetPos(), m.Self.Pos) < d2(best.GetPos(), m.Self.Pos) {
			best = n
		}
	}
	return best
}

// NPCName is what an NPC in view is called, else its id.
func (m *Model) NPCName(id string) string {
	if n := m.NPC(id); n != nil && n.GetName() != "" {
		return n.GetName()
	}
	if n := m.npcNames[id]; n != "" {
		return n
	}
	return m.label(id)
}

// Said is the last dialogue the NPC gave the character, nil if it has none.
func (m *Model) Said(npc string) *Dialogue { return m.Dialogues[npc] }

// EventName is a world event's name as the server announced it (D55); its id
// when the event has not been announced to this model.
func (m *Model) EventName(id string) string {
	if s := m.WorldEvents[id]; s != nil && s.Name != "" {
		return s.Name
	}
	return id
}

// Title is the event's name, its id if the server sent none.
func (s *WorldEventState) Title() string {
	if s.Name != "" {
		return s.Name
	}
	return s.Event
}

// ActiveEvents lists the world events announced as started and not yet over,
// sorted by id.
func (m *Model) ActiveEvents() []*WorldEventState {
	var out []*WorldEventState
	for _, e := range m.WorldEvents {
		if e.Active {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Event < out[j].Event })
	return out
}

// RecentEvent is the world event announced last, if it was told within window
// ms of game time before now; nil otherwise. A banner shows it.
func (m *Model) RecentEvent(now, window int64) *WorldEventState {
	var best *WorldEventState
	for _, e := range m.WorldEvents {
		if !e.Active && now-e.At > window {
			continue
		}
		if best == nil || e.At > best.At || (e.At == best.At && e.Event < best.Event) {
			best = e
		}
	}
	return best
}

// owedEventPay says a world event's pay is owed pay (D55): the event ended while
// the character was away, so the pay waited for its next join and arrives with
// it, with no announcement of the event's end before it.
func (m *Model) owedEventPay(w *pb.Reward) bool {
	e := m.WorldEvents[w.GetDetail()]
	return e == nil || e.Active
}

// joinPay is the log line of a snapshot that follows a join and shows more XP or
// crowns than the character had before it (D55): what a world event owed it, or
// a refund, paid by the join. The server tells nothing else of it: a stream
// starts from the snapshot, which already shows the join's tick, so the reward
// events of the join are not sent.
func (m *Model) joinPay(old Self) {
	xp, crowns := m.Self.XP-old.XP, m.Self.Crowns-old.Crowns
	if xp <= 0 && crowns <= 0 {
		return
	}
	text := "owed pay arrived with the join: " + m.RewardWords(&pb.RewardInfo{Xp: max(xp, 0), Crowns: max(crowns, 0)})
	m.Log = append(m.Log, LogEntry{Seq: m.last, GameTimeMs: m.GameTimeMs, Kind: "reward", Text: text, Actor: m.Self.ID})
	if over := len(m.Log) - m.logCap; over > 0 {
		m.Log = append([]LogEntry(nil), m.Log[over:]...)
	}
}

// applyDialogue keeps what an NPC said to the character.
func (m *Model) applyDialogue(d *pb.Dialogue) {
	if d.GetActor() != m.Self.ID {
		return
	}
	if m.Dialogues == nil {
		m.Dialogues = map[string]*Dialogue{}
	}
	m.Talks++
	opts := d.GetOptions()
	if m.Companion() == nil && len(m.questOrder) == 0 { // the house companion is shelved (D56): its option leads nowhere
		opts = slices.DeleteFunc(slices.Clone(opts), func(o *pb.DialogueOption) bool { return o.GetAction() == "companion_role" })
	}
	m.Dialogues[d.GetNpc()] = &Dialogue{NPC: d.GetNpc(), Lines: d.GetLines(), Options: opts, Talks: m.Talks}
}

// applyWorldEvent keeps the last announcement of a world event.
func (m *Model) applyWorldEvent(w *pb.WorldEvent, at int64) {
	if m.WorldEvents == nil {
		m.WorldEvents = map[string]*WorldEventState{}
	}
	s := &WorldEventState{Event: w.GetEvent(), Name: w.GetName(), Kind: w.GetKind(), Text: w.GetText(), Leaks: w.GetLeaks(), At: at}
	s.Active = w.GetKind() == "started" || w.GetKind() == "leak"
	m.WorldEvents[w.GetEvent()] = s
}

// Place is what the character has seen of a space, kept for the map: the NPCs
// (they stand where they stand).
type Place struct {
	NPCs map[string]*pb.NpcInfo
}

func (m *Model) place() *Place {
	if m.Seen == nil {
		m.Seen = map[string]*Place{}
	}
	p := m.Seen[m.Space]
	if p == nil {
		p = &Place{NPCs: map[string]*pb.NpcInfo{}}
		m.Seen[m.Space] = p
	}
	return p
}

// noteNPC remembers an NPC seen in the character's space, and what it is called.
func (m *Model) noteNPC(n *pb.NpcInfo) {
	if m.npcNames == nil {
		m.npcNames = map[string]string{}
	}
	m.npcNames[n.GetId()] = n.GetName()
	for _, t := range n.GetTrains() {
		if t.GetInfo().GetId() != "" {
			m.Lessons[t.GetInfo().GetId()] = t.GetInfo()
		}
	}
	m.noteQuestOffers(n)
	if m.Space != "" {
		m.place().NPCs[n.GetId()] = n
	}
}

// npcAppeared adds an NPC that came into view (D52), in place of one with its
// id (a fresh description of it).
func (m *Model) npcAppeared(n *pb.NpcInfo) {
	if n.GetId() == "" {
		return
	}
	m.noteNPC(n)
	for i, o := range m.NPCs {
		if o.GetId() == n.GetId() {
			m.NPCs[i] = n
			return
		}
	}
	m.NPCs = append(m.NPCs, n)
}

// npcMoved puts an NPC the server streams (a guard on its patrol, D56) where
// it now is.
func (m *Model) npcMoved(id string, pos *pb.Pos) {
	for i, n := range m.NPCs {
		if n.GetId() == id {
			c := proto.Clone(n).(*pb.NpcInfo)
			c.Pos = clonePos(pos)
			m.NPCs[i] = c
			m.noteNPC(c)
			return
		}
	}
}

// owedAtLogin lists the owed pay the first snapshot of a login shows, in the
// log, one line for each (D55, D56): it is paid at the character's next join.
func (m *Model) owedAtLogin() {
	for _, o := range m.Self.Owed {
		what := "the " + o.GetName()
		if o.GetKind() == "refund" {
			what = "a refund for " + o.GetName()
		}
		text := fmt.Sprintf("owed pay waiting from %s: %s, paid at your next join", what, m.RewardWords(&pb.RewardInfo{Xp: o.GetXp(), Crowns: o.GetCrowns()}))
		m.Log = append(m.Log, LogEntry{Seq: m.last, GameTimeMs: m.GameTimeMs, Kind: "reward", Text: text, Actor: m.Self.ID})
	}
	if over := len(m.Log) - m.logCap; over > 0 {
		m.Log = append([]LogEntry(nil), m.Log[over:]...)
	}
}

// npcLeft removes an NPC that went out of view. What the character has seen of
// it stays in Seen for the map.
func (m *Model) npcLeft(id string) {
	m.NPCs = slices.DeleteFunc(slices.Clone(m.NPCs), func(n *pb.NpcInfo) bool { return n.GetId() == id })
}

// Marker is where a held contract's scout marker is: the zone map's marker of
// that id (D52), else the position the contract itself carries. Nil when the
// contract has no marker, it is reached, or the character is not in its zone.
func (m *Model) Marker(c *pb.ContractInfo) *pb.Pos {
	if c.GetMarker() == "" || c.GetHave() >= c.GetNeed() {
		return nil
	}
	for _, k := range m.Map.GetMarkers() {
		if k.GetId() == c.GetMarker() {
			return k.GetPos()
		}
	}
	return c.GetMarkerPos()
}

// MarkerName is a marker's display name from the zone map (D56), else its id.
func (m *Model) MarkerName(id string) string {
	for _, k := range m.Map.GetMarkers() {
		if k.GetId() == id && k.GetName() != "" {
			return k.GetName()
		}
	}
	return id
}

// Camps are where the zone's mobs are found, from its map (D52).
func (m *Model) Camps() []*pb.Camp { return m.Map.GetCamps() }

// Known is what the character has seen of its space (never nil).
func (m *Model) Known() *Place {
	if p := m.Seen[m.Space]; p != nil {
		return p
	}
	return &Place{}
}

// KnownNPCs lists the NPCs seen in the character's space and those its zone map
// lists, sorted by id: those in view now are the snapshot's, the rest are
// remembered or the map's.
func (m *Model) KnownNPCs() []*pb.NpcInfo {
	byID := map[string]*pb.NpcInfo{}
	for id, n := range m.Known().NPCs {
		byID[id] = n
	}
	for _, n := range m.NPCs {
		byID[n.GetId()] = n
	}
	// The zone map lists the NPCs that stand where they are (D69), seen or not.
	for _, n := range m.Map.GetNpcs() {
		if byID[n.GetId()] == nil {
			byID[n.GetId()] = &pb.NpcInfo{Id: n.GetId(), Name: n.GetName(), Role: n.GetRole(), Pos: n.GetPos()}
		}
	}
	out := make([]*pb.NpcInfo, 0, len(byID))
	for _, n := range byID {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetId() < out[j].GetId() })
	return out
}

// Progress says how far a held contract has got in words: "2 of 3 killed",
// "1 of 3 carried", "go to the marker" or "marker reached", "ready to hand in".
func Progress(c *pb.ContractInfo) string {
	switch c.GetKind() {
	case "hunt", "bounty":
		return fmt.Sprintf("%d of %d killed", c.GetHave(), c.GetNeed())
	case "gather":
		return fmt.Sprintf("%d of %d carried", c.GetHave(), c.GetNeed())
	case "scout":
		if c.GetHave() >= c.GetNeed() {
			return "marker reached"
		}
		return "go to the marker"
	}
	return fmt.Sprintf("%d of %d", c.GetHave(), c.GetNeed())
}

// RewardWords says what a contract or event pays: "30 XP, 12 crowns and Tin Ring".
func (m *Model) RewardWords(r *pb.RewardInfo) string {
	var parts []string
	if r.GetXp() > 0 {
		parts = append(parts, fmt.Sprintf("%d XP", r.GetXp()))
	}
	if r.GetCrowns() > 0 {
		parts = append(parts, crowns(r.GetCrowns()))
	}
	if r.GetItem() != "" {
		parts = append(parts, m.ItemName(r.GetItem()))
	}
	switch len(parts) {
	case 0:
		return "nothing"
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

// describeContracts is the log line of a talk, a world event, a contract
// reward or a rare coming into view.
func (m *Model) describeContracts(e *pb.Event) (kind, text, reason string, ok bool) {
	switch b := e.GetBody().(type) {
	case *pb.Event_EntityAppeared:
		if en := b.EntityAppeared.GetEntity(); IsRare(en) {
			return "rare_appeared", fmt.Sprintf("a rare, %s (Lv %d), is in view (%s)", en.GetName(), en.GetLevel(), en.GetId()), "", true
		}
	case *pb.Event_NpcAppeared:
		n := b.NpcAppeared.GetNpc()
		return "npc_appeared", fmt.Sprintf("%s (%s) is in view", n.GetName(), n.GetRole()), "", true
	case *pb.Event_NpcLeft:
		return "npc_left", m.NPCName(b.NpcLeft.GetId()) + " is out of view", "", true
	case *pb.Event_Dialogue:
		d := b.Dialogue
		who := m.NPCName(d.GetNpc())
		if d.GetActor() != m.Self.ID {
			return "dialogue", fmt.Sprintf("%s spoke to %s", who, m.label(d.GetActor())), "", true
		}
		return "dialogue", fmt.Sprintf("%s says: %s", who, strings.Join(d.GetLines(), " ")), "", true
	case *pb.Event_WorldEvent:
		w := b.WorldEvent
		name := w.GetName()
		if name == "" {
			name = m.EventName(w.GetEvent())
		}
		switch w.GetKind() {
		case "started":
			return "world_event", fmt.Sprintf("World event, %s, has begun: %s", name, w.GetText()), "", true
		case "leak":
			return "world_event", fmt.Sprintf("%s: a raider got through (%d so far). %s", name, w.GetLeaks(), w.GetText()), "", true
		case "won":
			return "world_event", fmt.Sprintf("%s is won: %s", name, w.GetText()), "", true
		case "lost":
			return "world_event", fmt.Sprintf("%s is lost: %s", name, w.GetText()), "", true
		}
		return "world_event", fmt.Sprintf("%s: %s %s", name, w.GetKind(), w.GetText()), "", true
	case *pb.Event_Reward:
		w := b.Reward
		who := m.label(w.GetRecipient())
		if w.GetRecipient() == m.Self.ID {
			who = "you"
		}
		name := m.ContractName(w.GetDetail())
		switch w.GetKind() {
		case "contract_accept":
			return "reward", fmt.Sprintf("%s took the contract %s", who, name), "", true
		case "contract_abandon":
			return "reward", fmt.Sprintf("%s gave up the contract %s", who, name), "", true
		case "contract_progress":
			c := m.Contract(w.GetDetail())
			switch {
			case c != nil && c.GetKind() == "scout":
				return "reward", fmt.Sprintf("contract %s: the marker is reached; return to the board", name), "", true
			case c != nil:
				return "reward", fmt.Sprintf("contract %s: %d of %d", name, w.GetAmount(), c.GetNeed()), "", true
			}
			return "reward", fmt.Sprintf("contract %s: %d so far", name, w.GetAmount()), "", true
		case "contract_turn_in":
			return "reward", fmt.Sprintf("%s handed in %s to %s and got %s", who, name, m.NPCName(w.GetSource()),
				m.RewardWords(&pb.RewardInfo{Xp: w.GetXp(), Crowns: w.GetAmount(), Item: w.GetItem()})), "", true
		case "world_event":
			pay := m.RewardWords(&pb.RewardInfo{Xp: w.GetXp(), Crowns: w.GetAmount(), Item: w.GetItem()})
			event := w.GetEventName() // the world event's own name (D56)
			if event == "" {
				event = m.EventName(w.GetDetail())
			}
			if m.owedEventPay(w) {
				return "reward", fmt.Sprintf("owed pay arrived for the %s, which ended while you were away: %s", event, pay), "", true
			}
			return "reward", fmt.Sprintf("%s was paid for the %s: %s", who, event, pay), "", true
		}
	}
	return "", "", "", false
}
