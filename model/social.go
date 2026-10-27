package model

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"google.golang.org/protobuf/proto"
)

// Playing together (M9, D58): chat, the party, who is online and friends. None of
// it is game state; the model keeps what the server said, and the screen shows it.

// ChatCap bounds the chat scrollback.
const ChatCap = 500

// ChatLine is a line of chat as the server told it. Mine says the character
// spoke it (a speaker sees its own lines).
type ChatLine struct {
	Channel  string // "say", "party" or "tell"
	From, To string
	Text     string
	AtMs     int64
	Mine     bool
	// Seq numbers the lines the session has seen, from 1: it keeps counting past
	// the scrollback's cap, so a mark "read up to Seq n" outlives the lines cut.
	Seq int
	// Mention says the line, said by someone else, has the character's own name in
	// it (a whole word, in any case).
	Mention bool
	// Speaker and Parts are the line's structure (D101 §8), empty from a server
	// that does not send it: who spoke, and the text again as ordered spans.
	SpeakerID, SpeakerKind string
	Parts                  []ChatPart
}

// The kinds a character reference has (D101 §8), and the kind of a span that is
// not one.
const (
	KindPlayer = "player"
	KindNPC    = "npc"
	KindGM     = "gm"
	KindServer = "server"
	KindItem   = "item" // an item link: later, the server does not send it yet
)

// ChatPart is one span of a line of chat: plain text (Kind ""), a character
// reference (Kind player, npc, gm or server) or an item (Kind item). Text is
// what is written, the name for a reference.
type ChatPart struct {
	Text, ID, Kind string
}

// Structured says the server sent the line's parts.
func (l ChatLine) Structured() bool { return len(l.Parts) > 0 }

// IsRef says the part is a character.
func (p ChatPart) IsRef() bool {
	return p.Kind == KindPlayer || p.Kind == KindNPC || p.Kind == KindGM || p.Kind == KindServer
}

// partsOf reads the parts of a line, leaving out a span it does not know.
func partsOf(ps []*pb.ChatPart) []ChatPart {
	var out []ChatPart
	for _, p := range ps {
		switch b := p.GetBody().(type) {
		case *pb.ChatPart_Text:
			if b.Text != "" {
				out = append(out, ChatPart{Text: b.Text})
			}
		case *pb.ChatPart_Character:
			out = append(out, ChatPart{Text: b.Character.GetName(), ID: b.Character.GetId(), Kind: firstText(b.Character.GetKind(), KindPlayer)})
		case *pb.ChatPart_Item:
			out = append(out, ChatPart{Text: b.Item.GetName(), ID: b.Item.GetId(), Kind: KindItem})
		}
	}
	return out
}

// Invite is a party invitation the character has been sent and not answered.
type Invite struct {
	From string
	// N counts the invitations seen, so a screen can tell a new one.
	N int
}

// Social is what the model knows of playing together.
type Social struct {
	// Chat is the scrollback, oldest first, bounded by ChatCap.
	Chat []ChatLine
	// Party is the party as the server last said it: nil before it has, an empty
	// id for none.
	Party *pb.PartyState
	// Invite is the invitation waiting for an answer, nil when there is none.
	Invite *Invite
	// LastTell is the name of the last character that told the character
	// something: /reply answers it.
	LastTell string
	// Known is what the who and friends lists last said of each character, by
	// lower-case name: its class, level and zone, and whether it is online.
	Known map[string]Known
	// Friends is the character's friends list as last told, in the server's
	// order; HasFriends says one has been.
	Friends    []*pb.FriendEntry
	HasFriends bool
	// MutedText is what the server last told the character of its mute, "" when
	// it has not.
	MutedText string
	// OOCOff says the character switched the ooc channel off (D101 §7): the
	// server tells it at login and on every change (SEQ-156); until then it is on.
	OOCOff bool
	// ReportLimitText is the server's last words on a report refused for an account limit.
	ReportLimitText string
	// ReportAck counts the server's acknowledgements of a /report seen.
	ReportAck int
	invites   int
	// ChatN is how many lines of chat the session has seen (the Seq of the
	// newest). TellPeer is the other side of the latest tell, either way.
	ChatN    int
	TellPeer string

	// fighting and fightFrom note the chat that came during a fight, for the line
	// that catches the player up when it ends (D101 §5): the fight began when
	// ChatN was fightFrom.
	fighting  bool
	fightFrom int
}

