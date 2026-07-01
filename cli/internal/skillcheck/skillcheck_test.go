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
