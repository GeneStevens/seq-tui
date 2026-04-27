package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newSmokeServer(t *testing.T, opts smokeServerOpts) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/world/zone/testzone", func(w http.ResponseWriter, r *http.Request) {
		if opts.zoneDown {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"process_name":"zone-testzone-1","message":"ok"}`)
	})

	mux.HandleFunc("/world/dev/zone/testzone/player/join", func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("X-Seq-Dev-Token")
		if opts.tokenMismatch && tok != "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if opts.devAPIDisabled {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	})

	mux.HandleFunc("/world/dev/zone/testzone/player/p1", func(w http.ResponseWriter, r *http.Request) {
		if opts.playerReadFails {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"found":true,"x":10,"y":20,"z":0,"active_encounter_id":""}`)
	})

	mux.HandleFunc("/world/zone/testzone/mob_positions", func(w http.ResponseWriter, r *http.Request) {
		if opts.mobsFail {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if opts.noMobs {
			fmt.Fprint(w, `{"result":{}}`)
			return
		}
		fmt.Fprint(w, `{"result":{"pid-goblin-1":{"mob_id":"goblin-1","live_mob_id":"live-goblin-1","mob_name":"a goblin","position":{"x":5,"y":5,"z":0}}}}`)
	})

	mux.HandleFunc("/world/dev/zone/testzone/player/p1/target/pid-goblin-1/proximity", func(w http.ResponseWriter, r *http.Request) {
		if opts.proxFail {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"found":true,"within_proximity":true,"distance_2d":3.5,"target_mob_name":"a goblin","target_mob_id":"goblin-1"}`)
	})

	return httptest.NewServer(mux)
}

type smokeServerOpts struct {
	zoneDown        bool
	devAPIDisabled  bool
	tokenMismatch   bool
	playerReadFails bool
	mobsFail        bool
	noMobs          bool
	proxFail        bool
}

func TestSmokeAllPass(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	if len(steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(steps))
	}
	for _, s := range steps {
		if s.Status != smokePass {
			t.Errorf("step %q: expected PASS, got %s — %s", s.Name, s.Status, s.Detail)
		}
	}

	assertStepName(t, steps, 0, "backend reachable")
	assertStepName(t, steps, 1, "player join")
	assertStepName(t, steps, 2, "player state")
	assertStepName(t, steps, 3, "mob positions")
	assertStepName(t, steps, 4, "target proximity")

	if !strings.Contains(steps[1].Detail, "joined at") {
		t.Errorf("player join detail should contain position: %s", steps[1].Detail)
	}
	if !strings.Contains(steps[4].Detail, "a goblin") {
		t.Errorf("target proximity should name mob: %s", steps[4].Detail)
	}
}

func TestSmokeBackendDown(t *testing.T) {
	target := backendTarget{
		BaseURL:  "http://127.0.0.1:1",
		Zone:     "testzone",
		Mode:     "RT",
		Player:   "p1",
		DevToken: "test-token",
	}
	steps := runSmoke(target)

	if len(steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(steps))
	}
	if steps[0].Status != smokeFail {
		t.Errorf("backend reachable should FAIL, got %s", steps[0].Status)
	}
	if !strings.Contains(steps[0].Detail, "connection refused") {
		t.Errorf("should report connection refused: %s", steps[0].Detail)
	}
	for _, s := range steps[1:] {
		if s.Status != smokeSkip {
			t.Errorf("step %q should be SKIP after backend down, got %s", s.Name, s.Status)
		}
	}
}

func TestSmokeDevAPIDisabled(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{devAPIDisabled: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	if steps[0].Status != smokePass {
		t.Errorf("backend should be reachable: %s", steps[0].Detail)
	}
	if steps[1].Status != smokeFail {
		t.Errorf("player join should FAIL when dev API disabled, got %s", steps[1].Status)
	}
	if !strings.Contains(steps[1].Detail, "not found") {
		t.Errorf("should mention not found / dev API: %s", steps[1].Detail)
	}
}

func TestSmokeTokenMismatch(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{tokenMismatch: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	if steps[0].Status != smokePass {
		t.Errorf("backend should be reachable: %s", steps[0].Detail)
	}
	if steps[1].Status != smokeFail {
		t.Errorf("player join should FAIL on token mismatch, got %s", steps[1].Status)
	}
	if !strings.Contains(steps[1].Detail, "forbidden") {
		t.Errorf("should mention forbidden / token: %s", steps[1].Detail)
	}
}

func TestSmokePlayerReadFails(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{playerReadFails: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	if steps[0].Status != smokePass {
		t.Errorf("backend should be reachable")
	}
	// Join succeeds but player state read returns 500 — joinAndReadPlayer
	// treats join-ok + read-fail as playerReadOK with HasPos=false
	if steps[1].Status != smokePass {
		t.Errorf("player join should still PASS (join succeeded): %s", steps[1].Detail)
	}
	if !strings.Contains(steps[1].Detail, "no position") {
		t.Errorf("should indicate no position: %s", steps[1].Detail)
	}
}

func TestSmokeMobsFail(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{mobsFail: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	if steps[0].Status != smokePass {
		t.Errorf("backend should be reachable")
	}
	if steps[1].Status != smokePass {
		t.Errorf("player join should PASS")
	}

	mobStep := findStep(steps, "mob positions")
	if mobStep == nil {
		t.Fatal("missing mob positions step")
	}
	if mobStep.Status != smokeFail {
		t.Errorf("mob positions should FAIL: %s", mobStep.Detail)
	}

	proxStep := findStep(steps, "target proximity")
	if proxStep == nil {
		t.Fatal("missing target proximity step")
	}
	if proxStep.Status != smokeSkip {
		t.Errorf("target proximity should SKIP when mobs fail: %s", proxStep.Detail)
	}
}

func TestSmokeNoMobs(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{noMobs: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	mobStep := findStep(steps, "mob positions")
	if mobStep == nil {
		t.Fatal("missing mob positions step")
	}
	if mobStep.Status != smokePass {
		t.Errorf("mob positions should PASS with 0 mobs: %s", mobStep.Detail)
	}

	proxStep := findStep(steps, "target proximity")
	if proxStep == nil {
		t.Fatal("missing target proximity step")
	}
	if proxStep.Status != smokeSkip {
		t.Errorf("target proximity should SKIP with no mobs: %s", proxStep.Detail)
	}
	if !strings.Contains(proxStep.Detail, "no mobs") {
		t.Errorf("should explain skip reason: %s", proxStep.Detail)
	}
}

func TestSmokeProximityFails(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{proxFail: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	for _, s := range steps[:4] {
		if s.Status != smokePass {
			t.Errorf("step %q should PASS: %s", s.Name, s.Detail)
		}
	}

	proxStep := findStep(steps, "target proximity")
	if proxStep == nil {
		t.Fatal("missing target proximity step")
	}
	if proxStep.Status != smokeFail {
		t.Errorf("target proximity should FAIL: %s", proxStep.Detail)
	}
}

func TestSmokeStepStringer(t *testing.T) {
	s := smokeStep{Name: "test", Status: smokePass, Detail: "ok"}
	got := s.String()
	if got != "[PASS] test — ok" {
		t.Errorf("unexpected String(): %s", got)
	}

	s2 := smokeStep{Name: "test", Status: smokeFail}
	got2 := s2.String()
	if got2 != "[FAIL] test" {
		t.Errorf("unexpected String() without detail: %s", got2)
	}
}

func TestSmokeHasFailed(t *testing.T) {
	passing := []smokeStep{
		{Status: smokePass},
		{Status: smokeSkip},
	}
	if smokeHasFailed(passing) {
		t.Error("should not report failure for pass+skip")
	}

	failing := []smokeStep{
		{Status: smokePass},
		{Status: smokeFail},
	}
	if !smokeHasFailed(failing) {
		t.Error("should report failure when a step fails")
	}
}

func TestSmokeStatusStringer(t *testing.T) {
	cases := []struct {
		s    smokeStatus
		want string
	}{
		{smokePass, "PASS"},
		{smokeFail, "FAIL"},
		{smokeSkip, "SKIP"},
		{smokeStatus(99), "UNKNOWN"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("smokeStatus(%d).String() = %q, want %q", tc.s, got, tc.want)
		}
	}
}

func TestSmokeConfigDisplay(t *testing.T) {
	target := backendTarget{
		BaseURL:  "http://localhost:9090",
		Zone:     "crushbone",
		Player:   "p1",
		DevToken: "abc",
	}
	got := smokeConfigSummary(target)
	if !strings.Contains(got, "localhost:9090") {
		t.Errorf("should contain URL: %s", got)
	}
	if !strings.Contains(got, "crushbone") {
		t.Errorf("should contain zone: %s", got)
	}
	if !strings.Contains(got, "p1") {
		t.Errorf("should contain player: %s", got)
	}
	if strings.Contains(got, "abc") {
		t.Errorf("should not expose raw token: %s", got)
	}
	if !strings.Contains(got, "***") {
		t.Errorf("should mask token: %s", got)
	}
}

// --- helpers ---

func assertStepName(t *testing.T, steps []smokeStep, idx int, name string) {
	t.Helper()
	if steps[idx].Name != name {
		t.Errorf("step[%d] name = %q, want %q", idx, steps[idx].Name, name)
	}
}

func findStep(steps []smokeStep, name string) *smokeStep {
	for i := range steps {
		if steps[i].Name == name {
			return &steps[i]
		}
	}
	return nil
}

// Verify JSON round-trip of smoke steps for structured output.
func TestSmokeStepsJSON(t *testing.T) {
	steps := []smokeStep{
		{Name: "backend reachable", Status: smokePass, Detail: "ok"},
		{Name: "player join", Status: smokeFail, Detail: "connection refused"},
	}
	data, err := json.Marshal(steps)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded []smokeStep
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(decoded))
	}
}
