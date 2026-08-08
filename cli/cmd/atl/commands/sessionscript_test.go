package commands

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/manifest"
)

// These tests lean on the READ side deliberately.
//
// The decision half of this mechanism fails loudly — a script that would not run
// simply produces no briefing, and the absence is visible the moment you look for
// it. The read half does not: collectSessionScripts returning nothing is
// byte-identical, at the session's output, to a script that ran and had nothing to
// report. So a broken layer walk, a path resolved against the wrong directory, or
// a manifest field silently dropped would all present as "the delivery team is
// quiet today", forever, on every machine. Pinning what the read RESOLVES, and
// pinning that each refusal is REPORTED, is the only thing standing between that
// failure and permanent invisibility.

// writeInstalledTeam writes a manifest at a scope layer and, when body is non-empty,
// the script it declares into that scope's .claude tree — the real two-place
// arrangement install produces (team.json's declaration in the manifest, the
// asset reflected into .claude).
func writeInstalledTeam(t *testing.T, root, name string, decls []string, rel, body string, mode os.FileMode) {
	t.Helper()
	m := &manifest.Manifest{
		SchemaVersion:  manifest.SchemaVersion,
		Handle:         "acme",
		Name:           name,
		Version:        "1.0.0",
		Scope:          "project",
		Files:          map[string]string{"skills/x/SKILL.md": "hash"},
		SessionScripts: decls,
	}
	if err := m.Write(filepath.Join(root, ".atl")); err != nil {
		t.Fatal(err)
	}
	if rel == "" {
		return
	}
	path := filepath.Join(root, ".claude", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

// isolatedHome points the GLOBAL layer at an empty temp dir, so a test only ever
// sees what it wrote. Without it the developer's own installed teams leak into
// every assertion here and the results depend on whose machine runs them.
func isolatedHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", t.TempDir())
	}
}

const briefScript = "#!/bin/sh\necho 'atl delivery: on delivery/s1/7'\n"

func TestCollectSessionScriptsResolvesADeclarationIntoTheClaudeTree(t *testing.T) {
	isolatedHome(t)
	root := t.TempDir()
	writeInstalledTeam(t, root, "acme-team", []string{"scripts/brief.sh"}, "scripts/brief.sh", briefScript, 0o755)

	got, problems := collectSessionScripts(root)
	if len(got) != 1 {
		t.Fatalf("collectSessionScripts() = %v (problems %v), want exactly one script", got, problems)
	}
	if want := filepath.Join(root, ".claude", "scripts", "brief.sh"); got[0].Path != want {
		t.Errorf("resolved path = %q, want %q", got[0].Path, want)
	}
	if got[0].Team != "acme/acme-team" {
		t.Errorf("Team = %q, want acme/acme-team", got[0].Team)
	}
	if len(problems) != 0 {
		t.Errorf("problems = %v, want none", problems)
	}
}

// Each of these makes the script un-runnable, and the run path treats all of them
// as silence. If the collect pass ALSO stayed silent, the failure would have no
// surface anywhere — so every one must arrive as a reported problem.
func TestCollectSessionScriptsReportsEveryRefusalRatherThanSwallowingIt(t *testing.T) {
	cases := map[string]struct {
		decls   []string
		rel     string
		mode    os.FileMode
		wantSub string
	}{
		"declared but never installed": {
			decls: []string{"scripts/brief.sh"}, rel: "", mode: 0o755,
			wantSub: "no such file",
		},
		"installed without the exec bit": {
			decls: []string{"scripts/brief.sh"}, rel: "scripts/brief.sh", mode: 0o644,
			wantSub: "not executable",
		},
		"a path escaping the team's assets": {
			decls: []string{"../../../etc/payload.sh"}, rel: "", mode: 0o755,
			wantSub: "not a path inside the team's assets",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.mode == 0o644 && runtime.GOOS == "windows" {
				t.Skip("windows has no exec bit to drop")
			}
			isolatedHome(t)
			root := t.TempDir()
			writeInstalledTeam(t, root, "acme-team", tc.decls, tc.rel, briefScript, tc.mode)

			got, problems := collectSessionScripts(root)
			if len(got) != 0 {
				t.Errorf("collectSessionScripts() returned %v, want nothing runnable", got)
			}
			if len(problems) != 1 || !strings.Contains(problems[0], tc.wantSub) {
				t.Fatalf("problems = %v, want one mentioning %q", problems, tc.wantSub)
			}
			// A refusal must name the team, or the report cannot be acted on.
			if !strings.Contains(problems[0], "acme-team") {
				t.Errorf("problem %q does not name the declaring team", problems[0])
			}
		})
	}
}

