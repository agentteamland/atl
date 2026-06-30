// Package guard implements `atl guard`, the PreToolUse enforcement hook.
//
// It promotes ATL's load-bearing prose discipline to deterministic enforcement
// (the enforcement-hooks Lane 1 decision). Two layers, split by reversibility:
//
//   - Catastrophe layer (DENY): a fixed list of irreversible Bash operations
//     (force-push, reset --hard, force-clean, destructive SQL, gate-bypass) is
//     blocked outright — welcome even mid-autonomous-flow, since it's the safety
//     net for the riskiest commands. `rm -rf /` and `rm -rf ~` are intentionally
//     NOT here: Claude Code blocks those itself even in bypass mode, so guard
//     targets only the gaps.
//   - Quality layer (non-blocking): the first edit of an existing file in a
//     session injects a grep-before-edit reminder as additionalContext — with no
//     permission decision, so it never interrupts the flow or overrides the
//     user's own permission settings.
//
// Decide is pure (the filesystem and per-session state are injected), so the
// policy is unit-tested without touching disk; the command layer is a thin
// stdin -> Decide -> stdout wrapper.
package guard

import (
	"regexp"
	"strings"
)

// Input is the subset of the Claude Code PreToolUse hook payload guard reads.
type Input struct {
	SessionID string    `json:"session_id"`
	ToolName  string    `json:"tool_name"`
	ToolInput ToolInput `json:"tool_input"`
}

// ToolInput carries the per-tool fields guard inspects: the Bash command, or the
// edited file path for Edit / MultiEdit / Write.
type ToolInput struct {
	Command  string `json:"command"`
	FilePath string `json:"file_path"`
}

// Action is what guard tells Claude Code to do for a tool call.
type Action string

const (
	// Noop emits nothing; the normal permission flow applies.
	Noop Action = ""
	// Deny blocks the tool call (catastrophe layer).
	Deny Action = "deny"
	// Context allows the call normally but injects additionalContext (quality layer).
	Context Action = "context"
)

// Result is guard's decision for one tool call.
type Result struct {
	Action Action
	Reason string // the deny reason, or the additionalContext text
}

// NudgeText is the grep-before-edit reminder injected on a file's first edit of
// the session. It reinforces existing discipline (grep-before-edit +
// surgical-change) at the action point — it does not introduce a new rule.
const NudgeText = "atl guard — first edit of this file this session: before changing it, grep for the " +
	"identifiers you're touching to see every caller and importer the change reaches, keep the edit " +
	"surgical (touch only what the task needs), and verify the blast radius rather than assuming it. " +
	"(grep-before-edit · surgical-change)"

// Decide routes a PreToolUse Input to a Result. fileExists reports whether a path
// exists on disk; firstEdit reports (and records) whether this is the first edit
// of a path in the session. Both are injected so the policy is testable without
// disk or per-session state.
func Decide(in Input, fileExists func(string) bool, firstEdit func(path string) bool) Result {
	switch in.ToolName {
	case "Bash":
		if reason, blocked := Catastrophe(in.ToolInput.Command); blocked {
			return Result{Action: Deny, Reason: reason}
		}
	case "Edit", "MultiEdit", "Write":
		p := in.ToolInput.FilePath
		if p == "" {
			return Result{}
		}
		if !fileExists(p) {
			// New-file creation (Write) or a path that isn't there: nothing to
			// grep, so the nudge doesn't apply.
			return Result{}
		}
		if firstEdit(p) {
			return Result{Action: Context, Reason: NudgeText}
		}
	}
	return Result{}
}

// --- Catastrophe layer -------------------------------------------------------

// catastrophe is one irreversible-operation rule for the Bash layer.
type catastrophe struct {
	name   string
	match  func(cmd string) bool
	reason string
}

// Catastrophe reports whether a Bash command is an irreversible operation that
// must be blocked, and the reason to show Claude. Pure; the primary unit-test
// target.
//
// Each rule is evaluated per shell SEGMENT (split on | ; & and newlines), not
// over the whole command, so a flag belonging to one command in a chain never
// leaks into another's decision — e.g. the -f in `git push origin main && rm -f
// x` must not be read as a force-push, and the -n in `git clean -fd && make -n`
// must not disarm the force-clean. Within a segment there are no separators, so
// the flags a rule sees belong to that rule's subcommand.
func Catastrophe(command string) (reason string, blocked bool) {
	for _, seg := range reSegment.Split(command, -1) {
		for _, c := range catastrophes {
			if c.match(seg) {
				return c.reason, true
			}
		}
	}
	return "", false
}

