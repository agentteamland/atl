package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/docsstate"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeRepoFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDocAffectingCommitsSince(t *testing.T) {
	repo := t.TempDir()
	gitRun(t, repo, "init", "-q")
	writeRepoFile(t, repo, "README.md", "root readme") // top-level, not under docs/core/cli
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-qm", "base")
	base := gitHEAD(repo)

	// a non-doc-affecting commit
	writeRepoFile(t, repo, "notes.txt", "x")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-qm", "notes")
	if docAffectingCommitsSince(repo, base) {
		t.Error("a notes.txt commit must not count as doc-affecting")
	}

	// a doc-affecting commit (under docs/)
	writeRepoFile(t, repo, "docs/site/cli/x.md", "x")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-qm", "docs")
	if !docAffectingCommitsSince(repo, base) {
		t.Error("a docs/ commit must count as doc-affecting")
	}
}

func TestDocsAuditDueRunawayGuard(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// An audit recorded just now is inside the ~1-day guard → not due, regardless
	// of any doc-affecting commits.
	if err := docsstate.Record("deadbeef", time.Now()); err != nil {
		t.Fatal(err)
	}
	if docsAuditDue(t.TempDir()) {
		t.Error("an audit recorded just now must not be due (runaway-guard)")
	}
}
