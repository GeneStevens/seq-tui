package model

import (
	"slices"

	"google.golang.org/protobuf/proto"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// A tick frame (SEQ-119, D89) is the messages of one tick in one: the events,
// then a snapshot or self-view deltas, then the intent results and the door
// states. The model applies it as it would have applied them, in that order, so
// the stream of old and the frames give the same model.

// applyFrame folds one tick frame into the model.
func (m *Model) applyFrame(f *pb.TickFrame) {
	m.inFrame = true
	for i, e := range f.GetEvents() {
		// An event in a frame carries no number of its own: it is the base's and
		// its place in the frame. The message is never written to, because the
		// recording holds it too.
		m.applyEventAt(e, f.GetBaseSeq()+uint64(i))
		m.trainLearned(e)
	}
	m.inFrame = false
	if s := f.GetSnapshot(); s != nil {
		m.applySnapshot(s)
	}
	bag := false
	for _, d := range f.GetDeltas() {
		m.applyDelta(d)
		switch d.GetPart().(type) {
		case *pb.SelfDelta_Inventory, *pb.SelfDelta_Items:
			bag = true
		}
	}
	if bag {
		m.noteTradeTip(m.GameTimeMs) // once the items it names are known
	}
	for _, r := range f.GetResults() {
		m.applyResult(r)
	}
	if d := f.GetDoors(); d != nil {
		m.applyDoors(d)
	}
}

func (m *Model) applyDoors(d *pb.DoorStates) {
	m.Doors = map[string]*pb.DoorState{}
	for _, d := range d.GetDoors() {
		m.Doors[d.GetId()] = d
	}
}

// applyDelta replaces the part of the self view a delta names with what it
// holds, as the snapshot it stands in for would have set it. The rest of the
// self view (the target, the hints, the sheet, what is in view) is the events'.
func (m *Model) applyDelta(d *pb.SelfDelta) {
	s := &m.Self
	switch p := d.GetPart().(type) {
	case *pb.SelfDelta_Inventory:
		v := p.Inventory
		before := s.Inventory
		defer func() { m.noteQuestItemsGone(before, m.GameTimeMs) }()
		s.Inventory, s.Equipped, s.Bank = slices.Clone(v.GetInventory()), slices.Clone(v.GetEquipped()), slices.Clone(v.GetBank())
		s.InventorySize, s.BagSlots, s.BindZone = v.GetInventorySize(), v.GetBagSlots(), v.GetBindZone()
		s.BankSize, s.BankTiers = v.GetBankSize(), v.GetBankTiers()
	case *pb.SelfDelta_Money:
		s.Crowns, s.Owed = p.Money.GetCrowns(), slices.Clone(p.Money.GetOwed())
	case *pb.SelfDelta_Stats:
		m.applyStats(p.Stats.GetState())
	case *pb.SelfDelta_Quest:
		m.noteQuestChange(s.Quest, p.Quest.GetQuest())
		s.Quest = p.Quest.GetQuest()
	case *pb.SelfDelta_Companion:
		s.Companion = cloneCompanion(p.Companion.GetCompanion())
	case *pb.SelfDelta_Offers:
		s.Contracts, m.NPCs = slices.Clone(p.Offers.GetContracts()), p.Offers.GetNpcs()
		m.noteContracts(p.Offers.GetContracts(), p.Offers.GetNpcs())
		for _, n := range m.NPCs {
			m.noteNPC(n)
		}
	case *pb.SelfDelta_Group:
		m.setGroup(p.Group.GetGroup(), m.GameTimeMs)
	case *pb.SelfDelta_Trade:
		m.Trade = p.Trade.GetTrade()
	case *pb.SelfDelta_Items:
		for _, it := range p.Items.GetItems() {
			m.Items[it.GetId()] = it
		}
	}
}

// applyStats takes the character's own numbers from a stats delta: what a
// snapshot sets from its self, less what the delta does not carry.
func (m *Model) applyStats(st *pb.SelfState) {
	old := m.Self
	e, s := st.GetEntity(), &m.Self
	s.Level, s.HP, s.MaxHP, s.Dead = e.GetLevel(), e.GetHp(), e.GetMaxHp(), e.GetDead()
	if e.GetName() != "" {
		s.Name = e.GetName()
	}
	s.XP, s.Pos, s.Stats = st.GetXp(), clonePos(e.GetPos()), st.GetStats()
	s.Ground, s.SpeedPct, s.EffectiveSpeedMmS = st.GetGround(), st.GetSpeedPct(), st.GetEffectiveSpeedMmS()
	s.setKit(st, m.GameTimeMs)
	m.SelfMoves++
	m.remember(e)
	if m.levelPending || s.Level > old.Level {
		if len(old.Abilities) > 0 {
			m.levelUp(old)
		}
	}
	m.levelPending, m.levelTold = false, false
}

// noteEffects remembers the effects on an entity in view (D50), the
// character's own excepted: they are in its kit.
func (m *Model) noteEffects(en *pb.EntityState, now int64) {
	if fx := effectsFrom(en.GetEffects(), now); len(fx) > 0 && en.GetId() != m.Self.ID {
		if m.TargetEffects == nil {
			m.TargetEffects = map[string][]Effect{}
		}
		m.TargetEffects[en.GetId()] = fx
	}
}

// setGroup replaces the character's group with the members listed.
func (m *Model) setGroup(group []*pb.GroupMember, now int64) {
	m.Group = map[string]*Member{}
	for _, g := range group {
		delete(m.Entities, g.GetEntity().GetId()) // in the group, not also a stranger in view
		m.remember(g.GetEntity())
		m.noteEffects(g.GetEntity(), now)
		m.Group[g.GetEntity().GetId()] = &Member{Entity: cloneEntity(g.GetEntity()), Target: g.GetTarget(), AutoAttack: g.GetAutoAttack(), XP: g.GetXp(), Mana: g.GetMana(), MaxMana: g.GetMaxMana()}
	}
}

// What a frame leaves to the events. The server follows an event with the
// deltas of the parts of the self view it changes, chosen by the kind of event
// (rewardSections); where a snapshot of old would have changed something no delta
// names, the model makes the change from the event, as the snapshot would have.
// The three: the house companion's place in the group and its mana, and a lesson
// learned shown as known on the trainer. Each is in effect only while a frame is
// applied, so a stream of messages and the recordings of one are read as they were.

// companionJoins puts the house companion that came into view in the group, with
// the mana it had when it left (the frame does not carry the group's, and the
// events tell its mana only when it changes).
func (m *Model) companionJoins(en *pb.EntityState) {
	g := &Member{Entity: cloneEntity(en)}
	if mana, ok := m.companionMana[en.GetId()]; ok {
		g.Mana, g.MaxMana = mana[0], mana[1]
	}
	m.Group[en.GetId()] = g
}

// companionLeaves takes the companion out of the group, keeping its mana.
func (m *Model) companionLeaves(id string, g *Member) {
	if g.MaxMana > 0 {
		if m.companionMana == nil {
			m.companionMana = map[string][2]int64{}
		}
		m.companionMana[id] = [2]int64{g.Mana, g.MaxMana}
	}
	delete(m.Group, id)
}

// trainLearned marks the lesson a train reward paid for as known on every
// trainer that offers it.
func (m *Model) trainLearned(e *pb.Event) {
	w := e.GetReward()
	if w.GetKind() != "train" || w.GetRecipient() != m.Self.ID {
		return
	}
	cloned := false
	for i, n := range m.NPCs {
		learned := false
		for _, t := range n.GetTrains() {
			learned = learned || (t.GetAbility() == w.GetDetail() && !t.GetKnown())
		}
		if !learned {
			continue
		}
		if !cloned { // the slice is the message's, which the recording holds too
			m.NPCs, cloned = slices.Clone(m.NPCs), true
		}
		c := proto.Clone(n).(*pb.NpcInfo)
		for _, t := range c.GetTrains() {
			if t.GetAbility() == w.GetDetail() {
				t.Known = true
			}
		}
		m.NPCs[i] = c
		m.noteNPC(c)
	}
}
