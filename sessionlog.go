package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// globalSessionLog is set at TUI startup for transport-level logging.
// Transport functions run in tea.Cmd goroutines and can't access the model,
// so they use this package-level reference. Thread-safe via sessionLogger's mutex.
var globalSessionLog *sessionLogger

// sessionLogger writes JSONL session events to a file when enabled.
// When disabled, all methods are no-ops with near-zero overhead.
// Thread-safe via mutex — transport calls happen concurrently.
type sessionLogger struct {
	mu      sync.Mutex
	file    *os.File
	enabled bool
	session string
}

// sessionEvent is the common envelope for all log events.
type sessionEvent struct {
	Ts   string `json:"ts"`
	Type string `json:"type"`
	// Remaining fields are event-specific and merged via json.Marshal of the full struct.
}

// newSessionLogger creates a session logger. If SEQ_TUI_SESSION_LOG is not "1",
// returns a disabled no-op logger. Uses SEQ_TUI_SESSION_LOG_DIR for output directory,
// defaulting to the current directory.
func newSessionLogger() *sessionLogger {
	if os.Getenv("SEQ_TUI_SESSION_LOG") != "1" {
		return &sessionLogger{enabled: false}
	}

	dir := os.Getenv("SEQ_TUI_SESSION_LOG_DIR")
	if dir == "" {
		dir = "."
	}

	sessionID := time.Now().UTC().Format("20060102-150405")
	filename := filepath.Join(dir, fmt.Sprintf("seq-tui-session-%s.jsonl", sessionID))

	f, err := os.Create(filename)
	if err != nil {
		// Silently degrade — don't crash the TUI for logging
		fmt.Fprintf(os.Stderr, "session log: failed to create %s: %v\n", filename, err)
		return &sessionLogger{enabled: false}
	}

	sl := &sessionLogger{
		file:    f,
		enabled: true,
		session: sessionID,
	}
	sl.logEvent("session_start", map[string]any{
		"session_id": sessionID,
		"log_file":   filename,
	})
	return sl
}

// newSessionLoggerTo creates a logger writing to a specific writer (for testing).
func newSessionLoggerTo(f *os.File) *sessionLogger {
	return &sessionLogger{
		file:    f,
		enabled: true,
		session: "test",
	}
}

// Close flushes and closes the session log file.
func (sl *sessionLogger) Close() {
	if sl == nil || !sl.enabled || sl.file == nil {
		return
	}
	sl.logEvent("session_end", nil)
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.file.Close()
}

// logEvent writes a single JSONL event line.
func (sl *sessionLogger) logEvent(eventType string, fields map[string]any) {
	if sl == nil || !sl.enabled {
		return
	}

	entry := map[string]any{
		"ts":   time.Now().UTC().Format(time.RFC3339Nano),
		"type": eventType,
	}
	for k, v := range fields {
		entry[k] = v
	}

	sl.mu.Lock()
	defer sl.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	sl.file.Write(data)
	sl.file.Write([]byte("\n"))
}

// --- Convenience methods for specific event types ---

// LogKey records a key press event.
func (sl *sessionLogger) LogKey(key string) {
	sl.logEvent("key", map[string]any{"key": key})
}

// LogIntent records a derived intent from user input.
func (sl *sessionLogger) LogIntent(name string, fields map[string]any) {
	merged := map[string]any{"name": name}
	for k, v := range fields {
		merged[k] = v
	}
	sl.logEvent("intent", merged)
}

// LogRequest records an outbound HTTP request (no secrets).
func (sl *sessionLogger) LogRequest(name, method, path string) {
	sl.logEvent("request", map[string]any{
		"name":   name,
		"method": method,
		"path":   scrubPath(path),
	})
}

// LogRequestWith records an outbound HTTP request with compact payload summary (no secrets).
func (sl *sessionLogger) LogRequestWith(name, method, path string, extra map[string]any) {
	fields := map[string]any{
		"name":   name,
		"method": method,
		"path":   scrubPath(path),
	}
	for k, v := range extra {
		fields[k] = v
	}
	sl.logEvent("request", fields)
}

// LogResponse records an inbound HTTP response.
func (sl *sessionLogger) LogResponse(name string, status int, ok bool, latencyMs int64) {
	sl.logEvent("response", map[string]any{
		"name":       name,
		"status":     status,
		"ok":         ok,
		"latency_ms": latencyMs,
	})
}

// LogResponseWith records an inbound HTTP response with extra summary fields.
func (sl *sessionLogger) LogResponseWith(name string, status int, ok bool, latencyMs int64, extra map[string]any) {
	fields := map[string]any{
		"name":       name,
		"status":     status,
		"ok":         ok,
		"latency_ms": latencyMs,
	}
	for k, v := range extra {
		fields[k] = v
	}
	sl.logEvent("response", fields)
}

// LogPlayerSnapshot records a compact player-state snapshot from polled data.
func (sl *sessionLogger) LogPlayerSnapshot(name string, joined bool, hasPos bool, x, y float64, hasEnc bool) {
	fields := map[string]any{
		"name":          name,
		"player_joined": joined,
	}
	if hasPos {
		fields["player_pos"] = []float64{x, y}
	}
	fields["has_active_encounter"] = hasEnc
	sl.logEvent("state", fields)
}

// LogPoll records a periodic poll read result.
func (sl *sessionLogger) LogPoll(name string, ok bool) {
	sl.logEvent("poll", map[string]any{
		"name": name,
		"ok":   ok,
	})
}

// LogState records a compact render-observable state snapshot.
func (sl *sessionLogger) LogState(fields map[string]any) {
	sl.logEvent("state", fields)
}

// scrubPath removes query parameters that may contain tokens.
// Keeps the path portion only for logging safety.
func scrubPath(path string) string {
	for i, c := range path {
		if c == '?' {
			return path[:i]
		}
	}
	return path
}
