package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIndexBuildRefusesToSaveWhenTheDeadlineHasPassed pins the invariant that
// atl#608 was filed for: a build whose context is done must leave the stored
// index EXACTLY as it found it.
//
// The defect it guards against is silent and permanent. When the deadline fired,
// every page the build had not reached carried a nil vector; that index was
// saved over the good one and the command exited 0 with "indexed N pages". The
// nils then survive forever, because the incremental reuse key is (path, text) —
// a page saved without a vector matches its own checksum on the next build and
// is reused as though it had been embedded.
//
// Asserting on the FILE rather than on the command's message is deliberate: the
// message was already reassuring while the damage was being done, so the only
// witness that means anything is the durable state.
func TestIndexBuildRefusesToSaveWhenTheDeadlineHasPassed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	project := t.TempDir()
	gitRepo(t, project)
	// One knowledge page, so the walk finds a corpus and the build actually runs.
	wiki := filepath.Join(project, ".atl", "wiki")
	if err := os.MkdirAll(wiki, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wiki, "page.md"), []byte("# a page\n\nsome indexable prose about dispatch and merges\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)

	root, err := projectKey()
	if err != nil {
		t.Fatal(err)
	}
	idxPath, err := indexPathFor(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(idxPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// A sentinel standing in for a good index. Its CONTENT is irrelevant — what is
	// under test is whether anything writes over it.
	sentinel := []byte("the index that was already here")
	if err := os.WriteFile(idxPath, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := retrieveIndexCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(os.NewFile(0, os.DevNull))
	if err := cmd.Flags().Set("timeout", "1ns"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("timeout", (90 * time.Minute).String()) })

	runErr := cmd.RunE(cmd, nil)

	got, err := os.ReadFile(idxPath)
	if err != nil {
		t.Fatalf("the index file is gone entirely: %v", err)
	}
	if string(got) != string(sentinel) {
		t.Fatalf("a build that ran out of time overwrote the stored index (%d bytes -> %d bytes); "+
			"every page it did not reach now carries a nil vector, and the reuse key makes that permanent",
			len(sentinel), len(got))
	}
	if runErr == nil {
		t.Fatal("a build cut short by its deadline reported success; the whole defect is that this is " +
			"indistinguishable from a completed build, so it must be an error")
	}
}

// TestIndexTimeoutDefaultCoversAColdBuild guards the constant itself.
//
// 15 minutes was correct for the English embedder and became roughly a third of
// a cold build when the multilingual model replaced it (2.7x slower per chunk,
// ~17 -> ~45 minutes for this project's corpus). The guard above turns that into
// a refusal rather than a corruption, but a default that cannot finish a cold
// build would refuse every time and the index would simply never be built.
//
// This asserts a FLOOR, not the number: any value at or above the measured cold
// build is fine, and the test says so, so a later re-derivation does not have to
// come back and edit an equality.
func TestIndexTimeoutDefaultCoversAColdBuild(t *testing.T) {
	const measuredColdBuild = 45 * time.Minute

	f := retrieveIndexCmd.Flags().Lookup("timeout")
	if f == nil {
		t.Fatal("retrieve index has no --timeout flag")
	}
	got, err := time.ParseDuration(f.DefValue)
	if err != nil {
		t.Fatalf("--timeout default %q is not a duration: %v", f.DefValue, err)
	}
	if got < measuredColdBuild {
		t.Fatalf("--timeout defaults to %s, below the measured cold build of %s — every cold build "+
			"would be refused and the index would never be written", got, measuredColdBuild)
	}
}
