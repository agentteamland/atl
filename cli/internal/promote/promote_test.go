package promote

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
)

func h(s string) string { return fanout.Hash([]byte(s)) }

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func byRel(actions []Action) map[string]Action {
	m := map[string]Action{}
	for _, a := range actions {
		m[a.Rel] = a
	}
	return m
}

// TestPlan exercises every branch: clean lift, conflict lift, new-file lift,
// new-file conflict, and each skip (unchanged, already-shared, pinned, deleted).
func TestPlan(t *testing.T) {
	proj := t.TempDir()
	glob := t.TempDir()

	pm := &manifest.Manifest{Handle: "acme", Name: "demo", Files: map[string]string{
		"agents/a/agent.md":   h("V1"),   // project modifies; global at baseline → Lift
		"agents/a/keep.md":    h("KEEP"), // unchanged → skip
		"agents/a/shared.md":  h("S"),    // proj==glob already → skip
		"agents/a/pinned.md":  h("P1"),   // modified but pinned → skip
		"skills/b/skill.md":   h("SK"),   // modified, global also moved → ConflictLift
		"agents/a/deleted.md": h("DEL"),  // gone from project → skip (doctor's lane)
	}}

	// project copies
	write(t, filepath.Join(proj, "agents/a/agent.md"), "V2")
	write(t, filepath.Join(proj, "agents/a/keep.md"), "KEEP")
	write(t, filepath.Join(proj, "agents/a/shared.md"), "SX")
	write(t, filepath.Join(proj, "agents/a/pinned.md"), "P2")
	write(t, filepath.Join(proj, "skills/b/skill.md"), "SKproj")
	write(t, filepath.Join(proj, "agents/a/children/new.md"), "NEW")        // new, global absent → Lift
	write(t, filepath.Join(proj, "agents/a/children/newconf.md"), "NCproj") // new, global present → ConflictLift
	// agents/a/deleted.md intentionally absent in project

	// global copies
	write(t, filepath.Join(glob, "agents/a/agent.md"), "V1") // at baseline
	write(t, filepath.Join(glob, "agents/a/keep.md"), "KEEP")
	write(t, filepath.Join(glob, "agents/a/shared.md"), "SX") // == project
	write(t, filepath.Join(glob, "agents/a/pinned.md"), "P1")
	write(t, filepath.Join(glob, "skills/b/skill.md"), "SKglob") // diverged from baseline
	write(t, filepath.Join(glob, "agents/a/children/newconf.md"), "NCglob")
	write(t, filepath.Join(glob, "agents/a/deleted.md"), "DEL")

	pinned := func(rel string) bool { return rel == "agents/a/pinned.md" }

	actions, err := Plan(pm, proj, glob, pinned)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	got := byRel(actions)
	if len(actions) != 4 {
		t.Fatalf("got %d actions, want 4: %+v", len(actions), actions)
	}

	want := map[string]Kind{
		"agents/a/agent.md":            Lift,
		"agents/a/children/new.md":     Lift,
		"agents/a/children/newconf.md": ConflictLift,
		"skills/b/skill.md":            ConflictLift,
	}
	for rel, k := range want {
		a, ok := got[rel]
		if !ok {
			t.Errorf("missing action for %s", rel)
			continue
		}
		if a.Kind != k {
			t.Errorf("%s: kind = %v, want %v", rel, a.Kind, k)
		}
	}
	for _, rel := range []string{"agents/a/keep.md", "agents/a/shared.md", "agents/a/pinned.md", "agents/a/deleted.md"} {
		if _, ok := got[rel]; ok {
			t.Errorf("%s should have been skipped", rel)
		}
	}

	// Hash detail: a clean lift carries the project hash as the new baseline and
	// the prior global hash for the (unused) archive slot.
	if a := got["agents/a/agent.md"]; a.ProjHash != h("V2") || a.PriorGlobHash != h("V1") {
		t.Errorf("agent.md hashes: proj=%s prior=%s", a.ProjHash, a.PriorGlobHash)
	}
	// A new file with no global copy archives nothing.
	if a := got["agents/a/children/new.md"]; a.PriorGlobHash != "" {
		t.Errorf("new.md PriorGlobHash = %q, want empty", a.PriorGlobHash)
	}
	// A conflict carries the prior global hash to archive.
	if a := got["skills/b/skill.md"]; a.PriorGlobHash != h("SKglob") {
		t.Errorf("skill.md PriorGlobHash = %s, want %s", a.PriorGlobHash, h("SKglob"))
	}
}

// TestPlanOwnedUnitIsolation: new-file discovery must stay within the team's own
// units, never attributing a sibling team's stray file.
func TestPlanOwnedUnitIsolation(t *testing.T) {
	proj := t.TempDir()
	glob := t.TempDir()

	pm := &manifest.Manifest{Handle: "acme", Name: "demo", Files: map[string]string{
		"agents/a/agent.md": h("V1"),
	}}
	write(t, filepath.Join(proj, "agents/a/agent.md"), "V1")           // unchanged
	write(t, filepath.Join(proj, "agents/a/children/mine.md"), "MINE") // owned new → Lift
	write(t, filepath.Join(proj, "agents/other/stray.md"), "STRAY")    // NOT owned by this team

	actions, err := Plan(pm, proj, glob, func(string) bool { return false })
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(actions) != 1 || actions[0].Rel != "agents/a/children/mine.md" {
		t.Fatalf("want only the owned new file, got %+v", actions)
	}
}

// TestPlanClean: when project, baseline, and global all agree, there is nothing
// to lift.
func TestPlanClean(t *testing.T) {
	proj := t.TempDir()
	glob := t.TempDir()
	pm := &manifest.Manifest{Handle: "acme", Name: "demo", Files: map[string]string{
		"agents/a/agent.md": h("SAME"),
	}}
	write(t, filepath.Join(proj, "agents/a/agent.md"), "SAME")
	write(t, filepath.Join(glob, "agents/a/agent.md"), "SAME")
	actions, err := Plan(pm, proj, glob, func(string) bool { return false })
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("want no actions, got %+v", actions)
	}
}