// Known is what is known of another character from the who list.
type Known struct {
	Name  string
	Class string
	Level uint32
	Zone  string
}

// Key is the form a name is compared in: characters are named in any case.
func Key(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

func (m *Model) applySocial(msg *pb.ServerMessage) {
	s := &m.Social
	switch b := msg.GetBody().(type) {
	case *pb.ServerMessage_ChatSettings:
		// The whole of the settings, for the character the session plays (the name may
		// not be known yet when they come, straight after the welcome).
		if n := b.ChatSettings.GetCharacter(); m.Self.Name == "" || n == "" || Key(n) == Key(m.Self.Name) {
			s.OOCOff = !b.ChatSettings.GetOocOn()
		}
	case *pb.ServerMessage_Chat:
		c := b.Chat
		l := ChatLine{Channel: c.GetChannel(), From: c.GetFrom(), To: c.GetTo(), Text: c.GetText(), AtMs: c.GetAtMs(), Mine: Key(c.GetFrom()) == Key(m.Self.Name) && m.Self.Name != "",
			SpeakerID: c.GetSpeaker().GetId(), SpeakerKind: c.GetSpeaker().GetKind(), Parts: partsOf(c.GetParts())}
		s.ChatN++
		l.Seq = s.ChatN
		l.Mention = !l.Mine && m.mentionsMe(l)
		if l.Channel == "tell" {
			s.TellPeer = l.To
			if !l.Mine {
				s.LastTell, s.TellPeer = l.From, l.From
			}
		}
		s.Chat = append(s.Chat, l)
		if over := len(s.Chat) - ChatCap; over > 0 {
			s.Chat = slices.Clone(s.Chat[over:])
		}
	case *pb.ServerMessage_Party:
		s.Party = b.Party
		if b.Party.GetId() != "" {
			s.Invite = nil
		}
	case *pb.ServerMessage_PartyVitals:
		s.Party = withVitals(s.Party, b.PartyVitals)
	case *pb.ServerMessage_Notice:
		m.applyNotice(b.Notice)
	case *pb.ServerMessage_Who:
		for _, e := range b.Who.GetEntries() {
			if s.Known == nil {
				s.Known = map[string]Known{}
			}
			s.Known[Key(e.GetName())] = Known{Name: e.GetName(), Class: e.GetClassId(), Level: e.GetLevel(), Zone: e.GetZone()}
		}
	case *pb.ServerMessage_Friends:
		s.Friends, s.HasFriends = b.Friends.GetFriends(), true
	}
}

// mentionsMe says the line has the character in it: a reference to it in the
// parts the server sent, else its name as a whole word in the text.
func (m *Model) mentionsMe(l ChatLine) bool {
	if !l.Structured() {
		return Mentions(l.Text, m.Self.Name)
	}
	for _, p := range l.Parts {
		if p.IsRef() && ((p.ID != "" && p.ID == m.Self.ID) || (m.Self.Name != "" && Key(p.Text) == Key(m.Self.Name))) {
			return true
		}
	}
	return false
}

// withVitals is the party with one member's changed fields applied (D60): only
// the fields the message carries, the rest as they were. The party is copied, so
// a Party already handed out stays as it was; a member or party not known yet is
// ignored, the whole state follows on its first sighting.
func withVitals(p *pb.PartyState, v *pb.PartyVitals) *pb.PartyState {
	if p == nil {
		return p
	}
	next := proto.Clone(p).(*pb.PartyState)
	for _, mi := range next.GetMembers() {
		if Key(mi.GetName()) != Key(v.GetName()) {
			continue
		}
		if v.Hp != nil {
			mi.Hp = v.GetHp()
		}
		if v.MaxHp != nil {
			mi.MaxHp = v.GetMaxHp()
		}
		if v.Mana != nil {
			mi.Mana = v.GetMana()
		}
		if v.MaxMana != nil {
			mi.MaxMana = v.GetMaxMana()
		}
	}
	return next
}

func (m *Model) applyNotice(n *pb.SocialNotice) {
	s := &m.Social
	from := n.GetFrom()
	text := n.GetText()
	switch n.GetKind() {
	case "party_invite":
		s.invites++
		s.Invite = &Invite{From: from, N: s.invites}
		text = from + " invites you to a party"
	case "party_invite_declined":
		text = from + " declined your party invitation"
	case "party_joined":
		text = from + " joined the party"
	case "party_left":
		text = from + " left the party"
	case "party_kicked":
		if Key(from) == Key(m.Self.Name) {
			text = "you were removed from the party"
		} else {
			text = from + " was removed from the party"
		}
	case "party_leader":
		text = from + " leads the party now"
	case "party_disbanded":
		text = "the party is disbanded"
		s.Invite = nil
	case "friend_online":
		text = "friend " + from + " logged in"
	case "friend_offline":
		text = "friend " + from + " logged out"
		for i, f := range s.Friends {
			if Key(f.GetName()) == Key(from) {
				s.Friends = slices.Clone(s.Friends)
				s.Friends[i] = &pb.FriendEntry{Name: f.GetName()}
			}
		}
	case "muted":
		// A game master's mute (D100): the server's text, kept for the refusal of
		// the next line the character tries to say.
		text = firstText(text, "A game master has muted you. Nobody can hear what you say until the mute is lifted.")
		s.MutedText = text
	case "report_limit":
		// The server's own words for a refused report (D112); kept for the refusal of /report.
		text = firstText(text, "You've filed several reports recently. A GM will review them.")
		s.ReportLimitText = text
	case "report_filed":
		s.ReportAck++
		text = firstText(text, "your report was filed with the game masters")
	default:
		if text == "" {
			text = n.GetKind() + " " + from
		}
	}
	if n.GetKind() == "friend_online" {
		for i, f := range s.Friends {
			if Key(f.GetName()) == Key(from) {
				s.Friends = slices.Clone(s.Friends)
				s.Friends[i] = &pb.FriendEntry{Name: f.GetName(), Online: true}
			}
		}
	}
	m.Log = append(m.Log, LogEntry{Kind: "notice_" + n.GetKind(), Text: text})
	m.trimLog()
}

// ClearInvite drops the invitation: it was answered, or it ran out.
func (m *Model) ClearInvite() { m.Social.Invite = nil }

// InParty reports whether the character is in a party.
func (m *Model) InParty() bool { return m.Social.Party.GetId() != "" }

// PartyMembers lists the party's members in the order they joined, or nil.
func (m *Model) PartyMembers() []*pb.PartyMemberInfo { return m.Social.Party.GetMembers() }

// IsLeader says whether the character leads its party.
func (m *Model) IsLeader() bool {
	return m.InParty() && Key(m.Social.Party.GetLeader()) == Key(m.Self.Name)
}

// Mate is a party member as the Party panel shows it: what the server's party
// state says of it (zone, level, health, mana), or, for the character itself,
// what the character knows of itself.
type Mate struct {
	Name          string
	Leader        bool
	Online        bool
	Me            bool
	Zone          string
	Level         int64
	HP, MaxHP     int64
	Mana, MaxMana int64
	Dead          bool
	Status        string // "away", "autopilot" or "" (D79)
}

// Mates are the party's members for the Party panel, in the order they joined.
func (m *Model) Mates() []Mate {
	var out []Mate
	for _, pm := range m.PartyMembers() {
		mt := Mate{Name: pm.GetName(), Leader: pm.GetLeader(), Online: pm.GetOnline(), Status: pm.GetStatus()}
		if Key(pm.GetName()) == Key(m.Self.Name) {
			mt.Status = m.Status
		}
		if Key(pm.GetName()) == Key(m.Self.Name) {
			mt.Me, mt.Level = true, m.Self.Level
			mt.HP, mt.MaxHP, mt.Mana, mt.MaxMana, mt.Dead = m.Self.HP, m.Self.MaxHP, m.Self.Mana, m.Self.MaxMana, m.Self.Dead
		} else if mt.Online {
			mt.Zone, mt.Level = pm.GetZone(), int64(pm.GetLevel())
			mt.HP, mt.MaxHP, mt.Mana, mt.MaxMana = int64(pm.GetHp()), int64(pm.GetMaxHp()), int64(pm.GetMana()), int64(pm.GetMaxMana())
			mt.Dead = mt.MaxHP > 0 && mt.HP <= 0
		}
		out = append(out, mt)
	}
	return out
}

// String is a line of chat as one string, for a log and a plain-text screen.
func (l ChatLine) String() string {
	switch l.Channel {
	case "party":
		return fmt.Sprintf("[party] %s: %s", l.From, l.Text)
	case "ooc":
		return fmt.Sprintf("[ooc] %s: %s", l.From, l.Text)
	case "tell":
		if l.Mine {
			return fmt.Sprintf("[tell] to %s: %s", l.To, l.Text)
		}
		return fmt.Sprintf("[tell] %s: %s", l.From, l.Text)
	}
	return fmt.Sprintf("[say] %s: %s", l.From, l.Text)
}

// SocialReason is a refusal reason of a social request in words.
func SocialReason(r string) string {
	w := socialWords(r)
	if w == "" {
		return Words(r)
	}
	return fmt.Sprintf("%s (%s)", w, r)
}

func socialWords(r string) string {
	switch r {
	case "empty":
		return "there is nothing to say"
	case "too_long":
		return "that line is too long (200 characters at most)"
	case "rate_limited":
		return "you are talking too fast: wait a moment"
	case "bad_channel":
		return "that is not a channel"
	case "not_online":
		return "that character is not online"
	case "no_such_character":
		return "there is no character by that name"
	case "not_in_party":
		return "you are not in a party"
	case "already_in_party":
		return "already in a party"
	case "party_full":
		return "the party is full (6)"
	case "not_leader":
		return "only the party's leader can do that"
	case "no_invitation":
		return "you have no invitation"
	case "invitation_expired":
		return "the invitation has run out"
	case "already_invited":
		return "that character has been invited already"
	case "not_a_member":
		return "that character is not in your party"
	case "cannot_target_self":
		return "not yourself"
	case "too_many":
		return "too many on that list"
	case "unknown_action":
		return "the server does not know that command"
	case "muted":
		return "a game master has muted you"
	case "ooc_off":
		return "ooc is switched off: /ooc on turns it back on"
	case "unavailable":
		return "not available now"
	}
	return ""
}

// PartyIDs are the character ids of the party's online members, found from the
// names the model has seen entities carry (the character's own id for itself).
// A member whose id the model never saw is listed by name.
func (m *Model) PartyIDs() []string {
	var ids []string
	for _, pm := range m.PartyMembers() {
		if !pm.GetOnline() {
			continue
		}
		k := Key(pm.GetName())
		id := pm.GetName()
		if k == Key(m.Self.Name) {
			id = m.Self.ID
		} else {
			for eid, n := range m.names {
				if Key(n) == k {
					id = eid
				}
			}
		}
		ids = append(ids, id)
	}
	return ids
}

// KnownCharacters lists the characters the who list has told of, by name.
func (m *Model) KnownCharacters() []Known {
	out := make([]Known, 0, len(m.Social.Known))
	for _, k := range m.Social.Known {
		out = append(out, k)
	}
	slices.SortFunc(out, func(a, b Known) int { return strings.Compare(Key(a.Name), Key(b.Name)) })
	return out
}

// Mentions says text has name in it as a whole word, in any case: "Mira," and
// "hi mira!" do, "Miranda" does not. An empty name is mentioned nowhere.
func Mentions(text, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	t, n := []rune(strings.ToLower(text)), []rune(strings.ToLower(name))
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\'' }
	for i := 0; i+len(n) <= len(t); i++ {
		if string(t[i:i+len(n)]) != string(n) {
			continue
		}
		if (i == 0 || !word(t[i-1])) && (i+len(n) == len(t) || !word(t[i+len(n)])) {
			return true
		}
	}
	return false
}

