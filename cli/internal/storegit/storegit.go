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

// noHooks is a path that must not exist, used as core.hooksPath so git finds no
// hook to run. Pointing at a nonexistent directory is the portable way to say
// "no hooks"; an empty value is not.
const noHooks = "/nonexistent/atl-no-hooks"

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
	// Resolve the reference points too, not just the declared path. Every check
	// below is a lexical comparison, so mixing a resolved path with an unresolved
	// home would reject a perfectly legitimate store the moment any component
	// above it is a symlink — which is the normal state of /var and /tmp on macOS,
	// and of a home directory that has been moved to another volume.
	home, cwd = resolve(home), resolve(cwd)
	if home == "" {
		return 0
	}

	seen := map[string]bool{}
	n := 0
	for _, d := range dirs {
		path := resolve(expand(d, home))
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
	if isEmptyDir(dir) {
		// Nothing to preserve yet. Checked BEFORE `git init` rather than after the
		// tree is built, because leaving a lone .git behind is not harmless: it
		// makes the directory non-empty, and a consumer that tests emptiness to
		// decide whether there is anything to work with — /profile-backup does
		// exactly that — then reports on a store that holds nothing.
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	switch {
	case !isRepoRoot(dir):
		// A store nested INSIDE some other repo is left alone: initialising here
		// would shadow the outer repo, and committing would be writing into a repo
		// this package does not own.
		if insideOtherRepo(dir) {
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
			// Leaving an unclaimed .git behind would be terminal: the next pass sees
			// a repo it does not own and skips the store FOREVER, silently. Undo the
			// init so the next pass gets a clean attempt.
			_ = os.RemoveAll(filepath.Join(dir, ".git"))
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
	} else if isEmptyTree(ctx, dir, tree) {
		// An empty store has nothing to preserve. Committing here would create a
		// repo and report "previous values stay recoverable" about a directory that
		// has never held a value.
		return false
	}

	// Read this BEFORE update-ref moves HEAD — afterwards the comparison would be
	// against the commit we are about to make, which tells us nothing about what
	// the user had staged.
	indexWasClean := !hasParent
	if hasParent {
		_, indexWasClean = git(ctx, dir, nil, "diff-index", "--cached", "--quiet", "HEAD")
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
	if _, ok := git(ctx, dir, nil, "update-ref", "HEAD", commit, expected); !ok {
		return false
	}

	// Bring the repo's REAL index up to the commit we just made — but only when it
	// is not carrying somebody's intent.
	//
	// Without this, the repo's own index stays empty, `git status` reports every
	// file as both staged-for-deletion and untracked, and `git checkout` refuses to
	// move. The docs send users here to recover an old value, so it has to be
	// legible when they arrive.
	//
	// The guard matters because "we created this repo" is not "nobody has state in
	// it": a half-staged `git add -p` lives in the index, and overwriting it would
	// destroy exactly the intermediate state this package claims to leave alone.
	// So the sync runs only when the index already agrees with the old HEAD (or
	// there was no HEAD at all) — i.e. when nothing is staged. A repo with staged
	// work stays slightly untidy, which is the right way to lose that trade.
	if indexWasClean {
		_, _ = git(ctx, dir, nil, "read-tree", tree)
	}
	return true
}

// isEmptyTree reports whether tree is git's canonical empty tree. This still
// matters after the isEmptyDir check above: a store holding only empty
// directories, or only ignored files, is non-empty on disk and empty to git.
func isEmptyTree(ctx context.Context, dir, tree string) bool {
	// --stdin with no input rather than reading /dev/null as a file: exec gives a
	// nil Stdin on every platform, where /dev/null is a unix path.
	empty, ok := git(ctx, dir, nil, "hash-object", "-t", "tree", "--stdin")
	return ok && empty == tree
}

// isEmptyDir reports whether dir holds no entries at all. A store git already
// owns is never empty (it has .git), so this only ever fires before the first
// snapshot.
func isEmptyDir(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return false // unreadable: let the normal path decline it
	}
	defer f.Close()
	names, err := f.Readdirnames(1)
	return err != nil && len(names) == 0
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

// resolve follows symlinks so vetting sees where the path actually LANDS. Every
// check below is lexical, and the git work is not: a symlink at the declared
// path would otherwise pass a check against its own harmless-looking name while
// git initialises a repo over whatever it points at. A user symlinking their
// store out to an external volume or a sync folder is ordinary, so this is a
// realistic route, not a contrived one. An unresolvable path is dropped.
func resolve(p string) string {
	if p == "" {
		return ""
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		if os.IsNotExist(err) {
			return p // does not exist yet — ensure() will decline it anyway
		}
		return ""
	}
	return real
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
//
// This walks the filesystem rather than asking git, deliberately. `git rev-parse
// --show-toplevel` reports failure identically for "not in a repo" and for every
// other reason it can decline — a dubious-ownership refusal under a bind mount,
// a ceiling-directory setting — and reading those as "not in a repo" fails OPEN,
// into exactly the shadowing init this guard exists to prevent. A walk has one
// meaning and no configuration.
func insideOtherRepo(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return true // cannot tell — decline rather than risk shadowing
	}
	for cur := filepath.Dir(abs); ; cur = filepath.Dir(cur) {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return true
		}
		if parent := filepath.Dir(cur); parent == cur {
			return false // reached the filesystem root
		}
	}
}

// git runs one git command in dir and returns its trimmed stdout. Silent by
// design: stderr is discarded and every failure is reported as ok=false, because
// there is no caller that could act on the difference.
//
// Two settings are forced on every invocation. core.hooksPath is pointed at
// nothing, because a hook belongs to the user's workflow and must never run on a
// machine-written snapshot — commit-tree runs none, but update-ref fires
// reference-transaction, which is enough for a configured hook to disable
// retention permanently and silently. And WaitDelay bounds the wait after the
// context expires: cancellation kills the direct child, but Output() blocks
// until every holder of the inherited pipe closes it, so a backgrounded
// grandchild would otherwise stall the user's prompt past the deadline — and
// report success while doing it.
func git(ctx context.Context, dir string, env []string, args ...string) (string, bool) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.hooksPath=" + noHooks}, args...)...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	if env != nil {
		cmd.Env = env
	}
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
