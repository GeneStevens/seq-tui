package main

import "fmt"

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

func runSmoke(target backendTarget) []smokeStep {
	var steps []smokeStep

	// Step 1: Backend reachable (zone status GET — no dev token required)
	zoneResult := fetchZoneStatus(target)
	if zoneResult.State != zoneReadOK {
		steps = append(steps, smokeStep{
			Name:   "backend reachable",
			Status: smokeFail,
			Detail: classifyJoinError(zoneResult.Error),
		})
		steps = append(steps, smokeStep{Name: "player join", Status: smokeSkip, Detail: "backend unreachable"})
		steps = append(steps, smokeStep{Name: "player state", Status: smokeSkip, Detail: "backend unreachable"})
		steps = append(steps, smokeStep{Name: "mob positions", Status: smokeSkip, Detail: "backend unreachable"})
		steps = append(steps, smokeStep{Name: "target proximity", Status: smokeSkip, Detail: "backend unreachable"})
		return steps
	}
	steps = append(steps, smokeStep{
		Name:   "backend reachable",
		Status: smokePass,
		Detail: fmt.Sprintf("zone=%s process=%s", target.Zone, zoneResult.Summary.ProcessName),
	})

	// Step 2: Join player (requires dev API + valid token)
	joinResult := joinAndReadPlayer(target)
	if joinResult.State != playerReadOK {
		steps = append(steps, smokeStep{
			Name:   "player join",
			Status: smokeFail,
			Detail: joinResult.Error,
		})
		steps = append(steps, smokeStep{Name: "player state", Status: smokeSkip, Detail: "join failed"})
		steps = append(steps, smokeStep{Name: "mob positions", Status: smokeSkip, Detail: "join failed"})
		steps = append(steps, smokeStep{Name: "target proximity", Status: smokeSkip, Detail: "join failed"})
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

	// Step 3: Read player state (refresh path)
	refreshResult := readPlayerState(target)
	if refreshResult.State != playerReadOK {
		steps = append(steps, smokeStep{
			Name:   "player state",
			Status: smokeFail,
			Detail: refreshResult.Error,
		})
	} else {
		stateDetail := "readable"
		if refreshResult.HasPos {
			stateDetail = fmt.Sprintf("pos=(%.0f, %.0f)", refreshResult.Position.X, refreshResult.Position.Y)
		}
		if refreshResult.HasActiveEncounter {
			stateDetail += " encounter=" + refreshResult.ActiveEncounterID
		}
		steps = append(steps, smokeStep{
			Name:   "player state",
			Status: smokePass,
			Detail: stateDetail,
		})
	}

	// Step 4: Mob positions
	mobResult := fetchMobPositions(target)
	if mobResult.State != mobReadOK {
		steps = append(steps, smokeStep{
			Name:   "mob positions",
			Status: smokeFail,
			Detail: mobResult.Error,
		})
		steps = append(steps, smokeStep{Name: "target proximity", Status: smokeSkip, Detail: "mob read failed"})
		return steps
	}
	steps = append(steps, smokeStep{
		Name:   "mob positions",
		Status: smokePass,
		Detail: fmt.Sprintf("%d mobs loaded", mobResult.Count),
	})

	// Step 5: Target proximity (first available mob)
	if mobResult.Count == 0 {
		steps = append(steps, smokeStep{
			Name:   "target proximity",
			Status: smokeSkip,
			Detail: "no mobs in zone",
		})
		return steps
	}
	mob := mobResult.Mobs[0]
	entry := rosterEntry{kind: "mb", id: mob.ProcessID, mobID: mob.MobID}
	proxResult := queryTargetProximity(target, entry)
	if proxResult.State != targetConfirmOK {
		steps = append(steps, smokeStep{
			Name:   "target proximity",
			Status: smokeFail,
			Detail: proxResult.Error,
		})
		return steps
	}
	proxDetail := fmt.Sprintf("mob=%s dist=%.1f", proxResult.MobName, proxResult.Distance)
	if proxResult.WithinProximity {
		proxDetail += " (in range)"
	}
	steps = append(steps, smokeStep{
		Name:   "target proximity",
		Status: smokePass,
		Detail: proxDetail,
	})

	return steps
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
