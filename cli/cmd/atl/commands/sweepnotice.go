package commands

import "fmt"

// sweepNotice builds the session-start signal for a periodic sweep that is due.
//
// One helper for all four sweeps, for the same reason autoDrainNotice is one
// helper for every channel: these strings are a contract with an agent, not
// decoration, and four independently-authored copies drift. The wording is
// deliberately parallel to the auto-drain signal — it names the action, not a
// slash command, and it cites the rule that owns the response — because the
// addressee is what decides who acts on a signal. The sweeps read as "run /X"
// for months, addressed to a human, and went unrun while the drains, addressed
// to the agent, ran unprompted.
//
// skill is the slash command to spawn; what completes "… to <what>".
func sweepNotice(prefix, skill, what string) string {
	return fmt.Sprintf("%s — run %s now in a background subagent to %s (per the sweep-dispatch rule)",
		prefix, skill, what)
}
