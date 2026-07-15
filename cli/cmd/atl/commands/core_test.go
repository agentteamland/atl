package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
)

// autoDrainNotice fires only when the queue is non-empty, and its text tells the
// agent to auto-drain in the background (not the old passive "run /drain").
func TestAutoDrainNotice(t *testing.T) {
	if autoDrainNotice(0) != "" {
		t.Error("an empty queue must produce no auto-drain signal (no false-fire)")
	}
	if autoDrainNotice(-1) != "" {
		t.Error("a negative count must produce no signal")
	}
	msg := autoDrainNotice(3)
	if !strings.Contains(msg, "3 learning") {
		t.Errorf("the signal must carry the count: %q", msg)
	}
	if !strings.Contains(msg, "auto-drain") || !strings.Contains(msg, "background") {
		t.Errorf("the signal must instruct a background auto-drain, not a manual /drain: %q", msg)
	}
	if strings.Contains(msg, "run /drain") {
		t.Errorf("the signal must not be the old passive 'run /drain' wording: %q", msg)
	}
}

// autoProfileDrainNotice is the profile-fact sibling: same background auto-drain
// shape, but its own channel noun and its own action-owning capture rule.
func TestAutoProfileDrainNotice(t *testing.T) {
	if autoProfileDrainNotice(0) != "" {
		t.Error("an empty profile-fact queue must produce no signal (no false-fire)")
	}
	if autoProfileDrainNotice(-1) != "" {
		t.Error("a negative count must produce no signal")
	}
	msg := autoProfileDrainNotice(2)
	if !strings.Contains(msg, "2 profile-fact") {
		t.Errorf("the signal must carry the count and channel: %q", msg)
	}
	if !strings.Contains(msg, "auto-drain") || !strings.Contains(msg, "background") {
		t.Errorf("the signal must instruct a background auto-drain: %q", msg)
	}
	if !strings.Contains(msg, "profile-capture") {
		t.Errorf("the signal must point at the profile-capture rule (its action owner): %q", msg)
	}
	if strings.Contains(msg, "run /profile-drain") {
		t.Errorf("the signal must not be the old passive 'run /profile-drain' wording: %q", msg)
	}
}

// ownedRuleNames must protect both the platform's core rules (global only) and
// any team-installed rule — a user rule that collides with either name must not
// be reflected over installed content.
func TestOwnedRuleNames(t *testing.T) {
	layer := t.TempDir() // stands in for a scope's .atl dir

	// A team installed at this layer owns rules/team-house.md.
	m := &manifest.Manifest{
		Handle: "acme", Name: "team",
		Files: map[string]string{
			"rules/team-house.md":  "sha",
			"agents/api/agent.md":  "sha", // a non-rule asset must not leak in
		},
	}
	if err := m.Write(layer); err != nil {
		t.Fatal(err)
	}

	// Global: core rule names AND the team rule name are protected.
	global, err := ownedRuleNames(scope.Global, layer)
	if err != nil {
		t.Fatal(err)
	}
	if !global["branch-hygiene.md"] {
		t.Error("global scope must protect core rule names (branch-hygiene.md)")
	}
	if !global["team-house.md"] {
		t.Error("global scope must protect a team-installed rule name")
	}
	if global["agent.md"] {
		t.Error("a non-rule manifest asset must not be treated as a protected rule")
	}

	// Project: core rules are global-only, so they must NOT appear; the team rule
	// still does.
	project, err := ownedRuleNames(scope.Project, layer)
	if err != nil {
		t.Fatal(err)
	}
	if project["branch-hygiene.md"] {
		t.Error("project scope must not carry global core rule names")
	}
	if !project["team-house.md"] {
		t.Error("project scope must protect a team-installed rule name")
	}
}

// A layer with no installs (no .atl/installed) yields only the core names at
// global, and an empty set at project — never an error.
func TestOwnedRuleNamesNoInstalls(t *testing.T) {
	layer := t.TempDir()
	global, err := ownedRuleNames(scope.Global, layer)
	if err != nil {
		t.Fatal(err)
	}
	if !global["karpathy-guidelines.md"] {
		t.Error("core names must be present even with no installs")
	}
	project, err := ownedRuleNames(scope.Project, filepath.Join(layer, "nope"))
	if err != nil {
		t.Fatalf("a missing layer must not error: %v", err)
	}
	if len(project) != 0 {
		t.Errorf("project scope with no installs must be empty, got %v", project)
	}
}
