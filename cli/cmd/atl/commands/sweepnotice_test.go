package commands

import (
	"strings"
	"testing"
)

// The sweep signal is a contract with an agent. These assertions pin the two
// properties that decide whether it gets acted on at all — both learned the
// expensive way, from four sweeps that sat unrun for months while the
// structurally identical auto-drain signal ran unprompted every time.
func TestSweepNoticeAddressesTheAgent(t *testing.T) {
	msg := sweepNotice("atl: a proactive observer sweep is due", "/observe", "surface ripe backlog triggers and latent gaps")

	want := "atl: a proactive observer sweep is due — run /observe now in a background subagent to surface ripe backlog triggers and latent gaps (per the sweep-dispatch rule)"
	if msg != want {
		t.Errorf("signal drifted:\n got %q\nwant %q", msg, want)
	}

	// It must name an ACTION, not merely a slash command. "run /observe" alone is
	// addressed to a human reading a terminal; an agent needs to be told what to
	// do with it, which is the whole difference between the sweeps and the drains.
	if !strings.Contains(msg, "background subagent") {
		t.Errorf("the signal must name the action (a background subagent), not just the command: %q", msg)
	}

	// It must cite the rule that owns the response. Without the citation the
	// instruction lives only inside the skill the signal is telling you to run —
	// reachable only by an agent that already obeyed, which is why these four
	// were unreachable in practice.
	if !strings.Contains(msg, "sweep-dispatch rule") {
		t.Errorf("the signal must cite the rule that owns the response: %q", msg)
	}
}

// Every sweep uses the one helper, so a later edit cannot make one of them drift
// back to the passive register while the other three stay correct.
func TestEverySweepUsesTheSameShape(t *testing.T) {
	for _, c := range []struct{ prefix, skill, what string }{
		{"atl: a proactive observer sweep is due", "/observe", "surface ripe backlog triggers and latent gaps"},
		{"atl docs: a full audit is due", "/docs-audit", "sweep the docs site for semantic drift"},
		{"atl skills: a stocktake is due", "/skill-stocktake", "sweep skills for obedience and redundancy"},
		{"atl rules: a distill is due", "/rules-distill", "mine recurring principles into core rules"},
	} {
		msg := sweepNotice(c.prefix, c.skill, c.what)
		if !strings.HasPrefix(msg, c.prefix+" — run "+c.skill+" now in a background subagent") {
			t.Errorf("%s: unexpected shape: %q", c.skill, msg)
		}
		if !strings.HasSuffix(msg, "(per the sweep-dispatch rule)") {
			t.Errorf("%s: missing the rule citation: %q", c.skill, msg)
		}
	}
}

// The brake removes the dispatch and keeps the report. Both halves are asserted
// because either one alone is a different, wrong feature: dropping the signal
// entirely would silence a cheap git-log report nobody asked to lose, and
// keeping the dispatch words would make the opt-out inert while reading as
// present — the shape this repo has recorded as worse than no control at all.
func TestSweepNoticeBrakeDropsTheDispatchAndKeepsTheReport(t *testing.T) {
	t.Setenv("ATL_NO_SWEEP_DISPATCH", "1")

	msg := sweepNotice("atl: a proactive observer sweep is due", "/observe", "surface ripe backlog triggers and latent gaps")

	// The dispatch half is gone — nothing here tells an agent to spawn anything.
	if strings.Contains(msg, "background subagent") {
		t.Errorf("the brake must drop the dispatch instruction: %q", msg)
	}
	if strings.Contains(msg, "sweep-dispatch rule") {
		t.Errorf("the brake must drop the rule citation that makes it binding: %q", msg)
	}

	// The report half survives — which sweep, and how to run it by hand.
	if !strings.Contains(msg, "atl: a proactive observer sweep is due") {
		t.Errorf("the brake must keep the report of WHICH sweep is due: %q", msg)
	}
	if !strings.Contains(msg, "/observe") {
		t.Errorf("the brake must keep the command so it stays runnable by hand: %q", msg)
	}
}

// An unset brake must behave exactly as before. Pinned separately from the
// assertions above so that a change to the brake's plumbing cannot quietly
// alter the default for every user who never sets it.
func TestSweepNoticeUnbrakedIsUnchanged(t *testing.T) {
	t.Setenv("ATL_NO_SWEEP_DISPATCH", "")

	want := "atl: a proactive observer sweep is due — run /observe now in a background subagent to surface ripe backlog triggers and latent gaps (per the sweep-dispatch rule)"
	if got := sweepNotice("atl: a proactive observer sweep is due", "/observe", "surface ripe backlog triggers and latent gaps"); got != want {
		t.Errorf("default signal drifted:\n got %q\nwant %q", got, want)
	}
}

// Every sweep honours the brake, not just the one that motivated it. A brake
// that covered /observe alone would leave three mechanisms dispatching in a
// project whose owner had already said no — and the opt-out would still read as
// present, which is the failure it exists to prevent.
func TestBrakeCoversEverySweep(t *testing.T) {
	t.Setenv("ATL_NO_SWEEP_DISPATCH", "1")

	for _, c := range []struct{ prefix, skill, what string }{
		{"atl: a proactive observer sweep is due", "/observe", "surface ripe backlog triggers and latent gaps"},
		{"atl docs: a full audit is due", "/docs-audit", "sweep the docs site for semantic drift"},
		{"atl skills: a stocktake is due", "/skill-stocktake", "sweep skills for obedience and redundancy"},
		{"atl rules: a distill is due", "/rules-distill", "mine recurring principles into core rules"},
	} {
		msg := sweepNotice(c.prefix, c.skill, c.what)
		if strings.Contains(msg, "background subagent") || strings.Contains(msg, "sweep-dispatch rule") {
			t.Errorf("%s: still dispatches under the brake: %q", c.skill, msg)
		}
		if !strings.Contains(msg, c.skill) {
			t.Errorf("%s: lost its command under the brake: %q", c.skill, msg)
		}
	}
}
