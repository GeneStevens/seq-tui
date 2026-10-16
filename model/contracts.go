package model

import (
	"fmt"
	"sort"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
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
	Kind   string
	Text   string
	Leaks  int64
	At     int64
	Active bool
}

// campCell is the size of the square, in millimetres, in which one camp is
// remembered: two mobs first seen closer than this are one camp.
const campCell = 12000

// rareNames are the roaming rares the content has. The protocol does not mark a
// mob as a rare (the snapshot has no such flag), so the client knows them by
// name: this is a guess the screen makes, and a design question for the report.
var rareNames = map[string]bool{"Old Grimtusk": true, "Ancient Scuttler": true}

// IsRare says whether the entity is a roaming rare, as far as the client knows.
func IsRare(e *pb.EntityState) bool {
	return e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB && rareNames[e.GetName()]
}

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
// its id made readable.
func (m *Model) ContractName(id string) string {
	if n := m.contractNames[id]; n != "" {
		return n
	}
	return readable(id)
}

// contractText is what the model has been told of what a contract is called, says
// and pays.
type contractText struct {
	name, description string
	reward            *pb.RewardInfo
}

// noteContracts remembers the names, texts and rewards of the contracts a
// snapshot offered or held, so a log line can name one that is gone by then, and
// a held contract can be told in full away from its board: the server describes
// a held contract from the zone it is in, and a zone without the board's contracts
// (the Downs) gives only its id.
func (m *Model) noteContracts(s *pb.Snapshot) {
	if m.contractNames == nil {
		m.contractNames = map[string]string{}
		m.contractTexts = map[string]contractText{}
	}
	for _, c := range s.GetSelf().GetContracts() {
		if c.GetName() != c.GetId() && c.GetName() != "" {
			m.contractNames[c.GetId()] = c.GetName()
			m.contractTexts[c.GetId()] = contractText{c.GetName(), c.GetDescription(), c.GetReward()}
		}
	}
	for _, n := range s.GetNpcs() {
		for _, o := range n.GetOffers() {
			m.contractNames[o.GetId()] = o.GetName()
			m.contractTexts[o.GetId()] = contractText{o.GetName(), o.GetDescription(), o.GetReward()}
		}
	}
}

// completeContracts fills in the name, text and reward of a held contract the
// server described by its id alone, from what an earlier snapshot told.
func (m *Model) completeContracts() {
	for i, c := range m.Self.Contracts {
		t, ok := m.contractTexts[c.GetId()]
		if !ok || (c.GetName() != c.GetId() && c.GetName() != "") {
			continue
		}
		full := &pb.ContractInfo{Id: c.GetId(), Name: t.name, Description: t.description, Kind: c.GetKind(), Have: c.GetHave(),
			Need: c.GetNeed(), Ready: c.GetReady(), Reward: c.GetReward(), Marker: c.GetMarker(), MarkerPos: c.GetMarkerPos()}
		if full.Reward == nil {
			full.Reward = t.reward
		}
		m.Self.Contracts[i] = full
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

// EventName is a world event's name: its id made readable ("harvest-raid" is
// "Harvest Raid").
func EventName(id string) string { return readable(id) }

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

// applyDialogue keeps what an NPC said to the character.
func (m *Model) applyDialogue(d *pb.Dialogue) {
	if d.GetActor() != m.Self.ID {
		return
	}
	if m.Dialogues == nil {
		m.Dialogues = map[string]*Dialogue{}
	}
	m.Talks++
	m.Dialogues[d.GetNpc()] = &Dialogue{NPC: d.GetNpc(), Lines: d.GetLines(), Options: d.GetOptions(), Talks: m.Talks}
}

// applyWorldEvent keeps the last announcement of a world event.
func (m *Model) applyWorldEvent(w *pb.WorldEvent, at int64) {
	if m.WorldEvents == nil {
		m.WorldEvents = map[string]*WorldEventState{}
	}
	s := &WorldEventState{Event: w.GetEvent(), Kind: w.GetKind(), Text: w.GetText(), Leaks: w.GetLeaks(), At: at}
	s.Active = w.GetKind() == "started" || w.GetKind() == "leak"
	m.WorldEvents[w.GetEvent()] = s
}

// Place is what the character has seen of a space, kept for the map: the NPCs
// (they stand where they stand) and the camps (see noteCamps).
type Place struct {
	NPCs  map[string]*pb.NpcInfo
	Camps []*pb.Pos
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

// noteNPCs remembers the NPCs in view in the character's space.
func (m *Model) noteNPCs() {
	if m.Space == "" {
		return
	}
	for _, n := range m.NPCs {
		m.place().NPCs[n.GetId()] = n
	}
}

// noteCamps remembers where mobs were first seen, as camps, in the space the
// character is in. It does not remember a mob that appears while a world event
// is on (the event's mobs march; they are not a camp), or a rare.
func (m *Model) noteCamps(e *pb.EntityState) {
	if e.GetKind() != pb.EntityKind_ENTITY_KIND_MOB || e.GetDead() || IsRare(e) || len(m.ActiveEvents()) > 0 || m.Space == "" {
		return
	}
	p := m.place()
	c := roundDiv(e.GetPos().GetX(), campCell)
	r := roundDiv(e.GetPos().GetY(), campCell)
	for _, q := range p.Camps {
		if roundDiv(q.GetX(), campCell) == c && roundDiv(q.GetY(), campCell) == r {
			return
		}
	}
	p.Camps = append(p.Camps, clonePos(e.GetPos()))
}

func roundDiv(a, b int64) int64 {
	if a < 0 {
		return -((-a + b/2) / b)
	}
	return (a + b/2) / b
}

// Known is what the character has seen of its space (never nil).
func (m *Model) Known() *Place {
	if p := m.Seen[m.Space]; p != nil {
		return p
	}
	return &Place{}
}

// KnownNPCs lists the NPCs seen in the character's space, sorted by id: those
// in view now are the snapshot's, the rest are remembered.
func (m *Model) KnownNPCs() []*pb.NpcInfo {
	byID := map[string]*pb.NpcInfo{}
	for id, n := range m.Known().NPCs {
		byID[id] = n
	}
	for _, n := range m.NPCs {
		byID[n.GetId()] = n
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
	case *pb.Event_Dialogue:
		d := b.Dialogue
		who := m.NPCName(d.GetNpc())
		if d.GetActor() != m.Self.ID {
			return "dialogue", fmt.Sprintf("%s spoke to %s", who, m.label(d.GetActor())), "", true
		}
		return "dialogue", fmt.Sprintf("%s says: %s", who, strings.Join(d.GetLines(), " ")), "", true
	case *pb.Event_WorldEvent:
		w := b.WorldEvent
		name := EventName(w.GetEvent())
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
			return "reward", fmt.Sprintf("%s was paid for the %s: %s", who, EventName(w.GetDetail()),
				m.RewardWords(&pb.RewardInfo{Xp: w.GetXp(), Crowns: w.GetAmount(), Item: w.GetItem()})), "", true
		}
	}
	return "", "", "", false
}
