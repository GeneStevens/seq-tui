package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/client"
)

// device is a server that approves on the second poll.
func device(t *testing.T, final string) *httptest.Server {
	polls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/device/start", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"device_code": "d", "user_code": "ABCD-EFGH", "verification_uri": "https://h/activate",
			"verification_uri_complete": "https://h/activate?code=ABCD-EFGH", "expires_in": 600, "interval": 1})
	})
	mux.HandleFunc("/auth/device/token", func(w http.ResponseWriter, r *http.Request) {
		polls++
		if polls == 1 {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			return
		}
		if final != "ok" {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": final})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"token": "seq_st_new", "account": "gene"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func testSignIn(t *testing.T, srv *httptest.Server, opened *[]string) (signIn, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return signIn{Gateway: "ws" + strings.TrimPrefix(srv.URL, "http") + "/seq/v1", Dir: t.TempDir(), Out: out,
		Open: func(u string) error { *opened = append(*opened, u); return nil },
		Now:  func() time.Time { return time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC) },
		Wait: func(context.Context, time.Duration) error { return nil }}, out
}

func TestLoginLogoutAndToken(t *testing.T) {
	var opened []string
	si, out := testSignIn(t, device(t, "ok"), &opened)
	if _, ok := si.Saved(); ok {
		t.Fatal("saved before login")
	}
	// no env, nothing saved: a login, then the saved token is used with no second login
	tok, saved, err := si.Token(context.Background(), "")
	if err != nil || tok != "seq_st_new" || saved {
		t.Fatalf("Token = %q, %v, %v", tok, saved, err)
	}
	for _, want := range []string{"ABCD-EFGH", "https://h/activate", "Signed in as gene"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if len(opened) != 1 || opened[0] != "https://h/activate?code=ABCD-EFGH" {
		t.Errorf("opened %v", opened)
	}
	if tok, saved, err = si.Token(context.Background(), ""); err != nil || tok != "seq_st_new" || !saved {
		t.Fatalf("second Token = %q, %v, %v", tok, saved, err)
	}
	if tok, saved, _ = si.Token(context.Background(), "env_tok"); tok != "env_tok" || saved {
		t.Errorf("SEQ_SESSION_TOKEN must win: %q %v", tok, saved)
	}
	if err := si.Logout(); err != nil {
		t.Fatal(err)
	}
	if _, ok := si.Saved(); ok {
		t.Error("still saved after logout")
	}
	out.Reset()
	si.Logout()
	if !strings.Contains(out.String(), "No saved sign-in") {
		t.Errorf("second logout said %q", out)
	}
}

func TestLoginFailures(t *testing.T) {
	for final, want := range map[string]string{"access_denied": "denied", "expired_token": "expired"} {
		var opened []string
		si, _ := testSignIn(t, device(t, final), &opened)
		_, err := si.Login(context.Background())
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v", final, err)
		}
		if _, ok := si.Saved(); ok {
			t.Errorf("%s: saved a session", final)
		}
	}
}

func TestSavedExpires(t *testing.T) {
	var opened []string
	si, _ := testSignIn(t, device(t, "ok"), &opened)
	if _, err := si.Login(context.Background()); err != nil {
		t.Fatal(err)
	}
	si.Now = func() time.Time {
		return time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC).Add(client.SavedLifetime + time.Hour)
	}
	if _, ok := si.Saved(); ok {
		t.Error("a session past its lifetime must not be used")
	}
}

func TestTLSFor(t *testing.T) {
	for _, tc := range []struct {
		addr     string
		insecure bool
		ok       bool
	}{{"wss://127.0.0.1:8443/seq/v1", true, true}, {"wss://play.spielengine.com/seq/v1", true, false}, {"wss://play.spielengine.com/seq/v1", false, true}} {
		if _, err := tlsFor(tc.addr, tc.insecure); (err == nil) != tc.ok {
			t.Errorf("tlsFor(%q, %v) = %v", tc.addr, tc.insecure, err)
		}
	}
}
