package publish

import (
	"os"
	"path/filepath"
	"testing"
)

func writeF(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOwns(t *testing.T) {
	if !Owns("mesut/my-team", "mesut") {
		t.Error("owner == login should own")
	}
	if Owns("agentteamland/atl", "mesut") {
		t.Error("org repo != personal login should not own (handle-namespace check)")
	}
	if Owns("", "mesut") || Owns("noslash", "mesut") {
		t.Error("malformed repo should not own")
	}
	if Owns("mesut/x", "") {
		t.Error("empty login should not own")
	}
}

func TestPlan(t *testing.T) {
	glob := t.TempDir()
	pub := t.TempDir()

	writeF(t, filepath.Join(glob, "agents/a/agent.md"), "V2") // modified vs published
	writeF(t, filepath.Join(pub, "agents/a/agent.md"), "V1")
	writeF(t, filepath.Join(glob, "agents/a/keep.md"), "SAME") // unchanged
	writeF(t, filepath.Join(pub, "agents/a/keep.md"), "SAME")
	writeF(t, filepath.Join(glob, "agents/a/children/new.md"), "NEW") // new (absent upstream)
	writeF(t, filepath.Join(pub, "agents/a/gone.md"), "GONE")         // absent in global → skip

	candidates := []string{
		"agents/a/agent.md", "agents/a/keep.md",
		"agents/a/children/new.md", "agents/a/gone.md",
	}
	changes, err := Plan(glob, pub, candidates)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("got %d changes, want 2: %+v", len(changes), changes)
	}
	// sorted by Rel: children/new.md, then agent.md? "agents/a/agent.md" vs
	// "agents/a/children/new.md": 'a' < 'c' so agent.md first.
	if changes[0].Rel != "agents/a/agent.md" || changes[0].New {
		t.Errorf("change[0] = %+v, want agent.md modified", changes[0])
	}
	if changes[1].Rel != "agents/a/children/new.md" || !changes[1].New {
		t.Errorf("change[1] = %+v, want children/new.md new", changes[1])
	}
}
