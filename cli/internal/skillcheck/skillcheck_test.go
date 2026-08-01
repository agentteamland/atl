package skillcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// validTeam lays out a well-formed team and returns the teams dir.
func validTeam(t *testing.T) string {
	root := t.TempDir()
	teams := filepath.Join(root, "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","agents":[{"name":"api"}],"skills":[{"name":"ship"}]}`)
	write(t, filepath.Join(base, "agents/api/agent.md"), "---\nname: api\ndescription: \"the api agent\"\n---\n# API\n")
	write(t, filepath.Join(base, "agents/api/children/topic.md"), "---\nknowledge-base-summary: \"a summary\"\n---\n# Topic\n")
	write(t, filepath.Join(base, "skills/ship/SKILL.md"), "---\nname: ship\ndescription: \"ship it\"\n---\n# Ship\n")
	return teams
}

func TestCleanTeamHasNoFindings(t *testing.T) {
	teams := validTeam(t)
	f := RunAll(Input{TeamsDir: teams})
	if len(f) != 0 {
		t.Fatalf("clean team should yield no findings, got %+v", f)
	}
}

func TestSkillFileAcceptsLowercaseSkillMd(t *testing.T) {
	// Core skills use SKILL.md; team skills use skill.md. Both must pass — and
	// case-sensitively (this once broke on Linux CI while passing on macOS).
	root := t.TempDir()
	teams := filepath.Join(root, "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","skills":[{"name":"ship"}]}`)
	write(t, filepath.Join(base, "skills/ship/skill.md"), "---\nname: ship\ndescription: \"ship it\"\n---\n")

	if f := RunAll(Input{TeamsDir: teams}); len(f) != 0 {
		t.Fatalf("a lowercase skill.md should be accepted, got %+v", f)
	}
}

func TestMissingFrontmatterFields(t *testing.T) {
	root := t.TempDir()
	teams := filepath.Join(root, "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","agents":[{"name":"api"}]}`)
	// agent.md with a frontmatter block but no description
	write(t, filepath.Join(base, "agents/api/agent.md"), "---\nname: api\n---\n# API\n")

	f := Frontmatter("", teams)
	found := false
	for _, x := range f {
		if x.Check == "frontmatter" && x.Detail == "agent frontmatter is missing `description`" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a missing-description finding, got %+v", f)
	}
}

func TestManifestDiskMismatchBothDirections(t *testing.T) {
	root := t.TempDir()
	teams := filepath.Join(root, "teams")
	base := filepath.Join(teams, "demo")
	// team.json declares agent "ghost" (no dir); disk has agent "rogue" (not declared)
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","agents":[{"name":"ghost"}]}`)
	write(t, filepath.Join(base, "agents/rogue/agent.md"), "---\nname: rogue\ndescription: \"x\"\n---\n")

	f := TeamManifest(teams)
	var declaredMissing, diskUndeclared bool
	for _, x := range f {
		if x.Check == "manifest" {
			if strings.Contains(x.Detail, "no agents/ghost dir") {
				declaredMissing = true
			}
			if strings.Contains(x.Detail, "not declared in team.json") {
				diskUndeclared = true
			}
		}
	}
	if !declaredMissing || !diskUndeclared {
		t.Fatalf("both directions should be flagged; got %+v", f)
	}
}

func TestChildMissingSummary(t *testing.T) {
	root := t.TempDir()
	teams := filepath.Join(root, "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "agents/api/children/bad.md"), "# no frontmatter here\n")

	f := Children(teams)
	if len(f) != 1 || f[0].Check != "children" {
		t.Fatalf("expected one children finding, got %+v", f)
	}
}

// The installed layer is the surface /drain actually writes to, and the one the
// teams/ walk structurally cannot see.
func TestInstalledChildMissingSummary(t *testing.T) {
	claude := filepath.Join(t.TempDir(), ".claude")
	write(t, filepath.Join(claude, "agents/api/agent.md"), "---\nname: api\ndescription: \"x\"\n---\n")
	write(t, filepath.Join(claude, "agents/api/children/good.md"), "---\nknowledge-base-summary: \"a summary\"\n---\n# Good\n")
	write(t, filepath.Join(claude, "agents/api/children/bad.md"), "# no frontmatter here\n")

	f := InstalledChildren(claude)
	if len(f) != 1 || f[0].Path != "agents/api/children/bad.md" {
		t.Fatalf("expected only bad.md flagged, got %+v", f)
	}
	// Warn, never Fail: this layer is invisible to CI, so it must not gate it.
	if f[0].Severity != Warn {
		t.Fatalf("installed-layer findings must be Warn, got %q", f[0].Severity)
	}
}

// A children/ dir with no agent.md beside it is not an ATL agent knowledge base —
// the false-positive guard that keeps the walk off unrelated content.
func TestInstalledChildrenSkipsDirWithoutAgentMd(t *testing.T) {
	claude := filepath.Join(t.TempDir(), ".claude")
	write(t, filepath.Join(claude, "agents/notanagent/children/bad.md"), "# no frontmatter here\n")

	if f := InstalledChildren(claude); len(f) != 0 {
		t.Fatalf("a children/ dir without agent.md must be ignored, got %+v", f)
	}
}

// The installed layer must stay out of the CI gate — RunAll walks the repo only.
func TestRunAllIgnoresInstalledLayer(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, ".claude")
	write(t, filepath.Join(claude, "agents/api/agent.md"), "---\nname: api\ndescription: \"x\"\n---\n")
	write(t, filepath.Join(claude, "agents/api/children/bad.md"), "# no frontmatter here\n")

	if f := RunAll(Input{CoreDir: filepath.Join(root, "core"), TeamsDir: filepath.Join(root, "teams")}); len(f) != 0 {
		t.Fatalf("RunAll must not reach the installed layer, got %+v", f)
	}
}

