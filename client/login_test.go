package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// fakeDevice is the server's device endpoints: the token poll answers with each
// of answers in turn (the last one repeats).
func fakeDevice(t *testing.T, answers []string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var polls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/device/start", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"device_code": "dev1", "user_code": "ABCD-EFGH",
			"verification_uri": "http://x/activate", "verification_uri_complete": "http://x/activate?code=ABCD-EFGH", "expires_in": 600, "interval": 5})
	})
	mux.HandleFunc("/auth/device/token", func(w http.ResponseWriter, r *http.Request) {
		n := int(polls.Add(1)) - 1
		if n >= len(answers) {
			n = len(answers) - 1
		}
		if a := answers[n]; a != "ok" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": a, "message": "m"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": "seq_st_x", "token": "seq_st_x", "account": "acc-1"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &polls
}

func TestDeviceLoginRun(t *testing.T) {
	for _, tc := range []struct {
		name    string
		answers []string
		waits   []time.Duration // the sleeps before each poll
		want    error
	}{
		{"approved after pending", []string{"authorization_pending", "authorization_pending", "ok"}, []time.Duration{5, 5, 5}, nil},
		{"slow_down adds five seconds", []string{"slow_down", "ok"}, []time.Duration{5, 10}, nil},
		{"denied", []string{"authorization_pending", "access_denied"}, []time.Duration{5, 5}, ErrLoginDenied},
		{"expired", []string{"expired_token"}, []time.Duration{5}, ErrLoginExpired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := fakeDevice(t, tc.answers)
			var waits []time.Duration
			shown := 0
			d := DeviceLogin{Base: srv.URL,
				Wait:  func(_ context.Context, d time.Duration) error { waits = append(waits, d/time.Second); return nil },
				Shown: func(dev Device) { shown++ }}
			g, err := d.Run(context.Background())
			if err != tc.want {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if tc.want == nil && (g.Token != "seq_st_x" || g.Account != "acc-1") {
				t.Errorf("grant = %+v", g)
			}
			if shown != 1 || len(waits) != len(tc.waits) {
				t.Fatalf("shown %d, waits %v, want %v", shown, waits, tc.waits)
			}
			for i := range waits {
				if waits[i] != tc.waits[i] {
					t.Errorf("waits = %v, want %v", waits, tc.waits)
				}
			}
		})
	}
}

func TestDeviceLoginNotOffered(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	_, err := (&DeviceLogin{Base: srv.URL}).Run(context.Background())
	if err == nil || err.Error() == "" {
		t.Fatal("a server without /auth/ must be an error")
	}
}

func TestHTTPBase(t *testing.T) {
	for in, want := range map[string]string{
		"wss://play.spielengine.com/seq/v1": "https://play.spielengine.com",
		"wss://127.0.0.1:8443/seq/v1":       "https://127.0.0.1:8443",
		"ws://localhost:8080/seq/v1":        "http://localhost:8080",
		"https://x.example/seq/v1":          "https://x.example",
		"ftp://x/y":                         "",
		"nonsense":                          "",
	} {
		got, err := HTTPBase(in)
		if (err != nil) != (want == "") || got != want {
			t.Errorf("HTTPBase(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestSavedSession(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "seq")
	path, err := SessionPath(dir, "wss://127.0.0.1:8443/seq/v1")
	if err != nil || filepath.Base(path) != "127.0.0.1_8443.session" {
		t.Fatalf("path = %q, %v", path, err)
	}
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	if _, ok := LoadSession(path, now); ok {
		t.Fatal("no file must not load")
	}
	if err := SaveSession(path, Saved{Token: "seq_st_a", Account: "a", Saved: now}); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		fi, _ := os.Stat(path)
		di, _ := os.Stat(dir)
		if fi.Mode().Perm() != 0o600 || di.Mode().Perm() != 0o700 {
			t.Errorf("modes: file %v, dir %v; want 0600, 0700", fi.Mode().Perm(), di.Mode().Perm())
		}
	}
	for _, tc := range []struct {
		name string
		at   time.Time
		ok   bool
	}{
		{"fresh", now.Add(time.Hour), true},
		{"just inside", now.Add(SavedLifetime), true},
		{"expired", now.Add(SavedLifetime + time.Second), false},
	} {
		if s, ok := LoadSession(path, tc.at); ok != tc.ok || (ok && s.Token != "seq_st_a") {
			t.Errorf("%s: ok = %v", tc.name, ok)
		}
	}
	// saving over a file with loose permissions tightens them
	os.Chmod(path, 0o644)
	if err := SaveSession(path, Saved{Token: "seq_st_b", Saved: now}); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(path); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("re-saved mode %v", fi.Mode().Perm())
	}
	if err := RemoveSession(path); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSession(path); err != nil {
		t.Errorf("removing twice: %v", err)
	}
	os.WriteFile(path, []byte("not json"), 0o600)
	if _, ok := LoadSession(path, now); ok {
		t.Error("garbage must not load")
	}
}

func TestSavedWho(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.session")
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ name, file, want string }{
		{"with email", `{"token":"seq_st_a","account":"p-1","email":"a@b.c","saved":"2026-10-31T12:00:00Z"}`, "a@b.c"},
		{"old file, no email", `{"token":"seq_st_a","account":"p-1","saved":"2026-10-31T12:00:00Z"}`, "p-1"},
		{"empty email", `{"token":"seq_st_a","account":"p-1","email":"","saved":"2026-10-31T12:00:00Z"}`, "p-1"},
	} {
		if err := os.WriteFile(path, []byte(tc.file), 0o600); err != nil {
			t.Fatal(err)
		}
		s, ok := LoadSession(path, now)
		if !ok || s.Token != "seq_st_a" || s.Who() != tc.want {
			t.Errorf("%s: ok=%v token=%q who=%q, want %q", tc.name, ok, s.Token, s.Who(), tc.want)
		}
	}
	if err := SaveSession(path, Saved{Token: "t", Account: "p-1", Email: "a@b.c", Saved: now}); err != nil {
		t.Fatal(err)
	}
	if s, _ := LoadSession(path, now); s.Email != "a@b.c" {
		t.Errorf("email not kept: %+v", s)
	}
}
