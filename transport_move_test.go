package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// testTargetWithServer returns a backendTarget pointing at the given test server.
func testTargetWithServer(serverURL string) backendTarget {
	return backendTarget{
		BaseURL:  serverURL,
		Zone:     "testzone",
		Mode:     "RT",
		Player:   "p1",
		DevToken: "test-token",
	}
}

func TestSubmitDirectionalMoveSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		// Verify request shape
		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		json.Unmarshal(body, &req)
		if req["player_id"] != "p1" {
			t.Fatalf("expected player_id=p1, got %s", req["player_id"])
		}
		if req["direction"] != "up" {
			t.Fatalf("expected direction=up, got %s", req["direction"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"x":0,"y":5,"z":0}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitDirectionalMove(target, "north") // north → up

	if !result.OK {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !result.HasPos {
		t.Fatal("expected position in result")
	}
	if result.Position.Y != 5.0 {
		t.Fatalf("expected Y=5, got %f", result.Position.Y)
	}
}

func TestSubmitDirectionalMoveBodyFalse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":false,"error":"player not found"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitDirectionalMove(target, "north")

	if result.OK {
		t.Fatal("expected failure when body ok=false")
	}
	if !strings.Contains(result.Error, "player not found") {
		t.Fatalf("expected backend error, got: %s", result.Error)
	}
}

func TestSubmitDirectionalMoveMalformedBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitDirectionalMove(target, "east")

	if result.OK {
		t.Fatal("expected failure on malformed body")
	}
}

func TestSubmitDirectionalMoveHTTPError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitDirectionalMove(target, "south")

	if result.OK {
		t.Fatal("expected failure on HTTP 500")
	}
}

func TestSubmitDirectionalMoveDirectionMapping(t *testing.T) {
	var capturedDirection string
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		json.Unmarshal(body, &req)
		capturedDirection = req["direction"]
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"x":0,"y":0,"z":0}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)

	cases := map[string]string{
		"north": "up",
		"south": "down",
		"east":  "right",
		"west":  "left",
	}
	for tuiDir, expectedBackend := range cases {
		submitDirectionalMove(target, tuiDir)
		if capturedDirection != expectedBackend {
			t.Fatalf("direction %q should map to %q, got %q", tuiDir, expectedBackend, capturedDirection)
		}
	}
}

func TestSubmitDirectionalMoveUsesCorrectEndpoint(t *testing.T) {
	var capturedPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"x":0,"y":0,"z":0}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	submitDirectionalMove(target, "north")

	if capturedPath != "/world/dev/zone/testzone/player/move" {
		t.Fatalf("expected /player/move endpoint, got: %s", capturedPath)
	}
}

func TestSubmitDirectionalMoveLogsBodyOutcome(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	oldLog := globalSessionLog
	globalSessionLog = newSessionLoggerTo(f)
	defer func() {
		globalSessionLog.Close()
		globalSessionLog = oldLog
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"x":5,"y":0,"z":0}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	submitDirectionalMove(target, "east")

	globalSessionLog.Close()
	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "submit_move") {
		t.Fatal("log should contain submit_move events")
	}
	if !strings.Contains(s, "body_ok") {
		t.Fatal("log should contain body_ok field")
	}
	// Should log direction, not from/to coordinates
	if !strings.Contains(s, `"direction"`) {
		t.Fatal("log should contain direction field")
	}
	if strings.Contains(s, `"from"`) {
		t.Fatal("log should not contain client-computed from coordinates")
	}
}

func TestSubmitDirectionalMoveNoTokenInLog(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	oldLog := globalSessionLog
	globalSessionLog = newSessionLoggerTo(f)
	defer func() {
		globalSessionLog.Close()
		globalSessionLog = oldLog
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"x":0,"y":0,"z":0}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	target.DevToken = "supersecrettoken123"
	submitDirectionalMove(target, "north")

	globalSessionLog.Close()
	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(content), "supersecrettoken123") {
		t.Fatal("session log must not contain dev token")
	}
}

func TestSubmitDirectionalMoveNoClientCoordinates(t *testing.T) {
	// Verify the request body has no x/y/z fields — only player_id + direction
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if strings.Contains(s, `"x"`) || strings.Contains(s, `"y"`) || strings.Contains(s, `"z"`) {
			t.Fatal("request body must not contain coordinate fields")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"x":5,"y":0,"z":0}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitDirectionalMove(target, "east")
	if !result.OK {
		t.Fatalf("expected success, got: %s", result.Error)
	}
}
