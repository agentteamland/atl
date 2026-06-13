package fanout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecide(t *testing.T) {
	cases := []struct {
		name                      string
		baseline, local, upstream string
		want                      Decision
	}{
		{"already up to date", "a", "b", "b", UpToDate},        // local == upstream
		{"unmodified gets refreshed", "a", "a", "b", Refresh},  // local == baseline, upstream differs
		{"user-modified is preserved", "a", "x", "b", Preserve}, // local diverged from baseline
		{"all identical", "a", "a", "a", UpToDate},             // up-to-date wins over refresh
		{"modified but matches upstream", "a", "b", "b", UpToDate},
	}
	for _, c := range cases {
		if got := Decide(c.baseline, c.local, c.upstream); got != c.want {
			t.Errorf("%s: Decide(%q,%q,%q) = %v, want %v",
				c.name, c.baseline, c.local, c.upstream, got, c.want)
		}
	}
}

func TestHashDeterministic(t *testing.T) {
	if Hash([]byte("x")) != Hash([]byte("x")) {
		t.Fatal("hash not deterministic")
	}
	if Hash([]byte("x")) == Hash([]byte("y")) {
		t.Fatal("different inputs produced the same hash")
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	if err := os.WriteFile(p, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := HashFile(p)
	if err != nil {
		t.Fatalf("hashfile: %v", err)
	}
	if h != Hash([]byte("hello")) {
		t.Fatalf("hashfile mismatch: %q", h)
	}

	missing, err := HashFile(filepath.Join(dir, "nope"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if missing != "" {
		t.Fatalf("missing file should hash to empty, got %q", missing)
	}
}
