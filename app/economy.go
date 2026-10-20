package app

import (
	"context"
	"fmt"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

// economy sends the intent for an action of the inventory, the traders or the
// trade window, and says in a word what it was for the status line.
func (a *App) economy(ctx context.Context, act Action) (what, id string, err error) {
	s := a.sess
	switch act.Kind {
	case Equip:
		what = "wear " + act.Note
		id, err = s.Equip(ctx, act.ID, "")
	case Unequip:
		what = "take off " + act.Note
		id, err = s.Unequip(ctx, act.Gear)
	case Buy:
		what = "buy " + act.Note
		id, err = s.Buy(ctx, act.NPC, act.ID, max(act.Count, 1))
	case Sell:
		what = "sell " + act.Note
		if act.Count > 1 {
			id, err = s.SellN(ctx, act.NPC, act.ID, act.Count)
		} else {
			id, err = s.Sell(ctx, act.NPC, act.ID)
		}
	case SellJunk:
		what = "sell all junk"
		id, err = s.SellJunk(ctx, act.NPC)
	case Train:
		what = "train " + act.Note
		id, err = s.Train(ctx, act.NPC, act.ID)
	case Bind:
		what = "bind"
		id, err = s.Bind(ctx, act.NPC)
	case BankDeposit:
		what = "bank " + act.Note
		id, err = s.BankDeposit(ctx, act.NPC, act.ID)
	case BankWithdraw:
		what = "take out " + act.Note
		id, err = s.BankWithdraw(ctx, act.NPC, act.ID)
	case TradeOpen:
		what = "ask " + act.Note + " to trade"
		id, err = s.TradeOpen(ctx, act.ID)
	case TradeAdd:
		what = "offer " + act.Note
		id, err = s.TradeAddItem(ctx, act.ID)
	case TradeCrowns:
		what = fmt.Sprintf("offer %d crowns", act.Amount)
		id, err = s.TradeSetCrowns(ctx, int(act.Amount))
	case TradeRemove:
		what = "take back " + act.Note
		id, err = s.TradeRemoveItem(ctx, act.ID)
	case TradeConfirm:
		what = "confirm the trade"
		id, err = s.TradeConfirm(ctx)
	case TradeCancel:
		what = "cancel the trade"
		id, err = s.TradeCancel(ctx)
	}
	return what, id, err
}

func up(k string) bool   { return k == "up" || k == "w" || k == "k" }
func down(k string) bool { return k == "down" || k == "s" || k == "j" }
func enter(k string) bool {
	return k == "enter" || k == "space" || k == " "
}

// step moves a cursor by the key, within n rows.
func step(cur int, k string, n int) int {
	switch {
	case up(k):
		cur--
	case down(k):
		cur++
	}
	return max(min(cur, n-1), 0)
}

// charKey handles a key on the character view: Tab (or 1 to 3) turns to a tab, the
// arrows move the cursor, and Enter takes off the gear under it or wears the item.
func (a *App) charKey(k string, m *model.Model) Action {
	sb := &a.mouse.charTab
	switch {
	case k == "esc" || k == "c" || k == "q":
		a.char = false
	case k == "ctrl+c":
		return Action{Kind: Quit}
	case k == "n":
		return Action{Kind: OpenTraders}
	case k == "t":
		return Action{Kind: OpenTrade}
	case k == "tab" || k == "right":
		return Action{Kind: SelectSide, Count: (sb.Tab + 1) % screen.Tabs}
	case k == "left" || k == "shift+tab":
		return Action{Kind: SelectSide, Count: (sb.Tab + screen.Tabs - 1) % screen.Tabs}
	case k == "1" || k == "2" || k == "3":
		return Action{Kind: SelectSide, Count: int(k[0] - '1')}
	case k == "pgdown":
		return Action{Kind: ScrollSide, Count: 4}
	case k == "pgup":
		return Action{Kind: ScrollSide, Count: -4}
	case up(k) || down(k):
		if sb.Tab == screen.TabStats {
			return Action{Kind: ScrollSide, Count: map[bool]int{true: -1, false: 1}[up(k)]}
		}
		sb.Cur[sb.Tab] = step(sb.Cur[sb.Tab], k, screen.CharRows(m, sb.Tab))
	case enter(k):
		inst, gear, slot := screen.CharSelected(m, sb)
		switch {
		case sb.Tab == screen.TabStats:
		case gear && inst == "":
			return Action{Note: "nothing is worn there"}
		case gear:
			return Action{Kind: Unequip, Gear: model.GearSlots[slot], Note: m.ItemName(inst)}
		case inst == "":
			return Action{Note: "you carry nothing to wear"}
		default:
			return Action{Kind: Equip, ID: inst, Note: m.ItemName(inst)}
		}
	}
	return Action{}
}

// tradersKey handles a key on the traders' screen: the list, then a window.
func (a *App) tradersKey(k string, m *model.Model) Action {
	t := a.traders
	list := screen.TradersInView(m)
	t.Sel = max(min(t.Sel, len(list)-1), 0)
	switch {
	case k == "ctrl+c":
		return Action{Kind: Quit}
	case k == "esc" && t.Open:
		t.Open = false
		return Action{}
	case k == "esc" || k == "q":
		a.traders = nil
		return Action{}
	}
	if !t.Open {
		switch {
		case up(k) || down(k):
			t.Sel = step(t.Sel, k, len(list))
		case enter(k):
			if len(list) == 0 {
				return Action{Note: "nobody here trades, teaches or binds"}
			}
			if list[t.Sel].GetRole() == "contract_board" {
				a.traders = nil
				return Action{Kind: OpenBoard, NPC: list[t.Sel].GetId()}
			}
			*t = screen.Traders{Sel: t.Sel, Open: true, NPC: list[t.Sel].GetId()}
		}
		return Action{}
	}
	n := m.NPC(t.NPC)
	if n == nil {
		t.Open = false
		return Action{}
	}
	switch n.GetRole() {
	case "vendor":
		stacks := m.Stacks(m.Self.Inventory)
		switch k {
		case "J":
			// One key sells all junk (D78): what has no gear slot and no use in a held contract.
			return Action{Kind: SellJunk, NPC: n.GetId()}
		case "a":
			// A sells the whole stack under the cursor.
			t.Cur[1] = max(min(t.Cur[1], len(stacks)-1), 0)
			if t.Pane != 1 || len(stacks) == 0 {
				return Action{Note: "A sells the stack under the cursor in the list you carry (Tab switches)"}
			}
			sk := stacks[t.Cur[1]]
			return Action{Kind: Sell, NPC: n.GetId(), ID: sk.Inst, Count: sk.Count, Note: m.ItemName(sk.Inst)}
		}
		return a.listKey(k, t, true, len(n.GetStock()), len(stacks), func(pane, i int) Action {
			if pane == 0 {
				s := n.GetStock()[i]
				return Action{Kind: Buy, NPC: n.GetId(), ID: s.GetItem(), Count: 1, Note: m.ItemName(s.GetItem())}
			}
			inst := stacks[i].Inst
			return Action{Kind: Sell, NPC: n.GetId(), ID: inst, Note: m.ItemName(inst)}
		})
	case "banker":
		held, banked := m.Stacks(m.Self.Inventory), m.Stacks(m.Self.Bank)
		return a.listKey(k, t, true, len(held), len(banked), func(pane, i int) Action {
			if pane == 0 {
				inst := held[i].Inst
				return Action{Kind: BankDeposit, NPC: n.GetId(), ID: inst, Note: m.ItemName(inst)}
			}
			inst := banked[i].Inst
			return Action{Kind: BankWithdraw, NPC: n.GetId(), ID: inst, Note: m.ItemName(inst)}
		})
	case "trainer":
		t.Pane = 0
		return a.listKey(k, t, false, len(n.GetTrains()), 0, func(_, i int) Action {
			ab := n.GetTrains()[i].GetAbility()
			return Action{Kind: Train, NPC: n.GetId(), ID: ab, Note: m.AbilityName(ab)}
		})
	case "binder":
		if enter(k) {
			return Action{Kind: Bind, NPC: n.GetId()}
		}
	}
	return Action{}
}

// listKey moves in the window's one or two lists (of n0 and n1 rows; Tab
// switches between two) and acts
// on the row under the cursor with Enter.
func (a *App) listKey(k string, t *screen.Traders, two bool, n0, n1 int, act func(pane, i int) Action) Action {
	ns := [2]int{n0, n1}
	t.Cur[0], t.Cur[1] = max(min(t.Cur[0], n0-1), 0), max(min(t.Cur[1], n1-1), 0)
	switch {
	case k == "tab" && two:
		t.Pane = 1 - t.Pane
	case up(k) || down(k):
		t.Cur[t.Pane] = step(t.Cur[t.Pane], k, ns[t.Pane])
	case enter(k):
		if ns[t.Pane] == 0 {
			return Action{Note: "there is nothing there to choose"}
		}
		return act(t.Pane, t.Cur[t.Pane])
	}
	return Action{}
}

// askerIndex is where the trade screen's cursor starts: on the character that
// asked for a trade, if one did.
func askerIndex(m *model.Model) int {
	if m.Trade == nil {
		return 0
	}
	for i, e := range screen.TraderPeers(m) {
		if e.GetId() == m.Trade.GetWith() {
			return i
		}
	}
	return 0
}

// tradeKey handles a key on the trade screen.
func (a *App) tradeKey(k string, m *model.Model) Action {
	t := a.trade
	tr := m.Trade
	peers := screen.TraderPeers(m)
	if k == "ctrl+c" {
		return Action{Kind: Quit}
	}
	if tr == nil || !tr.GetOpen() {
		t.Who = max(min(t.Who, len(peers)-1), 0)
		switch {
		case k == "esc" || k == "q":
			a.trade = nil
		case up(k) || down(k):
			t.Who = step(t.Who, k, len(peers))
		case enter(k):
			if len(peers) == 0 {
				return Action{Note: "nobody else is in view"}
			}
			return Action{Kind: TradeOpen, ID: peers[t.Who].GetId(), Note: peers[t.Who].GetName()}
		}
		return Action{}
	}
	carried := m.Stacks(m.Self.Inventory)
	n, mine := len(carried), tr.GetMine().GetItems()
	t.Item, t.Mine = max(min(t.Item, n-1), 0), max(min(t.Mine, len(mine)-1), 0)
	if t.Typing {
		switch {
		case k == "esc":
			t.Typing = false
		case k == "backspace":
			if r := []rune(t.Crowns); len(r) > 0 {
				t.Crowns = string(r[:len(r)-1])
			}
		case len(k) == 1 && k[0] >= '0' && k[0] <= '9' && len(t.Crowns) < 9:
			t.Crowns += k
		case k == "enter":
			t.Typing = false
			var amount int64
			_, _ = fmt.Sscan(t.Crowns, &amount)
			return Action{Kind: TradeCrowns, Amount: amount}
		}
		return Action{}
	}
	switch {
	case k == "esc" || k == "q":
		a.trade = nil
	case k == "tab" || k == "left" || k == "right":
		t.Offer = !t.Offer
	case up(k) || down(k):
		if t.Offer {
			t.Mine = step(t.Mine, k, len(mine))
		} else {
			t.Item = step(t.Item, k, n)
		}
	case k == "c":
		t.Typing, t.Crowns = true, ""
	case k == "y":
		return Action{Kind: TradeConfirm}
	case k == "x":
		return Action{Kind: TradeCancel}
	case enter(k) && t.Offer:
		if len(mine) == 0 {
			return Action{Note: "the offer holds no items (C sets crowns)"}
		}
		return Action{Kind: TradeRemove, ID: mine[t.Mine], Note: m.ItemName(mine[t.Mine])}
	case enter(k):
		if n == 0 {
			return Action{Note: "you carry nothing to offer"}
		}
		inst := carried[t.Item].Inst
		return Action{Kind: TradeAdd, ID: inst, Note: m.ItemName(inst)}
	}
	return Action{}
}
