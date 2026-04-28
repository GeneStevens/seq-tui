package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type smokeServerOpts struct {
	zoneDown        bool
	zoneNotFound    bool
	devAPIDisabled  bool
	tokenMismatch   bool
	playerReadFails bool
	mobsFail        bool
	noMobs          bool
	proxFail        bool
	moveFail        bool
	moveNoProgress  bool
	attackFail      bool
	encounterFail   bool
	noEncounters    bool
	inventoryFail   bool
	mobStartDist    float64
	mobInRange      bool
}

func newSmokeServer(t *testing.T, opts smokeServerOpts) *httptest.Server {
	t.Helper()

	mobDist := opts.mobStartDist
	if mobDist == 0 {
		mobDist = 3.5
	}
	moveCount := 0
	attacked := false

	mux := http.NewServeMux()

	mux.HandleFunc("/world/zone/testzone", func(w http.ResponseWriter, r *http.Request) {
		if opts.zoneNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
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
		inRange := opts.mobInRange || mobDist <= 1.0
		fmt.Fprintf(w, `{"found":true,"within_proximity":%t,"distance_2d":%.1f,"target_mob_name":"a goblin","target_mob_id":"goblin-1"}`, inRange, mobDist)
	})

	mux.HandleFunc("/world/dev/zone/testzone/player/move", func(w http.ResponseWriter, r *http.Request) {
		if opts.moveFail {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		moveCount++
		if opts.moveNoProgress {
			fmt.Fprint(w, `{"ok":true,"x":10,"y":20,"z":0}`)
			return
		}
		newDist := mobDist - 1.0
		if newDist < 0.5 {
			newDist = 0.5
		}
		mobDist = newDist
		newX := 10.0 - float64(moveCount)
		newY := 20.0 - float64(moveCount)
		fmt.Fprintf(w, `{"ok":true,"x":%.0f,"y":%.0f,"z":0}`, newX, newY)
	})

	mux.HandleFunc("/world/dev/zone/testzone/intent", func(w http.ResponseWriter, r *http.Request) {
		if opts.attackFail {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":false,"error":"attack denied"}`)
			return
		}
		attacked = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	})

	mux.HandleFunc("/world/call/testzone", func(w http.ResponseWriter, r *http.Request) {
		msg := r.URL.Query().Get("message")
		w.Header().Set("Content-Type", "application/json")
		switch msg {
		case "encounters":
			if opts.encounterFail {
				http.Error(w, "error", http.StatusInternalServerError)
				return
			}
			if opts.noEncounters || !attacked {
				fmt.Fprint(w, `{"result":[]}`)
				return
			}
			fmt.Fprint(w, `{"result":[{"encounter_id":"enc-1","state":"active","player_ids":["p1"],"mob_ids":["goblin-1"],"mobs_alive_count":1,"mobs_dead_count":0,"action_index":1,"timeline_length":1,"latest_result_kind":"damage_applied","latest_result_actor":"p1","latest_result_target":"goblin-1","latest_result_value":15,"mob_threat":[{"mob_id":"goblin-1","live_mob_id":"live-goblin-1","process_id":"pid-goblin-1","hp_current":85,"hp_max":100}]}]}`)
		case "gameplay_status":
			if opts.inventoryFail {
				http.Error(w, "error", http.StatusInternalServerError)
				return
			}
			fmt.Fprint(w, `{"result":{"items":[],"can_act":true,"blocked_reason":"","hp_current":100,"hp_max":100,"players":[{"player_id":"p1","hp_current":100,"hp_max":100}]}}`)
		default:
			http.Error(w, "unknown message", http.StatusBadRequest)
		}
	})

	return httptest.NewServer(mux)
}

func TestSmokeAllPass(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{mobStartDist: 3.5})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	if len(steps) != 9 {
		for _, s := range steps {
			t.Logf("  %s", s)
		}
		t.Fatalf("expected 9 steps, got %d", len(steps))
	}

	wantNames := []string{
		"backend reachable", "player join", "player visible",
		"mob visible", "target selection", "distance known",
		"movement", "range check", "attack feedback",
	}
	for i, name := range wantNames {
		assertStepName(t, steps, i, name)
	}

	for _, s := range steps {
		if s.Status == smokeFail {
			t.Errorf("step %q: expected PASS or SKIP, got FAIL — %s", s.Name, s.Detail)
		}
	}

	if !strings.Contains(steps[2].Detail, "HP=") {
		t.Errorf("player visible should contain HP: %s", steps[2].Detail)
	}
	if !strings.Contains(steps[4].Detail, "a goblin") {
		t.Errorf("target selection should name mob: %s", steps[4].Detail)
	}
	if !strings.Contains(steps[8].Detail, "damage_applied") {
		t.Errorf("attack feedback should contain event kind: %s", steps[8].Detail)
	}
}

