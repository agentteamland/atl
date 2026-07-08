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
	RepoDir   string // the delivery repo root (where `dev` lives)
	Root      string // base dir under which per-unit worktrees are created
	BaseRef   string // the branch worktrees fork from + merge back to (e.g. "dev")
	RemoteRef string // the fetched ref to branch off (e.g. "origin/dev")
	Run       Runner
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
//
// If the unit's canonical branch/worktree already exists, it is a leftover from a
// prior run that must be freed before `worktree add -b` (which errors on an existing
// branch or path). This happens on an engine RESTART mid-pipeline: with the per-unit
// pipeline, a unit holds a committed-but-unmerged branch through its tester + tech-lead
// stages, and Run()'s Reconcile PRESERVES an unmerged leftover in place under the
// canonical name (it never deletes unmerged work) — so re-admission would collide.
// Quarantine applies the branch-hygiene asymmetry (a clean leftover is reclaimed; a
// leftover with real work is moved aside + renamed, never deleted), freeing the name so
// the re-drive gets a fresh worktree off dev. On the in-process retry path recover()
// already quarantined, so branchExists is false there and this is a no-op.
func (w *Worktree) Create(slug string, id int) (string, error) {
	branch := BranchName(slug, id)
	path := w.path(slug, id)
	if w.branchExists(branch) {
		if _, err := w.Quarantine(slug, id, "restart"); err != nil {
			return "", err
		}
	}
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

// MergedToBase reports whether the unit's branch is fully contained in the
// freshly-fetched base — every commit on the branch already reachable from
// origin/dev, so the branch carries NO work that is not integrated. It fetches
// first (the local BaseRef is never fast-forwarded, so it would be stale) and
// compares against RemoteRef. This is the supervisor's DETERMINISTIC merge
// confirmation: it does not trust a worker's exit code, it verifies the durable
// git state before any force-delete. (It requires the delivery workers to
// integrate to dev with a history-reachable strategy — a merge commit or
// rebase/ff, not a squash — so a merged branch's commits are ancestors of dev.)
func (w *Worktree) MergedToBase(slug string, id int) (bool, error) {
	branch := BranchName(slug, id)
	if _, err := w.git("fetch", "origin", w.BaseRef); err != nil {
		return false, err
	}
	out, err := w.git("rev-list", "--count", w.RemoteRef+".."+branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "0", nil
}

// branchExists reports whether a local branch of the given name exists.
func (w *Worktree) branchExists(name string) bool {
	_, err := w.git("show-ref", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// Quarantine retires the worktree + branch of a unit whose worker the recovery
// ladder (#12) just killed — the reclaim step that must free the canonical
// delivery/<slug>/<id> name for a retry WITHOUT ever destroying real work. It
// applies the same branch-hygiene asymmetry as Reconcile: a worktree with NO
// commits beyond BaseRef AND a clean working tree is reclaimed outright (only
// transient status.json telemetry could be lost); a worktree with real unmerged
// commits OR uncommitted work is PRESERVED — the checkout is moved aside with
// `git worktree move` (which keeps the working tree, uncommitted changes and
// all) and its branch renamed with the suffix, so the canonical name is freed
// for the retry while the stalled work survives for diagnosis, never
// force-deleted. Returns the Orphan record (reclaimed or preserved) to surface.
func (w *Worktree) Quarantine(slug string, id int, suffix string) (Orphan, error) {
	branch := BranchName(slug, id)
	path := w.path(slug, id)

	unmerged, err := w.hasUnmergedCommits(branch)
	if err != nil {
		return Orphan{}, err
	}
	dirty := false
	if !unmerged {
		dirty, err = w.hasUncommittedWork(path)
		if err != nil {
			return Orphan{}, err
		}
	}

	if !unmerged && !dirty {
		// Proven safe: no commits beyond BaseRef and a clean working tree.
		if _, err := w.git("worktree", "remove", "--force", path); err != nil {
			return Orphan{}, err
		}
		if _, err := w.git("branch", "-D", branch); err != nil {
			return Orphan{}, err
		}
		return Orphan{
			Branch: branch, Path: path, Unmerged: false,
			Detail: "no commits beyond " + w.BaseRef + " and a clean working tree; worktree + branch reclaimed",
		}, nil
	}

	// Real work — preserve, never delete. Move the checkout aside (working tree
	// intact) and rename the branch so the canonical name is free for the retry.
	// Probe for a free suffix: a persisted quarantine from a PRIOR run (Reconcile
	// preserves it) would otherwise collide — a fixed suffix makes the branch -m
	// fail and abort the sprint.
	qSuffix := suffix
	for n := 2; w.branchExists(branch + "-" + qSuffix); n++ {
		qSuffix = fmt.Sprintf("%s-%d", suffix, n)
	}
	qPath := filepath.Join(w.Root, ".quarantine", slug, strconv.Itoa(id)+"-"+qSuffix)
	qBranch := branch + "-" + qSuffix
	if err := os.MkdirAll(filepath.Dir(qPath), 0o755); err != nil {
		return Orphan{}, err
	}
	if _, err := w.git("worktree", "move", path, qPath); err != nil {
		return Orphan{}, err
	}
	if _, err := w.git("branch", "-m", branch, qBranch); err != nil {
		return Orphan{}, err
	}
	detail := "has commits not in " + w.BaseRef
	if !unmerged {
		detail = "has uncommitted working-tree changes"
	}
	return Orphan{
		Branch: qBranch, Path: qPath, Unmerged: true,
		Detail: detail + "; preserved at " + qPath + " for diagnosis, not deleted",
	}, nil
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
		// "Unmerged work" is not commits-only: a worker killed mid-implement by
		// the reclaim ladder has real UNCOMMITTED work in its working tree, which
		// rev-list reports as zero. Check the working tree too, or a --force
		// removal would silently destroy that work.
		dirty := false
		if !unmerged {
			dirty, err = w.hasUncommittedWork(e.path)
			if err != nil {
				return nil, err
			}
		}
		if unmerged || dirty {
			detail := "has commits not in " + w.BaseRef
			if !unmerged {
				detail = "has uncommitted working-tree changes"
			}
			orphans = append(orphans, Orphan{
				Branch: e.branch, Path: e.path, Unmerged: true,
				Detail: detail + "; preserved for diagnosis, not deleted",
			})
			continue
		}
		// Proven safe: no commits beyond BaseRef AND no uncommitted work (only
		// transient telemetry). --force here can discard nothing but status.json.
		if _, err := w.git("worktree", "remove", "--force", e.path); err != nil {
			return nil, err
		}
		if _, err := w.git("branch", "-D", e.branch); err != nil {
			return nil, err
		}
		orphans = append(orphans, Orphan{
			Branch: e.branch, Path: e.path, Unmerged: false,
			Detail: "no commits beyond " + w.BaseRef + " and a clean working tree; worktree + branch reclaimed",
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

// hasUncommittedWork reports whether the worktree at path has any modified or
// untracked file OTHER than the supervisor's own status.json telemetry — i.e.
// real uncommitted work that a --force removal would destroy. Runs against the
// worktree dir itself (not RepoDir).
func (w *Worktree) hasUncommittedWork(worktreePath string) (bool, error) {
	out, err := w.Run("git", "-C", worktreePath, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status %s: %w\n%s", worktreePath, err, strings.TrimSpace(string(out)))
	}
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 3 {
			continue // blank / too short to be a porcelain entry
		}
		// Porcelain: two status chars, a space, then the path.
		if strings.TrimSpace(line[2:]) == StatusFileName {
			continue // transient telemetry, not work
		}
		return true, nil
	}
	return false, nil
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
