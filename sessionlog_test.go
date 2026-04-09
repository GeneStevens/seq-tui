package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSessionLoggerDisabledByDefault(t *testing.T) {
	// Ensure env is not set
	os.Unsetenv("SEQ_TUI_SESSION_LOG")
	sl := newSessionLogger()
	if sl.enabled {
		t.Fatal("session logger should be disabled when env not set")
	}
	// All methods should be no-ops (no panic)
	sl.LogKey("a")
	sl.LogIntent("move", map[string]any{"dir": "north"})
	sl.LogRequest("test", "GET", "/foo?token=secret")
	sl.LogResponse("test", 200, true, 10)
	sl.LogPoll("player", true)
	sl.LogState(map[string]any{"joined": true})
	sl.Close()
}

func TestSessionLoggerNilSafe(t *testing.T) {
	var sl *sessionLogger
	// All methods should be no-ops on nil
	sl.LogKey("a")
	sl.LogIntent("move", nil)
	sl.LogRequest("test", "GET", "/foo")
	sl.LogResponse("test", 200, true, 10)
	sl.LogPoll("player", true)
	sl.LogState(nil)
	sl.Close()
}

func TestSessionLoggerWritesJSONL(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sl := newSessionLoggerTo(f)

	sl.LogKey("k")
	sl.LogIntent("move", map[string]any{"dir": "north"})
	sl.LogRequest("submit_intent", "POST", "/world/dev/zone/crushbone/intent")
	sl.LogResponse("submit_intent", 200, true, 24)
	sl.LogPoll("player_state", true)
	sl.LogState(map[string]any{"player_joined": true, "encounter_state": "Active"})
	sl.Close()

	// Re-read the file and validate JSONL
	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineCount := 0
	eventTypes := make(map[string]bool)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("line %d is not valid JSON: %s", lineCount+1, line)
		}
		// Every event must have ts and type
		if _, ok := entry["ts"]; !ok {
			t.Fatalf("line %d missing ts field", lineCount+1)
		}
		if et, ok := entry["type"].(string); ok {
			eventTypes[et] = true
		} else {
			t.Fatalf("line %d missing or non-string type field", lineCount+1)
		}
		lineCount++
	}

	// Should have at least 6 events (key, intent, request, response, poll, state)
	if lineCount < 6 {
		t.Fatalf("expected at least 6 events, got %d", lineCount)
	}

	// Check expected event types
	expected := []string{"key", "intent", "request", "response", "poll", "state"}
	for _, et := range expected {
		if !eventTypes[et] {
			t.Fatalf("missing event type: %s (got: %v)", et, eventTypes)
		}
	}
}

func TestSessionLogEventShapes(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sl := newSessionLoggerTo(f)

	sl.LogKey("a")
	sl.LogIntent("attack", map[string]any{"target": "orc-1"})
	sl.LogRequest("basic_attack", "POST", "/world/dev/zone/crushbone/intent?token=abc")
	sl.LogResponse("basic_attack", 200, true, 15)
	sl.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	// Key event
	var keyEvent map[string]any
	json.Unmarshal([]byte(lines[0]), &keyEvent)
	if keyEvent["type"] != "key" || keyEvent["key"] != "a" {
		t.Fatalf("unexpected key event: %v", keyEvent)
	}

	// Intent event
	var intentEvent map[string]any
	json.Unmarshal([]byte(lines[1]), &intentEvent)
	if intentEvent["type"] != "intent" || intentEvent["name"] != "attack" {
		t.Fatalf("unexpected intent event: %v", intentEvent)
	}

	// Request event — path should be scrubbed
	var reqEvent map[string]any
	json.Unmarshal([]byte(lines[2]), &reqEvent)
	if reqEvent["type"] != "request" {
		t.Fatalf("unexpected request event: %v", reqEvent)
	}
	path := reqEvent["path"].(string)
	if strings.Contains(path, "token") {
		t.Fatalf("request path should be scrubbed of query params: %s", path)
	}

	// Response event
	var respEvent map[string]any
	json.Unmarshal([]byte(lines[3]), &respEvent)
	if respEvent["type"] != "response" || respEvent["ok"] != true {
		t.Fatalf("unexpected response event: %v", respEvent)
	}
}