// ChatSince counts the lines of chat others said after Seq n, in all, by party
// and by tell.
func (m *Model) ChatSince(n int) (all, party, tell int) {
	for _, l := range m.Social.Chat {
		if l.Seq <= n || l.Mine {
			continue
		}
		all++
		switch l.Channel {
		case "party":
			party++
		case "tell":
			tell++
		}
	}
	return all, party, tell
}

// watchFight notes a fight beginning and ending (D101 §5). When it ends with chat
// having come, one line in the log says how much, so the player who looked only at
// the fight is caught up. The model is locked by the caller.
func (m *Model) watchFight() {
	s := &m.Social
	switch under := m.UnderAttack(); {
	case under && !s.fighting:
		s.fighting, s.fightFrom = true, s.ChatN
	case !under && s.fighting:
		s.fighting = false
		if line := CatchUp(m.ChatSince(s.fightFrom)); line != "" {
			m.addLog(LogEntry{GameTimeMs: m.GameTimeMs, Kind: "chat_catchup", Text: line})
		}
	}
}

// CatchUp is the line that says how much chat came during a fight: "" when none.
func CatchUp(all, party, tell int) string {
	if all == 0 {
		return ""
	}
	var parts []string
	if party > 0 {
		parts = append(parts, fmt.Sprintf("%d party", party))
	}
	if tell > 0 {
		parts = append(parts, fmt.Sprintf("%d tells", tell))
	}
	line := fmt.Sprintf("While fighting: %d chat", all)
	if len(parts) > 0 {
		line += " (" + strings.Join(parts, ", ") + ")"
	}
	return line
}