// Every team's scripts/ reflects into ONE shared .claude/scripts/, so two teams
// declaring the same relative path name the same file — running it twice would
// print the same briefing twice.
func TestCollectSessionScriptsRunsAResolvedPathOnlyOnce(t *testing.T) {
	isolatedHome(t)
	root := t.TempDir()
	writeInstalledTeam(t, root, "team-a", []string{"scripts/brief.sh"}, "scripts/brief.sh", briefScript, 0o755)
	writeInstalledTeam(t, root, "team-b", []string{"scripts/brief.sh"}, "", "", 0o755)

	got, _ := collectSessionScripts(root)
	if len(got) != 1 {
		t.Errorf("collectSessionScripts() = %v, want the shared path claimed once", got)
	}
}

// The other way a duplicate arises, and the one keying on the RESOLVED path would
// miss: one team installed at BOTH layers has two copies of its own script at two
// different paths. That is still one team asking for one briefing.
func TestCollectSessionScriptsClaimsADeclarationOnceAcrossLayers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	root := t.TempDir()
	writeInstalledTeam(t, home, "acme-team", []string{"scripts/brief.sh"}, "scripts/brief.sh", briefScript, 0o755)
	writeInstalledTeam(t, root, "acme-team", []string{"scripts/brief.sh"}, "scripts/brief.sh", briefScript, 0o755)

	got, problems := collectSessionScripts(root)
	if len(got) != 1 {
		t.Fatalf("collectSessionScripts() = %v, want one briefing, not one per layer", got)
	}
	// Global is walked first, so its copy is the one claimed — the same first-wins
	// order collectChannels uses for a channel name declared at both layers.
	if want := filepath.Join(home, ".claude", "scripts", "brief.sh"); got[0].Path != want {
		t.Errorf("claimed %q, want the global layer's copy %q", got[0].Path, want)
	}
	// A shadowed duplicate is not a PROBLEM — both copies are the same team asking
	// for the same thing, so reporting it would train the user to ignore the check.
	if len(problems) != 0 {
		t.Errorf("problems = %v, want none", problems)
	}
}

// The overwhelmingly common case — no team declares one — must cost nothing and
// say nothing, including from the doctor check.
func TestNoDeclarationIsSilentEverywhere(t *testing.T) {
	isolatedHome(t)
	root := t.TempDir()
	writeInstalledTeam(t, root, "quiet-team", nil, "", "", 0o755)

	got, problems := collectSessionScripts(root)
	if len(got) != 0 || len(problems) != 0 {
		t.Errorf("collectSessionScripts() = %v, %v; want both empty", got, problems)
	}
	if r := sessionScriptsCheck(root)(); r.Status != doctor.OK {
		t.Errorf("doctor status = %v (%s), want OK", r.Status, r.Detail)
	}
}

// Without a resolved project root the project layer must not be read at all:
// scope.LayerDir(Project, "") returns a RELATIVE ".atl", which would execute
// whatever happens to sit under the current working directory.
func TestCollectSessionScriptsSkipsTheProjectLayerWithNoRoot(t *testing.T) {
	isolatedHome(t)
	root := t.TempDir()
	writeInstalledTeam(t, root, "acme-team", []string{"scripts/brief.sh"}, "scripts/brief.sh", briefScript, 0o755)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if got, _ := collectSessionScripts(""); len(got) != 0 {
		t.Errorf("collectSessionScripts(\"\") = %v, want nothing — the project layer is unresolvable", got)
	}
}

// The doctor check is the ONLY surface on which a broken declaration is visible,
// because the run path is silent by contract. Warn, never Fail: a third-party
// team's mistake must not make `atl doctor` exit non-zero.
func TestSessionScriptsCheckWarnsRatherThanFails(t *testing.T) {
	isolatedHome(t)
	root := t.TempDir()
	writeInstalledTeam(t, root, "acme-team", []string{"scripts/gone.sh"}, "", "", 0o755)

	r := sessionScriptsCheck(root)()
	if r.Status != doctor.Warn {
		t.Errorf("status = %v, want Warn", r.Status)
	}
	if !strings.Contains(r.Detail, "gone.sh") {
		t.Errorf("detail = %q, want it to name the missing script", r.Detail)
	}
}

func TestRunSessionScriptForwardsStdout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture is a POSIX shell script")
	}
	root := t.TempDir()
	s := writeScript(t, root, "#!/bin/sh\necho hello\necho world\n", 0o755)

	ctx, cancel := context.WithTimeout(context.Background(), sessionScriptBudget)
	defer cancel()
	if got := runSessionScript(ctx, s, root); got != "hello\nworld" {
		t.Errorf("runSessionScript() = %q, want %q", got, "hello\nworld")
	}
}

