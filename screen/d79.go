package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The group rules (D79, D81) on the screen: the Instance panel says how many
// players set the tier and what the instance stirred, the Party panel lists the
// slots, the status shows beside names, and the help has the published rules.
// All of it is what the server said; nothing here decides.

// playersWords is "1 player" or "3 players".
func playersWords(n int64) string {
	if n == 1 {
		return "1 player"
	}
	return fmt.Sprintf("%d players", n)
}

// groupLines are the Instance panel's rows about the group: the tier and the
// number of players that set it, then what the instance stirred, then the
// server's last refusal. They are wrapped to w cells.
func groupLines(m *model.Model, w int) []string {
	var rows []string
	players, tier := m.Players(), m.GroupTierName()
	preset := ""
	switch {
	case m.Instance != nil:
		preset = m.Instance.GetPreset()
	case m.Opened != nil:
		preset = m.Opened.Preset
	}
	var rule *pb.GroupTierRule
	if p := m.Preset(preset); p != nil {
		rule = p.GetGroupTier()
	}
	switch {
	case tier != "":
		if rule != nil && rule.GetId() == tier {
			tier = rule.GetName() // the wire names the tier by its id
		}
		if line := fmt.Sprintf("%s | %s", tier, playersWords(players)); width(line) <= w {
			rows = append(rows, line)
		} else {
			rows = append(rows, tier, playersWords(players))
		}
	case players > 0 && rule != nil && m.SlotCount() != 1:
		rows = append(rows, "no group tier | "+playersWords(players),
			fmt.Sprintf("  %s needs %d", rule.GetName(), rule.GetMinPlayers()))
	case players > 1:
		rows = append(rows, playersWords(players))
	}
	for _, s := range m.Stirred() {
		rows = append(rows, wrapIndent("stirred: "+clean(s), w, "  ")...)
	}
	if r := m.Refusal; r != nil {
		rows = append(rows, wrapIndent("refused: "+clean(r.Text), w, "  ")...)
	}
	return rows
}

// slotRows are the rows of the Party panel that list the instance's slots, one
// each, then when a replacement may join. They are the server's holder list
// (D82): the player's handle, "Player 2", is the same for everyone and tells
// nothing of the account, so a holder is told as yours only by the characters
// this client plays.
func slotRows(m *model.Model, iw int, st Styler) []string {
	slots := m.Slots()
	if len(slots) < 2 { // a solo instance has one slot and nothing to list
		return nil
	}
	rows := []string{fit(fmt.Sprintf(" Slots: %d, %s", len(slots), playersWords(m.Players())), iw)}
	for _, s := range slots {
		player := s.Player
		if player == "" {
			player = "a player"
		}
		var row string
		switch {
		case s.State == model.HolderReserved:
			row = st.Dim(fit(fmt.Sprintf("  %d  reserved  %s, not yet in", s.N, player), iw))
		case s.State == model.HolderEmpty:
			row = st.Dim(fit(fmt.Sprintf("  %d  empty  was %s's, open", s.N, player), iw))
		case s.Me:
			row = fit(fmt.Sprintf("  %d  %s (you)  %s, you", s.N, s.Holder, player), iw)
		case s.Own:
			row = fit(fmt.Sprintf("  %d  %s  %s, yours", s.N, s.Holder, player), iw)
		default:
			row = fit(fmt.Sprintf("  %d  %s  %s", s.N, s.Holder, player), iw)
		}
		rows = append(rows, row)
	}
	words := m.ReplacementWords()
	if m.EncounterOpen() {
		return append(rows, st.Bold(fit(" "+words, iw)))
	}
	return append(rows, st.Dim(fit(" "+words, iw)))
}

// withStatus adds the You panel's status row when the character has set one.
func withStatus(m *model.Model, rows []string, w int, st Styler) []string {
	if m.Status == "" {
		return rows
	}
	return append(rows, fit(" Status: "+model.StatusWords(m.Status), w))
}

// statusTag is a name's status as it follows it in a list ("" for none).
func statusTag(status string) string {
	if status == "" {
		return ""
	}
	return " [" + model.StatusWords(status) + "]"
}

// groupRulesPage is the help page of the party-size rules (D79, D81), drawn
// from what the zone map's instance menu says: the factors for each party size,
// the group tier's threshold and the personal reward. Nothing in it is the
// client's own.
func groupRulesPage(m *model.Model, w int) []string {
	out := []string{"  Two things count in an instance, and they are different:"}
	for _, p := range []string{
		"The size of the party, in characters, sets the difficulty: the mobs change by the factors below for a party of that size (the largest size at or below yours). They are fixed when the instance opens and never ease if somebody leaves.",
		"The number of different players, counted by account, sets the group tier. Two characters of one account are one player: a party of two, not a group.",
		"Each player earns one personal reward for each encounter at the group tier, usable on either of its characters. One entitlement per account per encounter; it is never rolled twice for a second character.",
		"Slots are fixed when the instance opens, one for each character that enters. A replacement joins between encounters only and fills a slot whose holder is gone; it never adds a slot. One expedition at a time for each account.",
		"/status away, /status autopilot and /status clear set a word beside your name. It has no reward effect: nothing in the game reads it.",
	} {
		out = append(out, wrapIndent("  "+p, w, "  ")...)
	}
	if len(m.Presets) == 0 {
		return append(out, "", "  This zone offers no instances, or its map has not arrived: the rules of each are listed here when it does.")
	}
	for _, p := range m.Presets {
		lines := agentkit.PartyRules(p)
		out = append(out, "", fmt.Sprintf("%s, %s tier, party up to %d", p.GetName(), presetTier(p), p.GetPartyCap()))
		if len(lines) == 0 {
			out = append(out, "  every party faces it as it stands; no group tier")
			continue
		}
		for _, l := range lines {
			out = append(out, wrapIndent("  "+strings.ToUpper(l[:1])+l[1:], w, "    ")...)
		}
	}
	return out
}

func presetTier(p *pb.InstancePreset) string {
	if t := p.GetTier(); t != "" {
		return strings.ToUpper(t[:1]) + t[1:]
	}
	return "no"
}
