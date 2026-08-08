package skillcheck

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// scriptTeam lays out a team dir declaring one session script. rel=="" writes no
// script file, which is the "declares a file it does not ship" case.
func scriptTeam(t *testing.T, decl, rel string, mode os.FileMode) string {
	t.Helper()
	teams := t.TempDir()
	base := filepath.Join(teams, "demo-team")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	tj := `{"name":"demo-team","capabilities":{"delivery":{"sessionScript":"` + decl + `"}}}`
	if err := os.WriteFile(filepath.Join(base, "team.json"), []byte(tj), 0o644); err != nil {
		t.Fatal(err)
	}
	if rel != "" {
		path := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho hi\n"), mode); err != nil {
			t.Fatal(err)
		}
	}
	return teams
}

func TestSessionScriptsAcceptsARunnableDeclaration(t *testing.T) {
	teams := scriptTeam(t, "scripts/brief.sh", "scripts/brief.sh", 0o755)
	if f := SessionScripts(teams); len(f) != 0 {
		t.Errorf("SessionScripts() = %v, want clean", f)
	}
}

// The failure this gate exists for. At runtime every one of these is SILENT — the
// session simply never gets a briefing, indistinguishable from a script that had
// nothing to report — so if CI does not catch it, nothing does until a user runs
// `atl doctor` on a hunch.
func TestSessionScriptsFailsOnADeclarationThatCanNeverRun(t *testing.T) {
	cases := map[string]struct {
		decl, rel string
		mode      os.FileMode
		wantSub   string
	}{
		"names a file the team does not ship": {
			decl: "scripts/brief.sh", rel: "", mode: 0o755,
			wantSub: "ships no such file",
		},
		"ships it without the exec bit": {
			decl: "scripts/brief.sh", rel: "scripts/brief.sh", mode: 0o644,
			wantSub: "not executable",
		},
		"escapes the team's assets": {
			decl: "../../../etc/payload.sh", rel: "", mode: 0o755,
			wantSub: "not a path inside the team's assets",
		},
		"is absolute": {
			decl: "/etc/payload.sh", rel: "", mode: 0o755,
			wantSub: "not a path inside the team's assets",
		},
		"is empty": {
			decl: "", rel: "", mode: 0o755,
			wantSub: "not a path inside the team's assets",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.mode == 0o644 && runtime.GOOS == "windows" {
				t.Skip("windows has no exec bit to drop")
			}
			f := SessionScripts(scriptTeam(t, tc.decl, tc.rel, tc.mode))
			if len(f) != 1 {
				t.Fatalf("SessionScripts() = %v, want exactly one finding", f)
			}
			if f[0].Severity != Fail {
				t.Errorf("severity = %v, want Fail — this breaks the CI gate on purpose", f[0].Severity)
			}
			if f[0].Check != "session-script" {
				t.Errorf("check = %q, want session-script", f[0].Check)
			}
			if !strings.Contains(f[0].Detail, tc.wantSub) {
				t.Errorf("detail = %q, want it to mention %q", f[0].Detail, tc.wantSub)
			}
		})
	}
}

// A team that declares nothing — every team but one — must produce no finding,
// including when it declares OTHER capabilities in other shapes.
func TestSessionScriptsIsSilentWithoutADeclaration(t *testing.T) {
	for name, tj := range map[string]string{
		"no capabilities":          `{"name":"demo-team"}`,
		"a bare-string capability": `{"name":"demo-team","capabilities":{"review":"tech-lead"}}`,
		"a store capability":       `{"name":"demo-team","capabilities":{"profile":{"store":"~/.atl/profiles"}}}`,
	} {
		teams := t.TempDir()
		base := filepath.Join(teams, "demo-team")
		if err := os.MkdirAll(base, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "team.json"), []byte(tj), 0o644); err != nil {
			t.Fatal(err)
		}
		if f := SessionScripts(teams); len(f) != 0 {
			t.Errorf("%s: SessionScripts() = %v, want clean", name, f)
		}
	}
}

// The gate has to be WIRED, not merely written: RunAll is what CI calls, and a
// check absent from it compiles clean, tests clean, and guards nothing.
func TestRunAllIncludesTheSessionScriptCheck(t *testing.T) {
	teams := scriptTeam(t, "scripts/brief.sh", "", 0o755)
	found := false
	for _, f := range RunAll(Input{TeamsDir: teams}) {
		if f.Check == "session-script" {
			found = true
		}
	}
	if !found {
		t.Error("RunAll did not report the broken session-script declaration — the check is not wired in")
	}
}

// Every shipped team, through the real gate: a declaration and its file must not
// have drifted apart.
func TestTheShippedTeamsPassTheSessionScriptGate(t *testing.T) {
	if f := SessionScripts("../../../teams"); len(f) != 0 {
		t.Errorf("SessionScripts(teams/) = %v, want clean", f)
	}
}
