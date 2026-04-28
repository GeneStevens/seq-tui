package main

import (
	"fmt"
	"io"
	"strings"
)

type smokeStatus int

const (
	smokePass smokeStatus = iota
	smokeFail
	smokeSkip
)

func (s smokeStatus) String() string {
	switch s {
	case smokePass:
		return "PASS"
	case smokeFail:
		return "FAIL"
	case smokeSkip:
		return "SKIP"
	default:
		return "UNKNOWN"
	}
}

type smokeStep struct {
	Name   string      `json:"name"`
	Status smokeStatus `json:"status"`
	Detail string      `json:"detail,omitempty"`
}

func (s smokeStep) String() string {
	if s.Detail != "" {
		return fmt.Sprintf("[%s] %s — %s", s.Status, s.Name, s.Detail)
	}
	return fmt.Sprintf("[%s] %s", s.Status, s.Name)
}

const smokeMaxMoves = 100

func runSmoke(target backendTarget) []smokeStep {
	var steps []smokeStep

	skipRemaining := func(reason string, from int) {
		names := []string{
			"backend reachable", "player join", "player visible",
			"mob visible", "target selection", "distance known",
			"movement", "range check", "attack feedback",
		}
		for i := from; i < len(names); i++ {
			steps = append(steps, smokeStep{Name: names[i], Status: smokeSkip, Detail: reason})
		}
	}

	// Step 1: Backend reachable
	zoneResult := fetchZoneStatus(target)
	if zoneResult.State != zoneReadOK {
		steps = append(steps, smokeStep{
			Name:   "backend reachable",
			Status: smokeFail,
			Detail: classifyZoneError(zoneResult.Error, target.Zone),
		})
		skipRemaining("backend unreachable", 1)
		return steps
	}
	steps = append(steps, smokeStep{
		Name:   "backend reachable",
		Status: smokePass,
		Detail: fmt.Sprintf("zone=%s process=%s", target.Zone, zoneResult.Summary.ProcessName),
	})

	// Step 2: Player join
	joinResult := joinAndReadPlayer(target)
	if joinResult.State != playerReadOK {
		steps = append(steps, smokeStep{
			Name:   "player join",
			Status: smokeFail,
			Detail: joinResult.Error,
		})
		skipRemaining("join failed", 2)
		return steps
	}
	posDetail := "joined (no position)"
	if joinResult.HasPos {
		posDetail = fmt.Sprintf("joined at (%.0f, %.0f)", joinResult.Position.X, joinResult.Position.Y)
	}
	steps = append(steps, smokeStep{
		Name:   "player join",
		Status: smokePass,
		Detail: posDetail,
	})

	// Step 3: Player visible (position + HP)
	invResult := fetchPlayerInventory(target)
	if !joinResult.HasPos {
		steps = append(steps, smokeStep{
			Name:   "player visible",
			Status: smokeFail,
			Detail: "no position data",
		})
	} else if invResult.State != inventoryReadOK || !invResult.HasLifecycle {
		steps = append(steps, smokeStep{
			Name:   "player visible",
			Status: smokeFail,
			Detail: "position ok but HP unavailable",
		})
	} else {
		steps = append(steps, smokeStep{
			Name:   "player visible",
			Status: smokePass,
			Detail: fmt.Sprintf("pos=(%.0f, %.0f) HP=%d/%d",
				joinResult.Position.X, joinResult.Position.Y,
				invResult.HPCurrent, invResult.HPMax),
		})
	}

	// Step 4: Mob visible
	mobResult := fetchMobPositions(target)
	if mobResult.State != mobReadOK {
		steps = append(steps, smokeStep{
			Name:   "mob visible",
			Status: smokeFail,
			Detail: mobResult.Error,
		})
		skipRemaining("mob read failed", 4)
		return steps
	}
	if mobResult.Count == 0 {
		steps = append(steps, smokeStep{
			Name:   "mob visible",
			Status: smokeFail,
			Detail: "no mobs in zone",
		})
		skipRemaining("no mobs", 4)
		return steps
	}
	steps = append(steps, smokeStep{
		Name:   "mob visible",
		Status: smokePass,
		Detail: fmt.Sprintf("%d mobs loaded", mobResult.Count),
	})

	// Step 5: Target selection
	mob := mobResult.Mobs[0]
	entry := rosterEntry{kind: "mb", id: mob.ProcessID, mobID: mob.MobID}
	proxResult := queryTargetProximity(target, entry)
	if proxResult.State != targetConfirmOK {
		steps = append(steps, smokeStep{
			Name:   "target selection",
			Status: smokeFail,
			Detail: proxResult.Error,
		})
		skipRemaining("target failed", 5)
		return steps
	}
	steps = append(steps, smokeStep{
		Name:   "target selection",
		Status: smokePass,
		Detail: fmt.Sprintf("mob=%s", proxResult.MobName),
	})

	// Step 6: Distance known
	steps = append(steps, smokeStep{
		Name:   "distance known",
		Status: smokePass,
		Detail: fmt.Sprintf("dist=%.1f", proxResult.Distance),
	})

	// Steps 7-8: Movement and range
	if proxResult.WithinProximity {
		steps = append(steps, smokeStep{
			Name:   "movement",
			Status: smokeSkip,
			Detail: "already in range",
		})
		steps = append(steps, smokeStep{
			Name:   "range check",
			Status: smokePass,
			Detail: fmt.Sprintf("already in range dist=%.1f", proxResult.Distance),
		})
	} else {
		moveSteps := smokeMovementLoop(target, entry, joinResult.Position, mob.Position, proxResult.Distance)
		steps = append(steps, moveSteps...)
		if smokeHasFailed(moveSteps) {
			steps = append(steps, smokeStep{Name: "attack feedback", Status: smokeSkip, Detail: "not in range"})
			return steps
		}
	}

	// Step 9: Attack feedback
	steps = append(steps, smokeAttackFeedback(target, mob, invResult)...)

	return steps
}

