package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// write creates a file (and its parents) under root.
func write(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func has(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

// The whole point of the change: a file the learning loop grew inside an
// installed agent is absent from the manifest (which records what INSTALL
// wrote), and must still be offered for publication. Without it, ring 2→3 is
// closed for global-origin growth — the growth most worth circulating.
func TestPublishCandidates_IncludesGrownFiles(t *testing.T) {
	root := t.TempDir()
	write(t, root, "agents/advisor/agent.md")
	write(t, root, "agents/advisor/children/installed-topic.md")
	write(t, root, "agents/advisor/children/grown-topic.md") // grown after install

	files := map[string]string{
		"agents/advisor/agent.md":                    "h1",
		"agents/advisor/children/installed-topic.md": "h2",
	}

	got := publishCandidates(root, files)

	if !has(got, "agents/advisor/children/grown-topic.md") {
		t.Errorf("grown file not offered for publication; got %v", got)
	}
	for rel := range files {
		if !has(got, rel) {
			t.Errorf("manifest file %q dropped from candidates; got %v", rel, got)
		}
	}
}

// Ownership is derived from the MANIFEST, not from the asset dirs, because
// agents/ and skills/ are shared by every installed team. Walking them wholesale
// would offer a neighbouring team's growth for publication under this team's
// name — publishing someone else's work to someone else's repo.
func TestPublishCandidates_IgnoresAnotherTeamsGrowth(t *testing.T) {
	root := t.TempDir()
	write(t, root, "agents/advisor/agent.md")         // this team's
	write(t, root, "agents/advisor/children/mine.md") // this team's growth
	write(t, root, "agents/other-agent/agent.md")     // a different team's
	write(t, root, "agents/other-agent/children/not-mine.md")
	write(t, root, "skills/other-skill/SKILL.md")

	files := map[string]string{"agents/advisor/agent.md": "h1"}

	got := publishCandidates(root, files)

	if !has(got, "agents/advisor/children/mine.md") {
		t.Errorf("own growth missing; got %v", got)
	}
	for _, foreign := range []string{
		"agents/other-agent/agent.md",
		"agents/other-agent/children/not-mine.md",
		"skills/other-skill/SKILL.md",
	} {
		if has(got, foreign) {
			t.Errorf("offered another team's file %q for publication; got %v", foreign, got)
		}
	}
}

// A team that has grown nothing must produce exactly its manifest set — no
// phantom entries, and no dependence on walk order.
func TestPublishCandidates_NoGrowthIsExactlyTheManifest(t *testing.T) {
	root := t.TempDir()
	write(t, root, "agents/advisor/agent.md")
	write(t, root, "skills/advisor/SKILL.md")

	files := map[string]string{
		"agents/advisor/agent.md": "h1",
		"skills/advisor/SKILL.md": "h2",
	}

	got := publishCandidates(root, files)
	if len(got) != len(files) {
		t.Errorf("expected exactly the manifest set, got %v", got)
	}
	// Sorted, so a caller diffing plans across runs sees a stable order.
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("candidates not sorted: %v", got)
			break
		}
	}
}

// Editor droppings and nested VCS directories are not gains. `.DS_Store` next to
// a real gain is the realistic case on the machine this runs on.
func TestPublishCandidates_SkipsHiddenAndNestedVCS(t *testing.T) {
	root := t.TempDir()
	write(t, root, "agents/advisor/agent.md")
	write(t, root, "agents/advisor/children/real-gain.md")
	write(t, root, "agents/advisor/.DS_Store")
	write(t, root, "agents/advisor/.git/config")

	files := map[string]string{"agents/advisor/agent.md": "h1"}

	got := publishCandidates(root, files)
	if !has(got, "agents/advisor/children/real-gain.md") {
		t.Errorf("real gain missing; got %v", got)
	}
	for _, junk := range []string{"agents/advisor/.DS_Store", "agents/advisor/.git/config"} {
		if has(got, junk) {
			t.Errorf("offered %q for publication; got %v", junk, got)
		}
	}
}

// A manifest may name a unit that has since been removed from disk. That is not
// an error — publish.Plan skips a candidate absent from the global layer — so the
// walk must tolerate it rather than failing the whole publish.
func TestPublishCandidates_ToleratesMissingUnit(t *testing.T) {
	root := t.TempDir()
	write(t, root, "agents/advisor/agent.md")

	files := map[string]string{
		"agents/advisor/agent.md": "h1",
		"agents/removed/agent.md": "h2", // never written to disk
	}

	got := publishCandidates(root, files)
	if !has(got, "agents/removed/agent.md") {
		t.Errorf("manifest entry dropped just because it is missing on disk; got %v", got)
	}
	if !has(got, "agents/advisor/agent.md") {
		t.Errorf("surviving unit dropped; got %v", got)
	}
}

// A manifest entry that IS the whole unit (rules/<name>.md) contributes no
// directory to walk — nothing grows inside a single file — and must not drag in
// every other team's rules.
func TestPublishCandidates_FlatUnitDoesNotClaimItsDirectory(t *testing.T) {
	root := t.TempDir()
	write(t, root, "rules/mine.md")
	write(t, root, "rules/someone-elses.md")

	files := map[string]string{"rules/mine.md": "h1"}

	got := publishCandidates(root, files)
	if has(got, "rules/someone-elses.md") {
		t.Errorf("a flat rule claimed the whole rules/ directory; got %v", got)
	}
	if !has(got, "rules/mine.md") {
		t.Errorf("own rule missing; got %v", got)
	}
}
