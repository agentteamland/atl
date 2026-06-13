package commands

import (
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
