package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/teampkg"
)

const (
	// envNoSessionScript turns the whole mechanism off.
	//
	// The declaration is accepted without a per-session confirm gate — that was
	// decided, with the concern named: this is automatic execution of installed
	// team content. An env brake is not that gate. It is the same opt-out every
	// other automatic user-facing behaviour here already carries
	// (ATL_NO_SELF_UPDATE, ATL_NO_TEAM_UPDATE, ATL_NO_RETRIEVE_INDEX,
	// ATL_NO_CAPTURE_WATCHDOG, ATL_NO_STORE_GIT, ATL_NO_SWEEP_DISPATCH),
	// and running a team's shell is a LARGER automatic
	// behaviour than any of them, so shipping it without one would be the
	// inconsistency rather than the caution.
	envNoSessionScript = "ATL_NO_SESSION_SCRIPT"

	// sessionScriptBudget bounds the WHOLE pass, not one script.
	//
	// Bounding the pass rather than each script is what keeps the promise that
	// matters: session start blocks the session, so the cost has to be a property
	// of the mechanism and not of how many teams happen to declare a script.
	//
	// 10s rather than the 3s the binary self-update uses, because the two are not
	// the same kind of work. A missed self-update check costs nothing and retries
	// next session; a briefing truncated mid-read is the entire output. The
	// expected cost is near-zero anyway — a script is supposed to decide it has
	// nothing to say from local state and exit, so the network is only reached in
	// the situation the briefing exists for.
	sessionScriptBudget = 10 * time.Second

	// sessionScriptWaitDelay bounds cmd.Wait() after the context expires. Without
	// it a script that leaves a grandchild holding the stdout pipe (a backgrounded
	// `gh`, say) hangs Wait past the deadline and the budget above is a fiction —
	// the same hazard internal/dispatch and internal/storegit already guard.
	sessionScriptWaitDelay = time.Second

	// sessionScriptMaxOutput caps what one script may put into the session's
	// context. A briefing is a few lines; anything approaching this is a
	// misbehaving script, and the head of the output is the useful part, so the
	// excess is dropped rather than the whole thing.
	sessionScriptMaxOutput = 8 << 10
)

// sessionScript is one runnable declaration: where it came from, and what to run.
type sessionScript struct {
	Team string // "<handle>/<name>" — for a doctor detail, never for a decision
	Rel  string // the declared path, relative to the .claude tree
	Path string // the resolved absolute path
}

// runDeclaredSessionScripts runs every session script an installed team declared
// and forwards each one's stdout into the session's context.
//
// This is the third "any team that declares X" contract, after
// capabilities.<name>.store and .channel. Core runs the script and prints what it
// says; it learns neither which team declared it nor what the output is about.
// A script reads whatever project state it cares about itself — which is the
// point: a new consumer needs no change here.
//
// A script speaks only by SUCCEEDING. Missing, not executable, non-zero exit,
// timed out, empty output — all resolve to silence, because this runs inside a
// hook and a hook must never block or fail. The cost of that contract is that a
// broken script is indistinguishable from one that had nothing to report, which
// is exactly why the failure is reported somewhere a person can see it on
// purpose: sessionScriptsCheck, under `atl doctor`.
func runDeclaredSessionScripts(projectRoot string) {
	if os.Getenv(envNoSessionScript) != "" {
		return
	}
	scripts, _ := collectSessionScripts(projectRoot)
	if len(scripts) == 0 {
		return
	}
	// Skip git worktrees, for the reason autoIndexRetrieval skips them: a
	// declaration is a COMMITTED file, so every session opened in any worktree of
	// one repo would land here. That is wrong in both directions — N sessions would
	// each run the script (and each script's network reads), and the output would
	// print into contexts with nobody sitting in them. A session briefing belongs
	// in the session a person is sitting in.
	// The git exec runs only after the cheap manifest reads above, matching the
	// ordering those two callers use.
	if projectRoot != "" && inGitWorktree(projectRoot) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sessionScriptBudget)
	defer cancel()
	for _, s := range scripts {
		if out := runSessionScript(ctx, s, projectRoot); out != "" {
			fmt.Println(out)
		}
	}
}

// runSessionScript runs one script and returns the text to forward, or "".
func runSessionScript(ctx context.Context, s sessionScript, dir string) string {
	cmd := exec.CommandContext(ctx, s.Path)
	cmd.Dir = dir // the script reads project state (its own config, the branch)
	cmd.WaitDelay = sessionScriptWaitDelay
	out, err := cmd.Output()
	if err != nil {
		return "" // non-zero exit, spawn failure, or the budget ran out — all silence
	}
	if len(out) > sessionScriptMaxOutput {
		out = out[:sessionScriptMaxOutput]
	}
	return strings.TrimRight(string(out), "\n")
}

