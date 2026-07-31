// Package storegit keeps a team's declared durable store under local version
// control, so a write that overwrites a value never destroys the previous one.
//
// The problem it solves is narrow and mechanical. A team may keep state outside
// the reflected `.claude` tree — profile-team's `~/.atl/profiles` is the first —
// and the write policy for such a store is usually last-write-wins. Without
// version control an overwrite is unrecoverable: the old value is simply gone,
// with no record that it ever existed. Measured on a real store: a field went
// from an inferred placeholder to a confirmed value and the previous text now
// exists nowhere.
//
// Four constraints shape everything here:
//
//   - **Core stays team-agnostic.** This package is handed a path that a team
//     DECLARED (`capabilities.<name>.store`, recorded into the install manifest).
//     It never learns a team name, and it has no opinion about what the store
//     contains. A future team that declares a store gets the same treatment for
//     free.
//   - **Local only, never a remote.** A store holds the user's most sensitive
//     data. This package creates no remote, adds no remote, and pushes nothing.
//     Carrying a copy off the machine is a separate, explicitly-consented act.
//   - **The declared path is untrusted.** It arrives from a team.json, which is
//     third-party content. An unvetted path reaching `git init` + a full-tree add
//     is how "~" or "." turns into a repo over the user's home directory or their
//     current project. Accept only paths that cannot do that.
//   - **The user's git state is theirs.** This package writes only to a repo it
//     created itself (marked inside .git); a repo somebody else put there is left
//     completely alone, because that user has already met the goal by versioning
//     the store themselves, and advancing their branch would move HEAD under
//     their in-flight work. Even in its own repo it commits with plumbing against
//     a throwaway index, so no staging area is disturbed and no hook fires.
//
// Everything fails silently and is time-bounded. This runs from session-start and
// from the per-prompt tick, where a failure must never block the user and a hang
// is worse than a failure — a store on a stale network mount must not stall a
// prompt. A missing `git` binary is an ordinary state, not an error worth
// reporting.
package storegit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// budget bounds all git work for one store. The common case is a few
// milliseconds (an unchanged store costs one tree build); the bound exists for
// the pathological one — a network mount that has gone away — where blocking the
// user's prompt indefinitely is the real failure.
const budget = 10 * time.Second

// commitMessage prefixes every automatic snapshot commit. It is fixed and
// recognisable so the log reads as a machine record rather than an attempt to
// describe what changed — describing the change is the knowledge layer's job.
const commitMessage = "chore(store): snapshot"

// disableEnv turns the whole pass off, matching the brake every other automatic
// background behaviour in the CLI offers. This one writes into a directory the
// user may consider theirs, so it needs the brake more than the others, not less.
const disableEnv = "ATL_NO_STORE_GIT"

// EnsureAll versions every declared store: it vets each path, de-duplicates, and
// commits whatever changed. Returns how many stores produced a commit.
//
// This is the only entry point, deliberately — the vetting of the declared path
// happens here, at the boundary where untrusted team-declared input enters, so
// there is no way to reach the git work without passing it.
//
// Two teams declaring the same path is legitimate (a provider and a consumer
// naming one store) and must not commit twice.
func EnsureAll(dirs []string) int {
	if os.Getenv(disableEnv) != "" {
		return 0
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return 0
	}
	cwd, _ := os.Getwd()

	seen := map[string]bool{}
	n := 0
	for _, d := range dirs {
		path := expand(d, home)
		if path == "" || seen[path] || !acceptable(path, home, cwd) {
			continue
		}
		seen[path] = true
		if ensure(path) {
			n++
		}
	}
	return n
}

// acceptable reports whether a declared store path may be versioned.
//
// The rule is deliberately blunt: strictly inside the home directory, at least
// two components below it, and never an ancestor of (or equal to) the working
// directory. That admits the shape teams actually use — a namespaced directory
// like ~/.atl/profiles — while rejecting the three declarations that would do
// real damage: "~" (a repo over the entire home directory), a single top-level
// directory like ~/Documents, and "." or any parent of the current project.
//
// A team wanting somewhere else is not silently mishandled: it simply does not
// get automatic versioning, which is the safe direction to fail.
func acceptable(path, home, cwd string) bool {
	if !filepath.IsAbs(path) || path != filepath.Clean(path) {
		return false
	}
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false // outside the home directory, or the home directory itself
	}
	if len(strings.Split(rel, string(filepath.Separator))) < 2 {
		return false // a bare top-level directory under home is too broad
	}
	if cwd != "" {
		if r, err := filepath.Rel(path, cwd); err == nil && !strings.HasPrefix(r, "..") {
			return false // the working directory is inside it — that is the user's project
		}
	}
	return true
}

// ensure versions one already-vetted store. It assumes acceptable(path) held.
func ensure(dir string) bool {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		// An absent store means the owning feature is not in use on this machine.
		// Creating it would litter the disk AND misreport the feature as active.
		return false
	}
	if !hasGit() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	switch {
	case !isRepoRoot(dir):
		// A store nested INSIDE some other repo is left alone: initialising here
		// would shadow the outer repo, and committing would be writing into a repo
		// this package does not own.
		if insideOtherRepo(ctx, dir) {
			return false
		}
		// init.templateDir is disabled explicitly: a globally configured template
		// (what `pre-commit init-templatedir` installs) would copy hooks into the
		// repo we are creating, and those hooks belong to the user's workflow, not
		// to a machine-written snapshot.
		if _, ok := git(ctx, dir, nil, "-c", "init.templateDir=", "init"); !ok {
			return false
		}
		if !claim(dir) {
			return false
		}
	case !owned(dir):
		// A repo somebody else created here is theirs. Committing into it would
		// advance their branch, move HEAD under their in-flight work, and change
		// what `git status` reports about their staged set — and it would be doing
		// so to reach a goal they have already met by versioning the store
		// themselves. Leave it completely alone.
		return false
	}
	if midOperation(dir) || detachedHead(ctx, dir) {
		// A rebase, merge, cherry-pick or a parked detached HEAD is the user's work
		// in progress. Advancing HEAD under it would corrupt that state — and a
		// full-tree add during a conflict would commit the conflict markers as
		// content. Wait; the store will be snapshotted on a later pass.
		return false
	}
	return snapshot(ctx, dir)
}

