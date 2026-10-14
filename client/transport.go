// Package client is the TUI's connection to a SEQ server: the transport, the
// session that feeds the model from it, and the commands that send intents.
//
// It uses the generated protocol types of the seq module and nothing else of
// the game. agentkit's Client is not used for the connection: it keeps what it
// receives but offers no tap on the raw incoming messages, which recording a
// session needs (see the report's Design questions).
package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// ProtocolVersion is the protocol version this client speaks.
const ProtocolVersion = 1

// Transport carries protocol messages. The WebSocket one is the real one; tests
// use an in-memory one.
type Transport interface {
	Send(ctx context.Context, m *pb.ClientMessage) error
	Recv(ctx context.Context) (*pb.ServerMessage, error)
	Close() error
}

// Dialer opens a new Transport (one per connection; a resume dials again).
type Dialer func(ctx context.Context) (Transport, error)

// WebSocketDialer dials the gateway at url (wss://host:port/seq/v1).
func WebSocketDialer(url string, cfg *tls.Config) Dialer {
	return func(ctx context.Context) (Transport, error) {
		hc := &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}
		conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPClient: hc})
		if err != nil {
			return nil, err
		}
		return &wsTransport{conn}, nil
	}
}

type wsTransport struct{ conn *websocket.Conn }

func (t *wsTransport) Send(ctx context.Context, m *pb.ClientMessage) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.conn.Write(ctx, websocket.MessageBinary, b)
}

func (t *wsTransport) Recv(ctx context.Context) (*pb.ServerMessage, error) {
	typ, b, err := t.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	var m pb.ServerMessage
	if typ != websocket.MessageBinary || proto.Unmarshal(b, &m) != nil {
		return nil, errBadMessage
	}
	return &m, nil
}

func (t *wsTransport) Close() error { return t.conn.Close(websocket.StatusNormalClosure, "bye") }

var errBadMessage = errors.New("the server sent something that is not a protocol message")

func describeErr(what string, err error) error { return fmt.Errorf("%s: %w", what, err) }
