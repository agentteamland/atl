package dispatch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Runner runs an external command and returns its combined output — the git
// seam (mirrors internal/publish's Runner) that keeps worktree management
// testable: real dispatch shells out via ExecRunner, tests inject a fake that
// asserts the exact git argv and proves no forced delete ever touches an
// unmerged branch.
type Runner func(name string, args ...string) ([]byte, error)

// ExecRunner is the real Runner (exec.Command combined output).
func ExecRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// Worktree owns the git worktree + branch lifecycle for dispatched work units,
// deterministically — workers never create or destroy their own worktree. Every
// git call routes through Run so the branch-hygiene asymmetry (merged branches
// deleted freely; unmerged branches NEVER auto-deleted) is testable.
type Worktree struct {
	RepoDir  string // the delivery repo root (where `dev` lives)
	Root     string // base dir under which per-unit worktrees are created
	BaseRef  string // the branch worktrees fork from + merge back to (e.g. "dev")
	RemoteRef string // the fetched ref to branch off (e.g. "origin/dev")
	Run      Runner
}

// Orphan is a leftover worktree found at startup that no live worker owns.
type Orphan struct {
	Branch   string
	Path     string
	Unmerged bool   // has commits not in BaseRef — NEVER auto-deleted
	Detail   string // diagnostic for the surfaced / mark-blocked path
}

// BranchName is the branch (and worktree dir name) for a unit:
// delivery/<sprint-slug>/<work-item-id>. The id ties every branch to exactly
// one Azure unit and one PR (branch-hygiene by construction).
func BranchName(slug string, id int) string {
	return fmt.Sprintf("delivery/%s/%d", slug, id)
}

func (w *Worktree) path(slug string, id int) string {
	return filepath.Join(w.Root, slug, strconv.Itoa(id))
}

// Create adds a fresh worktree for the unit, branched off a freshly-fetched
// base so the branch never starts from a stale local ref (#11). Returns the
// worktree path.
func (w *Worktree) Create(slug string, id int) (string, error) {
	branch := BranchName(slug, id)
	path := w.path(slug, id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if _, err := w.git("fetch", "origin", w.BaseRef); err != nil {
		return "", err
	}
	if _, err := w.git("worktree", "add", "-b", branch, path, w.RemoteRef); err != nil {
		return "", err
	}
	return path, nil
}

// Teardown removes a MERGED unit's worktree and deletes its branch. Call it only
// after the unit's PR has merged to BaseRef — merge is the caller's confirmed
// precondition, which is why the branch delete is forced (`-D`): a squash-merge
// leaves the branch's commits absent from BaseRef by SHA, so `-d` would wrongly
// refuse. The worktree checkout is disposable (its work is already in BaseRef),
// so it is removed with --force.
func (w *Worktree) Teardown(slug string, id int) error {
	branch := BranchName(slug, id)
	path := w.path(slug, id)
	if _, err := w.git("worktree", "remove", "--force", path); err != nil {
		return err
	}
	if _, err := w.git("branch", "-D", branch); err != nil {
		return err
	}
	// Best-effort remote branch delete — a missing remote branch is not an error.
	_, _ = w.git("push", "origin", "--delete", branch)
	return nil
}

// Reconcile classifies leftover delivery/* worktrees no live worker owns
// (active = branch names currently in use) and applies the branch-hygiene
// asymmetry. A worktree with NO commits beyond BaseRef is safely reclaimed
// (worktree + branch removed). A worktree WITH commits beyond BaseRef is
// preserved and surfaced — never deleted. Detection is deliberately
// conservative: any commit in the branch but not in BaseRef counts as unmerged,
// even one that was actually squash-merged (a safe false-positive — it only
// preserves+surfaces an already-merged branch; it never risks dropping real
// unmerged work).
func (w *Worktree) Reconcile(active map[string]bool) ([]Orphan, error) {
	entries, err := w.listWorktrees()
	if err != nil {
		return nil, err
	}
	var orphans []Orphan
	for _, e := range entries {
		if !strings.HasPrefix(e.branch, "delivery/") || active[e.branch] {
			continue
		}
		unmerged, err := w.hasUnmergedCommits(e.branch)
		if err != nil {
			return nil, err
		}
		if unmerged {
			orphans = append(orphans, Orphan{
				Branch: e.branch, Path: e.path, Unmerged: true,
				Detail: "has commits not in " + w.BaseRef + "; preserved for diagnosis, not deleted",
			})
			continue
		}
		if _, err := w.git("worktree", "remove", "--force", e.path); err != nil {
			return nil, err
		}
		if _, err := w.git("branch", "-D", e.branch); err != nil {
			return nil, err
		}
		orphans = append(orphans, Orphan{
			Branch: e.branch, Path: e.path, Unmerged: false,
			Detail: "no commits beyond " + w.BaseRef + "; worktree + branch reclaimed",
		})
	}
	return orphans, nil
}

// hasUnmergedCommits reports whether branch carries any commit not in BaseRef.
func (w *Worktree) hasUnmergedCommits(branch string) (bool, error) {
	out, err := w.git("rev-list", "--count", w.BaseRef+".."+branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "0", nil
}

type worktreeEntry struct {
	path   string
	branch string
}

// listWorktrees parses `git worktree list --porcelain` into (path, branch)
// pairs (branch is the short name, e.g. delivery/s1/42; empty for detached).
func (w *Worktree) listWorktrees() ([]worktreeEntry, error) {
	out, err := w.git("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var entries []worktreeEntry
	var cur worktreeEntry
	flush := func() {
		if cur.path != "" {
			entries = append(entries, cur)
		}
		cur = worktreeEntry{}
	}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur.path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			cur.branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		}
	}
	flush()
	return entries, nil
}

func (w *Worktree) git(args ...string) (string, error) {
	out, err := w.Run("git", append([]string{"-C", w.RepoDir}, args...)...)
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
