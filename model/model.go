// Package model is the state the screen shows. It is a pure Go type built only
// from protocol messages (seq-design doc 07 §1): the snapshot, events, intent
// results, the zone map. It computes no game rule; where it changes a number it
// does what the event says (a hit of N takes N from the target's hit points).
package model

import (
	"slices"
	"sort"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Default bounds of the event log, the intent list and the error list.
const (
	DefaultLogCap     = 200
	DefaultIntentsCap = 32
	errorsCap         = 16
)

// IntentState is where an intent stands.
type IntentState int

const (
	IntentPending IntentState = iota
	IntentAccepted
	IntentDenied
)

func (s IntentState) String() string {
	return [...]string{"pending", "accepted", "denied"}[s]
}

// IntentStatus is one recent intent and the server's answer to it.
type IntentStatus struct {
	ID     string
	Seq    uint64 // the client's submission sequence (D33)
	Kind   string // "move", "target", ...; "?" for a result to an intent not recorded as sent
	State  IntentState
	Reason string // set exactly when denied
	// Unknown: the server answered from its high-water mark (D41): the
	// intent was applied, but the original result was forgotten, so it may
	// have been denied when it was judged.
	Unknown bool
}

// Self is the character's own state.
type Self struct {
	ID, Name       string
	Level          int64
	HP, MaxHP      int64
	Dead           bool
	Released       bool  // a dead character released; it respawns after RespawnInMs
	RespawnInMs    int64 // game time from the released event
	XP             int64
	Pos            *pb.Pos
	Target         string
	AutoAttack     bool
	HasDestination bool
	Destination    *pb.Pos
	Stats          *pb.Stats
	Inventory      []string

	// The economy (SEQ-040): crowns, what is worn (one entry for each gear
	// slot in the order of GearSlots, "" for an empty one; empty when nothing
	// is), the bank, the zone of the bind point ("" until the character binds)
	// and how many items the inventory holds.
	Crowns        int64
	Contracts     []*pb.ContractInfo // the contracts held, with their progress (SEQ-044)
	Companion     *pb.CompanionState // the house companion (SEQ-049); nil before the quest is begun
	Equipped      []string
	Bank          []string
	BindZone      string
	InventorySize int64

	// Class, and the XP the level began at and ends at (XPNext 0: the cap).
	Class           string
	XPStart, XPNext int64
	// The combat model (SEQ-038): mana, the global cooldown, what the
	// character knows, what is in each slot and when each ability is ready,
	// the cast in progress and the effects on it. LastCastEnd is the last cast
	// that did not land, until another begins.
	Mana, MaxMana int64
	GCDUntilMs    int64
	GCDMs         int64
	Abilities     []*pb.AbilityInfo
	Slots         [Slots]SlotState
	ReadyAtMs     map[string]int64
	Cast          *Cast
	Effects       []Effect
	LastCastEnd   *CastEnd
}

// SlotState is what an ability slot holds.
type SlotState struct{ Ability string }

// Member is a group member, seen wherever it is.
type Member struct {
	Entity     *pb.EntityState
	Target     string
	AutoAttack bool
	XP         int64
}

// Model is what a client knows of the world.
type Model struct {
	Space      string
	Tick       uint64
	GameTimeMs int64
	Self       Self
	Group      map[string]*Member
	Entities   map[string]*pb.EntityState // in view, not the character or its group
	Drops      map[string]*pb.LootDropState
	Encounters map[string]*pb.EncounterState
	Map        *pb.ZoneMap // nil until it arrives, which is after the snapshot (D40)

	// Presets are the instance menu of the last zone map that listed one: the
	// world zone's, kept while the character is inside an instance.
	Presets []*pb.InstancePreset
	// Opened is the instance the character's group opened, until it ends;
	// Ended is the last one that ended. Round is the turn-based round, nil in
	// a real-time space.
	Opened *Opened
	Ended  *Ended
	Round  *Round
	// TargetEffects are the effects on the other entities in view, as the
	// events told (the snapshot lists only the character's own).
	TargetEffects map[string][]Effect
	// Instance is what the server says of the instance the character is in
	// (D49): its preset, party cap, time mode, difficulty and sleep limit. Nil
	// in a world zone.
	Instance *pb.InstanceInfo

	// Items are the items the snapshots described (the server sends every item
	// the character carries, wears, banks, offers or sees on sale), NPCs the
	// vendors, trainers, binders and bankers in view, and Trade the character's
	// trade window: nil when it has none.
	Items map[string]*pb.ItemInfo
	// Lessons are the abilities the trainers in view teach, as they described
	// them (D52): the names and words of abilities the character does not know.
	Lessons map[string]*pb.AbilityInfo
	NPCs    []*pb.NpcInfo
	Trade   *pb.TradeState

	// Dialogues are what each NPC last said to the character, by NPC id, and
	// Talks counts them; WorldEvents are the last announcement of each timed
	// event of the zone; Seen is what the character has seen of each space
	// (its NPCs and where mobs first stood), remembered for the map.
	Dialogues   map[string]*Dialogue
	Talks       int
	WorldEvents map[string]*WorldEventState
	Seen        map[string]*Place

	// Welcome is the latest welcome; Connections counts the welcomes seen
	// (a resume after a drop is a second connection).
	Welcome     *pb.Welcome
	Connections int

	// Heard counts every message applied, a pong included: the screen's
	// "last heard" clock, since a quiet zone sends no events.
	Heard uint64

	// Log is the bounded, ordered event log, oldest first.
	Log []LogEntry
	// Intents are the most recent intents, oldest first.
	Intents []IntentStatus
	// Errors are the protocol errors received, most recent last.
	Errors []*pb.Error

	// Duplicates counts events dropped as already applied; Faults lists gaps in
	// the stream numbers. A healthy session has neither (D39).
	Duplicates int
	Faults     []string

	names map[string]string // id to name, for every entity ever seen, so the log can name those that left
	// npcNames and contractNames are the names of the NPCs and contracts the
	// snapshots told, kept after they leave view.
	npcNames      map[string]string
	contractNames map[string]string
	// questGiver is the quest giver seen (an NPC with that role), questOffered the
	// contract ids it offers now and questSteps/questOrder every one it has offered.
	questGiver   string
	questOffered map[string]bool
	questSteps   map[string]*pb.ContractOffer
	questOrder   []string
	last         uint64 // the stream number of the last event applied, or of the snapshot
	selected     bool   // a snapshot has arrived
	// levelPending: a reward raised the level and the snapshot that says what it
	// taught has not come yet.
	levelPending bool
	logCap       int
	intentCap    int
}

// New returns an empty model with the default bounds.
func New() *Model { return NewBounded(DefaultLogCap, DefaultIntentsCap) }

// NewBounded returns an empty model with the given event log and intent list bounds.
func NewBounded(logCap, intentCap int) *Model {
	return &Model{logCap: max(logCap, 1), intentCap: max(intentCap, 1), Group: map[string]*Member{}, names: map[string]string{},
		Items: map[string]*pb.ItemInfo{}, Lessons: map[string]*pb.AbilityInfo{}, Entities: map[string]*pb.EntityState{}, Drops: map[string]*pb.LootDropState{}, Encounters: map[string]*pb.EncounterState{}}
}

// Last is the character's stream number of the last event applied, or of the
// snapshot: what a resume names (D39).
func (m *Model) Last() uint64 { return m.last }

// Ready reports whether a snapshot has arrived.
func (m *Model) Ready() bool { return m.selected }

// IntentSent records an intent as sent, pending its result.
func (m *Model) IntentSent(id string, seq uint64, kind string) {
	m.addIntent(IntentStatus{ID: id, Seq: seq, Kind: kind})
}

// PendingIntents lists the ids of the intents with no result yet, oldest first.
func (m *Model) PendingIntents() []string {
	var ids []string
	for _, i := range m.Intents {
		if i.State == IntentPending {
			ids = append(ids, i.ID)
		}
	}
	return ids
}

// Intent returns the status of the intent with id.
func (m *Model) Intent(id string) (IntentStatus, bool) {
	for _, i := range m.Intents {
		if i.ID == id {
			return i, true
		}
	}
	return IntentStatus{}, false
}

func (m *Model) addIntent(s IntentStatus) {
	m.Intents = append(m.Intents, s)
	if over := len(m.Intents) - m.intentCap; over > 0 {
		m.Intents = slices.Clone(m.Intents[over:])
	}
}

// Apply folds one server message into the model.
func (m *Model) Apply(msg *pb.ServerMessage) {
	m.Heard++
	switch b := msg.GetBody().(type) {
	case *pb.ServerMessage_Welcome:
		m.Welcome = b.Welcome
		m.Connections++
	case *pb.ServerMessage_Snapshot:
		m.applySnapshot(b.Snapshot)
	case *pb.ServerMessage_ZoneMap:
		if !m.selected || m.Space == "" || b.ZoneMap.GetSpace() == m.Space {
			m.Map = b.ZoneMap
			if len(b.ZoneMap.GetInstancePresets()) > 0 {
				m.Presets = b.ZoneMap.GetInstancePresets()
			}
		}
	case *pb.ServerMessage_Event:
		m.applyEvent(b.Event)
	case *pb.ServerMessage_IntentResult:
		m.applyResult(b.IntentResult)
	case *pb.ServerMessage_Error:
		m.Errors = append(m.Errors, b.Error)
		if over := len(m.Errors) - errorsCap; over > 0 {
			m.Errors = slices.Clone(m.Errors[over:])
		}
	case *pb.ServerMessage_Pong:
		// nothing to show
	}
}

func (m *Model) applyResult(r *pb.IntentResult) {
	for i := range m.Intents {
		if m.Intents[i].ID == r.GetIntentId() {
			m.Intents[i].State, m.Intents[i].Reason, m.Intents[i].Unknown = IntentAccepted, "", r.GetResultUnknown()
			if !r.GetAccepted() {
				m.Intents[i].State, m.Intents[i].Reason = IntentDenied, r.GetReason()
			}
			return
		}
	}
	s := IntentStatus{ID: r.GetIntentId(), Kind: "?", State: IntentAccepted, Unknown: r.GetResultUnknown()}
	if !r.GetAccepted() {
		s.State, s.Reason = IntentDenied, r.GetReason()
	}
	m.addIntent(s)
}

func (m *Model) applySnapshot(s *pb.Snapshot) {
	if m.Map != nil && m.Map.GetSpace() != s.GetSpace() {
		m.Map = nil // the map of the space the character left
	}
	if m.Space != s.GetSpace() {
		m.Round = nil // a round belongs to its instance
	}
	self := s.GetSelf()
	e := self.GetEntity()
	old := m.Self
	m.Space, m.Tick, m.GameTimeMs, m.Instance = s.GetSpace(), s.GetTick(), s.GetGameTimeMs(), s.GetInstance()
	m.Self = Self{
		ID: e.GetId(), Name: e.GetName(), Level: e.GetLevel(), HP: e.GetHp(), MaxHP: e.GetMaxHp(), Dead: e.GetDead(),
		XP: self.GetXp(), Pos: clonePos(e.GetPos()), Target: self.GetTarget(), AutoAttack: self.GetAutoAttack(),
		HasDestination: self.GetHasDestination(), Destination: clonePos(self.GetDestination()),
		Stats: self.GetStats(), Inventory: slices.Clone(self.GetInventory()),
		Crowns: self.GetCrowns(), Equipped: slices.Clone(self.GetEquipped()), Bank: slices.Clone(self.GetBank()),
		BindZone: self.GetBindZone(), InventorySize: self.GetInventorySize(), Contracts: slices.Clone(self.GetContracts()), Companion: cloneCompanion(self.GetCompanion()),
	}
	m.Self.setKit(self, s.GetGameTimeMs())
	m.Group, m.Entities = map[string]*Member{}, map[string]*pb.EntityState{}
	m.remember(e)
	// Everything in view carries its effects (D50): a fresh snapshot, a login in
	// the middle of a fight, tells what lies on each of them.
	m.TargetEffects = map[string][]Effect{}
	noteEffects := func(en *pb.EntityState) {
		if fx := effectsFrom(en.GetEffects(), s.GetGameTimeMs()); len(fx) > 0 && en.GetId() != m.Self.ID {
			m.TargetEffects[en.GetId()] = fx
		}
	}
	for _, it := range s.GetItems() {
		m.Items[it.GetId()] = it
	}
	m.NPCs, m.Trade = s.GetNpcs(), s.GetTrade()
	m.noteContracts(s)
	for _, n := range m.NPCs {
		m.noteNPC(n)
	}
	m.Drops, m.Encounters = map[string]*pb.LootDropState{}, map[string]*pb.EncounterState{}
	for _, g := range s.GetGroup() {
		m.remember(g.GetEntity())
		noteEffects(g.GetEntity())
		m.Group[g.GetEntity().GetId()] = &Member{Entity: cloneEntity(g.GetEntity()), Target: g.GetTarget(), AutoAttack: g.GetAutoAttack(), XP: g.GetXp()}
	}
	for _, en := range s.GetEntities() {
		m.remember(en)
		noteEffects(en)
		m.Entities[en.GetId()] = cloneEntity(en)
	}
	for _, d := range s.GetDrops() {
		m.Drops[d.GetId()] = d
	}
	for _, en := range s.GetEncounters() {
		m.Encounters[en.GetId()] = en
	}
	m.last = s.GetSeq()
	m.selected = true
	if old.ID == m.Self.ID && (m.levelPending || m.Self.Level > old.Level) && len(old.Abilities) > 0 {
		m.levelUp(old)
	}
	m.levelPending = false
}

func (m *Model) applyEvent(e *pb.Event) {
	if _, ok := e.GetBody().(*pb.Event_SpaceChanged); ok {
		// A crossing is announced outside the numbering of either stream
		// (seq 0), before the new space's snapshot (D40).
		m.apply(e)
		return
	}
	n := e.GetSeq()
	switch {
	case n <= m.last:
		m.Duplicates++ // a resume repeats nothing (D39); if it does, once is enough
		return
	case n != m.last+1:
		m.Faults = append(m.Faults, "event "+itoa(n)+" after "+itoa(m.last))
	}
	m.last = n
	m.apply(e)
}

func (m *Model) apply(e *pb.Event) {
	m.GameTimeMs = max(m.GameTimeMs, e.GetGameTimeMs())
	m.noteDescribed(e)      // before the log line, which names the item with its tier
	if !m.repeatsRound(e) { // the server says the open round again after a snapshot: once in the log is enough
		m.logEvent(e)
	}
	switch b := e.GetBody().(type) {
	case *pb.Event_SpaceChanged:
		// The snapshot of the new space follows and replaces this view.
		m.Space, m.Map, m.Round, m.Instance = b.SpaceChanged.GetSpace(), nil, nil, nil
	case *pb.Event_EntityAppeared:
		en := b.EntityAppeared.GetEntity()
		m.remember(en)
		switch {
		case en.GetId() == m.Self.ID:
		case m.Group[en.GetId()] != nil:
			m.Group[en.GetId()].Entity = cloneEntity(en)
		default:
			m.Entities[en.GetId()] = cloneEntity(en)
		}
	case *pb.Event_EntityMoved:
		mv := b.EntityMoved
		if en := m.entity(mv.GetId()); en != nil {
			en.Pos = clonePos(mv.GetPos())
		} else if mv.GetId() == m.Self.ID {
			m.Self.Pos = clonePos(mv.GetPos())
			if m.Self.HasDestination && m.Self.Destination.GetX() == mv.GetPos().GetX() && m.Self.Destination.GetY() == mv.GetPos().GetY() {
				m.Self.HasDestination = false
			}
		}
	case *pb.Event_EntityLeft:
		delete(m.Entities, b.EntityLeft.GetId())
	case *pb.Event_NpcAppeared:
		m.npcAppeared(b.NpcAppeared.GetNpc())
	case *pb.Event_NpcLeft:
		m.npcLeft(b.NpcLeft.GetId())
	case *pb.Event_Combat:
		m.combat(b.Combat)
	case *pb.Event_EncounterOpened:
		o := b.EncounterOpened
		m.Encounters[o.GetId()] = &pb.EncounterState{Id: o.GetId(), Mob: o.GetMob(), Owner: o.GetOwner()}
	case *pb.Event_EncounterClosed:
		delete(m.Encounters, b.EncounterClosed.GetId())
	case *pb.Event_Reward:
		w := b.Reward
		if w.GetKind() == "loot" || w.GetKind() == "crowns" {
			delete(m.Drops, w.GetSource()) // a loot reward names the drop it took (sim: Target is the drop)
		}
		if w.GetRecipient() == m.Self.ID {
			switch w.GetKind() {
			case "xp":
				m.Self.XP += w.GetAmount()
				if w.GetLevel() > m.Self.Level {
					m.Self.Level, m.levelPending = w.GetLevel(), true // the fresh snapshot follows
				}
			case "loot":
				m.Self.Inventory = append(m.Self.Inventory, w.GetItem())
			case "crowns":
				m.Self.Crowns += w.GetAmount()
			}
			// The other economic rewards (buy, sell, train, bind, fee, refund,
			// equip, bank, trade) are followed by a fresh snapshot with the new
			// state; the log line says what happened until it arrives.
		} else if g := m.Group[w.GetRecipient()]; g != nil && w.GetKind() == "xp" {
			g.XP += w.GetAmount()
		}
	case *pb.Event_TargetChanged:
		t := b.TargetChanged
		if t.GetActor() == m.Self.ID {
			m.Self.Target = t.GetTarget()
		} else if g := m.Group[t.GetActor()]; g != nil {
			g.Target = t.GetTarget()
		}
	case *pb.Event_TargetCleared:
		t := b.TargetCleared
		if t.GetActor() == m.Self.ID {
			m.Self.Target = ""
		} else if g := m.Group[t.GetActor()]; g != nil {
			g.Target = ""
		}
	case *pb.Event_AutoAttackChanged:
		a := b.AutoAttackChanged
		if a.GetActor() == m.Self.ID {
			m.Self.AutoAttack = a.GetOn()
		} else if g := m.Group[a.GetActor()]; g != nil {
			g.AutoAttack = a.GetOn()
		}
	case *pb.Event_LootDropped:
		d := b.LootDropped
		m.Drops[d.GetDrop()] = &pb.LootDropState{Id: d.GetDrop(), Item: d.GetItem(), Pos: d.GetPos(), CanLoot: d.GetCanLoot(), Crowns: d.GetCrowns()}
	case *pb.Event_LootGone:
		delete(m.Drops, b.LootGone.GetDrop()) // expired or taken (D41): gone now, not at the next snapshot
	case *pb.Event_Released:
		if r := b.Released; r.GetActor() == m.Self.ID {
			m.Self.Released, m.Self.RespawnInMs = true, r.GetRespawnInMs()
		}
	case *pb.Event_Respawned:
		// The character itself is sent a fresh snapshot after this event; the
		// rest are told here.
		r := b.Respawned
		if r.GetActor() == m.Self.ID {
			m.Self.Dead, m.Self.Released, m.Self.HP, m.Self.Pos = false, false, r.GetHp(), clonePos(r.GetPos())
			m.Self.Mana = m.Self.MaxMana // a respawn restores the mana as well as the hit points
		} else if en := m.entity(r.GetActor()); en != nil {
			en.Dead, en.Hp, en.Pos = false, r.GetHp(), clonePos(r.GetPos())
		}
	case *pb.Event_MobReset:
		if en := m.Entities[b.MobReset.GetId()]; en != nil {
			en.Hp, en.Dead = b.MobReset.GetHp(), false
		}
	case *pb.Event_Dialogue:
		m.applyDialogue(b.Dialogue)
	case *pb.Event_WorldEvent:
		m.applyWorldEvent(b.WorldEvent, e.GetGameTimeMs())
	case *pb.Event_InstanceOpened, *pb.Event_InstanceEnded, *pb.Event_RoundStarted, *pb.Event_RoundResolved:
		m.applyInstanceEvent(e)
	default:
		m.applyCombatEvent(e)
	}
}

func (m *Model) remember(e *pb.EntityState) {
	if e.GetName() != "" {
		m.names[e.GetId()] = e.GetName()
	}
}

// entity finds a non-self entity in view or in the group.
func (m *Model) entity(id string) *pb.EntityState {
	if en := m.Entities[id]; en != nil {
		return en
	}
	if g := m.Group[id]; g != nil {
		return g.Entity
	}
	return nil
}

func (m *Model) combat(c *pb.Combat) {
	hp := func(f func(hp, max int64) int64) {
		if c.GetTarget() == m.Self.ID {
			m.Self.HP = f(m.Self.HP, m.Self.MaxHP)
		} else if en := m.entity(c.GetTarget()); en != nil {
			en.Hp = f(en.Hp, en.MaxHp)
		}
	}
	switch c.GetKind() {
	case pb.CombatKind_COMBAT_KIND_HIT:
		hp(func(h, _ int64) int64 { return max(h-c.GetAmount(), 0) })
	case pb.CombatKind_COMBAT_KIND_DEATH:
		hp(func(int64, int64) int64 { return 0 })
		if c.GetTarget() == m.Self.ID {
			m.Self.Dead = true
		} else if en := m.entity(c.GetTarget()); en != nil {
			en.Dead = true
		}
		delete(m.TargetEffects, c.GetTarget())
		// Nothing else is inferred from a death: the server says when the
		// target is cleared (target_cleared, then auto_attack_changed off).
	case pb.CombatKind_COMBAT_KIND_HEAL:
		hp(func(h, mx int64) int64 { return min(h+c.GetAmount(), mx) })
	}
}

// EntityIDs lists the ids of the entities in view, sorted.
func (m *Model) EntityIDs() []string { return sortedKeys(m.Entities) }

// DropIDs lists the ids of the drops, sorted.
func (m *Model) DropIDs() []string { return sortedKeys(m.Drops) }

// EncounterIDs lists the ids of the open encounters, sorted.
func (m *Model) EncounterIDs() []string { return sortedKeys(m.Encounters) }

// GroupIDs lists the ids of the group members, sorted.
func (m *Model) GroupIDs() []string { return sortedKeys(m.Group) }

func sortedKeys[V any](mp map[string]V) []string {
	ids := make([]string, 0, len(mp))
	for id := range mp {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func clonePos(p *pb.Pos) *pb.Pos { return &pb.Pos{X: p.GetX(), Y: p.GetY(), Z: p.GetZ()} }

func cloneEntity(e *pb.EntityState) *pb.EntityState {
	return &pb.EntityState{Id: e.GetId(), Kind: e.GetKind(), Name: e.GetName(), Pos: clonePos(e.GetPos()),
		Hp: e.GetHp(), MaxHp: e.GetMaxHp(), Level: e.GetLevel(), Dead: e.GetDead(), Rare: e.GetRare(), Owner: e.GetOwner()}
}

func cloneCompanion(c *pb.CompanionState) *pb.CompanionState {
	if c == nil {
		return nil
	}
	return &pb.CompanionState{Stage: c.GetStage(), Unlocked: c.GetUnlocked(), Role: c.GetRole(), Stance: c.GetStance(),
		Summoned: c.GetSummoned(), Present: c.GetPresent(), Id: c.GetId(), Order: c.GetOrder()}
}
