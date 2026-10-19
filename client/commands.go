package client

import (
	"context"
	"errors"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Commands: one function for each intent the protocol has. Each sends the
// intent and returns its id at once; Result waits for the server's answer and
// matches it by that id. The server decides the outcome; the client does not
// check a rule first (doc 07 §1).

// Move walks toward a point (millimetres).
func (s *Session) Move(ctx context.Context, x, y int64) (string, error) {
	return s.intent(ctx, "move", func(i *pb.Intent) { i.Body = &pb.Intent_Move{Move: &pb.Move{Destination: &pb.Pos{X: x, Y: y}}} })
}

// Stop cancels a move in progress.
func (s *Session) Stop(ctx context.Context) (string, error) {
	return s.intent(ctx, "stop", func(i *pb.Intent) { i.Body = &pb.Intent_Stop{Stop: &pb.Stop{}} })
}

// Target selects a target; "" clears it.
func (s *Session) Target(ctx context.Context, id string) (string, error) {
	return s.intent(ctx, "target", func(i *pb.Intent) { i.Body = &pb.Intent_Target{Target: &pb.SetTarget{Target: id}} })
}

// Attack turns auto-attack on or off.
func (s *Session) Attack(ctx context.Context, on bool) (string, error) {
	return s.intent(ctx, "attack", func(i *pb.Intent) { i.Body = &pb.Intent_Attack{Attack: &pb.Attack{On: on}} })
}

// Loot takes the item from a drop.
func (s *Session) Loot(ctx context.Context, drop string) (string, error) {
	return s.intent(ctx, "loot", func(i *pb.Intent) { i.Body = &pb.Intent_Loot{Loot: &pb.Loot{Drop: drop}} })
}

// UseDoor crosses a zone line or enters or leaves an instance (D39).
func (s *Session) UseDoor(ctx context.Context, door string) (string, error) {
	return s.intent(ctx, "use_door", func(i *pb.Intent) { i.Body = &pb.Intent_UseDoor{UseDoor: &pb.UseDoor{Door: door}} })
}

// Release gives up a dead character's body; it respawns at the zone's start
// point after the zone's delay (D41).
func (s *Session) Release(ctx context.Context) (string, error) {
	return s.intent(ctx, "release", func(i *pb.Intent) { i.Body = &pb.Intent_Release{Release: &pb.Release{}} })
}

// Logout leaves the world at once (D39); the server ends the connection when
// the logout is journaled.
func (s *Session) Logout(ctx context.Context) error {
	c := s.client()
	if c == nil {
		return errors.New("not connected")
	}
	s.mu.Lock()
	s.out = true
	s.mu.Unlock()
	return c.Logout(ctx)
}

// LoggedOut reports whether Logout was sent.
func (s *Session) LoggedOut() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.out
}

type directedKey struct{}

// Directing makes the commands sent with the returned context ones for another
// online character of the account (D56, directed play): the server plays them
// as that character's and answers the directing session.
func Directing(ctx context.Context, character string) context.Context {
	return context.WithValue(ctx, directedKey{}, character)
}

// intent sends a new intent through the toolkit. The toolkit numbers it and
// calls the body with the id and sequence number set, just before the write;
// the body is where the model learns of it, so a fast result finds it there.
func (s *Session) intent(ctx context.Context, kind string, fill func(*pb.Intent)) (string, error) {
	c := s.client()
	if c == nil {
		return "", errors.New("not connected")
	}
	who, _ := ctx.Value(directedKey{}).(string)
	body := func(i *pb.Intent) {
		fill(i)
		i.Character = who
		s.mu.Lock()
		if _, ok := s.m.Intent(i.IntentId); !ok { // a resend names it again
			s.m.IntentSent(i.IntentId, i.Seq, kind)
		}
		s.mu.Unlock()
	}
	sent, err := c.Send(ctx, "tui", body)
	if sent.ID != "" {
		s.mu.Lock()
		s.sent[sent.ID] = sent
		s.mu.Unlock()
	}
	return sent.ID, err
}

// Writing reports whether an intent is being written: the model knows of it, as
// a pending one, from just before the write, and Send has not yet returned. A
// scenario that moves a clock on the server's side waits for this to end, so the
// intent is on the wire ahead of its next ping.
func (s *Session) Writing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range s.m.PendingIntents() {
		if _, ok := s.sent[id]; !ok {
			return true
		}
	}
	return false
}

// Result waits for the intent_result of the intent with id and returns its status.
func (s *Session) Result(ctx context.Context, id string) (model.IntentStatus, error) {
	var st model.IntentStatus
	err := s.wait(ctx, "the result of "+id, func() bool {
		i, ok := s.m.Intent(id)
		return ok && i.State != model.IntentPending
	}, func() error {
		st, _ = s.m.Intent(id)
		return nil
	})
	return st, err
}

