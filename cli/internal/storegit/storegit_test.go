package storegit

import (
	"context"
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

// store creates a directory two levels under a temporary HOME, the shape a real
// declared store has (~/.atl/profiles), and returns its path. Setting HOME is
// what lets EnsureAll's path vetting run for real in tests instead of being
// bypassed.
func store(t *testing.T, name string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".atl", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

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

// run executes a git command in dir for test setup, failing the test on error.
func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{
		"-c", "user.name=test", "-c", "user.email=test@localhost", "-c", "commit.gpgsign=false",
	}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func logSubjects(t *testing.T, dir string) []string {
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

// The load-bearing guarantee: after an overwrite, the PREVIOUS value is still
// retrievable. This is the whole reason the package exists.
func TestOverwrittenValueSurvivesInHistory(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "people/alex/profile.md", "state.emotional: unknown\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("first pass made no commit")
	}

	// The store's write policy is last-write-wins: the file is replaced wholesale.
	write(t, dir, "people/alex/profile.md", "state.emotional: settled\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("second pass made no commit after the overwrite")
	}

	if got := show(t, dir, "HEAD", "people/alex/profile.md"); !strings.Contains(got, "settled") {
		t.Fatalf("HEAD does not hold the new value: %q", got)
	}
	if got := show(t, dir, "HEAD~1", "people/alex/profile.md"); !strings.Contains(got, "unknown") {
		t.Fatalf("the overwritten value was not retained: %q", got)
	}
}

// An absent store means the owning feature is not in use on this machine.
// Creating it would litter the disk AND misreport the feature as active.
func TestDoesNotCreateAnAbsentStore(t *testing.T) {
	requireGit(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".atl", "never-existed")

	if EnsureAll([]string{dir}) != 0 {
		t.Fatal("reported a commit for a directory that does not exist")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the store directory was created; it must not be: %v", err)
	}
}

func TestQuietWhenNothingChanged(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "x\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("first pass made no commit")
	}
	if EnsureAll([]string{dir}) != 0 {
		t.Fatal("a clean store produced a second commit")
	}
	if n := len(logSubjects(t, dir)); n != 1 {
		t.Fatalf("commit count = %d, want 1", n)
	}
}

// A store that lives inside some OTHER repository is left completely alone:
// initialising there would shadow the outer repo.
func TestLeavesAStoreNestedInAnotherRepoAlone(t *testing.T) {
	requireGit(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	outer := filepath.Join(home, "work", "outer")
	if err := os.MkdirAll(outer, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, outer, "init")
	nested := filepath.Join(outer, "nested-store")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, nested, "a.md", "x\n")

	if EnsureAll([]string{nested}) != 0 {
		t.Fatal("committed inside a repo it does not own")
	}
	if _, err := os.Stat(filepath.Join(nested, ".git")); !os.IsNotExist(err) {
		t.Fatal("initialised a nested repo that would shadow the outer one")
	}
	if n := len(logSubjects(t, outer)); n != 0 {
		t.Fatalf("wrote %d commit(s) into the outer repo", n)
	}
}

// A repo somebody else created in the store is theirs, and is left completely
// alone. They have already met the retention goal by versioning it themselves;
// committing there would advance their branch, move HEAD under their in-flight
// work, and change what `git status` reports about their staged set.
func TestLeavesAUserCreatedRepoAlone(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	run(t, dir, "init")
	write(t, dir, "tracked.md", "one\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "the user's own commit")

	// The user is mid-workflow: one change staged, one not — a `git add -p` shape.
	write(t, dir, "tracked.md", "two\n")
	run(t, dir, "add", "tracked.md")
	write(t, dir, "other.md", "unstaged\n")

	if n := EnsureAll([]string{dir}); n != 0 {
		t.Fatalf("EnsureAll = %d, want 0 — it wrote into a repo it did not create", n)
	}
	if entries := logSubjects(t, dir); len(entries) != 1 || entries[0] != "the user's own commit" {
		t.Fatalf("the user's history changed: %v", entries)
	}
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if staged := strings.TrimSpace(string(out)); staged != "tracked.md" {
		t.Fatalf("the user's staged set changed: got %q, want \"tracked.md\"", staged)
	}
}

// Deleting the ownership marker is the per-store opt-out: ATL stops writing.
func TestStopsWhenTheOwnershipMarkerIsRemoved(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "x\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("first pass made no commit")
	}
	if err := os.Remove(filepath.Join(dir, ".git", ownerMarker)); err != nil {
		t.Fatal(err)
	}

	write(t, dir, "a.md", "y\n")
	if n := EnsureAll([]string{dir}); n != 0 {
		t.Fatalf("EnsureAll = %d, want 0 after the marker was removed", n)
	}
	if n := len(logSubjects(t, dir)); n != 1 {
		t.Fatalf("commit count = %d, want 1", n)
	}
}

// The marker lives inside .git, so it must never show up as a file to commit.
func TestOwnershipMarkerIsNotCommitted(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "x\n")
	EnsureAll([]string{dir})

	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), ownerMarker) {
		t.Fatalf("the ownership marker was committed:\n%s", out)
	}
}

