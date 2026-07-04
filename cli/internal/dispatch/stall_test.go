package dispatch

import (
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	cfg := StallConfig{HeartbeatThreshold: 5 * time.Minute, PhaseStallThreshold: 15 * time.Minute}

	cases := []struct {
		name          string
		heartbeatAgo  time.Duration
		phaseEnterAgo time.Duration
		want          Breach
	}{
		{"fresh + young phase", 30 * time.Second, 2 * time.Minute, Alive},
		{"stale heartbeat", 6 * time.Minute, 2 * time.Minute, HeartbeatBreach},
		{"fresh heartbeat but stuck in phase", 30 * time.Second, 16 * time.Minute, PhaseStallBreach},
		{"heartbeat breach dominates phase stall", 6 * time.Minute, 16 * time.Minute, HeartbeatBreach},
		{"exactly at heartbeat threshold is alive", 5 * time.Minute, 2 * time.Minute, Alive},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := &Status{Phase: "implement", HeartbeatTS: now.Add(-c.heartbeatAgo)}
			got := Classify(st, now.Add(-c.phaseEnterAgo), now, cfg)
			if got != c.want {
				t.Errorf("Classify = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDefaultStallConfigMobileWidened(t *testing.T) {
	general := DefaultStallConfig("implement")
	mobile := DefaultStallConfig("mobile-self-test")
	if mobile.HeartbeatThreshold <= general.HeartbeatThreshold {
		t.Errorf("mobile heartbeat threshold (%v) should exceed general (%v)", mobile.HeartbeatThreshold, general.HeartbeatThreshold)
	}
	if mobile.PhaseStallThreshold <= general.PhaseStallThreshold {
		t.Errorf("mobile phase-stall threshold (%v) should exceed general (%v)", mobile.PhaseStallThreshold, general.PhaseStallThreshold)
	}
}

func TestIsMobileTestPhase(t *testing.T) {
	mobile := []string{"mobile-self-test", "run-emulator", "iOS Simulator boot", "sim-test"}
	for _, p := range mobile {
		if !isMobileTestPhase(p) {
			t.Errorf("%q should be a mobile-test phase", p)
		}
	}
	for _, p := range []string{"implement", "web-test", "pr"} {
		if isMobileTestPhase(p) {
			t.Errorf("%q should NOT be a mobile-test phase", p)
		}
	}
}

func TestBreachString(t *testing.T) {
	if HeartbeatBreach.String() != "heartbeat-breach" || PhaseStallBreach.String() != "phase-stall-breach" || Alive.String() != "alive" {
		t.Error("Breach.String mismatch")
	}
}
