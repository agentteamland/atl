package index

import "testing"

func TestSeedLoads(t *testing.T) {
	ix, err := Seed()
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if ix.SchemaVersion != 1 {
		t.Errorf("schemaVersion = %d, want 1", ix.SchemaVersion)
	}
	if len(ix.Teams) < 2 {
		t.Fatalf("teams = %d, want >= 2", len(ix.Teams))
	}
	for _, e := range ix.Teams {
		if e.Handle == "" || e.Name == "" || e.Source.Repo == "" || e.Source.Ref == "" {
			t.Errorf("seed entry %q has empty required field: %+v", e.Ref(), e)
		}
	}
}

func TestLookup(t *testing.T) {
	ix, err := Seed()
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	e, err := ix.Lookup("agentteamland", "software-project-team")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if e.Source.Repo != "agentteamland/software-project-team" {
		t.Errorf("source repo = %q", e.Source.Repo)
	}
	if e.Ref() != "agentteamland/software-project-team" {
		t.Errorf("Ref() = %q", e.Ref())
	}
	if _, err := ix.Lookup("nobody", "ghost"); err == nil {
		t.Error("expected not-found error for unknown team")
	}
}

func TestLoadInvalid(t *testing.T) {
	if _, err := Load([]byte("{not json")); err == nil {
		t.Error("expected parse error on malformed JSON")
	}
}

func TestParseRef(t *testing.T) {
	h, n, err := ParseRef("mesut/my-team")
	if err != nil || h != "mesut" || n != "my-team" {
		t.Errorf("ParseRef(mesut/my-team) = %q,%q,%v", h, n, err)
	}
	for _, bad := range []string{"", "noslash", "/empty", "empty/", "a/b/c"} {
		if _, _, err := ParseRef(bad); err == nil {
			t.Errorf("ParseRef(%q) expected error, got nil", bad)
		}
	}
}
