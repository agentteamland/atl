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
