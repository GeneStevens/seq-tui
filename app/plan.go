// Package app is the Bubble Tea program: it turns keys into the client's
// commands and the model into the screen. It holds no game rule; where a key
// needs a choice (the next target, the nearest drop) it only picks among
// things the model shows, and the server decides whether the intent is allowed.
package app

import (
	"fmt"
	"sort"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// StepMm is how far one press of a move key aims: the destination sent is this
// far from where the character stands.
const StepMm = 3000

// Kind says what a key does.
type Kind int

const (
	None Kind = iota
	Move
	Target // set the target to ID ("" clears it)
	Attack // turn auto-attack On
	Loot
	UseDoor
	Release
	Logout
	Quit
	ToggleHelp
	ToggleVerbose
	Stop       // a plan step: stop where you are
	OpenMenu   // the instance menu
	PlanAdd    // add Step to the plan
	PlanUndo   // remove the last step
	PlanClear  // remove every step
	PlanSubmit // send the plan
	OpenInstance
	UseAbility // use the ability in Slot (1 to 8); ID is a target, "" the current one
	SetSlot    // put the ability ID in Slot ("" empties it)
	OpenSlots  // the slot screen
	CreateCharacter
	OpenInventory // the inventory and gear screen
	OpenTraders   // the screen for the traders in view
	OpenTrade     // the trade window
	Equip         // wear the item ID
	Unequip       // take off the gear slot Gear
	Buy           // buy Count of the item ID from the trader NPC
	Sell          // sell the item ID to the trader NPC
	Train         // learn the ability ID from the trader NPC
	Bind          // set the bind point at the trader NPC
	BankDeposit   // bank the item ID at the trader NPC
	BankWithdraw  // take the item ID out of the bank at the trader NPC
	TradeOpen     // ask the character ID for a trade, or agree to its ask
	TradeAdd      // put the item ID in the offer
	TradeCrowns   // offer Amount crowns in all
	TradeRemove   // take the item ID back out of the offer
	TradeConfirm
	TradeCancel
	Talk            // talk to the NPC NPC
	OpenTradersAt   // the traders' window of the NPC NPC
	OpenBoard       // the contract board of the NPC NPC (Count 1: on the held contracts)
	OpenContracts   // the Contracts screen
	OpenOverview    // the whole-zone map
	ContractAccept  // take the contract ID from the board NPC
	ContractTurnIn  // hand the contract ID in at the board NPC
	ContractAbandon // give the contract ID up
	OpenRole        // the screen for choosing the recruit's role at the quest giver NPC
	CompanionSummon
	CompanionDismiss
	CompanionStance // set the companion's stance to ID
	CompanionRole   // the quest giver NPC makes the recruit's role ID
	CompanionOrder  // order the companion: ID is "attack" (Target), "move" (X, Y), "stop" or "clear"
	HelperNext      // the companion keys talk to the other helper
	DuoSummon       // bring the second character into play
	DuoDismiss      // log the second character out
	DuoStance       // set the second character's stance to ID
	DuoOrder        // order the second character, as CompanionOrder
	Focus           // switch the keys and the screen between the two characters (D56)
	OpenLine        // the command line for chat and the social commands
	SocialDo        // a request about playing together: ID is the verb, Note its argument
)

// Action is a key's meaning, with the choice it needs already made.
type Action struct {
	Kind Kind
	X, Y int64  // Move: the destination
	ID   string // Target, Loot, UseDoor: what it names
	// Target is the mob an order to attack names.
	Target string
	On     bool   // Attack
	Slot   int    // UseAbility, SetSlot: 1 to 8
	NPC    string // Buy, Sell, Train, Bind, BankDeposit, BankWithdraw: the trader
	Gear   string // Unequip: the gear slot
	// Count is how many Buy buys; Amount is the crowns of TradeCrowns.
	Count  int
	Amount int64
	Note   string  // None: why nothing happens, for the status line
	Step   *Action // PlanAdd: the step to add

	// PlanSubmit: the plan as the toolkit takes it, and the round and edit it
	// is a copy of.
	Steps []agentkit.Step
	Round uint64
	Ver   int
	// For is the character the action is for when it is not the player's own (D56).
	For string
}

// Plan decides what a key does against the model. It is pure.
func Plan(key string, m *model.Model) Action {
	sp := m.Self.Pos
	switch key {
	case "up", "w":
		return Action{Kind: Move, X: sp.GetX(), Y: sp.GetY() + StepMm}
	case "down", "s":
		return Action{Kind: Move, X: sp.GetX(), Y: sp.GetY() - StepMm}
	case "left", "a":
		return Action{Kind: Move, X: sp.GetX() - StepMm, Y: sp.GetY()}
	case "right", "d":
		return Action{Kind: Move, X: sp.GetX() + StepMm, Y: sp.GetY()}
	case "tab":
		if id := nextTarget(m); id != "" {
			return Action{Kind: Target, ID: id}
		}
		return Action{Note: "no target in view"}
	case "esc":
		if m.Self.Target != "" {
			return Action{Kind: Target}
		}
	case "f", "space", " ":
		return Action{Kind: Attack, On: !m.Self.AutoAttack}
	case "l":
		return lootAction(m, m.Self.Pos)
	case "e", "enter":
		if id := nearestDoor(m); id != "" {
			return Action{Kind: UseDoor, ID: id}
		}
		return Action{Note: "no door on this map"}
	case "r":
		return Action{Kind: Release} // the server denies it unless the character is dead
	case "g":
		return talkAction(m)
	case "k":
		return Action{Kind: OpenContracts}
	case "m":
		return Action{Kind: OpenOverview}
	}
	if n := slotKey(key); n > 0 {
		return Action{Kind: UseAbility, Slot: n, Note: slotLabel(m, n)}
	}
	return common(key)
}

// common are the keys that mean the same in real time and in a plan.
func common(key string) Action {
	switch key {
	case "o":
		return Action{Kind: Logout}
	case "q", "ctrl+c":
		return Action{Kind: Quit}
	case "?":
		return Action{Kind: ToggleHelp}
	case "v":
		return Action{Kind: ToggleVerbose}
	case "i":
		return Action{Kind: OpenMenu}
	case "b":
		return Action{Kind: OpenSlots}
	case "c":
		return Action{Kind: OpenInventory}
	case "n":
		return Action{Kind: OpenTraders}
	case "t":
		return Action{Kind: OpenTrade}
	case "/":
		return Action{Kind: OpenLine}
	}
	return Action{}
}

// slotKey is the slot a number key names (1 to 8), 0 for any other key.
func slotKey(key string) int {
	if len(key) == 1 && key[0] >= '1' && key[0] <= '8' {
		return int(key[0] - '0')
	}
	return 0
}

// slotLabel names a slot for the status line and the plan: "key 1 Flame Dart".
func slotLabel(m *model.Model, n int) string {
	if id := m.Self.Slots[n-1].Ability; id != "" {
		return fmt.Sprintf("key %d %s", n, m.AbilityName(id))
	}
	return fmt.Sprintf("key %d", n)
}

func d2(a, b *pb.Pos) int64 {
	dx, dy := a.GetX()-b.GetX(), a.GetY()-b.GetY()
	return dx*dx + dy*dy
}

// nextTarget is the living mob after the current target, nearest first, and
// round again.
func nextTarget(m *model.Model) string { return nextTargetFrom(m, m.Self.Target) }

// nextTargetFrom is the living mob after cur, nearest first.
func nextTargetFrom(m *model.Model, cur string) string {
	var ids []string
	for _, id := range m.EntityIDs() {
		if e := m.Entities[id]; e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB && !e.GetDead() {
			ids = append(ids, id)
		}
	}
	sort.SliceStable(ids, func(i, j int) bool {
		return d2(m.Entities[ids[i]].GetPos(), m.Self.Pos) < d2(m.Entities[ids[j]].GetPos(), m.Self.Pos)
	})
	for i, id := range ids {
		if id == cur {
			return ids[(i+1)%len(ids)]
		}
	}
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// lootAction loots the nearest drop this character may take from a point. A
// drop the server marks as not its to take (can_loot false: a Rare drop it is
// not entitled to) is not sent; the key says so.
func lootAction(m *model.Model, from *pb.Pos) Action {
	id := nearestDropTo(m, from)
	switch {
	case id == "":
		return Action{Note: "nothing to loot in view"}
	case !m.Drops[id].GetCanLoot():
		return Action{Note: "not lootable: the drops in view are not yours to take"}
	}
	return Action{Kind: Loot, ID: id}
}

// nearestDrop prefers a drop this character may take.
func nearestDrop(m *model.Model) string { return nearestDropTo(m, m.Self.Pos) }

func nearestDropTo(m *model.Model, from *pb.Pos) string {
	best, bestOK, bestD := "", false, int64(0)
	for _, id := range m.DropIDs() {
		d := m.Drops[id]
		ok, dd := d.GetCanLoot(), d2(d.GetPos(), from)
		if best == "" || (ok && !bestOK) || (ok == bestOK && dd < bestD) {
			best, bestOK, bestD = id, ok, dd
		}
	}
	return best
}

func nearestDoor(m *model.Model) string { return nearestDoorTo(m, m.Self.Pos) }

func nearestDoorTo(m *model.Model, from *pb.Pos) string {
	best, bestD := "", int64(0)
	for _, d := range m.Map.GetDoors() {
		if dd := d2(d.GetPos(), from); best == "" || dd < bestD {
			best, bestD = d.GetId(), dd
		}
	}
	return best
}
