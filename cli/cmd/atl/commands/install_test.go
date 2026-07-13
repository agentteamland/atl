package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/teampkg"
)

func makeSrc(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("team.json", `{"name":"demo","version":"2.0.0","scope":"project"}`)
	write("agents/api/agent.md", "API agent")
	write("skills/build/skill.md", "build skill")
	return src
}

func demoEntry() *index.Entry {
	return &index.Entry{
		Handle: "acme", Name: "demo", Version: "2.0.0", Scope: "project",
		Source: index.Source{Repo: "acme/demo", Subpath: "", Ref: "v2.0.0"},
	}
}

func TestInstallAtProject(t *testing.T) {
	src := makeSrc(t)
	tm, err := teampkg.ReadManifest(src)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := installAt(scope.Project, root, "acme", "demo", demoEntry(), tm, src); err != nil {
		t.Fatalf("installAt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "agents", "api", "agent.md")); err != nil {
		t.Errorf("agent not reflected into .claude: %v", err)
	}
	m, err := manifest.Read(filepath.Join(root, ".atl"), "acme", "demo")
	if err != nil {
		t.Fatalf("manifest.Read: %v", err)
	}
	if m.Version != "2.0.0" || m.Scope != "project" {
		t.Errorf("manifest = %+v", m)
	}
	if m.Files["agents/api/agent.md"] == "" {
		t.Error("manifest files map missing the agent")
	}
}

// TestInstallAtReinstallPreservesEdits proves the v2 idempotency guarantee: a
// second installAt over an already-installed team (a repeat install, or a
// transitive dependency re-pulled) reflects under fan-out discipline and never
// clobbers a locally-grown/edited file.
func TestInstallAtReinstallPreservesEdits(t *testing.T) {
	src := makeSrc(t)
	tm, err := teampkg.ReadManifest(src)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := installAt(scope.Project, root, "acme", "demo", demoEntry(), tm, src); err != nil {
		t.Fatalf("first installAt: %v", err)
	}
	agentPath := filepath.Join(root, ".claude", "agents", "api", "agent.md")
	// The learning loop (or the user) grows the installed file locally.
	if err := os.WriteFile(agentPath, []byte("LOCALLY GROWN"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A re-install (directly or via a transitive dependency) must preserve it.
	if err := installAt(scope.Project, root, "acme", "demo", demoEntry(), tm, src); err != nil {
		t.Fatalf("reinstall installAt: %v", err)
	}
	if b, _ := os.ReadFile(agentPath); string(b) != "LOCALLY GROWN" {
		t.Errorf("reinstall clobbered a local edit: got %q, want LOCALLY GROWN", b)
	}
	m, err := manifest.Read(filepath.Join(root, ".atl"), "acme", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "2.0.0" {
		t.Errorf("manifest version = %q, want 2.0.0", m.Version)
	}
}

func TestInstallAtGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // windows
	src := makeSrc(t)
	tm, _ := teampkg.ReadManifest(src)
	if err := installAt(scope.Global, "/unused", "acme", "demo", demoEntry(), tm, src); err != nil {
		t.Fatalf("installAt global: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "api", "agent.md")); err != nil {
		t.Errorf("global agent not reflected: %v", err)
	}
	if _, err := manifest.Read(filepath.Join(home, ".atl"), "acme", "demo"); err != nil {
		t.Errorf("global manifest not written: %v", err)
	}
}

func TestInstallTargets(t *testing.T) {
	if got := installTargets(scope.Both); len(got) != 2 {
		t.Errorf("Both -> %v, want 2 layers", got)
	}
	if got := installTargets(scope.Project); len(got) != 1 || got[0] != scope.Project {
		t.Errorf("Project -> %v", got)
	}
	if got := installTargets(scope.Global); len(got) != 1 || got[0] != scope.Global {
		t.Errorf("Global -> %v", got)
	}
}

func TestScopeLabel(t *testing.T) {
	if scopeLabel([]scope.Scope{scope.Global, scope.Project}) != "both" {
		t.Error("two targets should label as both")
	}
	if scopeLabel([]scope.Scope{scope.Project}) != "project" {
		t.Error("single project target should label as project")
	}
}

func TestClaudeDirProject(t *testing.T) {
	d, err := scope.ClaudeDir(scope.Project, filepath.FromSlash("/proj"))
	if err != nil || d != filepath.FromSlash("/proj/.claude") {
		t.Errorf("ClaudeDir project = %q, %v", d, err)
	}
}

// makeSrcWith writes a fixture team dir with the given team.json plus one asset.
func makeSrcWith(t *testing.T, teamJSON string) string {
	t.Helper()
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "agents", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "team.json"), []byte(teamJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "agents", "x", "agent.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return src
}

func TestInstallWithDeps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // windows
	root := t.TempDir()

	ix := &index.Index{Teams: []index.Entry{
		{Handle: "acme", Name: "consumer", Version: "1.0.0", Scope: "project",
			Source: index.Source{Repo: "acme/consumer"}},
		{Handle: "agentteamland", Name: "dep-team", Version: "1.1.0", Scope: "global", Verified: true,
			Source: index.Source{Repo: "agentteamland/dep-team"}},
	}}

	// consumer depends on core (skipped) + dep-team; dep-team depends back on
	// consumer (a cycle — must not loop).
	consumerSrc := makeSrcWith(t, `{"name":"consumer","version":"1.0.0","scope":"project","dependencies":{"core":"^1.0.0","dep-team":"^1.0.0"}}`)
	depSrc := makeSrcWith(t, `{"name":"dep-team","version":"1.1.0","scope":"global","dependencies":{"consumer":"^1.0.0"}}`)
	fetch := func(repo, subpath, ref string) (string, error) {
		switch repo {
		case "acme/consumer":
			return consumerSrc, nil
		case "agentteamland/dep-team":
			return depSrc, nil
		}
		return "", fmt.Errorf("unexpected fetch %q", repo)
	}

	consumer, _ := ix.Lookup("acme", "consumer")
	visited := map[string]bool{}
	var installed []installedTeam
	if err := installWithDeps(ix, consumer, scope.NoOverride, root, fetch, visited, &installed, false); err != nil {
		t.Fatalf("installWithDeps: %v", err)
	}

	// consumer + dep pulled transitively; the cycle did not re-install consumer.
	if len(installed) != 2 {
		t.Fatalf("installed %d, want 2 (consumer + dep, cycle-safe): %+v", len(installed), installed)
	}
	byRef := map[string]installedTeam{}
	for _, it := range installed {
		byRef[it.ref] = it
	}
	if c := byRef["acme/consumer"]; c.dep || c.scopeLabel != "project" {
		t.Errorf("consumer = %+v, want project scope + not a dep", c)
	}
	// the dependency installs at ITS OWN scope (global), not the consumer's project.
	if d := byRef["agentteamland/dep-team"]; !d.dep || d.scopeLabel != "global" {
		t.Errorf("dep-team = %+v, want global scope + dep=true", d)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "x", "agent.md")); err != nil {
		t.Errorf("dependency not installed at global scope: %v", err)
	}
	if _, err := manifest.Read(filepath.Join(home, ".atl"), "agentteamland", "dep-team"); err != nil {
		t.Errorf("dependency manifest not written at global: %v", err)
	}
	// "core" is never resolved/installed — it is the platform core.
	if visited["agentteamland/core"] {
		t.Error("core must be skipped, not resolved as a team")
	}
}