// channelTeam lays out a team that declares one capture channel, with the rule
// and skill it names actually present. ch is spliced into team.json verbatim so
// a test can declare a broken variant.
func channelTeam(t *testing.T, ch string) string {
	t.Helper()
	root := t.TempDir()
	teams := filepath.Join(root, "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"),
		`{"name":"demo","skills":[{"name":"note-drain"}],"capabilities":{"note":{"channel":`+ch+`}}}`)
	write(t, filepath.Join(base, "skills/note-drain/SKILL.md"), "---\nname: note-drain\ndescription: \"drain notes\"\n---\n# Drain\n")
	write(t, filepath.Join(base, "rules/note-capture.md"), "# Note capture\n")
	return teams
}

const wholeChannel = `{"name":"note","drain":"/note-drain","rule":"note-capture","describes":"notes"}`

// A declaration whose rule and skill both exist is clean — the check must not
// fire on the shape it is meant to bless.
func TestChannelsAcceptsAResolvableDeclaration(t *testing.T) {
	if f := Channels(channelTeam(t, wholeChannel)); len(f) != 0 {
		t.Fatalf("a resolvable declaration should yield no findings, got %+v", f)
	}
}

// The gate this check exists for. A channel naming a rule the team does not ship
// is accepted by every runtime surface: the channel goes active, its markers ARE
// captured, and nothing ever tells an agent to drain them. The only symptom is a
// queue that grows forever, so it has to fail here, in CI, against the source tree.
func TestChannelsRejectsARuleTheTeamDoesNotShip(t *testing.T) {
	teams := channelTeam(t, `{"name":"note","drain":"/note-drain","rule":"note-captre","describes":"notes"}`)
	f := Channels(teams)
	if len(f) != 1 || f[0].Severity != Fail || f[0].Check != "channel" {
		t.Fatalf("want one fail-level channel finding, got %+v", f)
	}
	if !strings.Contains(f[0].Detail, "note-captre") {
		t.Errorf("finding should name the unresolved rule: %q", f[0].Detail)
	}
}

// Same for the drain skill — the declared value is a slash-command, the asset is
// the skill dir it resolves to.
func TestChannelsRejectsADrainSkillTheTeamDoesNotShip(t *testing.T) {
	teams := channelTeam(t, `{"name":"note","drain":"/note-drian","rule":"note-capture","describes":"notes"}`)
	f := Channels(teams)
	if len(f) != 1 || f[0].Severity != Fail {
		t.Fatalf("want one fail-level finding, got %+v", f)
	}
	if !strings.Contains(f[0].Detail, "note-drian") {
		t.Errorf("finding should name the unresolved skill: %q", f[0].Detail)
	}
}

// A field a signal sentence needs, missing. `name` is the sharpest: the read
// path drops a nameless declaration entirely, so without this check it looks
// declared in team.json and is invisible at every other surface.
func TestChannelsRejectsAnIncompleteDeclaration(t *testing.T) {
	for _, tc := range []struct{ name, ch, want string }{
		{"no name", `{"drain":"/note-drain","rule":"note-capture","describes":"notes"}`, "name"},
		{"no rule", `{"name":"note","drain":"/note-drain","describes":"notes"}`, "rule"},
		{"no describes", `{"name":"note","drain":"/note-drain","rule":"note-capture"}`, "describes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := Channels(channelTeam(t, tc.ch))
			if len(f) != 1 || f[0].Severity != Fail {
				t.Fatalf("want one fail-level finding, got %+v", f)
			}
			if !strings.Contains(f[0].Detail, tc.want) {
				t.Errorf("finding should name the missing field %q: %q", tc.want, f[0].Detail)
			}
		})
	}
}

// A capability that declares no channel at all is the common case (`store`,
// `review`, a bare string) and must stay silent — the check reports broken
// declarations, not the absence of one.
func TestChannelsIgnoresCapabilitiesWithoutAChannel(t *testing.T) {
	root := t.TempDir()
	teams := filepath.Join(root, "teams")
	write(t, filepath.Join(teams, "demo", "team.json"),
		`{"name":"demo","capabilities":{"profile":{"store":"~/.atl/x"},"review":"code-reviewer"}}`)
	if f := Channels(teams); len(f) != 0 {
		t.Fatalf("a team declaring no channel should yield no findings, got %+v", f)
	}
}

// The registration assertion. Every test above calls Channels directly, which
// proves the body and says nothing about the wiring — and `atl skills check`
// runs RunAll, not Channels. Unwiring the check from RunAll leaves a correct
// function nothing calls: it compiles, every other test stays green, and the
// gate silently stops gating.
func TestRunAllRunsTheChannelCheck(t *testing.T) {
	teams := channelTeam(t, `{"name":"note","drain":"/note-drain","rule":"note-captre","describes":"notes"}`)
	for _, f := range RunAll(Input{TeamsDir: teams}) {
		if f.Check == "channel" {
			return
		}
	}
	t.Fatal("RunAll surfaced no channel finding — the check is not wired into the gate")
}
