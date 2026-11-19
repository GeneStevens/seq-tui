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
// Each takes the item links of the line (D206): an id and the span of text it
// stands for, never markup.
func (s *Session) Say(ctx context.Context, text string, links ...*pb.ItemLink) (*pb.SocialResult, error) {
	return s.Social(ctx, linked(agentkit.ChatSay(text), links))
}

func (s *Session) PartyChat(ctx context.Context, text string, links ...*pb.ItemLink) (*pb.SocialResult, error) {
	return s.Social(ctx, linked(agentkit.ChatParty(text), links))
}

func (s *Session) Tell(ctx context.Context, to, text string, links ...*pb.ItemLink) (*pb.SocialResult, error) {
	return s.Social(ctx, linked(agentkit.ChatTell(to, text), links))
}

// linked adds the links to a chat request; a line with none is sent as it was.
func linked(req agentkit.SocialRequest, links []*pb.ItemLink) agentkit.SocialRequest {
	if len(links) == 0 {
		return req
	}
	return agentkit.WithLinks(req, links...)
}

// ItemDetails asks for a linked item's details by id (D206) and keeps the
// answer in the model; the refusal, in words, is "" when the item came.
func (s *Session) ItemDetails(ctx context.Context, item string) (string, error) {
	c := s.client()
	if c == nil {
		return "", errors.New("not connected")
	}
	d, err := c.ItemDetails(ctx, item)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	why := s.m.NoteItemDetails(d)
	s.wake()
	return why, nil
}

// News asks for one entry of the release log (D211), the latest or, with more,
// the next older one than the log last showed, and puts it in the log. The words
// are "" when it came, else why nothing was shown.
func (s *Session) News(ctx context.Context, more bool) (string, error) {
	c := s.client()
	if c == nil {
		return "", errors.New("not connected")
	}
	s.mu.Lock()
	offset := s.m.NewsOffset(more)
	s.mu.Unlock()
	r, err := c.ReleaseLog(ctx, 1, offset)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	why := s.m.NoteNews(offset, r)
	s.wake()
	return why, nil
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
func (s *Session) OOC(ctx context.Context, text string, links ...*pb.ItemLink) (*pb.SocialResult, error) {
	return s.Social(ctx, linked(agentkit.ChatOOC(text), links))
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

// Feedback sends feedback to the game masters, as Bug does a bug; the server's
// words, with the short id, come as a notice.
func (s *Session) Feedback(ctx context.Context, text, kind string) (*pb.SocialResult, error) {
	return s.Social(ctx, func(m *pb.Social) { m.Body = &pb.Social_Feedback{Feedback: &pb.Feedback{Text: text, Client: kind}} })
}