func TestSmokeAlreadyInRange(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{mobInRange: true, mobStartDist: 1.0})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	moveStep := findStep(steps, "movement")
	if moveStep == nil {
		t.Fatal("missing movement step")
	}
	if moveStep.Status != smokeSkip {
		t.Errorf("movement should be SKIP when already in range, got %s", moveStep.Status)
	}

	rangeStep := findStep(steps, "range check")
	if rangeStep == nil {
		t.Fatal("missing range check step")
	}
	if rangeStep.Status != smokePass {
		t.Errorf("range check should PASS when already in range, got %s — %s", rangeStep.Status, rangeStep.Detail)
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

	if len(steps) != 9 {
		t.Fatalf("expected 9 steps, got %d", len(steps))
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

	mobStep := findStep(steps, "mob visible")
	if mobStep == nil {
		t.Fatal("missing mob visible step")
	}
	if mobStep.Status != smokeFail {
		t.Errorf("mob visible should FAIL: %s", mobStep.Detail)
	}

	targetStep := findStep(steps, "target selection")
	if targetStep == nil {
		t.Fatal("missing target selection step")
	}
	if targetStep.Status != smokeSkip {
		t.Errorf("target selection should SKIP when mobs fail: %s", targetStep.Detail)
	}
}

func TestSmokeNoMobs(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{noMobs: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	mobStep := findStep(steps, "mob visible")
	if mobStep == nil {
		t.Fatal("missing mob visible step")
	}
	if mobStep.Status != smokeFail {
		t.Errorf("mob visible should FAIL with 0 mobs: %s", mobStep.Detail)
	}
	if !strings.Contains(mobStep.Detail, "no mobs") {
		t.Errorf("should explain reason: %s", mobStep.Detail)
	}

	targetStep := findStep(steps, "target selection")
	if targetStep == nil {
		t.Fatal("missing target selection step")
	}
	if targetStep.Status != smokeSkip {
		t.Errorf("target selection should SKIP with no mobs: %s", targetStep.Detail)
	}
}

func TestSmokeProximityFails(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{proxFail: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	for _, s := range steps[:3] {
		if s.Status == smokeFail {
			t.Errorf("step %q should not FAIL: %s", s.Name, s.Detail)
		}
	}

	targetStep := findStep(steps, "target selection")
	if targetStep == nil {
		t.Fatal("missing target selection step")
	}
	if targetStep.Status != smokeFail {
		t.Errorf("target selection should FAIL: %s", targetStep.Detail)
	}
}

func TestSmokeMoveFails(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{moveFail: true, mobStartDist: 10.0})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	moveStep := findStep(steps, "movement")
	if moveStep == nil {
		t.Fatal("missing movement step")
	}
	if moveStep.Status != smokeFail {
		t.Errorf("movement should FAIL: %s", moveStep.Detail)
	}

	rangeStep := findStep(steps, "range check")
	if rangeStep == nil {
		t.Fatal("missing range check step")
	}
	if rangeStep.Status != smokeSkip {
		t.Errorf("range check should SKIP when movement fails: %s", rangeStep.Detail)
	}
}

func TestSmokeMoveNoProgress(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{moveNoProgress: true, mobStartDist: 10.0})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	moveStep := findStep(steps, "movement")
	if moveStep == nil {
		t.Fatal("missing movement step")
	}
	if moveStep.Status != smokeFail {
		t.Errorf("movement should FAIL when no progress: %s", moveStep.Detail)
	}
	if !strings.Contains(moveStep.Detail, "did not decrease") {
		t.Errorf("should explain distance didn't decrease: %s", moveStep.Detail)
	}
}

func TestSmokeAttackFails(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{attackFail: true, mobInRange: true, mobStartDist: 1.0})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	atkStep := findStep(steps, "attack feedback")
	if atkStep == nil {
		t.Fatal("missing attack feedback step")
	}
	if atkStep.Status != smokeFail {
		t.Errorf("attack feedback should FAIL: %s", atkStep.Detail)
	}
	if !strings.Contains(atkStep.Detail, "attack denied") {
		t.Errorf("should report attack error: %s", atkStep.Detail)
	}
}

func TestSmokeNoEncounterAfterAttack(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{noEncounters: true, mobInRange: true, mobStartDist: 1.0})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	atkStep := findStep(steps, "attack feedback")
	if atkStep == nil {
		t.Fatal("missing attack feedback step")
	}
	if atkStep.Status != smokeFail {
		t.Errorf("attack feedback should FAIL with no encounters: %s", atkStep.Detail)
	}
	if !strings.Contains(atkStep.Detail, "no encounter") {
		t.Errorf("should explain no encounter: %s", atkStep.Detail)
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

func TestSmokeZoneNotFound(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{zoneNotFound: true})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	steps := runSmoke(target)

	if len(steps) != 9 {
		t.Fatalf("expected 9 steps, got %d", len(steps))
	}
	if steps[0].Status != smokeFail {
		t.Errorf("backend reachable should FAIL on zone not found, got %s", steps[0].Status)
	}
	if !strings.Contains(steps[0].Detail, "not found") {
		t.Errorf("should mention not found: %s", steps[0].Detail)
	}
	if !strings.Contains(steps[0].Detail, "testzone") {
		t.Errorf("should mention zone name: %s", steps[0].Detail)
	}
	for _, s := range steps[1:] {
		if s.Status != smokeSkip {
			t.Errorf("step %q should be SKIP after zone not found, got %s", s.Name, s.Status)
		}
	}
}

