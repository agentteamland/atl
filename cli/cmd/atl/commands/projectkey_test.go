package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/digest"
)

// gitRepo builds a real repository with one commit — real, because projectKey's
// whole job is to ask git a question, and a fake would only prove the fake.
func gitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("-c", "user.email=t@e", "-c", "user.name=t", "commit", "--allow-empty", "-qm", "init")
}

// resolved is what projectKey returns for a path: it evaluates symlinks, and on
// macOS t.TempDir() lives under a symlinked /var, so comparing against the raw
// path would fail for a reason that has nothing to do with the code.
func resolved(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// The property every ~/.atl store depends on: one repository, one key. Without
// it a session started in a subdirectory writes to a partition no other session
// in the same project will ever look in.
func TestProjectKeyCollapsesASubdirectoryToTheRepositoryRoot(t *testing.T) {
	repo := t.TempDir()
	gitRepo(t, repo)
	sub := filepath.Join(repo, "cli", "internal", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(sub)
	got, err := projectKey()
	if err != nil {
		t.Fatal(err)
	}
	if want := resolved(t, repo); got != want {
		t.Errorf("projectKey() from a subdirectory = %q, want the repository root %q", got, want)
	}
}

// The case the queue fix was built for: `atl work dispatch` cuts a worktree per
// unit and deletes it on completion, so anything keyed to the worktree's own
// path becomes unreachable the moment the unit finishes. Resolving through the
// COMMON dir is what makes a worker's items outlive its worktree.
func TestProjectKeyResolvesAWorktreeToItsMainRepository(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRepo(t, repo)

	wt := filepath.Join(base, "wt")
	c := exec.Command("git", "worktree", "add", "-q", "-b", "unit-1", wt)
	c.Dir = repo
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	t.Chdir(wt)
	got, err := projectKey()
	if err != nil {
		t.Fatal(err)
	}
	if want := resolved(t, repo); got != want {
		t.Errorf("projectKey() inside a worktree = %q, want the main repository %q", got, want)
	}
}

// Fails toward the old behaviour rather than refusing: a store that partitions
// oddly outside git is recoverable, one that will not open is not.
//
// Asserted against os.Getwd rather than against a resolved path, because that
// is the contract two channel tests already pin by writing to the store with a
// raw path. Normalising this arm would re-key every existing non-git bucket.
func TestProjectKeyFallsBackToTheWorkingDirectoryOutsideGit(t *testing.T) {
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectKey()
	if err != nil {
		t.Fatalf("projectKey outside git must not error: %v", err)
	}
	if got != cwd {
		t.Errorf("projectKey() outside git = %q, want the working directory %q", got, cwd)
	}
}

// The regression this key sharing exists to prevent, stated as the user sees it:
// session-start counts the findings at the repository root and tells you to run
// `atl digest`. If that command resolved its own root differently, it would
// answer "nothing waiting" about a digest the signal had just counted — the
// reader and the signal disagreeing about which project they are in.
func TestDigestFindingIsReachableFromASubdirectory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitRepo(t, repo)
	sub := filepath.Join(repo, "teams", "delivery-team")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	// Written the way a sweep writes it: at the root the signal reports from.
	if _, err := digest.Add(resolved(t, repo), digest.Finding{
		Sweep: "observe",
		Title: "a finding recorded from the repository root",
	}); err != nil {
		t.Fatal(err)
	}

	// Read the way a person reads it: from wherever they happen to be standing.
	t.Chdir(sub)
	root, err := projectKey()
	if err != nil {
		t.Fatal(err)
	}
	findings, err := digest.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("digest from a subdirectory holds %d finding(s), want the 1 recorded at the root", len(findings))
	}
}
