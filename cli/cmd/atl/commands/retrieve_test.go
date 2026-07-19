package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/retrieve"
)

func TestFormatResults(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, ".atl", "wiki", "x.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# Title Here\n\nBody line one is the excerpt."), 0o644); err != nil {
		t.Fatal(err)
	}
	out := formatResults(root, []retrieve.Result{{Path: p, Title: "Title Here"}})
	for _, want := range []string{"atl#140", "Title Here", ".atl/wiki/x.md", "Body line one"} {
		if !strings.Contains(out, want) {
			t.Errorf("formatResults missing %q in:\n%s", want, out)
		}
	}
}

func TestExcerptSkipsFrontmatterAndHeadings(t *testing.T) {
	p := filepath.Join(t.TempDir(), "y.md")
	if err := os.WriteFile(p, []byte("---\nknowledge-base-summary: meta text\n---\n# Heading\n\nReal prose body."), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := excerpt(p)
	if strings.Contains(ex, "knowledge-base-summary") {
		t.Errorf("frontmatter leaked into excerpt: %q", ex)
	}
	if strings.Contains(ex, "Heading") {
		t.Errorf("heading leaked into excerpt: %q", ex)
	}
	if !strings.Contains(ex, "Real prose body") {
		t.Errorf("excerpt missing body: %q", ex)
	}
}

func TestIndexPathFor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p, err := indexPathFor("/some/project")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, "index.gob") || !strings.Contains(filepath.ToSlash(p), "cache/retrieve") {
		t.Fatalf("unexpected index path: %s", p)
	}
}

func TestCorpusStale(t *testing.T) {
	dir := t.TempDir()
	wiki := filepath.Join(dir, "wiki")
	if err := os.MkdirAll(wiki, 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(wiki, "p.md")
	if err := os.WriteFile(page, []byte("# P\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx := filepath.Join(dir, "index.gob")
	dirs := []string{wiki}

	if !corpusStale(dirs, idx) {
		t.Error("a missing index with a page present should be stale")
	}

	if err := os.WriteFile(idx, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(page, past, past); err != nil {
		t.Fatal(err)
	}
	if corpusStale(dirs, idx) {
		t.Error("an index newer than the corpus should not be stale")
	}

	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(page, future, future); err != nil {
		t.Fatal(err)
	}
	if !corpusStale(dirs, idx) {
		t.Error("a page newer than the index should be stale")
	}

	if corpusStale([]string{filepath.Join(dir, "does-not-exist")}, idx) {
		t.Error("an empty corpus should never be stale")
	}
}

func TestInGitWorktreeNonRepo(t *testing.T) {
	if inGitWorktree(t.TempDir()) {
		t.Error("a non-git directory is not a worktree")
	}
}

// TestInGitWorktreeReal guards the worktree-skip that keeps `atl work dispatch`
// workers from each storming a full index build — and the symlink-immune
// comparison (a main repo reached through a symlinked path must NOT be misread as
// a worktree, which would silently disable auto-indexing).
func TestInGitWorktreeReal(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git(repo, "init", "-q")
	git(repo, "-c", "user.email=t@e", "-c", "user.name=t", "commit", "--allow-empty", "-qm", "init")

	if inGitWorktree(repo) {
		t.Error("main repo root wrongly detected as a worktree")
	}
	sub := filepath.Join(repo, "src", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if inGitWorktree(sub) {
		t.Error("main repo subdir wrongly detected as a worktree")
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(repo, link); err == nil { // skip if the FS forbids symlinks
		if inGitWorktree(filepath.Join(link, "src", "deep")) {
			t.Error("main repo reached via a symlink wrongly detected as a worktree")
		}
	}
	wt := filepath.Join(base, "wt")
	git(repo, "worktree", "add", "-q", wt)
	if !inGitWorktree(wt) {
		t.Error("a real linked worktree was not detected")
	}
}

func TestCorpusDirsGatesDocsOnDelivery(t *testing.T) {
	root := t.TempDir()

	hasDocs := func(dirs []string) bool {
		for _, d := range dirs {
			if filepath.Base(d) == "docs" {
				return true
			}
		}
		return false
	}

	dirs, err := corpusDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	if hasDocs(dirs) {
		t.Error("docs/ must be excluded without a .delivery marker")
	}

	if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirs, err = corpusDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDocs(dirs) {
		t.Error("docs/ must be included for a delivery project")
	}
}
