package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/fanout"
)

func TestVersionLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.2.0", "1.2.1", true},
		{"1.2.1", "1.2.1", false},
		{"1.2.1", "1.2.0", false},
		{"1.9.0", "1.10.0", true}, // numeric, not lexicographic
		{"v1.0.0", "v2.0.0", true},
		{"0.8.1", "0.8.1", false},
		{"1.0.0-beta", "1.0.0", true},  // a pre-release is older than its final release → upgrades
		{"1.0.0", "1.0.0-beta", false}, // and the final is never "older" than its own pre-release
		{"2.0.0-alpha.1", "2.0.0", true},
		{"1.0.0-beta", "1.0.1", true}, // numeric triple still wins first
	}
	for _, c := range cases {
		if got := versionLess(c.a, c.b); got != c.want {
			t.Errorf("versionLess(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestReflectWithFanout(t *testing.T) {
	src := t.TempDir()
	claude := t.TempDir()
	w := func(dir, rel, body string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// upstream = the new version
	w(src, "agents/a/unchanged.md", "NEW-A")
	w(src, "agents/b/edited.md", "NEW-B")
	w(src, "agents/c/brandnew.md", "C")
	w(src, "knowledge/adapter.md", "NEW-K") // non-agent/skill/rule asset must refresh too
	// installed copy
	w(claude, "agents/a/unchanged.md", "OLD-A")  // == baseline → refresh
	w(claude, "agents/b/edited.md", "USER-EDIT") // != baseline → preserve
	w(claude, "knowledge/adapter.md", "OLD-K")   // == baseline → refresh
	baseline := map[string]string{
		"agents/a/unchanged.md": fanout.Hash([]byte("OLD-A")),
		"agents/b/edited.md":    fanout.Hash([]byte("OLD-B")),
		"knowledge/adapter.md":  fanout.Hash([]byte("OLD-K")),
	}

	next, err := reflectWithFanout(src, claude, baseline)
	if err != nil {
		t.Fatalf("reflectWithFanout: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "agents/a/unchanged.md")); string(b) != "NEW-A" {
		t.Errorf("unchanged should refresh to NEW-A, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "agents/b/edited.md")); string(b) != "USER-EDIT" {
		t.Errorf("edited should be preserved, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "agents/c/brandnew.md")); string(b) != "C" {
		t.Errorf("brandnew should be added, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "knowledge/adapter.md")); string(b) != "NEW-K" {
		t.Errorf("knowledge asset should refresh to NEW-K, got %q", b)
	}
	if next["knowledge/adapter.md"] != fanout.Hash([]byte("NEW-K")) {
		t.Error("knowledge asset must stay in the manifest (advance to NEW-K), not be dropped")
	}
	if next["agents/a/unchanged.md"] != fanout.Hash([]byte("NEW-A")) {
		t.Error("baseline a should advance to NEW-A")
	}
	if next["agents/b/edited.md"] != fanout.Hash([]byte("OLD-B")) {
		t.Error("baseline b should stay at the original install point (preserve)")
	}
	if next["agents/c/brandnew.md"] != fanout.Hash([]byte("C")) {
		t.Error("baseline c should be the new file's hash")
	}
}
