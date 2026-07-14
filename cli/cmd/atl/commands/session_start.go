package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/agentteamland/atl/cli/internal/buildinfo"
	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/gc"
	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/selfupdate"
	"github.com/spf13/cobra"
)

var sessionStartCmd = &cobra.Command{
	Use:   "session-start",
	Short: "Session-start maintenance (run by the SessionStart hook)",
	Long: "Run by the SessionStart hook. Drains the previous session's transcripts\n" +
		"into the queue, runs the doctor self-check, and signals any pending\n" +
		"learnings so Claude can fold them into the knowledge base via /drain.\n" +
		"Whatever it prints reaches Claude's context (SessionStart delivers stdout),\n" +
		"so it stays quiet unless there's something worth surfacing.\n\n" +
		"Never fails: a hook must not block the session.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Refresh the platform core (rules + skills) into the global layer — it
		// ships in the binary, so this keeps ~/.claude in lockstep with the binary
		// version. Non-blocking; a hook must never fail.
		if n, _ := reflectCore(); n > 0 {
			fmt.Printf("atl: refreshed %d core file(s)\n", n)
		}
		// Reflect the user's own global rules (authored via `/rule --global` into
		// ~/.atl/rules) into ~/.claude/rules so Claude Code actually loads them.
		// Kept above the queue gate — like reflectCore — so a locked/contended
		// queue can never skip it. Non-blocking; a hook must never fail.
		if n, _ := reflectUserRules(scope.Global, ""); n > 0 {
			fmt.Printf("atl: reflected %d global user rule(s) into the Claude load surface\n", n)
		}

		st, project, err := openQueue()
		if err != nil {
			return nil // non-blocking: never fail a hook
		}

		// Same for project rules (authored via `/rule` into <project>/.atl/rules);
		// this needs the project root that openQueue resolved.
		if project != "" {
			if n, _ := reflectUserRules(scope.Project, project); n > 0 {
				fmt.Printf("atl: reflected %d project rule(s) into the Claude load surface\n", n)
			}
		}

		// Drain the previous session's transcripts (no throttle at session start).
		if _, _, enqueued, _, derr := drainProjectTranscripts(st, project); derr == nil && enqueued > 0 {
			fmt.Printf("atl: captured %d new learning(s) from the previous session\n", enqueued)
		}

		// Doctor self-check + asset integrity restore — surface non-OK / healed.
		checks := append(doctor.QueueChecks(st, project, time.Now()), integrityCheck(project), hooksCheck())
		for _, r := range doctor.Run(checks) {
			if r.Status != doctor.OK || r.Healed {
				fmt.Printf("atl doctor: %s — %s\n", r.Status, r.Detail)
			}
		}

		// Signal pending learnings before releasing the queue lock (below).
		var learningPending, profilePending int
		if counts, cerr := st.Counts(project); cerr == nil {
			learningPending = counts[queue.ChannelLearning]
			profilePending = counts[queue.ChannelProfileFact]
		}

		// Release the queue's exclusive lock before the non-queue scans (gc + the
		// docs/skills/rules signals) so a concurrent session isn't blocked on the
		// 1s open timeout while this one runs unrelated work.
		st.Close()

		// Reclamation awareness — surface only high-signal orphans (gains/edits
		// beside an installed unit), not wholly-unowned dirs (usually the user's own
		// non-ATL Claude Code assets — noise). Awareness only; `atl gc` is the action.
		if orphans, oerr := gc.Scan(project, time.Now()); oerr == nil {
			n := 0
			for _, o := range orphans {
				if o.Owned {
					n++
				}
			}
			if n > 0 {
				fmt.Printf("atl: %d orphaned file(s) beside installed units — run `atl gc` to review (reversible)\n", n)
			}
		}

		// Signal pending learnings so Claude folds them in via /drain (counts read
		// above, before the queue was closed). The skill is LLM work the CLI can't
		// run itself (the CLI/Skill boundary) — surfacing the count here is how it
		// gets triggered without the user remembering to.
		if msg := autoDrainNotice(learningPending); msg != "" {
			fmt.Println(msg)
		}
		// profile-team's channel: only fires when profile-team is installed and a
		// session dropped profile-fact markers. /profile-drain is a team skill; core
		// /drain stays learning-only (its documented boundary).
		if profilePending > 0 {
			fmt.Printf("atl: %d profile-fact(s) pending — run /profile-drain to fold them into the profiles\n", profilePending)
		}

		// Docs-correctness signal — fires only in a repo that has a docs site
		// (monorepo-internal): a deterministic drift count plus a /docs-audit
		// "sweep due" signal. Silent everywhere else.
		docsSessionSignal()

		// Skill/asset content-quality signal — monorepo-internal, same as docs.
		skillsSessionSignal()

		// Rules-distill "distill due" signal — monorepo-internal, same shape.
		rulesSessionSignal()

		// Binary self-update — once/24h, check for a newer stable release and, if
		// there is one, spawn a detached `atl upgrade` so the download + swap runs
		// independently and the next session runs the new binary. Bounded (short
		// timeout) + throttled + never-fail.
		sctx, scancel := context.WithTimeout(context.Background(), 3*time.Second)
		if notice := selfupdate.AutoApply(sctx, buildinfo.Version); notice != "" {
			fmt.Println(notice)
		}
		scancel()

		return nil
	},
}
