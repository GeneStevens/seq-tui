package main

import (
	"encoding/json"
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

func TestSubmitMoveSuccess(t *testing.T) {
	// Mock: position endpoint returns ok:true, state endpoint returns position
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/position", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/world/dev/zone/testzone/player/p1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":{"Position":{"Pos":{"X":120.0,"Y":200.0,"Z":0}}}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitMoveAndReadback(target, playerPosResult{X: 100, Y: 200}, 20, 0)

	if !result.OK {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !result.HasPos {
		t.Fatal("expected position in result")
	}
	if result.Position.X != 120.0 {
		t.Fatalf("expected X=120, got %f", result.Position.X)
	}
}

func TestSubmitMoveBodyFalse(t *testing.T) {
	// Mock: position endpoint returns HTTP 200 but ok:false
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/position", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":false,"error":"player not in zone"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitMoveAndReadback(target, playerPosResult{X: 100, Y: 200}, 20, 0)

	if result.OK {
		t.Fatal("expected failure when body ok=false")
	}
	if !strings.Contains(result.Error, "player not in zone") {
		t.Fatalf("expected backend error message, got: %s", result.Error)
	}
}

func TestSubmitMoveBodyFalseNoError(t *testing.T) {
	// Mock: position endpoint returns ok:false without error field
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/position", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":false}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitMoveAndReadback(target, playerPosResult{X: 100, Y: 200}, 20, 0)

	if result.OK {
		t.Fatal("expected failure when body ok=false")
	}
	if result.Error == "" {
		t.Fatal("expected a default error message")
	}
}

func TestSubmitMoveMalformedBody(t *testing.T) {
	// Mock: returns HTTP 200 but garbage body
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/position", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitMoveAndReadback(target, playerPosResult{X: 100, Y: 200}, 20, 0)

	if result.OK {
		t.Fatal("expected failure on malformed body")
	}
	if !strings.Contains(result.Error, "decode") {
		t.Fatalf("expected decode error, got: %s", result.Error)
	}
}

func TestSubmitMoveHTTPError(t *testing.T) {
	// Mock: returns HTTP 500
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/position", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	result := submitMoveAndReadback(target, playerPosResult{X: 100, Y: 200}, 20, 0)

	if result.OK {
		t.Fatal("expected failure on HTTP 500")
	}
}

func TestSubmitMoveLogsBodyOutcome(t *testing.T) {
	// Set up session logger to capture events
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

	// Mock: returns ok:false
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/position", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":false,"error":"test_error"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	submitMoveAndReadback(target, playerPosResult{X: 100, Y: 200}, 20, 0)

	globalSessionLog.Close()
	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	// Should contain request event
	if !strings.Contains(s, "submit_move") {
		t.Fatal("log should contain submit_move events")
	}
	// Should contain body_ok:false in response
	if !strings.Contains(s, "body_ok") {
		t.Fatal("log should contain body_ok field")
	}

	// Parse the response event to verify shape
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for _, line := range lines {
		var entry map[string]any
		json.Unmarshal([]byte(line), &entry)
		if entry["type"] == "response" && entry["name"] == "submit_move" {
			if entry["body_ok"] != false {
				t.Fatalf("response should log body_ok=false, got: %v", entry["body_ok"])
			}
			return
		}
	}
	t.Fatal("no submit_move response event found in log")
}

func TestSubmitMoveNoTokenInLog(t *testing.T) {
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
	mux.HandleFunc("/world/dev/zone/testzone/player/position", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/world/dev/zone/testzone/player/p1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":{"Position":{"Pos":{"X":120,"Y":200,"Z":0}}}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	target.DevToken = "supersecrettoken123"
	submitMoveAndReadback(target, playerPosResult{X: 100, Y: 200}, 20, 0)

	globalSessionLog.Close()
	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(content), "supersecrettoken123") {
		t.Fatal("session log must not contain dev token")
	}
}