// Ping asks the server for a pong, so a quiet connection is seen to be alive.
func (s *Session) Ping(ctx context.Context) error {
	c := s.client()
	if c == nil {
		return errors.New("not connected")
	}
	s.mu.Lock()
	s.nping++
	n := 100 + s.nping // 1 is the toolkit's own, on a reconnect
	s.mu.Unlock()
	return c.Ping(ctx, n)
}

// OpenInstance asks for an instance of the zone from one of the presets its
// map lists (D11). An accepted intent means the request was taken: the
// instance_opened or instance_open_failed event follows.
func (s *Session) OpenInstance(ctx context.Context, preset string) (string, error) {
	return s.intent(ctx, "instance_open", agentkit.InstanceOpen(preset))
}

// SubmitPlan sends the plan for the next round of a turn-based instance (D30).
// The intent's result is the plan's durable receipt (D37). A human session
// never marks a plan as a stand-in.
func (s *Session) SubmitPlan(ctx context.Context, steps ...agentkit.Step) (string, error) {
	return s.intent(ctx, "plan", agentkit.TurnSubmit(steps...))
}

// UseAbility uses the ability in a slot (1 to 8) on the target; "" is the
// current target, or the character itself for a self ability. The server
// decides whether it can be used now; the answer is the intent's result.
func (s *Session) UseAbility(ctx context.Context, slot int, target string) (string, error) {
	return s.intent(ctx, "use_ability", agentkit.UseAbility(slot, target))
}

// SetSlot puts a known ability in a slot (1 to 8); "" empties the slot. The
// server allows it out of combat only.
func (s *Session) SetSlot(ctx context.Context, slot int, ability string) (string, error) {
	return s.intent(ctx, "set_slot", agentkit.SetSlot(slot, ability))
}

// Equip wears an item from the inventory (as the inventory lists it); gear
// names the slot, "" lets the server choose (a ring goes to the first free
// one). Out of combat only.
func (s *Session) Equip(ctx context.Context, item, gear string) (string, error) {
	return s.intent(ctx, "equip", agentkit.Equip(item, gear))
}

// Unequip takes off what is worn in a gear slot.
func (s *Session) Unequip(ctx context.Context, gear string) (string, error) {
	return s.intent(ctx, "unequip", agentkit.Unequip(gear))
}

// Buy buys count of an item from a vendor within reach.
func (s *Session) Buy(ctx context.Context, npc, item string, count int) (string, error) {
	return s.intent(ctx, "buy", agentkit.Buy(npc, item, count))
}

// Sell sells one copy of an item from the inventory to a vendor.
func (s *Session) Sell(ctx context.Context, npc, item string) (string, error) {
	return s.intent(ctx, "sell", agentkit.Sell(npc, item))
}

// SellN sells count copies of an item (a stack, D78) to a vendor.
func (s *Session) SellN(ctx context.Context, npc, item string, count int) (string, error) {
	return s.intent(ctx, "sell", agentkit.SellN(npc, item, count))
}

// SellJunk sells every junk item at once to a vendor (D78): what has no gear
// slot and no use in a held contract.
func (s *Session) SellJunk(ctx context.Context, npc string) (string, error) {
	return s.intent(ctx, "sell_junk", func(i *pb.Intent) { i.Body = &pb.Intent_SellJunk{SellJunk: &pb.SellJunk{Npc: npc}} })
}

// LootAll takes every drop of the character's within reach (D78): the loot
// intent with no drop named.
func (s *Session) LootAll(ctx context.Context) (string, error) {
	return s.Loot(ctx, "")
}

// Train buys an ability from a trainer of the character's class.
func (s *Session) Train(ctx context.Context, npc, ability string) (string, error) {
	return s.intent(ctx, "train", agentkit.Train(npc, ability))
}

// Bind sets the bind point at a binder.
func (s *Session) Bind(ctx context.Context, npc string) (string, error) {
	return s.intent(ctx, "bind", agentkit.Bind(npc))
}

// BankDeposit puts an item from the inventory in the bank.
func (s *Session) BankDeposit(ctx context.Context, npc, item string) (string, error) {
	return s.intent(ctx, "bank_deposit", agentkit.BankDeposit(npc, item))
}

// BankWithdraw takes an item out of the bank.
func (s *Session) BankWithdraw(ctx context.Context, npc, item string) (string, error) {
	return s.intent(ctx, "bank_withdraw", agentkit.BankWithdraw(npc, item))
}

// TradeOpen asks another character for a trade, or agrees to its ask.
func (s *Session) TradeOpen(ctx context.Context, with string) (string, error) {
	return s.intent(ctx, "trade_open", agentkit.TradeOpen(with))
}