func smokeMovementLoop(target backendTarget, entry rosterEntry, playerPos playerPosResult, mobPos mobPosVec3, initialDist float64) []smokeStep {
	var steps []smokeStep

	dir := directionTowardMob(playerPos, mobPos)
	if dir == "" {
		steps = append(steps, smokeStep{
			Name:   "movement",
			Status: smokeFail,
			Detail: "at mob position but not in range",
		})
		steps = append(steps, smokeStep{Name: "range check", Status: smokeSkip, Detail: "movement failed"})
		return steps
	}

	moveRes := submitDirectionalMove(target, dir)
	if !moveRes.OK {
		steps = append(steps, smokeStep{
			Name:   "movement",
			Status: smokeFail,
			Detail: moveRes.Error,
		})
		steps = append(steps, smokeStep{Name: "range check", Status: smokeSkip, Detail: "movement failed"})
		return steps
	}

	postProx := queryTargetProximity(target, entry)
	if postProx.State != targetConfirmOK {
		steps = append(steps, smokeStep{
			Name:   "movement",
			Status: smokeFail,
			Detail: "proximity query failed after move",
		})
		steps = append(steps, smokeStep{Name: "range check", Status: smokeSkip, Detail: "movement failed"})
		return steps
	}

	if postProx.Distance >= initialDist {
		steps = append(steps, smokeStep{
			Name:   "movement",
			Status: smokeFail,
			Detail: fmt.Sprintf("distance did not decrease: %.1f → %.1f", initialDist, postProx.Distance),
		})
		steps = append(steps, smokeStep{Name: "range check", Status: smokeSkip, Detail: "movement ineffective"})
		return steps
	}

	steps = append(steps, smokeStep{
		Name:   "movement",
		Status: smokePass,
		Detail: fmt.Sprintf("dist %.1f → %.1f", initialDist, postProx.Distance),
	})

	if postProx.WithinProximity {
		steps = append(steps, smokeStep{
			Name:   "range check",
			Status: smokePass,
			Detail: fmt.Sprintf("in range dist=%.1f", postProx.Distance),
		})
		return steps
	}

	currentPos := moveRes.Position
	currentDist := postProx.Distance
	for range smokeMaxMoves {
		dir = directionTowardMob(currentPos, mobPos)
		if dir == "" {
			break
		}
		mr := submitDirectionalMove(target, dir)
		if !mr.OK {
			break
		}
		if mr.HasPos {
			currentPos = mr.Position
		}
		pr := queryTargetProximity(target, entry)
		if pr.State != targetConfirmOK {
			break
		}
		currentDist = pr.Distance
		if pr.WithinProximity {
			steps = append(steps, smokeStep{
				Name:   "range check",
				Status: smokePass,
				Detail: fmt.Sprintf("in range dist=%.1f", currentDist),
			})
			return steps
		}
	}

	steps = append(steps, smokeStep{
		Name:   "range check",
		Status: smokeFail,
		Detail: fmt.Sprintf("not in range after moves dist=%.1f", currentDist),
	})
	return steps
}

