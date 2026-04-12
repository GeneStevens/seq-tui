package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionIDHeaderSentOnMove(t *testing.T) {
	oldID := globalSessionID
	globalSessionID = "test-ksuid-abc123"
	defer func() { globalSessionID = oldID }()

	var capturedHeader string
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get(sessionIDHeader)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"x":0,"y":5,"z":0}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	submitDirectionalMove(target, "north")

	if capturedHeader != "test-ksuid-abc123" {
		t.Fatalf("expected session ID header %q, got %q", "test-ksuid-abc123", capturedHeader)
	}
}

func TestSessionIDHeaderSentOnAttack(t *testing.T) {
	oldID := globalSessionID
	globalSessionID = "test-session-atk"
	defer func() { globalSessionID = oldID }()

	var capturedHeader string
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/intent", func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get(sessionIDHeader)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	submitBasicAttack(target, "orc_01")

	if capturedHeader != "test-session-atk" {
		t.Fatalf("expected session ID header %q, got %q", "test-session-atk", capturedHeader)
	}
}

func TestSessionIDHeaderSentOnJoin(t *testing.T) {
	oldID := globalSessionID
	globalSessionID = "test-session-join"
	defer func() { globalSessionID = oldID }()

	var capturedHeaders []string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = append(capturedHeaders, r.Header.Get(sessionIDHeader))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"found":true,"player":{"position":{"x":0,"y":0,"z":0},"active_encounter_id":""}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	joinAndReadPlayer(target)

	// Both join POST and state GET should include session ID
	for i, h := range capturedHeaders {
		if h != "test-session-join" {
			t.Fatalf("request %d: expected session ID header %q, got %q", i, "test-session-join", h)
		}
	}
	if len(capturedHeaders) < 2 {
		t.Fatalf("expected at least 2 requests, got %d", len(capturedHeaders))
	}
}

func TestSessionIDHeaderSentOnEncounterRead(t *testing.T) {
	oldID := globalSessionID
	globalSessionID = "test-session-enc"
	defer func() { globalSessionID = oldID }()

	var capturedHeader string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get(sessionIDHeader)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":[]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	fetchZoneEncounters(target)

	if capturedHeader != "test-session-enc" {
		t.Fatalf("expected session ID header %q, got %q", "test-session-enc", capturedHeader)
	}
}

func TestSessionIDHeaderEmptyWhenNoID(t *testing.T) {
	oldID := globalSessionID
	globalSessionID = ""
	defer func() { globalSessionID = oldID }()

	var capturedHeader string
	mux := http.NewServeMux()
	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get(sessionIDHeader)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"x":0,"y":0,"z":0}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	submitDirectionalMove(target, "north")

	if capturedHeader != "" {
		t.Fatalf("expected empty session ID header when globalSessionID is empty, got %q", capturedHeader)
	}
}

func TestGenerateSessionIDIsKSUID(t *testing.T) {
	id := generateSessionID()
	if id == "" {
		t.Fatal("generateSessionID should return non-empty string")
	}
	// KSUIDs are 27 characters
	if len(id) != 27 {
		t.Fatalf("expected KSUID length 27, got %d: %q", len(id), id)
	}
}

func TestGenerateSessionIDUnique(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()
	if id1 == id2 {
		t.Fatal("two generated session IDs should be unique")
	}
}
