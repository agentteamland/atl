package dispatch

import (
	"strings"
	"time"
)

// Breach is the supervisor's liveness verdict for a worker (#12). A worker is
// alive only while its status.json shows BOTH a fresh heartbeat AND forward
// phase progress; a breach of either triggers the reclaim ladder.
type Breach int

const (
	Alive Breach = iota
	// HeartbeatBreach — no heartbeat within the (phase-aware) threshold; the
	// worker is likely dead or wedged.
	HeartbeatBreach
	// PhaseStallBreach — heartbeat is fresh but the phase has not advanced in a
	// much longer window; the worker is looping in-place (e.g. an unbreakable
	// self-fix loop) that keeps ticking without progress.
	PhaseStallBreach
)

func (b Breach) String() string {
	switch b {
	case HeartbeatBreach:
		return "heartbeat-breach"
	case PhaseStallBreach:
		return "phase-stall-breach"
	default:
		return "alive"
	}
}

// StallConfig holds the two (phase-aware) stall thresholds for a worker.
type StallConfig struct {
	// HeartbeatThreshold — how long since the last heartbeat before the worker
	// is considered dead. Generous, always measured from the last observed
	// heartbeat (never a fixed-sleep liveness assumption).
	HeartbeatThreshold time.Duration
	// PhaseStallThreshold — how long a worker may sit in the same phase (a fresh
	// heartbeat notwithstanding) before it counts as stuck; ~3x a phase's
	// expected budget.
	PhaseStallThreshold time.Duration
}

// Classify is the pure liveness decision (#12) over a worker's current status,
// the time the supervisor first observed the current phase, and the clock —
// `now` and `phaseEnteredAt` are injected so this is a deterministic,
// zero-IO function (the gc.Scan pattern). Heartbeat breach takes precedence: a
// dead worker is not making phase progress either.
func Classify(status *Status, phaseEnteredAt, now time.Time, cfg StallConfig) Breach {
	if now.Sub(status.HeartbeatTS) > cfg.HeartbeatThreshold {
		return HeartbeatBreach
	}
	if now.Sub(phaseEnteredAt) > cfg.PhaseStallThreshold {
		return PhaseStallBreach
	}
	return Alive
}

// DefaultStallConfig returns the phase-aware thresholds (#12): a generous
// ~5 min heartbeat / ~15 min phase-stall in general, widened to ~10 min /
// ~30 min for a mobile-emulator test phase where an iOS sim boot legitimately
// takes 30-90s+. The phase heuristic keys off the worker's phase string; it can
// be tightened once the delivery-team's worker phase names are finalized.
func DefaultStallConfig(phase string) StallConfig {
	if isMobileTestPhase(phase) {
		return StallConfig{HeartbeatThreshold: 10 * time.Minute, PhaseStallThreshold: 30 * time.Minute}
	}
	return StallConfig{HeartbeatThreshold: 5 * time.Minute, PhaseStallThreshold: 15 * time.Minute}
}

func isMobileTestPhase(phase string) bool {
	p := strings.ToLower(phase)
	return strings.Contains(p, "mobile") ||
		strings.Contains(p, "emulator") ||
		strings.Contains(p, "simulator") ||
		strings.Contains(p, "sim-test")
}
