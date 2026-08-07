package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// chdir moves into dir for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestFindRepoRootUpward(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "core", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, deep)
	t.Setenv(repoRootEnv, "")

	got, err := findRepoRoot(filepath.Join("core", "skills"))
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	if resolve(t, got) != resolve(t, root) {
		t.Fatalf("got %q, want %q", got, root)
	}
}

// TestFindRepoRootDoesNotProbeDown pins the decision that cost the most to
// reach: the hub layout puts the marker BELOW the cwd, and searching downward
// for it is not merely unhelpful, it is wrong. In a real hub the same marker
// matches archived v1 clones as well as the live monorepo, and readdir order
// decides which one wins — so a probing build ran the quality gates against a
// dead repo and reported drift that was real about content nobody ships.
// Resolution must fail here, and the caller's message names ATL_REPO_ROOT.
func TestFindRepoRootDoesNotProbeDown(t *testing.T) {
	hub := t.TempDir()
	for _, name := range []string{"live", "archived"} {
		if err := os.MkdirAll(filepath.Join(hub, "repos", name, "core", "skills"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	chdir(t, hub)
	t.Setenv(repoRootEnv, "")

	if got, err := findRepoRoot(filepath.Join("core", "skills")); err == nil {
		t.Fatalf("resolved %q by searching downward — with two candidates the winner is readdir order, so this must fail instead", got)
	}
}

func TestFindRepoRootEnvOverride(t *testing.T) {
	hub := t.TempDir()
	live := filepath.Join(hub, "repos", "live")
	if err := os.MkdirAll(filepath.Join(live, "core", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, hub)
	t.Setenv(repoRootEnv, live)

	got, err := findRepoRoot(filepath.Join("core", "skills"))
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	if resolve(t, got) != resolve(t, live) {
		t.Fatalf("got %q, want %q", got, live)
	}
}

// A named root that does not hold the marker is an error, never a silent
// fallthrough to the upward walk: someone who names a root meant that one, and
// resolving a different one would be the wrong-subject failure this file exists
// to prevent.
func TestFindRepoRootEnvWrongIsAnError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "core", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, root)
	t.Setenv(repoRootEnv, filepath.Join(root, "nowhere"))

	if _, err := findRepoRoot(filepath.Join("core", "skills")); err == nil {
		t.Fatal("a set-but-wrong ATL_REPO_ROOT must fail rather than fall through to the upward walk")
	}
}

// resolve normalises symlinked temp dirs (/var vs /private/var on darwin) so the
// comparison is about the directory, not about how it was spelled.
func resolve(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return r
}