// A script speaks only by SUCCEEDING. A non-zero exit throws away whatever it
// printed — otherwise a half-finished read reaches the session's context as if it
// were a finished one, and the reader cannot tell.
func TestRunSessionScriptDropsOutputOnANonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture is a POSIX shell script")
	}
	root := t.TempDir()
	s := writeScript(t, root, "#!/bin/sh\necho 'half an answer'\nexit 3\n", 0o755)

	ctx, cancel := context.WithTimeout(context.Background(), sessionScriptBudget)
	defer cancel()
	if got := runSessionScript(ctx, s, root); got != "" {
		t.Errorf("runSessionScript() = %q, want \"\" — a failed script must not speak", got)
	}
}

// Stderr must not reach the session. A script's diagnostics are for its author,
// and a hook's output is context an agent reads as fact.
func TestRunSessionScriptDoesNotForwardStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture is a POSIX shell script")
	}
	root := t.TempDir()
	s := writeScript(t, root, "#!/bin/sh\necho 'warning: could not reach the board' >&2\necho ok\n", 0o755)

	ctx, cancel := context.WithTimeout(context.Background(), sessionScriptBudget)
	defer cancel()
	if got := runSessionScript(ctx, s, root); got != "ok" {
		t.Errorf("runSessionScript() = %q, want %q", got, "ok")
	}
}

// The budget is what keeps a hook from blocking the session. A hung script has to
// resolve to silence within it, not to a stalled session start.
func TestRunSessionScriptIsBoundedByTheContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture is a POSIX shell script")
	}
	root := t.TempDir()
	s := writeScript(t, root, "#!/bin/sh\nsleep 30\necho 'too late'\n", 0o755)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	got := runSessionScript(ctx, s, root)
	elapsed := time.Since(start)

	if got != "" {
		t.Errorf("runSessionScript() = %q, want \"\" on a timeout", got)
	}
	// Generous, but far under the 30s the script asked for: the assertion is that
	// the deadline is enforced at all, not how fast the machine is.
	if elapsed > 10*time.Second {
		t.Errorf("took %v — the context deadline is not bounding the run", elapsed)
	}
}

// A runaway script must not flood the session's context.
func TestRunSessionScriptCapsItsOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture is a POSIX shell script")
	}
	root := t.TempDir()
	s := writeScript(t, root, "#!/bin/sh\nawk 'BEGIN{while(i++<100000) print \"flood\"}'\n", 0o755)

	ctx, cancel := context.WithTimeout(context.Background(), sessionScriptBudget)
	defer cancel()
	got := runSessionScript(ctx, s, root)
	if len(got) > sessionScriptMaxOutput {
		t.Errorf("output was %d bytes, want it capped at %d", len(got), sessionScriptMaxOutput)
	}
	if got == "" {
		t.Error("the cap dropped everything — the head of the output is the useful part")
	}
}

// The script reads project state (.delivery/config.json, the current branch), so
// it has to start in the project root rather than wherever the session's hook
// happened to be invoked from.
func TestRunSessionScriptRunsInTheProjectRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture is a POSIX shell script")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "marker"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := writeScript(t, root, "#!/bin/sh\n[ -f marker ] && echo found\n", 0o755)

	ctx, cancel := context.WithTimeout(context.Background(), sessionScriptBudget)
	defer cancel()
	if got := runSessionScript(ctx, s, root); got != "found" {
		t.Errorf("runSessionScript() = %q, want %q — cmd.Dir is not the project root", got, "found")
	}
}

// The brake is the whole opt-out for a mechanism that executes installed team
// content automatically. If it stopped working, nothing would say so.
func TestTheBrakeStopsThePassEntirely(t *testing.T) {
	isolatedHome(t)
	root := t.TempDir()
	sentinel := filepath.Join(root, "ran")
	writeInstalledTeam(t, root, "acme-team", []string{"scripts/brief.sh"}, "scripts/brief.sh",
		"#!/bin/sh\ntouch '"+sentinel+"'\n", 0o755)
	t.Setenv(envNoSessionScript, "1")

	runDeclaredSessionScripts(root)
	if _, err := os.Stat(sentinel); err == nil {
		t.Errorf("%s was set and the script ran anyway", envNoSessionScript)
	}
}

func writeScript(t *testing.T, dir, body string, mode os.FileMode) sessionScript {
	t.Helper()
	path := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	return sessionScript{Team: "acme/demo", Rel: "scripts/script.sh", Path: path}
}
