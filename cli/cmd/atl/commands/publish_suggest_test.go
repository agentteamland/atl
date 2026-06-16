package commands

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
)

// TestDetectPublishable proves the scan over the global layer: a team whose
// global copy diverged from published is suggested, a team that still matches
// published is not, and a team whose source can't be fetched is skipped
// (best-effort) without failing the whole pass.
func TestDetectPublishable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	globalLayer := filepath.Join(home, ".atl")
	globalClaude := filepath.Join(home, ".claude")

	install := func(handle, name, repo, rel string) {
		m := &manifest.Manifest{
			Handle: handle, Name: name, Scope: "global",
			Source: manifest.Source{Repo: repo, Subpath: "", Ref: "v1.0.0"},
			// The hash value is irrelevant here — Plan diffs disk vs disk and uses
			// only the manifest's file *keys* as the candidate set.
			Files: map[string]string{rel: "ignored"},
		}
		if err := m.Write(globalLayer); err != nil {
			t.Fatal(err)
		}
	}
	install("acme", "has-gains", "acme/has-gains", "agents/a/agent.md")
	install("acme", "clean", "acme/clean", "agents/b/agent.md")
	install("acme", "offline", "acme/offline", "agents/c/agent.md")

	// Global .claude state.
	writeF(t, filepath.Join(globalClaude, "agents/a/agent.md"), "EVOLVED")   // diverged → gain
	writeF(t, filepath.Join(globalClaude, "agents/b/agent.md"), "PUBLISHED") // matches published
	writeF(t, filepath.Join(globalClaude, "agents/c/agent.md"), "EVOLVED")   // diverged, but fetch fails

	// Fake fetch: mimic each team's published version; the network is "down" for
	// the offline team.
	fetch := func(repo, subpath, ref string) (string, error) {
		switch repo {
		case "acme/has-gains":
			d := t.TempDir()
			writeF(t, filepath.Join(d, "agents/a/agent.md"), "ORIGINAL") // != global EVOLVED
			return d, nil
		case "acme/clean":
			d := t.TempDir()
			writeF(t, filepath.Join(d, "agents/b/agent.md"), "PUBLISHED") // == global
			return d, nil
		default:
			return "", fmt.Errorf("network down")
		}
	}

	got, err := detectPublishable(globalLayer, globalClaude, fetch)
	if err != nil {
		t.Fatalf("detectPublishable: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d suggestion(s), want 1: %+v", len(got), got)
	}
	if got[0].Ref != "acme/has-gains" {
		t.Errorf("Ref = %q, want acme/has-gains", got[0].Ref)
	}
	if got[0].Count != 1 {
		t.Errorf("Count = %d, want 1", got[0].Count)
	}
}

// TestDetectPublishableEmpty: no global installs → no suggestions, no error,
// and fetch is never called.
func TestDetectPublishableEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	fetch := func(repo, subpath, ref string) (string, error) {
		t.Fatalf("fetch should not be called when nothing is installed")
		return "", nil
	}
	got, err := detectPublishable(filepath.Join(home, ".atl"), filepath.Join(home, ".claude"), fetch)
	if err != nil {
		t.Fatalf("detectPublishable: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d suggestion(s), want 0", len(got))
	}
}
