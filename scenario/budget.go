package scenario

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// A speed gate that holds up on a busy host (TUI-032, after seq's SEQ-090). The
// engine's helper (testserver.Budget, internal/cpushare) is not importable, so
// this mirrors its rule minimally, on the host's load average alone: the CPU
// spin it also reads under-reads a busy macOS host, and the load tracked it.
//
// The limit grows with the load (TUI-045, ISSUES T8): a run may take limit times
// (1 + load per CPU), so a moderately busy host (the suite took 15-18 s at a
// load of 3.7 on 10 CPUs, limit 15 s, with no test failing) is not failed for
// the host's sake, while on a quiet host the limit is the one asked for and a
// real slowdown fails. On a contended host (load per CPU of Contended or more) a run over its
// limit is a logged warning with the load measured, and only fails past
// Ceiling: a gate that fails on the host's load says nothing about the code.
const (
	// Contended is the load per CPU from which a host counts as shared. It is
	// lower than seq's 1.5: TUI-031's suite failed at 19-23 s (limit 15 s) at a
	// load of 6-15 on 10 CPUs, 0.6-1.5 per CPU, and passed at 12.9 s at 0.7.
	Contended = 0.5
	// HardFactor is how many times its limit a run may take on a contended host
	// before it fails all the same, at the least; past it the ceiling is
	// CeilingPerShare times the load per CPU.
	HardFactor      = 5.0
	CeilingPerShare = 3.0
)

// Allowed is the longest a run with limit may take on a host with load per CPU
// load without a word: the limit, stretched by the load. A run past it is a
// failure on a host below Contended, and past it on a contended one a warning
// up to Ceiling.
func Allowed(limit time.Duration, load float64) time.Duration {
	return time.Duration(float64(limit) * (1 + max(load, 0)))
}

// Ceiling is the longest a run with limit may take on a host with load per CPU
// load before it fails though contended.
func Ceiling(limit time.Duration, load float64) time.Duration {
	return time.Duration(float64(limit) * max(HardFactor, CeilingPerShare*load))
}

// Judge is the verdict on a run that took took against limit on a host with load
// per CPU load: whether it fails, and whether it is a warning only.
func Judge(took, limit time.Duration, load float64) (fail, warn bool) {
	if took <= Allowed(limit, load) {
		return false, false
	}
	if load >= Contended && took <= Ceiling(limit, load) {
		return false, true
	}
	return true, false
}

// LoadPerCPU is the host's one-minute load average per CPU, 0 when it does not
// say (a platform with neither /proc/loadavg nor sysctl).
func LoadPerCPU() float64 {
	var raw string
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		raw = string(b)
	} else if b, err := exec.Command("sysctl", "-n", "vm.loadavg").Output(); err == nil {
		raw = string(b)
	}
	for _, f := range strings.Fields(strings.NewReplacer("{", " ", "}", " ").Replace(raw)) {
		if v, err := strconv.ParseFloat(f, 64); err == nil {
			return v / float64(runtime.NumCPU())
		}
	}
	return 0
}

// Verdict is TooSlow made load-aware: "" if the run took no longer than limit
// or only a warning is due (returned as warning, for the caller to log), else
// the failure.
func Verdict(what string, took, limit time.Duration, load float64) (failure, warning string) {
	fail, warn := Judge(took, limit, load)
	msg := TooSlow(what, took, limit)
	switch {
	case msg == "":
		return "", ""
	case warn:
		return "", fmt.Sprintf("WARNING: %s, over its budget on a host with load %.1f per CPU, so this is not a failure (it fails past %v)", msg, load, Ceiling(limit, load).Round(time.Millisecond))
	case fail:
		return msg, ""
	}
	return "", ""
}

// startLoad is the load when the package began: the load average lags by about
// a minute, so a gate reads the higher of it and the load at the check.
var startLoad = LoadPerCPU()

// gate holds a run to limit: a failure goes to fail, a warning to warn.
func gate(what string, took, limit time.Duration, fail, warn func(string)) {
	failure, warning := Verdict(what, took, limit, max(startLoad, LoadPerCPU()))
	if warning != "" {
		warn(warning)
	}
	if failure != "" {
		fail(failure)
	}
}