// NameKind is what a name in the chat is, as far as the client can tell (D101 §8).
type NameKind int

const (
	NameOther  NameKind = iota // not known
	NameSelf                   // the character's own
	NamePlayer                 // a character in the party, in view, or on the who list
	NameNPC                    // a person of the world in view or met
)

// KindOfName says what name is: the character itself, a player the client has
// seen (in the party, in view, or on the who list), an NPC, or none of them.
func (m *Model) KindOfName(name string) NameKind {
	k := Key(name)
	switch {
	case k == "":
		return NameOther
	case k == Key(m.Self.Name) && m.Self.Name != "":
		return NameSelf
	}
	for _, pm := range m.PartyMembers() {
		if Key(pm.GetName()) == k {
			return NamePlayer
		}
	}
	for _, g := range m.Group {
		if g.Entity != nil && Key(g.Entity.GetName()) == k {
			return NamePlayer
		}
	}
	for _, e := range m.Entities {
		if e.GetKind() == pb.EntityKind_ENTITY_KIND_CHARACTER && Key(e.GetName()) == k {
			return NamePlayer
		}
	}
	if _, ok := m.Social.Known[k]; ok {
		return NamePlayer
	}
	for _, n := range m.NPCs {
		if Key(n.GetName()) == k {
			return NameNPC
		}
	}
	return NameOther
}

// PosOfName is where the character called name is, if the client can see it: the
// character itself, a group member or another character in view. nil if not.
func (m *Model) PosOfName(name string) *pb.Pos {
	k := Key(name)
	switch {
	case k == "":
		return nil
	case k == Key(m.Self.Name) && m.Self.Name != "":
		return m.Self.Pos
	}
	for _, g := range m.Group {
		if g.Entity != nil && Key(g.Entity.GetName()) == k {
			return g.Entity.GetPos()
		}
	}
	for _, e := range m.Entities {
		if e.GetKind() == pb.EntityKind_ENTITY_KIND_CHARACTER && Key(e.GetName()) == k {
			return e.GetPos()
		}
	}
	return nil
}
