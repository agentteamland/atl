package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
)

// autoDrainNotice fires only when the channel is non-empty, and its text tells
// the agent to auto-drain in the background (not the old passive "run /drain").
// The wording is byte-identical to the pre-declaration output for core's own
// channel — a signal the rules already tell agents to act on must not shift
// because the channel now arrives as a value.
func TestAutoDrainNotice(t *testing.T) {
	if autoDrainNotice(coreChannel, 0) != "" {
		t.Error("an empty queue must produce no auto-drain signal (no false-fire)")
	}
	if autoDrainNotice(coreChannel, -1) != "" {
		t.Error("a negative count must produce no signal")
	}
	msg := autoDrainNotice(coreChannel, 3)
	want := "atl: 3 learning(s) pending — auto-drain them now in a background subagent (per the learning-capture rule)"
	if msg != want {
		t.Errorf("core-channel signal drifted:\n got %q\nwant %q", msg, want)
	}
	if strings.Contains(msg, "run /drain") {
		t.Errorf("the signal must not be the old passive 'run /drain' wording: %q", msg)
	}
}

// The same function serves a team-declared channel, taking its noun and its
// action-owning rule from the declaration — core emits the signal without ever
// learning which team ships that rule. The text is byte-identical to the
// hardcoded profile-fact signal it replaces.
func TestAutoDrainNoticeForADeclaredChannel(t *testing.T) {
	ch := manifest.Channel{
		Name: "profile-fact", Drain: "/profile-drain",
		Rule: "profile-capture", Describes: "durable entity facts",
	}
	if autoDrainNotice(ch, 0) != "" {
		t.Error("an empty profile-fact queue must produce no signal (no false-fire)")
	}
	msg := autoDrainNotice(ch, 2)
	want := "atl: 2 profile-fact(s) pending — auto-drain them now in a background subagent (per the profile-capture rule)"
	if msg != want {
		t.Errorf("declared-channel signal drifted:\n got %q\nwant %q", msg, want)
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
			"rules/team-house.md": "sha",
			"agents/api/agent.md": "sha", // a non-rule asset must not leak in
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