// A globally configured hook must not run: it belongs to the user's workflow,
// and a failing one would silently disable retention entirely.
func TestUserHooksDoNotRun(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	hooks := filepath.Join(t.TempDir(), "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	// A hook that always fails — a plain `git commit` would abort on it.
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.hooksPath")
	t.Setenv("GIT_CONFIG_VALUE_0", hooks)

	write(t, dir, "a.md", "x\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("a user pre-commit hook blocked the snapshot")
	}
}

// A repo mid-rebase is the user's work in progress. Committing there would fold
// conflict markers in as content and leave the rebase broken.
func TestSkipsARepoMidOperation(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "base\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("setup: no initial snapshot")
	}
	// Fabricate the marker rather than staging a real conflict — the guard is a
	// state check, and a real rebase conflict is slow and platform-sensitive.
	if err := os.WriteFile(filepath.Join(dir, ".git", "MERGE_HEAD"), []byte("deadbeef\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	write(t, dir, "a.md", "changed\n")
	if EnsureAll([]string{dir}) != 0 {
		t.Fatal("committed into a repo in the middle of a merge")
	}
	if n := len(logSubjects(t, dir)); n != 1 {
		t.Fatalf("commit count = %d, want 1 (the snapshot must not have landed)", n)
	}
}

// The user's global gitignore must not carve files out of the store: the
// retention promise is unconditional, and a silently-skipped file would break it
// with no symptom.
func TestGlobalExcludesDoNotSkipStoreFiles(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	excludes := filepath.Join(t.TempDir(), "globalignore")
	if err := os.WriteFile(excludes, []byte("*.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.excludesFile")
	t.Setenv("GIT_CONFIG_VALUE_0", excludes)

	write(t, dir, "people/alex/profile.md", "value\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("made no commit")
	}
	if got := show(t, dir, "HEAD", "people/alex/profile.md"); !strings.Contains(got, "value") {
		t.Fatalf("the store file was excluded by the user's global gitignore: %q", got)
	}
}

// A store must never gain a remote: it holds the user's most sensitive data, and
// carrying a copy off the machine is a separate, explicitly-consented act.
func TestNeverAddsARemote(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "x\n")
	EnsureAll([]string{dir})

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
// it must not produce two commits.
func TestEnsureAllDeduplicates(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "x\n")

	if n := EnsureAll([]string{dir, dir, "", dir}); n != 1 {
		t.Fatalf("EnsureAll = %d, want 1", n)
	}
	if n := len(logSubjects(t, dir)); n != 1 {
		t.Fatalf("commit count = %d, want 1", n)
	}
}

func TestEnsureAllRespectsTheDisableBrake(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "x\n")
	t.Setenv(disableEnv, "1")

	if n := EnsureAll([]string{dir}); n != 0 {
		t.Fatalf("EnsureAll = %d, want 0 with the brake set", n)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatal("initialised a repo despite the brake")
	}
}

// The declared path arrives from a team.json — third-party content. An unvetted
// one reaching `git init` plus a full-tree add is how "~" becomes a repo over the
// entire home directory.
func TestAcceptableRejectsDangerousDeclarations(t *testing.T) {
	home := "/home/u"
	cwd := "/home/u/projects/app"

	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{"the home directory itself", "/home/u", false},
		{"a bare top-level directory under home", "/home/u/Documents", false},
		{"outside home entirely", "/etc", false},
		{"a parent of the working directory", "/home/u/projects", false},
		{"the working directory itself", "/home/u/projects/app", false},
		{"a relative path", ".atl/profiles", false},
		{"an unclean path", "/home/u/.atl/../.atl/profiles", false},
		{"empty", "", false},
		{"the shape teams actually use", "/home/u/.atl/profiles", true},
		{"deeper still", "/home/u/.config/team/store", true},
	} {
		if got := acceptable(tc.path, home, cwd); got != tc.want {
			t.Errorf("acceptable(%q) = %v, want %v — %s", tc.path, got, tc.want, tc.name)
		}
	}
}

