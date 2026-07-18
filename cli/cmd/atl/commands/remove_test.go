package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
)

// TestRemoveTeam: removeTeam deletes the manifest's files, prunes the emptied
// dirs, and drops the manifest — leaving a sibling team's files untouched.
func TestRemoveTeam(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()
	projLayer := filepath.Join(proj, ".atl")
	projClaude := filepath.Join(proj, ".claude")

	// team to remove
	writeF(t, filepath.Join(projClaude, "agents/a/agent.md"), "X")
	writeF(t, filepath.Join(projClaude, "agents/a/children/c.md"), "Y")
	writeF(t, filepath.Join(projClaude, "skills/s/SKILL.md"), "Z")
	m := &manifest.Manifest{Handle: "acme", Name: "demo", Scope: "project", Files: map[string]string{
		"agents/a/agent.md":      fanout.Hash([]byte("X")),
		"agents/a/children/c.md": fanout.Hash([]byte("Y")),
		"skills/s/SKILL.md":      fanout.Hash([]byte("Z")),
	}}
	if err := m.Write(projLayer); err != nil {
		t.Fatal(err)
	}
	// a sibling team that must survive
	writeF(t, filepath.Join(projClaude, "agents/b/agent.md"), "KEEP")
	sib := &manifest.Manifest{Handle: "acme", Name: "other", Scope: "project",
		Files: map[string]string{"agents/b/agent.md": fanout.Hash([]byte("KEEP"))}}
	if err := sib.Write(projLayer); err != nil {
		t.Fatal(err)
	}

	n, err := removeTeam("acme", "demo", scope.Project, proj)
	if err != nil {
		t.Fatalf("removeTeam: %v", err)
	}
	if n != 3 {
		t.Errorf("removed %d files, want 3", n)
	}
	// removed team's files + emptied dirs gone
	for _, rel := range []string{"agents/a/agent.md", "agents/a/children/c.md", "skills/s/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(projClaude, rel)); !os.IsNotExist(err) {
			t.Errorf("%s should be gone", rel)
		}
	}
	if _, err := os.Stat(filepath.Join(projClaude, "agents/a")); !os.IsNotExist(err) {
		t.Error("agents/a should be pruned (empty)")
	}
	// manifest gone
	if _, err := manifest.Read(projLayer, "acme", "demo"); err == nil {
		t.Error("manifest should be removed")
	}
	// sibling survives
	if _, err := os.Stat(filepath.Join(projClaude, "agents/b/agent.md")); err != nil {
		t.Error("sibling team's file should survive")
	}
	if _, err := manifest.Read(projLayer, "acme", "other"); err != nil {
		t.Error("sibling manifest should survive")
	}
}

// TestRemoveSummary: the success line promises `atl gc --undo` reversibility only
// when files were actually soft-deleted (n>0). For n==0 nothing moved to gc-trash
// (the manifest's files were already absent on disk), so the summary must omit the
// hollow undo promise and state the files were already absent instead.
func TestRemoveSummary(t *testing.T) {
	cases := []struct {
		name    string
		n       int
		want    []string // substrings the summary must contain
		notWant []string // substrings it must not contain
	}{
		{
			name: "files soft-deleted → count + reversible promise",
			n:    3,
			want: []string{"acme/demo", "3 files", "reversible with `atl gc --undo`"},
		},
		{
			name:    "nothing soft-deleted → already-absent, no undo promise",
			n:       0,
			want:    []string{"acme/demo", "already", "absent"},
			notWant: []string{"reversible", "atl gc --undo", "undo"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := removeSummary("acme", "demo", tc.n, scope.Project)
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("summary must contain %q: %q", w, got)
				}
			}
			for _, nw := range tc.notWant {
				if strings.Contains(got, nw) {
					t.Errorf("summary must not contain %q: %q", nw, got)
				}
			}
		})
	}
}

func TestRemoveTeamNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()
	if _, err := removeTeam("acme", "ghost", scope.Project, proj); err == nil {
		t.Error("expected error removing a team that isn't installed")
	}
}