func smokeAttackFeedback(target backendTarget, mob mobPosition, preInv inventoryReadResult) []smokeStep {
	preEncounters := fetchZoneEncounters(target)
	preMobHP := -1
	if preEncounters.State == encounterReadOK {
		for _, enc := range preEncounters.Encounters {
			for _, mt := range enc.MobThreat {
				if mt.ProcessID == mob.ProcessID || mt.LiveMobID == mob.LiveMobID {
					preMobHP = mt.HPCurrent
				}
			}
		}
	}

	attackTarget := mob.LiveMobID
	if attackTarget == "" {
		attackTarget = mob.ProcessID
	}
	atkResult := submitBasicAttack(target, attackTarget)
	if atkResult.State != attackStateSent {
		return []smokeStep{{
			Name:   "attack feedback",
			Status: smokeFail,
			Detail: fmt.Sprintf("attack failed: %s", atkResult.Error),
		}}
	}

	postEncounters := fetchZoneEncounters(target)
	if postEncounters.State != encounterReadOK {
		return []smokeStep{{
			Name:   "attack feedback",
			Status: smokeFail,
			Detail: "encounter read failed after attack",
		}}
	}
	if postEncounters.Count == 0 {
		return []smokeStep{{
			Name:   "attack feedback",
			Status: smokeFail,
			Detail: "no encounter after attack",
		}}
	}

	var foundEvent bool
	var foundHP bool
	var eventDetail string
	for _, enc := range postEncounters.Encounters {
		if enc.LatestResultKind != "" {
			foundEvent = true
			eventDetail = fmt.Sprintf("event=%s value=%d", enc.LatestResultKind, enc.LatestResultValue)
		}
		for _, mt := range enc.MobThreat {
			if mt.ProcessID == mob.ProcessID || mt.LiveMobID == mob.LiveMobID {
				if preMobHP >= 0 && mt.HPCurrent != preMobHP {
					foundHP = true
				} else if preMobHP < 0 && mt.HPCurrent < mt.HPMax {
					foundHP = true
				}
			}
		}
	}

	if !foundHP && preInv.State == inventoryReadOK && preInv.HasLifecycle {
		postInv := fetchPlayerInventory(target)
		if postInv.State == inventoryReadOK && postInv.HasLifecycle && postInv.HPCurrent != preInv.HPCurrent {
			foundHP = true
		}
	}

	if foundEvent && foundHP {
		return []smokeStep{{
			Name:   "attack feedback",
			Status: smokePass,
			Detail: eventDetail,
		}}
	}
	if foundEvent {
		return []smokeStep{{
			Name:   "attack feedback",
			Status: smokeFail,
			Detail: "event found but no HP change — " + eventDetail,
		}}
	}
	return []smokeStep{{
		Name:   "attack feedback",
		Status: smokeFail,
		Detail: "attack sent but no event in encounter",
	}}
}

func smokeConfigSummary(target backendTarget) string {
	masked := "***"
	if target.DevToken == "" {
		masked = "(empty)"
	}
	return fmt.Sprintf("url=%s zone=%s player=%s token=%s", target.BaseURL, target.Zone, target.Player, masked)
}

func smokeHasFailed(steps []smokeStep) bool {
	for _, s := range steps {
		if s.Status == smokeFail {
			return true
		}
	}
	return false
}

func classifyZoneError(raw, zone string) string {
	if strings.Contains(raw, "connection refused") {
		return "connection refused (is backend running?)"
	}
	if strings.Contains(raw, "timeout") || strings.Contains(raw, "deadline exceeded") {
		return "request timeout"
	}
	if raw == "HTTP 404" {
		return fmt.Sprintf("zone %q not found (check SEQ_ZONE)", zone)
	}
	if raw == "HTTP 503" {
		return "backend unavailable (service starting?)"
	}
	if len(raw) > 80 {
		return raw[:80]
	}
	return raw
}

func runSmokeCmd(target backendTarget, w io.Writer) int {
	fmt.Fprintln(w, smokeConfigSummary(target))
	fmt.Fprintln(w, "---")
	steps := runSmoke(target)
	for _, s := range steps {
		fmt.Fprintln(w, s)
	}
	fmt.Fprintln(w, "---")
	if smokeHasFailed(steps) {
		fmt.Fprintln(w, "FAIL")
		return 1
	}
	fmt.Fprintln(w, "PASS")
	return 0
}
