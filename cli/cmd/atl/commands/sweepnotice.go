package commands

import (
	"fmt"
	"os"
)

// sweepDispatchBrake opts out of the automatic half of every sweep signal.
//
// It names the DISPATCH rather than the sweep, and that is the whole contract:
// someone who does not want background subagents spawned unasked has not asked
// to stop being told a sweep is due. So the brake drops the instruction and
// keeps the report.
//
// Why it exists at all. Every one of these mechanisms ships to users, and the
// signal's addressee IS its behaviour — an agent reads "run /observe now in a
// background subagent (per the sweep-dispatch rule)" and dispatches, which in
// someone else's project is a different decision from making it in the one that
// authored the rule. *Quality over cost* is this project's maintainer's rule,
// not necessarily its users'. Every other automatic user-facing behaviour here
// already carries a brake (ATL_NO_SELF_UPDATE, ATL_NO_TEAM_UPDATE,
// ATL_NO_RETRIEVE_INDEX, ATL_NO_CAPTURE_WATCHDOG, ATL_NO_STORE_GIT); this one
// shipped without one, which is the gap rather than a deliberate exception.
const sweepDispatchBrake = "ATL_NO_SWEEP_DISPATCH"

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
// Under the brake it returns exactly that passive form again. The fallback is
// therefore a state this project ran in for months rather than a new mode
// invented for the opt-out — the sweep stays discoverable, and whether to run
// it goes back to being a person's call.
//
// skill is the slash command to spawn; what completes "… to <what>".
func sweepNotice(prefix, skill, what string) string {
	if os.Getenv(sweepDispatchBrake) != "" {
		return fmt.Sprintf("%s — run %s to %s", prefix, skill, what)
	}
	return fmt.Sprintf("%s — run %s now in a background subagent to %s (per the sweep-dispatch rule)",
		prefix, skill, what)
}