var (
	// reSegment splits a command line into shell segments at pipes, command
	// separators, and newlines.
	reSegment   = regexp.MustCompile(`[|;&\n]+`)
	reGit       = regexp.MustCompile(`\bgit\b`)
	reGitPush   = regexp.MustCompile(`\bgit\b.*\bpush\b`)
	reGitReset  = regexp.MustCompile(`\bgit\b.*\breset\b`)
	reGitClean  = regexp.MustCompile(`\bgit\b.*\bclean\b`)
	reResetHard = regexp.MustCompile(`--hard\b`)
	// reShortF matches a short flag cluster containing f: -f, -rf, -fd, -Rf, ...
	reShortF = regexp.MustCompile(`(?:^|\s)-[a-zA-Z]*f`)
	// reRefspecForce matches a +-prefixed refspec in push args — Git's force form
	// (`git push origin +main`), equivalent to --force.
	reRefspecForce = regexp.MustCompile(`\bpush\b.*\s\+[\w./]`)
	// reCleanDryRun matches git-clean's dry-run flag: --dry-run, or a short cluster
	// built from clean's own flag letters where one is n (so an unrelated dash-led
	// token like -enode does not count as a dry-run).
	reCleanDryRun = regexp.MustCompile(`(?:^|\s)(?:--dry-run\b|-[dfiqxX]*n[dfiqxX]*(?:\s|$))`)
	// reDropSQL matches an executed destructive SQL statement (word-bounded so
	// `tablet` / `truncate file` don't trip it).
	reDropSQL = regexp.MustCompile(`(?i)\b(?:drop\s+table\b|drop\s+database\b|truncate\s+table\b)`)
	// reSQLClient gates destructive-SQL detection to an actual client invocation,
	// so prose that merely names DROP TABLE (a commit message, a grep) is not blocked.
	reSQLClient = regexp.MustCompile(`\b(?:psql|mysql|mariadb|sqlite3|sqlplus|cockroach|clickhouse-client|mongosh|sqlcmd)\b`)
)

var catastrophes = []catastrophe{
	{
		name:  "force-push",
		match: isForcePush,
		reason: "atl guard — `git push --force` rewrites remote history irreversibly and can clobber " +
			"others' commits. If you must force, use `--force-with-lease`; otherwise reconsider. " +
			"(Catastrophe layer: irreversible.)",
	},
	{
		name:  "reset-hard",
		match: isResetHard,
		reason: "atl guard — `git reset --hard` permanently discards all uncommitted changes in the " +
			"working tree. Stash or commit anything you might need first. (Catastrophe layer: irreversible.)",
	},
	{
		name:  "force-clean",
		match: isForceClean,
		reason: "atl guard — `git clean -f` permanently deletes untracked files (no trash, no history). " +
			"Preview with `git clean -n` first. (Catastrophe layer: irreversible.)",
	},
	{
		name:  "destructive-sql",
		match: isDestructiveSQL,
		reason: "atl guard — a destructive SQL statement (DROP / TRUNCATE) was detected; it irreversibly " +
			"destroys data and schema. Confirm the target database and intent before running it. " +
			"(Catastrophe layer: irreversible.)",
	},
	{
		name:  "no-verify",
		match: isNoVerify,
		reason: "atl guard — `--no-verify` skips the commit/push gate (formatters, linters, tests, hooks). " +
			"Fix what the gate would catch instead of bypassing it. (Catastrophe layer: don't weaken the gate.)",
	},
}

// The match functions below each receive a single shell segment (see Catastrophe).

func isForcePush(seg string) bool {
	if !reGitPush.MatchString(seg) {
		return false
	}
	if strings.Contains(seg, "--force-with-lease") {
		return false // the safe, non-destructive variant
	}
	return strings.Contains(seg, "--force") || reShortF.MatchString(seg) || reRefspecForce.MatchString(seg)
}

func isResetHard(seg string) bool {
	return reGitReset.MatchString(seg) && reResetHard.MatchString(seg)
}

func isForceClean(seg string) bool {
	if !reGitClean.MatchString(seg) {
		return false
	}
	if reCleanDryRun.MatchString(seg) {
		return false // `git clean -n` / --dry-run only previews; it deletes nothing
	}
	return strings.Contains(seg, "--force") || reShortF.MatchString(seg)
}

func isDestructiveSQL(seg string) bool {
	return reSQLClient.MatchString(seg) && reDropSQL.MatchString(seg)
}

func isNoVerify(seg string) bool {
	return strings.Contains(seg, "--no-verify") && reGit.MatchString(seg)
}
