package sweepstate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var allKinds = []struct {
	name string
	k    Kind
	file string
}{
	{"docs", Docs, "docs-audit-state.json"},
	{"skills", Skills, "skill-stocktake-state.json"},
	{"rules", Rules, "rules-distill-state.json"},
	{"observe", Observe, "observe-state.json"},
}

func TestLoadMissingIsEmpty(t *testing.T) {
	for _, tc := range allKinds {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			c, err := tc.k.Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if c.LastSHA != "" || c.LastAt != "" {
				t.Errorf("missing state should be empty, got %+v", c)
			}
			if c.SchemaVersion != SchemaVersion {
				t.Errorf("SchemaVersion = %d, want %d", c.SchemaVersion, SchemaVersion)
			}
		})
	}
}

func TestRecordRoundTrip(t *testing.T) {
	when := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	for _, tc := range allKinds {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			if err := tc.k.Record("abc123", when); err != nil {
				t.Fatalf("Record: %v", err)
			}
			c, err := tc.k.Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if c.LastSHA != "abc123" {
				t.Errorf("LastSHA = %q, want abc123", c.LastSHA)
			}
			if c.LastAt != "2026-06-23T10:00:00Z" {
				t.Errorf("LastAt = %q, want 2026-06-23T10:00:00Z", c.LastAt)
			}
		})
	}
}

// TestPathsAreDistinct guards the load-bearing per-kind divergence: each cursor must
// persist to its own file, or a recorded sweep of one kind would mark another swept.
func TestPathsAreDistinct(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	seen := map[string]bool{}
	for _, tc := range allKinds {
		p, err := tc.k.Path()
		if err != nil {
			t.Fatalf("%s Path: %v", tc.name, err)
		}
		if filepath.Base(p) != tc.file {
			t.Errorf("%s Path = %q, want basename %q", tc.name, p, tc.file)
		}
		if seen[p] {
			t.Errorf("%s reuses path %q — kinds must not collide", tc.name, p)
		}
		seen[p] = true
	}
}

func TestAffectingCommitsSince(t *testing.T) {
	repo := t.TempDir()
	gitRun(t, repo, "init", "-q")
	writeRepoFile(t, repo, "README.md", "root readme") // top-level: neither kind's path
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-qm", "base")
	base := headSHA(t, repo)

	// a commit under neither scan path
	writeRepoFile(t, repo, "notes.txt", "x")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-qm", "notes")
	if Docs.affectingCommitsSince(repo, base) {
		t.Error("notes.txt must not count as docs-affecting")
	}
	if Skills.affectingCommitsSince(repo, base) {
		t.Error("notes.txt must not count as skills-affecting")
	}

	// a docs/ commit: affects Docs (docs core cli), NOT Skills (core teams)
	writeRepoFile(t, repo, "docs/site/cli/x.md", "x")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-qm", "docs")
	if !Docs.affectingCommitsSince(repo, base) {
		t.Error("a docs/ commit must count as docs-affecting")
	}
	if Skills.affectingCommitsSince(repo, base) {
		t.Error("a docs/ commit must NOT count as skills-affecting (core teams only)")
	}

	// a teams/ commit: affects Skills, NOT Docs
	teamsBase := headSHA(t, repo)
	writeRepoFile(t, repo, "teams/x/team.json", "{}")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-qm", "teams")
	if !Skills.affectingCommitsSince(repo, teamsBase) {
		t.Error("a teams/ commit must count as skills-affecting")
	}
	if Docs.affectingCommitsSince(repo, teamsBase) {
		t.Error("a teams/ commit must NOT count as docs-affecting (docs core cli)")
	}
}

func TestDueRunawayGuard(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// A sweep recorded just now is inside the ~1-day guard → not due, regardless of
	// any affecting commits.
	if err := Docs.Record("deadbeef", time.Now()); err != nil {
		t.Fatal(err)
	}
	if Docs.Due(t.TempDir()) {
		t.Error("a sweep recorded just now must not be due (runaway-guard)")
	}
}

// --- test helpers (moved from commands/docs_test.go) ---

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

func headSHA(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}