func TestScrubPath(t *testing.T) {
	cases := []struct {
		input, expected string
	}{
		{"/world/dev/zone/crushbone/intent", "/world/dev/zone/crushbone/intent"},
		{"/world/dev/zone/crushbone/intent?token=abc&mode=Async", "/world/dev/zone/crushbone/intent"},
		{"/foo?secret=bar", "/foo"},
		{"", ""},
		{"/no-query", "/no-query"},
	}
	for _, tc := range cases {
		got := scrubPath(tc.input)
		if got != tc.expected {
			t.Errorf("scrubPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestSessionLoggerDeterministic(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sl := newSessionLoggerTo(f)
	sl.LogKey("k")
	sl.LogKey("j")
	sl.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	// All should be valid JSON; first two should be type "key"
	for i, line := range lines[:2] {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("line %d not valid JSON: %s", i, line)
		}
		if entry["type"] != "key" {
			t.Fatalf("line %d type should be 'key', got %v", i, entry["type"])
		}
	}
}

func TestSessionLoggerNoSecretInTokenFields(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sl := newSessionLoggerTo(f)

	// Log a request with a URL that contains token/auth params
	sl.LogRequest("join", "POST", "/world/dev/zone/crushbone/player/join?token=supersecret123")
	sl.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(content), "supersecret") {
		t.Fatal("session log should not contain secret token values")
	}
}

// --- Movement Path Observability Tests (M20260409-18) ---

func TestLogRequestWithPayload(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sl := newSessionLoggerTo(f)
	sl.LogRequestWith("submit_move", "POST", "/world/dev/zone/cb/position?token=abc", map[string]any{
		"action": "move",
		"from":   []float64{100, 200},
		"to":     []float64{120, 200},
	})
	sl.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	// Check shape
	var entry map[string]any
	lines := strings.Split(strings.TrimSpace(s), "\n")
	json.Unmarshal([]byte(lines[0]), &entry)
	if entry["type"] != "request" {
		t.Fatal("should be request type")
	}
	if entry["name"] != "submit_move" {
		t.Fatal("should have name submit_move")
	}
	if entry["action"] != "move" {
		t.Fatal("should have action field")
	}
	// Token should be scrubbed
	path := entry["path"].(string)
	if strings.Contains(path, "token") {
		t.Fatalf("path should be scrubbed: %s", path)
	}
}

func TestLogResponseWithExtra(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sl := newSessionLoggerTo(f)
	sl.LogResponseWith("submit_move", 200, true, 24, map[string]any{"result": "accepted"})
	sl.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry map[string]any
	json.Unmarshal([]byte(lines[0]), &entry)
	if entry["type"] != "response" {
		t.Fatal("should be response type")
	}
	if entry["result"] != "accepted" {
		t.Fatal("should have extra result field")
	}
}

func TestLogPlayerSnapshot(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sl := newSessionLoggerTo(f)
	sl.LogPlayerSnapshot("player_state", true, true, 123.4, 456.7, false)
	sl.LogPlayerSnapshot("player_state", true, false, 0, 0, true)
	sl.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	// First: with position
	var e1 map[string]any
	json.Unmarshal([]byte(lines[0]), &e1)
	if e1["type"] != "state" {
		t.Fatal("should be state type")
	}
	if e1["name"] != "player_state" {
		t.Fatal("should have name")
	}
	if e1["player_joined"] != true {
		t.Fatal("should show joined")
	}
	pos := e1["player_pos"].([]any)
	if len(pos) != 2 {
		t.Fatal("player_pos should be [x, y]")
	}

	// Second: without position
	var e2 map[string]any
	json.Unmarshal([]byte(lines[1]), &e2)
	if _, hasPosField := e2["player_pos"]; hasPosField {
		t.Fatal("should not have player_pos when hasPos is false")
	}
	if e2["has_active_encounter"] != true {
		t.Fatal("should show encounter status")
	}
}

func TestMovementChainEventTypes(t *testing.T) {
	// Simulate a full movement chain through the logger
	f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sl := newSessionLoggerTo(f)

	// 1. Key
	sl.LogKey("k")
	// 2. Intent
	sl.LogIntent("move", map[string]any{"dir": "north"})
	// 3. Request
	sl.LogRequestWith("submit_move", "POST", "/world/dev/zone/cb/position", map[string]any{
		"action": "move", "from": []float64{100, 200}, "to": []float64{100, 220},
	})
	// 4. Response
	sl.LogResponse("submit_move", 200, true, 24)
	// 5. Poll
	sl.LogPoll("player_state", true)
	// 6. State snapshot
	sl.LogPlayerSnapshot("player_state", true, true, 100, 220, false)
	// 7. Render state
	sl.LogState(map[string]any{
		"name": "render_player", "move_ok": true, "move_dir": "north",
		"player_pos": []float64{100, 220}, "has_pos": true,
	})
	sl.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	expectedTypes := []string{"key", "intent", "request", "response", "poll", "state", "state"}
	if len(lines) < len(expectedTypes) {
		t.Fatalf("expected at least %d lines, got %d", len(expectedTypes), len(lines))
	}
	for i, et := range expectedTypes {
		var entry map[string]any
		json.Unmarshal([]byte(lines[i]), &entry)
		if entry["type"] != et {
			t.Fatalf("line %d: expected type %q, got %q", i, et, entry["type"])
		}
	}
}
