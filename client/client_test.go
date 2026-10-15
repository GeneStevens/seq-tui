package client_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// fake is an in-memory server end of a Transport.
type fake struct {
	in   chan *pb.ClientMessage // what the client sent
	out  chan *pb.ServerMessage // what the server says
	done chan struct{}
}

func newFake() *fake {
	return &fake{in: make(chan *pb.ClientMessage, 64), out: make(chan *pb.ServerMessage, 64), done: make(chan struct{})}
}

func (f *fake) Send(_ context.Context, m *pb.ClientMessage) error {
	select {
	case <-f.done:
		return errors.New("closed")
	default:
	}
	f.in <- m
	return nil
}

func (f *fake) Recv(ctx context.Context) (*pb.ServerMessage, error) {
	select {
	case m := <-f.out:
		return m, nil
	case <-f.done:
		return nil, errors.New("closed")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *fake) Close() error {
	select {
	case <-f.done:
	default:
		close(f.done)
	}
	return nil
}

func (f *fake) say(b any) {
	switch v := b.(type) {
	case *pb.Welcome:
		f.out <- &pb.ServerMessage{Body: &pb.ServerMessage_Welcome{Welcome: v}}
	case *pb.Snapshot:
		f.out <- &pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: v}}
	case *pb.Event:
		f.out <- &pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: v}}
	case *pb.IntentResult:
		f.out <- &pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: v}}
	case *pb.Error:
		f.out <- &pb.ServerMessage{Body: &pb.ServerMessage_Error{Error: v}}
	}
}

func (f *fake) next(t *testing.T) *pb.ClientMessage {
	t.Helper()
	select {
	case m := <-f.in:
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("the client sent nothing")
		return nil
	}
}

func snap(seq uint64) *pb.Snapshot {
	return &pb.Snapshot{Seq: seq, Space: "outskirts", Self: &pb.SelfState{Entity: &pb.EntityState{Id: "char-aria", Name: "Aria", Hp: 100, MaxHp: 120, Pos: &pb.Pos{}}}}
}

func dialer(fs ...*fake) client.Dialer {
	i := 0
	return func(context.Context) (client.Transport, error) {
		f := fs[i]
		i++
		return f, nil
	}
}

func ctx(t *testing.T) context.Context {
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return c
}

// login runs the hello/select exchange against f, which answers it.
func login(t *testing.T, s *client.Session, f *fake) {
	t.Helper()
	go func() {
		h := f.next(t).GetHello()
		if h.GetCredential() != "seq_st_human" || h.GetProtocolVersion() != 1 {
			t.Errorf("hello %v", h)
		}
		f.say(&pb.Welcome{ProtocolVersion: 1})
		sel := f.next(t).GetSelect()
		if sel.GetCharacter() != "me-1" || sel.GetSession() != "sess" {
			t.Errorf("select %v", sel)
		}
		f.say(snap(10))
	}()
	if err := s.Login(ctx(t)); err != nil {
		t.Fatal(err)
	}
}

func newSession(d client.Dialer, rec *record.Writer) *client.Session {
	return client.NewSession(client.Config{Dial: d, Credential: "seq_st_human", Character: "me-1", Session: "sess", Record: rec})
}