// TradeAddItem puts an item from the inventory in the trade offer.
func (s *Session) TradeAddItem(ctx context.Context, item string) (string, error) {
	return s.intent(ctx, "trade_add", agentkit.TradeAddItem(item))
}

// TradeSetCrowns sets the total of crowns in the trade offer.
func (s *Session) TradeSetCrowns(ctx context.Context, crowns int) (string, error) {
	return s.intent(ctx, "trade_add", agentkit.TradeSetCrowns(crowns))
}

// TradeRemoveItem takes an item back out of the offer.
func (s *Session) TradeRemoveItem(ctx context.Context, item string) (string, error) {
	return s.intent(ctx, "trade_remove", agentkit.TradeRemoveItem(item))
}

// TradeConfirm confirms the trade as it stands; any change clears both
// confirmations.
func (s *Session) TradeConfirm(ctx context.Context) (string, error) {
	return s.intent(ctx, "trade_confirm", agentkit.TradeConfirm())
}

// TradeCancel closes the trade window.
func (s *Session) TradeCancel(ctx context.Context) (string, error) {
	return s.intent(ctx, "trade_cancel", agentkit.TradeCancel())
}

// Talk speaks to an NPC within reach; the answer is a dialogue event.
func (s *Session) Talk(ctx context.Context, npc string) (string, error) {
	return s.intent(ctx, "talk", agentkit.Talk(npc))
}

// ContractAccept takes a contract the board offers now.
func (s *Session) ContractAccept(ctx context.Context, npc, contract string) (string, error) {
	return s.intent(ctx, "contract_accept", agentkit.ContractAccept(npc, contract))
}

// ContractAbandon gives a held contract up, anywhere.
func (s *Session) ContractAbandon(ctx context.Context, contract string) (string, error) {
	return s.intent(ctx, "contract_abandon", agentkit.ContractAbandon(contract))
}

// ContractTurnIn hands a finished contract in at its board.
func (s *Session) ContractTurnIn(ctx context.Context, npc, contract string) (string, error) {
	return s.intent(ctx, "contract_turn_in", agentkit.ContractTurnIn(npc, contract))
}

// The house companion (SEQ-049, doc 05 section 5). The server judges each
// intent: the quest, the role, a free party slot.

// CompanionSummon calls the house companion to your side.
func (s *Session) CompanionSummon(ctx context.Context) (string, error) {
	return s.intent(ctx, "companion_summon", agentkit.CompanionSummon())
}

// CompanionDismiss sends the house companion away and keeps it away.
func (s *Session) CompanionDismiss(ctx context.Context) (string, error) {
	return s.intent(ctx, "companion_dismiss", agentkit.CompanionDismiss())
}

// CompanionStance sets what the companion does by itself: one of
// agentkit.CompanionStances.
func (s *Session) CompanionStance(ctx context.Context, stance string) (string, error) {
	return s.intent(ctx, "companion_stance", agentkit.CompanionStance(stance))
}

// CompanionRole chooses the companion's role at the quest giver npc.
func (s *Session) CompanionRole(ctx context.Context, npc, role string) (string, error) {
	return s.intent(ctx, "companion_role", agentkit.CompanionRole(npc, role))
}

// CompanionOrder gives the companion a direct command that overrides its stance:
// "attack" (target), "move" (x, y), "stop" or "clear".
func (s *Session) CompanionOrder(ctx context.Context, order, target string, x, y int64) (string, error) {
	var dest *pb.Pos
	if order == "move" {
		dest = &pb.Pos{X: x, Y: y}
	}
	return s.intent(ctx, "companion_order", agentkit.CompanionOrder(order, target, dest, 0))
}

// Send sends an intent and returns its id, without waiting for its result: Result
// does that.
func (s *Session) Send(ctx context.Context, kind string, in agentkit.Intent) (string, error) {
	return s.intent(ctx, kind, in)
}

// Replica runs f on the toolkit's own view of the character, the one a stance
// reads; f is not called before a login.
func (s *Session) Replica(f func(*agentkit.Replica)) {
	s.mu.Lock()
	a := s.agent
	s.mu.Unlock()
	if a != nil {
		a.Client().View(f)
	}
}

// QuestDismiss gives a quest chain up for good (D62): quest is the chain's id,
// empty for the one the character is on.
func (s *Session) QuestDismiss(ctx context.Context, quest string) (string, error) {
	return s.intent(ctx, "quest_dismiss", agentkit.QuestDismiss(quest))
}

// Picked waits a moment for the target the server picked for an untargeted
// attack or ability (D66), and returns its name: "" if none came.
func (s *Session) Picked(ctx context.Context) string {
	var name string
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	_ = s.wait(ctx, "a target", func() bool { return s.m.Self.Target != "" }, func() error {
		name = s.m.Name(s.m.Self.Target)
		return nil
	})
	return name
}