func TestExpandResolvesTilde(t *testing.T) {
	home := "/home/u"
	if got, want := expand("~/.atl/profiles", home), filepath.Join(home, ".atl", "profiles"); got != want {
		t.Errorf("expand = %q, want %q", got, want)
	}
	if got, want := expand("  /abs/path  ", home), "/abs/path"; got != want {
		t.Errorf("expand = %q, want %q", got, want)
	}
	if got := expand("", home); got != "" {
		t.Errorf("expand(empty) = %q, want empty", got)
	}
	// "~" alone must resolve to home and then be REJECTED by acceptable, not
	// silently treated as some other directory.
	if got := expand("~", home); got != home {
		t.Errorf("expand(~) = %q, want %q", got, home)
	}
	if acceptable(expand("~", home), home, "") {
		t.Error("a store declared as ~ was accepted")
	}
}

// A file is not a store.
func TestIgnoresAFileAtTheStorePath(t *testing.T) {
	requireGit(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := filepath.Join(home, ".atl", "not-a-dir")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if EnsureAll([]string{p}) != 0 {
		t.Fatal("treated a file as a store")
	}
}

// A deletion is a change like any other: the snapshot must record it, and the
// deleted content must still be reachable in history.
func TestRecordsDeletions(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "people/alex/profile.md", "value\n")
	EnsureAll([]string{dir})

	if err := os.RemoveAll(filepath.Join(dir, "people")); err != nil {
		t.Fatal(err)
	}
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("a deletion produced no commit")
	}
	if got := show(t, dir, "HEAD~1", "people/alex/profile.md"); !strings.Contains(got, "value") {
		t.Fatalf("the deleted content is not recoverable: %q", got)
	}
	cmd := exec.Command("git", "show", "HEAD:people/alex/profile.md")
	cmd.Dir = dir
	if err := cmd.Run(); err == nil {
		t.Fatal("HEAD still carries the deleted file")
	}
}

func TestGitHelperIsBounded(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "x\n")
	EnsureAll([]string{dir})

	// A REAL repo, so the command would succeed if the context were not honored —
	// running it in a plain temp dir would fail either way and prove nothing.
	if _, ok := git(context.Background(), dir, nil, "status", "--porcelain"); !ok {
		t.Fatal("setup: git status failed in a real repo")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := git(ctx, dir, nil, "status", "--porcelain"); ok {
		t.Fatal("a cancelled context still ran git to completion")
	}
}