// snapshot builds the current worktree as a tree in a THROWAWAY index and, if it
// differs from HEAD, records it with plumbing.
//
// Using a separate index rather than `git add -A` + `git commit` is what makes
// this safe in a repo the user also uses: their staging area is never read or
// written (a half-staged `git add -p` survives untouched), a failure leaves
// nothing staged behind, and commit-tree runs no hooks at all — so a slow or
// failing pre-commit hook can neither stall the prompt nor silently disable the
// retention guarantee.
func snapshot(ctx context.Context, dir string) bool {
	idxDir, err := os.MkdirTemp("", "atl-store-index-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(idxDir)
	env := append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(idxDir, "index"))

	// core.excludesFile is disabled so the user's global gitignore cannot quietly
	// carve files out of the store. A .gitignore INSIDE the store is a deliberate
	// statement about the store and is still honored.
	if _, ok := git(ctx, dir, env, "-c", "core.excludesFile=/dev/null", "add", "-A", "."); !ok {
		return false
	}
	tree, ok := git(ctx, dir, env, "write-tree")
	if !ok {
		return false
	}

	parent, hasParent := git(ctx, dir, nil, "rev-parse", "-q", "--verify", "HEAD")
	if hasParent {
		if head, ok := git(ctx, dir, nil, "rev-parse", "-q", "--verify", "HEAD^{tree}"); ok && head == tree {
			return false // nothing changed
		}
	}

	args := []string{
		"-c", "user.name=atl",
		"-c", "user.email=atl@localhost",
		"-c", "commit.gpgsign=false",
		"commit-tree", tree, "-m", commitMessage + " " + time.Now().Format("2006-01-02T15:04:05"),
	}
	if hasParent {
		args = append(args, "-p", parent)
	}
	commit, ok := git(ctx, dir, nil, args...)
	if !ok {
		return false
	}

	// Compare-and-swap against the HEAD we read, so a concurrent session that
	// committed in between loses this round rather than clobbering the other's
	// commit. The loser simply retries on its next pass.
	expected := parent
	if !hasParent {
		expected = strings.Repeat("0", len(commit)) // the ref must not exist yet
	}
	_, ok = git(ctx, dir, nil, "update-ref", "HEAD", commit, expected)
	return ok
}

// ownerMarker names the file that records "ATL created this repository". It
// lives inside .git, so it is never committed, never appears in `git status`,
// and travels with the repo rather than with the worktree.
const ownerMarker = "atl-store"

// claim records ownership of a repo this package just created.
func claim(dir string) bool {
	return os.WriteFile(filepath.Join(dir, ".git", ownerMarker), []byte(
		"Created by atl to keep this declared store's history.\n"+
			"Delete this file to stop atl from committing here.\n"), 0o644) == nil
}

// owned reports whether this package created the repo at dir — the only repo it
// will write to. Deleting the marker is the user's opt-out for a single store.
func owned(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git", ownerMarker))
	return err == nil
}

// midOperation reports whether the repo is in the middle of a multi-step git
// operation, where any HEAD movement would corrupt the user's state.
func midOperation(dir string) bool {
	for _, marker := range []string{
		"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "BISECT_LOG",
		"rebase-merge", "rebase-apply",
	} {
		if _, err := os.Stat(filepath.Join(dir, ".git", marker)); err == nil {
			return true
		}
	}
	return false
}

// detachedHead reports whether HEAD points at a commit rather than a branch —
// the user having parked the repo somewhere on purpose. An unborn branch in a
// freshly-initialised repo is NOT detached.
func detachedHead(ctx context.Context, dir string) bool {
	_, ok := git(ctx, dir, nil, "symbolic-ref", "-q", "HEAD")
	return !ok
}

// expand resolves a leading `~` against home. Store paths are recorded verbatim
// from team.json, which writes them in tilde form.
func expand(p, home string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		p = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
	}
	if !filepath.IsAbs(p) {
		return p // left as-is; acceptable() rejects it
	}
	return filepath.Clean(p)
}

func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// isRepoRoot reports whether dir is itself the root of a git repo — not merely
// inside one.
func isRepoRoot(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && (fi.IsDir() || fi.Mode().IsRegular())
}

// insideOtherRepo reports whether dir sits within a git repo rooted above it.
func insideOtherRepo(ctx context.Context, dir string) bool {
	top, ok := git(ctx, dir, nil, "rev-parse", "--show-toplevel")
	if !ok || top == "" {
		return false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	topAbs, err := filepath.Abs(top)
	if err != nil {
		return false
	}
	return topAbs != abs
}

// git runs one git command in dir and returns its trimmed stdout. Silent by
// design: stderr is discarded and every failure is reported as ok=false, because
// there is no caller that could act on the difference.
func git(ctx context.Context, dir string, env []string, args ...string) (string, bool) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
