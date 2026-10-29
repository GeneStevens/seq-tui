package client

import (
	"context"
	"errors"

	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Playing together (M9, D58): chat, parties, who, friends and mutes. None of it
// is an intent: a request goes out as a social message and is answered by one
// result, which carries the refusal when there is one. What the server tells
// (chat, the party, notices, the lists) reaches the model through the same
// hook as everything else.

// Social sends the request and waits for its result. A refusal is the result's
// Reason, not an error.
func (s *Session) Social(ctx context.Context, req agentkit.SocialRequest) (*pb.SocialResult, error) {
	c := s.client()
	if c == nil {
		return nil, errors.New("not connected")
	}
	return c.Social(ctx, req)
}

// Say speaks to the characters near this one; PartyChat to the party; Tell to
// one online character by name.
func (s *Session) Say(ctx context.Context, text string) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.ChatSay(text))
}

func (s *Session) PartyChat(ctx context.Context, text string) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.ChatParty(text))
}

func (s *Session) Tell(ctx context.Context, to, text string) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.ChatTell(to, text))
}

// Party runs a party command: invite, accept, decline, kick, leave or list.
func (s *Session) Party(ctx context.Context, action, name string) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.PartyDo(action, name))
}

// Mute runs a mute command: mute, unmute or list (the names are in the result).
func (s *Session) Mute(ctx context.Context, action, name string) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.MuteDo(action, name))
}

// Anonymous hides or shows this character's zone in who and friends lists.
func (s *Session) Anonymous(ctx context.Context, on bool) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.Anonymous(on))
}

// Status sets the character's optional status (D79): "away", "autopilot" or ""
// to clear it. It has no effect on play.
func (s *Session) Status(ctx context.Context, status string) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.Status(status))
}

// Who lists the characters online; every filter is optional.
func (s *Session) Who(ctx context.Context, zone, class string, minLevel, maxLevel uint32) (*pb.WhoList, *pb.SocialResult, error) {
	res, err := s.Social(ctx, agentkit.WhoIs(zone, class, minLevel, maxLevel))
	if err != nil || !res.GetOk() {
		return nil, res, err
	}
	return s.client().WhoList(res.GetRequestId()), res, nil
}

// Friends runs a friends command: add, remove or list. A list's entries come
// back with the result.
func (s *Session) Friends(ctx context.Context, action, name string) (*pb.FriendList, *pb.SocialResult, error) {
	res, err := s.Social(ctx, agentkit.FriendDo(action, name))
	if err != nil || !res.GetOk() {
		return nil, res, err
	}
	return s.client().FriendList(res.GetRequestId()), res, nil
}

// OOC speaks on the server-wide out-of-character channel (D101).
func (s *Session) OOC(ctx context.Context, text string) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.ChatOOC(text))
}

// SetOOC switches hearing and speaking on the ooc channel on or off; the server
// keeps it with the character.
func (s *Session) SetOOC(ctx context.Context, on bool) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.SetOOC(on))
}

// Whois asks for one character's public card. The reply is nil for a name no
// character has (the result then says why).
func (s *Session) Whois(ctx context.Context, name string) (*pb.WhoisReply, *pb.SocialResult, error) {
	res, err := s.Social(ctx, agentkit.WhoisOf(name))
	if err != nil || !res.GetOk() {
		return nil, res, err
	}
	return s.client().WhoisReply(res.GetRequestId()), res, nil
}

// Report files a report of a character with the game masters; the server's words
// about it come as a notice.
func (s *Session) Report(ctx context.Context, name, reason string) (*pb.SocialResult, error) {
	return s.Social(ctx, agentkit.ReportOf(name, reason))
}

// Bug files a bug with the game masters; kind is the client filing it ("tui" or
// "browser"). The server's words, with the bug's id, come as a notice.
func (s *Session) Bug(ctx context.Context, text, kind string) (*pb.SocialResult, error) {
	return s.Social(ctx, func(m *pb.Social) { m.Body = &pb.Social_Bug{Bug: &pb.Bug{Text: text, Client: kind}} })
}