func TestClassifyZoneError(t *testing.T) {
	cases := []struct {
		raw, zone, want string
	}{
		{"connection refused", "crushbone", "connection refused (is backend running?)"},
		{"HTTP 404", "crushbone", `zone "crushbone" not found (check SEQ_ZONE)`},
		{"HTTP 503", "crushbone", "backend unavailable (service starting?)"},
		{"timeout", "crushbone", "request timeout"},
		{"deadline exceeded", "crushbone", "request timeout"},
		{"something else", "crushbone", "something else"},
	}
	for _, tc := range cases {
		got := classifyZoneError(tc.raw, tc.zone)
		if got != tc.want {
			t.Errorf("classifyZoneError(%q, %q) = %q, want %q", tc.raw, tc.zone, got, tc.want)
		}
	}
}

func TestClassifyZoneErrorTruncation(t *testing.T) {
	long := strings.Repeat("x", 100)
	got := classifyZoneError(long, "z")
	if len(got) != 80 {
		t.Errorf("expected truncation to 80 chars, got %d", len(got))
	}
}

func TestSmokeCmdAllPass(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{mobStartDist: 3.5})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	var buf bytes.Buffer
	code := runSmokeCmd(target, &buf)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "PASS") {
		t.Errorf("output should contain PASS: %s", out)
	}
	if strings.Contains(out, "[FAIL]") {
		t.Errorf("output should not contain [FAIL]: %s", out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("output should contain separator: %s", out)
	}
}

func TestSmokeCmdFail(t *testing.T) {
	target := backendTarget{
		BaseURL:  "http://127.0.0.1:1",
		Zone:     "testzone",
		Mode:     "RT",
		Player:   "p1",
		DevToken: "test-token",
	}
	var buf bytes.Buffer
	code := runSmokeCmd(target, &buf)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "FAIL") {
		t.Errorf("output should contain FAIL: %s", out)
	}
	if !strings.Contains(out, "connection refused") {
		t.Errorf("output should contain connection refused: %s", out)
	}
}

func TestSmokeCmdShowsConfig(t *testing.T) {
	srv := newSmokeServer(t, smokeServerOpts{mobStartDist: 3.5})
	defer srv.Close()

	target := testTargetWithServer(srv.URL)
	var buf bytes.Buffer
	runSmokeCmd(target, &buf)
	out := buf.String()
	if !strings.Contains(out, "testzone") {
		t.Errorf("output should show zone in config: %s", out)
	}
	if !strings.Contains(out, "***") {
		t.Errorf("output should mask token: %s", out)
	}
}

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

func TestSmokeFailureClarity(t *testing.T) {
	cases := []struct {
		name     string
		opts     smokeServerOpts
		stepName string
		wantFail bool
		wantIn   string
	}{
		{"backend not running", smokeServerOpts{zoneDown: true}, "backend reachable", true, "unavailable"},
		{"dev API disabled", smokeServerOpts{devAPIDisabled: true}, "player join", true, "not found"},
		{"token mismatch", smokeServerOpts{tokenMismatch: true}, "player join", true, "forbidden"},
		{"no mobs", smokeServerOpts{noMobs: true}, "mob visible", true, "no mobs"},
		{"target failure", smokeServerOpts{proxFail: true}, "target selection", true, ""},
		{"movement ineffective", smokeServerOpts{moveNoProgress: true, mobStartDist: 10.0}, "movement", true, "did not decrease"},
		{"attack denied", smokeServerOpts{attackFail: true, mobInRange: true, mobStartDist: 1.0}, "attack feedback", true, "attack denied"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newSmokeServer(t, tc.opts)
			defer srv.Close()

			target := testTargetWithServer(srv.URL)
			steps := runSmoke(target)

			step := findStep(steps, tc.stepName)
			if step == nil {
				for _, s := range steps {
					t.Logf("  %s", s)
				}
				t.Fatalf("missing step %q", tc.stepName)
			}
			if tc.wantFail && step.Status != smokeFail {
				t.Errorf("step %q: expected FAIL, got %s — %s", tc.stepName, step.Status, step.Detail)
			}
			if tc.wantIn != "" && !strings.Contains(step.Detail, tc.wantIn) {
				t.Errorf("step %q detail should contain %q: %s", tc.stepName, tc.wantIn, step.Detail)
			}
		})
	}
}