// collectSessionScripts returns the runnable declared scripts, plus the reasons a
// declaration was refused — one pass, so the run path and the doctor check can
// never disagree about which scripts are active.
//
// Layer order is global then project, and a declaration already claimed is
// skipped — first wins, exactly as collectChannels claims a channel name. The key
// is the DECLARED path rather than the resolved one, which covers both ways a
// duplicate arises: two teams naming `scripts/brief.sh` resolve to one shared
// .claude/scripts/brief.sh (same file, obviously once), and one team installed at
// BOTH layers resolves to two copies of its own script (different files, still one
// briefing). Keying on the resolved path would catch only the first.
//
// Never fails: a layer this pass cannot read contributes no scripts and one
// problem line.
func collectSessionScripts(projectRoot string) ([]sessionScript, []string) {
	var active []sessionScript
	var problems []string
	seen := map[string]bool{}

	layers := []scope.Scope{scope.Global}
	if projectRoot != "" {
		// Without a resolved project root, scope.LayerDir(Project, "") returns a
		// RELATIVE ".atl" and would read whatever sits under the current directory —
		// the same refusal collectChannels makes.
		layers = append(layers, scope.Project)
	}
	for _, sc := range layers {
		claudeDir, err := scope.ClaudeDir(sc, projectRoot)
		if err != nil {
			problems = append(problems, fmt.Sprintf(
				"could not resolve the %s layer (%v); a session script declared there does not run", sc, err))
			continue
		}
		layerDir, err := scope.LayerDir(sc, projectRoot)
		if err != nil {
			problems = append(problems, fmt.Sprintf(
				"could not resolve the %s layer (%v); a session script declared there does not run", sc, err))
			continue
		}
		manifests, err := manifest.List(layerDir)
		if err != nil {
			// "Cannot tell" is a third outcome, not "nothing declared here". Saying so
			// keeps the check below from reporting every script healthy while a whole
			// layer went unexamined.
			problems = append(problems, fmt.Sprintf(
				"could not read the installed teams at the %s layer (%v); a session script declared there does not run", sc, err))
			continue
		}
		for _, m := range manifests {
			for _, decl := range m.SessionScripts {
				rel, ok := teampkg.SessionScriptRel(decl)
				if !ok {
					problems = append(problems, fmt.Sprintf(
						"team %s declares session script %q, which is not a path inside the team's assets; it does not run",
						m.Name, decl))
					continue
				}
				if seen[rel] {
					continue // already claimed by an earlier layer or team — first wins
				}
				path := filepath.Join(claudeDir, filepath.FromSlash(rel))
				if why := sessionScriptUnrunnable(path); why != "" {
					problems = append(problems, fmt.Sprintf(
						"team %s declares session script %q but %s; it does not run", m.Name, rel, why))
					continue
				}
				seen[rel] = true
				active = append(active, sessionScript{Team: m.Handle + "/" + m.Name, Rel: rel, Path: path})
			}
		}
	}
	return active, problems
}

// sessionScriptUnrunnable reports why path cannot be executed, or "" when it can.
//
// The executable bit is checked rather than assumed because losing it is the
// realistic way this breaks: CopyFile preserves the source mode on purpose, so a
// script committed without +x reflects without +x and then fails at exec — where
// the failure is, by contract, silent.
func sessionScriptUnrunnable(path string) string {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "there is no such file in the installed tree"
	}
	if err != nil {
		return "it cannot be read (" + err.Error() + ")"
	}
	if info.IsDir() {
		return "it is a directory"
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "it is not executable"
	}
	return ""
}

// sessionScriptsCheck reports session scripts an installed team declared that
// cannot run: a path escaping the team's assets, a file missing from the
// installed tree, one that lost its executable bit, or a layer the pass could not
// read.
//
// It exists because the run path is silent by contract. Every one of those is
// otherwise indistinguishable from a script that ran and had nothing to say — the
// failure mode this whole design chose, and the one thing it therefore owes a
// diagnostic somewhere a person looks on purpose.
//
// Warn, never Fail: a broken third-party declaration must not make `atl doctor`
// exit non-zero.
func sessionScriptsCheck(projectRoot string) doctor.Check {
	return func() doctor.Result {
		_, problems := collectSessionScripts(projectRoot)
		if len(problems) == 0 {
			return doctor.Result{Name: "session-scripts", Status: doctor.OK,
				Detail: "every declared session script can run"}
		}
		return doctor.Result{Name: "session-scripts", Status: doctor.Warn,
			Detail: strings.Join(problems, "; ")}
	}
}
