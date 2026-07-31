package storegit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if !hasGit() {
		t.Skip("git not on PATH")
	}
}

// write puts a file in dir, creating parents.
func write(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// log returns the repo's commit subjects, newest first.
func log(t *testing.T, dir string) []string {
	t.Helper()
	cmd := exec.Command("git", "log", "--format=%s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// show returns a file's content at HEAD~n.
func show(t *testing.T, dir, rev, rel string) string {
	t.Helper()
	cmd := exec.Command("git", "show", rev+":"+rel)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show %s:%s: %v", rev, rel, err)
	}
	return string(out)
}

// An absent store means the owning feature simply is not in use on this machine.
// Creating it would litter the filesystem AND misreport the feature as active.
func TestEnsureDoesNotCreateAnAbsentStore(t *testing.T) {
	requireGit(t)
	dir := filepath.Join(t.TempDir(), "never-existed")
	if Ensure(dir) {
		t.Fatal("reported a commit for a directory that does not exist")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the store directory was created; it must not be: %v", err)
	}
}

// The load-bearing guarantee: after an overwrite, the PREVIOUS value is still
// retrievable. This is the whole reason the package exists.
func TestOverwrittenValueSurvivesInHistory(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	write(t, dir, "people/alex/profile.md", "state.emotional: unknown\n")
	if !Ensure(dir) {
		t.Fatal("first pass made no commit")
	}

	// The store's write policy is last-write-wins: the file is replaced wholesale.
	write(t, dir, "people/alex/profile.md", "state.emotional: settled\n")
	if !Ensure(dir) {
		t.Fatal("second pass made no commit after the overwrite")
	}

	if got := show(t, dir, "HEAD", "people/alex/profile.md"); !strings.Contains(got, "settled") {
		t.Fatalf("HEAD does not hold the new value: %q", got)
	}
	if got := show(t, dir, "HEAD~1", "people/alex/profile.md"); !strings.Contains(got, "unknown") {
		t.Fatalf("the overwritten value was not retained: %q", got)
	}
}

func TestEnsureIsQuietWhenNothingChanged(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	write(t, dir, "a.md", "x\n")
	if !Ensure(dir) {
		t.Fatal("first pass made no commit")
	}
	if Ensure(dir) {
		t.Fatal("a clean store produced a second commit")
	}
	if n := len(log(t, dir)); n != 1 {
		t.Fatalf("commit count = %d, want 1", n)
	}
}

// A store that lives inside some OTHER repository is left completely alone:
// initialising there would shadow the outer repo, and committing would be
// writing into a repo this package does not own.
func TestEnsureLeavesAStoreNestedInAnotherRepoAlone(t *testing.T) {
	requireGit(t)
	outer := t.TempDir()
	if !run(outer, "init") {
		t.Fatal("could not init the outer repo")
	}
	store := filepath.Join(outer, "nested-store")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, store, "a.md", "x\n")

	if Ensure(store) {
		t.Fatal("committed inside a repo it does not own")
	}
	if _, err := os.Stat(filepath.Join(store, ".git")); !os.IsNotExist(err) {
		t.Fatal("initialised a nested repo that would shadow the outer one")
	}
	if n := len(log(t, outer)); n != 0 {
		t.Fatalf("wrote %d commit(s) into the outer repo", n)
	}
}

// An existing repo is adopted rather than re-initialised — the user may already
// keep the store under their own version control.
func TestEnsureAdoptsAnExistingRepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	if !run(dir, "init") {
		t.Fatal("could not init")
	}
	write(t, dir, "a.md", "x\n")
	if !run(dir, "add", "-A") || !run(dir, "commit", "-m", "the user's own commit") {
		t.Fatal("could not seed a user commit")
	}

	write(t, dir, "a.md", "y\n")
	if !Ensure(dir) {
		t.Fatal("made no commit in an adopted repo")
	}
	entries := log(t, dir)
	if len(entries) != 2 {
		t.Fatalf("commit count = %d, want 2", len(entries))
	}
	if entries[1] != "the user's own commit" {
		t.Fatalf("the user's own commit was disturbed: %q", entries[1])
	}
}

// A store must never gain a remote: it holds the user's most sensitive data, and
// carrying a copy off the machine is a separate, explicitly-consented act.
func TestEnsureNeverAddsARemote(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	write(t, dir, "a.md", "x\n")
	Ensure(dir)

	cmd := exec.Command("git", "remote")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("a remote was configured: %q", out)
	}
}

// Two teams naming the same store is legitimate (a provider and its consumer);
// it must not produce two commits, or a second empty one.
func TestEnsureAllDeduplicates(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	write(t, dir, "a.md", "x\n")

	if n := EnsureAll([]string{dir, dir, "", dir}); n != 1 {
		t.Fatalf("EnsureAll = %d, want 1", n)
	}
	if n := len(log(t, dir)); n != 1 {
		t.Fatalf("commit count = %d, want 1", n)
	}
}

// Store paths are recorded verbatim from team.json, which writes them in tilde
// form — so the tilde has to resolve, and dedup has to see two spellings of one
// path as one path.
func TestExpandResolvesTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	if got, want := expand("~/.atl/profiles"), filepath.Join(home, ".atl", "profiles"); got != want {
		t.Fatalf("expand = %q, want %q", got, want)
	}
	if got, want := expand("  /abs/path  "), "/abs/path"; got != want {
		t.Fatalf("expand = %q, want %q", got, want)
	}
	if got := expand(""); got != "" {
		t.Fatalf("expand(empty) = %q, want empty", got)
	}
}

// A file is not a store. Guarding on IsDir keeps a stray file at the declared
// path from being treated as one.
func TestEnsureIgnoresAFileAtTheStorePath(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if Ensure(p) {
		t.Fatal("treated a file as a store")
	}
}
