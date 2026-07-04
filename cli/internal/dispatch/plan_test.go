package dispatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.json")
	content := `{
  "sprintSlug": "s14",
  "granularity": "pbi",
  "units": [
    {"id": 4821, "title": "Login screen", "predecessors": [], "stackRank": 1},
    {"id": 4822, "title": "Session API", "predecessors": [4821], "stackRank": 2}
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.SprintSlug != "s14" {
		t.Errorf("SprintSlug = %q, want s14", p.SprintSlug)
	}
	if p.Granularity != GranularityPBI {
		t.Errorf("Granularity = %q, want pbi", p.Granularity)
	}
	if len(p.Units) != 2 {
		t.Fatalf("Units len = %d, want 2", len(p.Units))
	}
	u := p.Units[1]
	if u.ID != 4822 || u.StackRank != 2 || len(u.Predecessors) != 1 || u.Predecessors[0] != 4821 {
		t.Errorf("unit[1] = %+v, want id 4822 rank 2 pred [4821]", u)
	}
}

func TestLoadPlanMissing(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Error("Load of a missing file should error")
	}
}

func TestLoadPlanMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("Load of malformed JSON should error")
	}
}

func TestPlanPath(t *testing.T) {
	got := PlanPath("/proj")
	want := filepath.Join("/proj", ".delivery", "plan.json")
	if got != want {
		t.Errorf("PlanPath = %q, want %q", got, want)
	}
}
