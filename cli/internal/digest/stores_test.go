package digest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A store records WHICH project it is for, because its filename cannot.
//
// Path hashes the root, and a hash is one-way — so without this field nothing can
// list the digests and name their projects. Identifying six stores on one machine
// took hashing 5,596 directories, and two could not be identified at all.
func TestAStoreRecordsTheProjectItIsFor(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "a finding", "body")

	p, err := Path(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		ProjectRoot string `json:"projectRoot"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	if f.ProjectRoot != filepath.Clean(root) {
		t.Errorf("the store records %q as its project and the root is %q — a per-project "+
			"artifact that cannot say which project it belongs to is one nothing can list",
			f.ProjectRoot, filepath.Clean(root))
	}
}

// An older store has no project recorded, and that is REPORTED rather than guessed.
//
// A reverse lookup would manufacture a path carrying exactly the confidence the
// field exists to earn. The absence is a fact about when the distinction started
// being recorded, and a listing that filled it in would look uniform and be wrong.
func TestAStoreWrittenBeforeTheFieldReportsNoProjectRatherThanAGuess(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "a finding", "body")

	p, err := Path(root)
	if err != nil {
		t.Fatal(err)
	}
	// Rewrite it in the old shape: schemaVersion + findings, no projectRoot.
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw, "projectRoot")
	out, _ := json.MarshalIndent(raw, "", "  ")
	if err := os.WriteFile(p, out, 0o644); err != nil {
		t.Fatal(err)
	}

	stores, err := Stores()
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 1 {
		t.Fatalf("expected 1 store, got %d", len(stores))
	}
	if stores[0].Root != "" {
		t.Errorf("Root is %q for a store that records none — it was inferred from somewhere, "+
			"and an inferred project reads exactly like a recorded one", stores[0].Root)
	}
	if stores[0].Total != 1 || stores[0].Unread != 1 {
		t.Errorf("counts came back %d/%d, want 1/1 — the old shape must still be readable",
			stores[0].Total, stores[0].Unread)
	}
}

// Stores() sees every project's digest, which is the whole point: the split is
// correct and its SILENCE is the defect.
func TestStoresSeesEveryProjectsDigestNotOnlyThisOne(t *testing.T) {
	// One HOME, two projects — the shape of a hub that clones repos beneath itself.
	t.Setenv("HOME", t.TempDir())
	a, b := t.TempDir(), t.TempDir()

	if _, err := Add(a, Finding{Sweep: "observe", Title: "in A", Body: "x"}); err != nil {
		t.Fatal(err)
	}
	for i, title := range []string{"in B one", "in B two"} {
		if _, err := Add(b, Finding{Sweep: "skill-stocktake", Title: title, Body: "x"}); err != nil {
			t.Fatalf("%d: %v", i, err)
		}
	}

	stores, err := Stores()
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 2 {
		t.Fatalf("%d store(s) — Stores must see BOTH projects, or the split stays as silent "+
			"as it was before this existed", len(stores))
	}

	total := 0
	roots := map[string]bool{}
	for _, s := range stores {
		total += s.Total
		roots[s.Root] = true
	}
	if total != 3 {
		t.Errorf("%d finding(s) across the stores, want 3", total)
	}
	for _, want := range []string{filepath.Clean(a), filepath.Clean(b)} {
		if !roots[want] {
			t.Errorf("no store names %q — the listing cannot tell the reader where to look", want)
		}
	}
}

// A corrupt store is still listed. Reporting zero for it would hide it exactly as
// the silence this listing exists to end.
func TestACorruptStoreIsListedRatherThanSkipped(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "a finding", "body")
	p, err := Path(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	stores, err := Stores()
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 1 {
		t.Fatalf("%d store(s) — a corrupt digest must still appear, or it becomes invisible "+
			"at the moment it most needs a human", len(stores))
	}
	if !strings.HasSuffix(stores[0].Path, ".json") {
		t.Errorf("the corrupt store came back without a usable path: %q", stores[0].Path)
	}
}