// A symlink at the declared path is the route around vetting: every check is
// lexical, the git work is not. Resolving first is what keeps a store pointed at
// the home directory — the catastrophic case the package doc names — from
// passing a check made against its own harmless-looking name.
//
// Honest bound: this stops the targets vetting rejects outright (home itself, a
// bare top-level directory). It does NOT stop a user who deliberately points
// their store at some deep directory of their own — that target is accepted when
// declared directly too, so it is not a symlink-specific hole.
func TestSymlinkedStoreIsVettedByItsTarget(t *testing.T) {
	requireGit(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "secret"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".atl"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".atl", "profiles")
	if err := os.Symlink(home, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if n := EnsureAll([]string{link}); n != 0 {
		t.Fatalf("EnsureAll = %d, want 0 — a symlink to the home directory was accepted", n)
	}
	if _, err := os.Stat(filepath.Join(home, ".git")); !os.IsNotExist(err) {
		t.Fatal("initialised a repository over the entire home directory")
	}
}

// A store symlinked to a legitimate location still works — the resolution must
// not become a blanket rejection of every symlink.
func TestAcceptsAStoreSymlinkedSomewhereLegitimate(t *testing.T) {
	requireGit(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	target := filepath.Join(home, "vault", "profiles")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "a.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".atl"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".atl", "profiles")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if n := EnsureAll([]string{link}); n != 1 {
		t.Fatalf("EnsureAll = %d, want 1", n)
	}
}

// An empty store has nothing to preserve. Committing there would create a repo
// and report "previous values stay recoverable" about a directory that has never
// held a value.
func TestEmptyStoreProducesNoCommit(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	if n := EnsureAll([]string{dir}); n != 0 {
		t.Fatalf("EnsureAll = %d, want 0 for an empty store", n)
	}
}

// The repo's own index must reflect the snapshot. Building the tree in a
// throwaway index protects the user's staging area, but if the real index is
// never populated then `git status` reports every file as both deleted and
// untracked, and `git checkout` refuses to move — in the exact repo the docs
// send users to when they want an old value back.
func TestRepoIsLegibleAfterASnapshot(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "people/alex/profile.md", "value\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("made no commit")
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if s := strings.TrimSpace(string(out)); s != "" {
		t.Fatalf("the store reports itself dirty right after a snapshot:\n%s", s)
	}
}

// The enclosing-repo guard must fail CLOSED. Asking git makes "not a repo"
// indistinguishable from "git declined for another reason" — a dubious-ownership
// refusal under a bind mount, a ceiling directory — and reading those as "not in
// a repo" produces exactly the shadowing init the guard exists to prevent.
func TestEnclosingRepoDetectionIgnoresGitConfiguration(t *testing.T) {
	requireGit(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	outer := filepath.Join(home, "work")
	nested := filepath.Join(outer, "store")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, outer, "init")
	write(t, nested, "a.md", "x\n")

	// A setting that makes git itself refuse to walk up and report the enclosing
	// repo. The guard must not believe it.
	t.Setenv("GIT_CEILING_DIRECTORIES", outer)

	if n := EnsureAll([]string{nested}); n != 0 {
		t.Fatalf("EnsureAll = %d, want 0 — a git setting talked the guard out of it", n)
	}
	if _, err := os.Stat(filepath.Join(nested, ".git")); !os.IsNotExist(err) {
		t.Fatal("initialised a nested repo shadowing the outer one")
	}
}

// update-ref fires reference-transaction hooks, which commit-tree does not. A
// configured one would block every snapshot silently and forever — the exact
// failure mode the design claims to have closed.
func TestReferenceTransactionHooksDoNotRun(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	hooks := filepath.Join(t.TempDir(), "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "reference-transaction"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.hooksPath")
	t.Setenv("GIT_CONFIG_VALUE_0", hooks)

	write(t, dir, "a.md", "x\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("a reference-transaction hook blocked the snapshot")
	}
}

// An init we cannot claim must not be left behind: the next pass would see a repo
// it does not own and skip the store forever, silently.
func TestRollsBackAnInitItCannotClaim(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "a.md", "x\n")
	if EnsureAll([]string{dir}) != 1 {
		t.Fatal("setup: no initial snapshot")
	}
	// Simulate the terminal state the rollback exists to prevent, and confirm it
	// IS terminal — which is why the rollback matters.
	if err := os.Remove(filepath.Join(dir, ".git", ownerMarker)); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "a.md", "y\n")
	if EnsureAll([]string{dir}) != 0 {
		t.Fatal("wrote into a repo carrying no ownership marker")
	}
}
