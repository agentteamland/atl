package storegit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bare creates a bare repo to push at, standing in for an off-machine remote.
func bare(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name+".git")
	run(t, "", "init", "--bare", "-q", dir)
	return dir
}

// TestReportsAStoreWithNoRemote is the live case this whole report exists for: a
// store faithfully versioned by EnsureAll, whose history has never had anywhere
// else to be. Every automatic mechanism reports success and the content is one
// disk failure from gone.
func TestReportsAStoreWithNoRemote(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "people/x/profile.md", "a")
	EnsureAll([]string{dir})

	got := Backups([]string{dir})
	if len(got) != 1 {
		t.Fatalf("Backups returned %d states, want 1", len(got))
	}
	if got[0].HasRemote {
		t.Fatalf("HasRemote = true, want false — the store has no remote configured")
	}
	if got[0].Unpushed == 0 {
		t.Fatalf("Unpushed = 0, want the whole history — with no remote nothing has been pushed")
	}
}

// TestSilentWhenEverythingIsPushed guards the half that decides whether this is a
// signal or wallpaper. A report that cannot go quiet once the user has acted is
// the constant-advisory shape this codebase has already measured as unread.
func TestSilentWhenEverythingIsPushed(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "people/x/profile.md", "a")
	EnsureAll([]string{dir})

	run(t, dir, "remote", "add", "origin", bare(t, "off"))
	run(t, dir, "push", "-q", "origin", "HEAD:refs/heads/main")
	run(t, dir, "fetch", "-q", "origin")

	got := Backups([]string{dir})
	if len(got) != 1 {
		t.Fatalf("Backups returned %d states, want 1", len(got))
	}
	if !got[0].HasRemote {
		t.Fatalf("HasRemote = false, want true")
	}
	if got[0].Unpushed != 0 {
		t.Fatalf("Unpushed = %d, want 0 — everything reachable from HEAD is on the remote", got[0].Unpushed)
	}
}

// TestCountsCommitsMadeSinceThePush is the drift case: the backup exists but is
// behind, which is invisible to anything that only asks "is there a remote?".
func TestCountsCommitsMadeSinceThePush(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "people/x/profile.md", "a")
	EnsureAll([]string{dir})

	run(t, dir, "remote", "add", "origin", bare(t, "off"))
	run(t, dir, "push", "-q", "origin", "HEAD:refs/heads/main")
	run(t, dir, "fetch", "-q", "origin")

	// Two more sessions' worth of writes land locally and nothing pushes them.
	write(t, dir, "people/x/profile.md", "b")
	EnsureAll([]string{dir})
	write(t, dir, "projects/investment/profile.md", "c")
	EnsureAll([]string{dir})

	got := Backups([]string{dir})
	if len(got) != 1 {
		t.Fatalf("Backups returned %d states, want 1", len(got))
	}
	if !got[0].HasRemote {
		t.Fatalf("HasRemote = false, want true")
	}
	if got[0].Unpushed != 2 {
		t.Fatalf("Unpushed = %d, want 2", got[0].Unpushed)
	}
}

// TestLeavesAUserCreatedRepoAlone: the ownership marker is what separates "a
// store ATL versions" from "a repo that happens to sit at a declared path". This
// report has no business commenting on the second one's remotes.
func TestBackupsLeaveAUserCreatedRepoAlone(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	run(t, dir, "init", "-q")
	write(t, dir, "notes.md", "mine")

	if got := Backups([]string{dir}); len(got) != 0 {
		t.Fatalf("Backups returned %d states, want 0 — the repo carries no ownership marker", len(got))
	}
}

// TestSaysNothingAboutAnAbsentStore: a declared store that does not exist means
// the owning feature is not in use here, not that a backup is missing.
func TestSaysNothingAboutAnAbsentStore(t *testing.T) {
	requireGit(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := Backups([]string{filepath.Join(home, ".atl", "profiles")}); len(got) != 0 {
		t.Fatalf("Backups returned %d states, want 0", len(got))
	}
}

// TestRejectsAnUnacceptablePath: the vetting that protects EnsureAll protects
// this pass too — a declaration naming the home directory itself must not cause
// a report about the user's whole home.
func TestRejectsAnUnacceptablePath(t *testing.T) {
	requireGit(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := Backups([]string{home}); len(got) != 0 {
		t.Fatalf("Backups returned %d states, want 0 — the home directory is not an acceptable store", len(got))
	}
}

// TestReportsAStoreTheUserIsStandingIn is the one place this pass deliberately
// diverges from EnsureAll's vetting. That clause exists to stop this package
// WRITING into a repo the user is working in; refusing to READ there would drop
// the report in the one direction that costs something.
func TestReportsAStoreTheUserIsStandingIn(t *testing.T) {
	requireGit(t)
	dir := store(t, "profiles")
	write(t, dir, "people/x/profile.md", "a")
	EnsureAll([]string{dir})

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if got := Backups([]string{dir}); len(got) != 1 {
		t.Fatalf("Backups returned %d states, want 1 — reading is not the hazard the cwd clause guards", len(got))
	}
}

// TestDisplayHidesTheAccountName: the signal's text reaches a terminal, a
// screenshot and a pasted bug report, so it must never carry the home path.
func TestDisplayHidesTheAccountName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := Display(filepath.Join(home, ".atl", "profiles"))
	if want := filepath.Join("~", ".atl", "profiles"); got != want {
		t.Fatalf("Display = %q, want %q", got, want)
	}
	if strings.Contains(got, home) {
		t.Fatalf("Display leaked the home path: %q", got)
	}

	outside := filepath.Join(t.TempDir(), "elsewhere")
	if got := Display(outside); got != outside {
		t.Fatalf("Display = %q, want the path unchanged when it is outside home", got)
	}
}
