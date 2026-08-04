package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/retrieve"
)

// An embedder swap makes no corpus file newer, so the modtime pass calls the
// index fresh while Load reports "no index" — retrieval would go silently dead
// on a quiet corpus and never rebuild. This is the check that closes it.
func TestIndexUnusableDetectsAnIndexThisBinaryCannotRead(t *testing.T) {
	p := filepath.Join(t.TempDir(), "index.gob")
	if err := os.WriteFile(p, []byte("not a gob at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !indexUnusable(p) {
		t.Fatal("an unreadable index file must be reported as unusable")
	}
}

// A missing index is NOT this check's case — the modtime pass already treats it
// as stale. Claiming it here too would announce a migration on every first build.
func TestIndexUnusableIgnoresAMissingIndex(t *testing.T) {
	if indexUnusable(filepath.Join(t.TempDir(), "index.gob")) {
		t.Fatal("a missing index must not be reported as unusable")
	}
}

// Built through the package's own writer, so the test cannot drift away from
// indexFormatVersion the way a hand-written fixture would.
func TestIndexUnusableAcceptsACurrentIndex(t *testing.T) {
	p := filepath.Join(t.TempDir(), "index.gob")
	ix, err := retrieve.Build(context.Background(),
		[]retrieve.Doc{{Path: "a.md", Title: "A", Text: "hello"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Save(p); err != nil {
		t.Fatal(err)
	}
	if indexUnusable(p) {
		t.Fatal("an index this binary just wrote must not be reported as unusable")
	}
}