func TestLoginAndEveryIntent(t *testing.T) {
	f := newFake()
	s := newSession(dialer(f), nil)
	login(t, s, f)
	s.View(func(m *model.Model) {
		if !m.Ready() || m.Self.Name != "Aria" || m.Last() != 10 {
			t.Errorf("model after login: %+v", m.Self)
		}
	})
	c := ctx(t)
	type cmd struct {
		kind string
		send func() (string, error)
		ok   func(*pb.Intent) bool
	}
	cmds := []cmd{
		{"move", func() (string, error) { return s.Move(c, 5000, 6000) }, func(i *pb.Intent) bool { return i.GetMove().GetDestination().GetY() == 6000 }},
		{"stop", func() (string, error) { return s.Stop(c) }, func(i *pb.Intent) bool { return i.GetStop() != nil }},
		{"target", func() (string, error) { return s.Target(c, "mob-1") }, func(i *pb.Intent) bool { return i.GetTarget().GetTarget() == "mob-1" }},
		{"attack", func() (string, error) { return s.Attack(c, true) }, func(i *pb.Intent) bool { return i.GetAttack().GetOn() }},
		{"loot", func() (string, error) { return s.Loot(c, "drop-1") }, func(i *pb.Intent) bool { return i.GetLoot().GetDrop() == "drop-1" }},
		{"use_door", func() (string, error) { return s.UseDoor(c, "line-east") }, func(i *pb.Intent) bool { return i.GetUseDoor().GetDoor() == "line-east" }},
	}
	for n, k := range cmds {
		id, err := k.send()
		if err != nil {
			t.Fatal(err)
		}
		got := f.next(t).GetIntent()
		if got.GetIntentId() != id || got.GetSeq() != uint64(n+1) || !k.ok(got) {
			t.Errorf("%s: sent %v for id %s", k.kind, got, id)
		}
		reason := ""
		if k.kind == "use_door" {
			reason = "door_unavailable"
		}
		f.say(&pb.IntentResult{IntentId: id, Accepted: reason == "", Reason: reason})
		st, err := s.Result(c, id)
		if err != nil {
			t.Fatal(err)
		}
		if st.Kind != k.kind || (reason == "") != (st.State == model.IntentAccepted) || st.Reason != reason {
			t.Errorf("%s: status %+v", k.kind, st)
		}
	}
	if err := s.Logout(c); err != nil {
		t.Fatal(err)
	}
	if f.next(t).GetLogout() == nil || !s.LoggedOut() {
		t.Error("logout not sent")
	}
}

func TestSelectRefused(t *testing.T) {
	f := newFake()
	s := newSession(dialer(f), nil)
	go func() {
		f.next(t)
		f.say(&pb.Welcome{ProtocolVersion: 1})
		f.next(t)
		f.say(&pb.Error{Code: pb.ErrorCode_ERROR_CODE_SELECT_REFUSED, Message: "not your character"})
	}()
	err := s.Login(ctx(t))
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("not your character")) {
		t.Fatalf("err %v", err)
	}
}

func TestDropResumeAndResend(t *testing.T) {
	f1, f2 := newFake(), newFake()
	var buf bytes.Buffer
	s := newSession(dialer(f1, f2), record.NewWriter(&buf))
	login(t, s, f1)
	c := ctx(t)
	f1.say(&pb.Event{Seq: 11, Body: &pb.Event_RulesTick{RulesTick: &pb.RulesTick{}}})
	id, err := s.Move(c, 1, 2) // in flight when the connection drops
	if err != nil {
		t.Fatal(err)
	}
	f1.next(t)
	f1.Close()
	deadline := time.After(5 * time.Second)
	for {
		if closed, _ := s.Closed(); closed {
			break
		}
		select {
		case <-deadline:
			t.Fatal("the drop was not noticed")
		case <-time.After(5 * time.Millisecond):
		}
	}
	go func() {
		f2.next(t) // hello
		f2.say(&pb.Welcome{ProtocolVersion: 1})
		r := f2.next(t).GetResume()
		if r.GetSession() != "sess" || r.GetLastSeq() != 11 {
			t.Errorf("resume %v: it must name the session and the last event applied", r)
		}
		again := f2.next(t).GetIntent()
		if again.GetIntentId() != id || again.GetMove().GetDestination().GetX() != 1 {
			t.Errorf("resent %v, want %s unchanged", again, id)
		}
		f2.say(&pb.Event{Seq: 12, Body: &pb.Event_RulesTick{RulesTick: &pb.RulesTick{}}})
		f2.say(&pb.IntentResult{IntentId: id, Accepted: true})
	}()
	if err := s.Reconnect(c); err != nil {
		t.Fatal(err)
	}
	if st, err := s.Result(c, id); err != nil || st.State != model.IntentAccepted {
		t.Fatalf("result %+v %v", st, err)
	}
	s.View(func(m *model.Model) {
		if m.Last() != 12 || m.Connections != 2 || len(m.Faults) != 0 || m.Duplicates != 0 {
			t.Errorf("last %d connections %d faults %v dups %d", m.Last(), m.Connections, m.Faults, m.Duplicates)
		}
	})
	// the session was recorded, across both connections, and replays to the same model
	msgs, err := record.Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	rm := model.New()
	for _, m := range msgs {
		rm.Apply(m)
	}
	if rm.Last() != 12 || rm.Connections != 2 || len(msgs) != 6 {
		t.Errorf("replay: last %d connections %d messages %d", rm.Last(), rm.Connections, len(msgs))
	}
}
