package dispatch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner records every git argv and returns canned output per subcommand —
// the publish apply_test.go fakeGH idiom, so worktree management is asserted
// without touching a real repo.
type fakeRunner struct {
	calls           [][]string
	worktreeList    string
	revListCount    string
	statusPorcelain string
	branches        map[string]bool // branch names that `show-ref` reports as existing
}

func (f *fakeRunner) run(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	joined := strings.Join(args, " ")
	switch {
	case strings.Contains(joined, "worktree list"):
		return []byte(f.worktreeList), nil
	case strings.Contains(joined, "rev-list"):
		return []byte(f.revListCount), nil
	case strings.Contains(joined, "status --porcelain"):
		return []byte(f.statusPorcelain), nil
	case strings.Contains(joined, "show-ref"):
		for b := range f.branches {
			if strings.HasSuffix(joined, "refs/heads/"+b) {
				return nil, nil // exit 0 = branch exists
			}
		}
		return nil, fmt.Errorf("no such ref") // exit != 0 = branch absent (the default)
	default:
		return nil, nil
	}
}

func (f *fakeRunner) called(sub string) bool {
	for _, c := range f.calls {
		if strings.Contains(strings.Join(c, " "), sub) {
			return true
		}
	}
	return false
}

func newWorktree(f *fakeRunner) *Worktree {
	return &Worktree{
		RepoDir:   "/repo",
		Root:      "/repo/.delivery/worktrees",
		BaseRef:   "dev",
		RemoteRef: "origin/dev",
		Run:       f.run,
	}
}

func TestBranchName(t *testing.T) {
	if got := BranchName("s14", 4821); got != "delivery/s14/4821" {
		t.Errorf("BranchName = %q, want delivery/s14/4821", got)
	}
}

func TestCreate(t *testing.T) {
	f := &fakeRunner{}
	w := newWorktree(f)
	w.Root = t.TempDir() // Create really mkdir's the parent (git worktree add needs it)
	path, err := w.Create("s1", 42)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "/s1/42") {
		t.Errorf("worktree path = %q, want …/s1/42", path)
	}
	if !f.called("fetch origin dev") {
		t.Error("Create must fetch the base ref first (no stale local dev)")
	}
	if !f.called("worktree add -b delivery/s1/42") || !f.called("origin/dev") {
		t.Errorf("Create must branch off origin/dev: %v", f.calls)
	}
}

// A mid-pipeline engine restart leaves the unit's canonical branch/worktree in place
// (Reconcile preserves unmerged work under the canonical name). Re-admission's Create
// must free the name WITHOUT deleting the unmerged work — quarantine it aside, then
// create a fresh worktree. A regression here re-introduces the sprint-wedge (Create's
// `worktree add -b` colliding on the existing branch) or, worse, deletes unmerged work.
func TestCreateQuarantinesUnmergedRestartLeftover(t *testing.T) {
	f := &fakeRunner{
		branches:     map[string]bool{"delivery/s1/5": true}, // a leftover from a prior run
		revListCount: "3",                                    // it carries unmerged commits
	}
	w := newWorktree(f)
	w.Root = t.TempDir()
	if _, err := w.Create("s1", 5); err != nil {
		t.Fatal(err)
	}
	if !f.called("worktree move") || !f.called("branch -m delivery/s1/5 delivery/s1/5-restart") {
		t.Errorf("an unmerged leftover must be moved aside + renamed (quarantined), not left to collide: %v", f.calls)
	}
	if f.called("branch -D delivery/s1/5") {
		t.Error("DATA LOSS: the unmerged leftover branch must NEVER be force-deleted")
	}
	if !f.called("worktree add -b delivery/s1/5") {
		t.Errorf("after freeing the name, Create must add the fresh worktree: %v", f.calls)
	}
}

// The clean-leftover variant: a merged/empty leftover is safely reclaimed (not moved
// aside), then the fresh worktree is created — no collision, no needless quarantine.
func TestCreateReclaimsCleanRestartLeftover(t *testing.T) {
	f := &fakeRunner{
		branches:        map[string]bool{"delivery/s1/6": true},
		revListCount:    "0", // no commits beyond dev
		statusPorcelain: "",  // and a clean working tree → safe to reclaim
	}
	w := newWorktree(f)
	w.Root = t.TempDir()
	if _, err := w.Create("s1", 6); err != nil {
		t.Fatal(err)
	}
	if !f.called("worktree remove --force") || !f.called("branch -D delivery/s1/6") {
		t.Errorf("a clean leftover should be reclaimed outright: %v", f.calls)
	}
	if !f.called("worktree add -b delivery/s1/6") {
		t.Errorf("after reclaiming, Create must add the fresh worktree: %v", f.calls)
	}
}

func TestTeardown(t *testing.T) {
	f := &fakeRunner{}
	w := newWorktree(f)
	if err := w.Teardown("s1", 42); err != nil {
		t.Fatal(err)
	}
	if !f.called("worktree remove --force") {
		t.Error("Teardown should force-remove the disposable checkout")
	}
	if !f.called("branch -D delivery/s1/42") {
		t.Error("Teardown should delete the (confirmed-merged) branch")
	}
	if !f.called("push origin --delete delivery/s1/42") {
		t.Error("Teardown should best-effort delete the remote branch")
	}
}

func TestReconcileReclaimsClean(t *testing.T) {
	f := &fakeRunner{
		worktreeList: "worktree /repo\nHEAD a\nbranch refs/heads/dev\n\n" +
			"worktree /repo/.delivery/worktrees/s1/99\nHEAD b\nbranch refs/heads/delivery/s1/99\n",
		revListCount: "0", // no commits beyond dev → safe to reclaim
	}
	w := newWorktree(f)
	orphans, err := w.Reconcile(map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || orphans[0].Unmerged {
		t.Fatalf("want 1 clean orphan, got %+v", orphans)
	}
	if !f.called("worktree remove --force /repo/.delivery/worktrees/s1/99") {
		t.Error("clean orphan should have its worktree reclaimed")
	}
	if !f.called("branch -D delivery/s1/99") {
		t.Error("clean orphan branch should be deleted")
	}
	// #158: classify against the freshly-fetched origin/dev, never the stale local
	// ref — else a remote-merged leftover surfaces as a false unmerged orphan.
	if !f.called("fetch origin dev") {
		t.Error("Reconcile must fetch the base before classifying (#158)")
	}
	if !f.called("rev-list --count origin/dev..delivery/s1/99") {
		t.Errorf("Reconcile must classify against origin/dev, not local dev (#158): %v", f.calls)
	}
}

// The load-bearing safety test: an orphan branch with UNMERGED commits must be
// preserved and surfaced — NEVER deleted. A regression here is a data-loss bug.
func TestReconcilePreservesUnmerged(t *testing.T) {
	f := &fakeRunner{
		worktreeList: "worktree /repo\nHEAD a\nbranch refs/heads/dev\n\n" +
			"worktree /repo/.delivery/worktrees/s1/77\nHEAD b\nbranch refs/heads/delivery/s1/77\n",
		revListCount: "3", // 3 commits not in dev → unmerged work
	}
	w := newWorktree(f)
	orphans, err := w.Reconcile(map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || !orphans[0].Unmerged {
		t.Fatalf("want 1 unmerged orphan, got %+v", orphans)
	}
	if f.called("branch -D") || f.called("branch -d") {
		t.Error("SAFETY VIOLATION: Reconcile deleted a branch with unmerged commits")
	}
	if f.called("worktree remove") {
		t.Error("SAFETY VIOLATION: Reconcile removed an unmerged worktree")
	}
}

// Regression for the adversarial-review CRITICAL: an orphan with NO commits
// beyond dev but a DIRTY working tree (a worker killed mid-implement before its
// first commit) holds real uncommitted work — it must be preserved, never
// --force-removed.
func TestReconcilePreservesDirtyWorktree(t *testing.T) {
	f := &fakeRunner{
		worktreeList:    "worktree /repo/.delivery/worktrees/s1/50\nHEAD b\nbranch refs/heads/delivery/s1/50\n",
		revListCount:    "0",                               // no commits beyond dev...
		statusPorcelain: " M feature.go\n?? scratch.txt\n", // ...but real uncommitted work
	}
	w := newWorktree(f)
	orphans, err := w.Reconcile(map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || !orphans[0].Unmerged {
		t.Fatalf("dirty orphan must be surfaced as unmerged, got %+v", orphans)
	}
	if f.called("worktree remove") || f.called("branch -D") || f.called("branch -d") {
		t.Error("SAFETY VIOLATION: Reconcile destroyed a worktree with uncommitted work")
	}
}

// A worktree whose only untracked file is the transient status.json telemetry is
// NOT real work — it stays reclaimable when it has no commits beyond dev.
func TestReconcileReclaimsIgnoringStatusJson(t *testing.T) {
	f := &fakeRunner{
		worktreeList:    "worktree /repo/.delivery/worktrees/s1/51\nHEAD b\nbranch refs/heads/delivery/s1/51\n",
		revListCount:    "0",
		statusPorcelain: "?? " + StatusFileName + "\n",
	}
	w := newWorktree(f)
	orphans, err := w.Reconcile(map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || orphans[0].Unmerged {
		t.Fatalf("status.json-only orphan should be reclaimable, got %+v", orphans)
	}
	if !f.called("worktree remove --force") || !f.called("branch -D delivery/s1/51") {
		t.Error("clean orphan (telemetry only) should be reclaimed")
	}
}

// Direct #158 regression (real git): a leftover branch whose work already landed
// on origin/dev out of band (a PR merged remotely) must be RECLAIMED — not
// surfaced as a false unmerged orphan. The local dev ref never fast-forwards, and
// the origin/dev tracking ref is left deliberately stale here, so ONLY Reconcile's
// own fetch + comparison against origin/dev (the fix) classifies it correctly; the
// pre-fix code compared the stale local dev and wrongly preserved every
// remote-merged leftover.
func TestReconcileReclaimsRemotelyMergedLeftover(t *testing.T) {
	repo := t.TempDir()
	run := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run(repo, "init", "-q", "-b", "dev")
	run(repo, "config", "user.email", "e2e@atl.local")
	run(repo, "config", "user.name", "atl")
	if err := os.WriteFile(filepath.Join(repo, "seed.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "add", "-A")
	run(repo, "commit", "-q", "-m", "seed")
	seed := run(repo, "rev-parse", "dev")

	origin := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", "--bare", origin).CombinedOutput(); err != nil {
		t.Fatalf("bare origin init: %v\n%s", err, out)
	}
	run(repo, "remote", "add", "origin", origin)
	run(repo, "push", "-q", "origin", "dev")

	// A worker unit committed real work on its branch.
	root := filepath.Join(repo, ".delivery", "worktrees")
	if err := os.MkdirAll(filepath.Join(root, "s1"), 0o755); err != nil {
		t.Fatal(err)
	}
	wtPath := filepath.Join(root, "s1", "88")
	run(repo, "worktree", "add", "-q", "-b", "delivery/s1/88", wtPath, "dev")
	if err := os.WriteFile(filepath.Join(wtPath, "done.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(wtPath, "add", "-A")
	run(wtPath, "commit", "-q", "-m", "work")

	// The PR merged remotely: the branch's work fast-forwards origin/dev. Then reset
	// this repo's origin/dev tracking ref stale (as if it never fetched), so the
	// fetch inside Reconcile is what discovers the merge.
	run(repo, "push", "-q", "origin", "delivery/s1/88:refs/heads/dev")
	run(repo, "update-ref", "refs/remotes/origin/dev", seed)

	w := &Worktree{RepoDir: repo, Root: root, BaseRef: "dev", RemoteRef: "origin/dev", Run: ExecRunner}
	orphans, err := w.Reconcile(map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || orphans[0].Unmerged {
		t.Fatalf("a remotely-merged leftover must be reclaimed (Unmerged=false), got %+v", orphans)
	}
	if out := run(repo, "branch", "--list", "delivery/s1/88"); out != "" {
		t.Errorf("the reclaimed branch should be deleted, still present: %q", out)
	}
}

// The safety half of the #158 fix (real git): a leftover branch with a committed
// commit that is NOT on origin/dev holds real un-integrated work and MUST be
// preserved+surfaced, never reclaimed — even after Reconcile's own fetch. Proves
// the fetch+RemoteRef switch did not weaken the never-delete-real-work invariant
// (the reclaim half is TestReconcileReclaimsRemotelyMergedLeftover).
func TestReconcilePreservesCommittedUnmergedAgainstRealOrigin(t *testing.T) {
	repo := t.TempDir()
	run := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run(repo, "init", "-q", "-b", "dev")
	run(repo, "config", "user.email", "e2e@atl.local")
	run(repo, "config", "user.name", "atl")
	if err := os.WriteFile(filepath.Join(repo, "seed.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "add", "-A")
	run(repo, "commit", "-q", "-m", "seed")

	origin := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", "--bare", origin).CombinedOutput(); err != nil {
		t.Fatalf("bare origin init: %v\n%s", err, out)
	}
	run(repo, "remote", "add", "origin", origin)
	run(repo, "push", "-q", "origin", "dev")

	// A worker committed real work that NEVER landed on origin/dev.
	root := filepath.Join(repo, ".delivery", "worktrees")
	if err := os.MkdirAll(filepath.Join(root, "s1"), 0o755); err != nil {
		t.Fatal(err)
	}
	wtPath := filepath.Join(root, "s1", "91")
	run(repo, "worktree", "add", "-q", "-b", "delivery/s1/91", wtPath, "dev")
	if err := os.WriteFile(filepath.Join(wtPath, "done.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(wtPath, "add", "-A")
	run(wtPath, "commit", "-q", "-m", "unmerged work")

	w := &Worktree{RepoDir: repo, Root: root, BaseRef: "dev", RemoteRef: "origin/dev", Run: ExecRunner}
	orphans, err := w.Reconcile(map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || !orphans[0].Unmerged {
		t.Fatalf("a committed-unmerged leftover must be preserved (Unmerged=true), got %+v", orphans)
	}
	if out := run(repo, "branch", "--list", "delivery/s1/91"); out == "" {
		t.Error("SAFETY VIOLATION: the unmerged branch was deleted, must be preserved")
	}
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("SAFETY VIOLATION: the unmerged worktree was removed: %v", err)
	}
}

func TestMergedToBase(t *testing.T) {
	// Contained in origin/dev (rev-list 0) → merged; must fetch first + compare
	// against RemoteRef (the local BaseRef is never fast-forwarded, so it is stale).
	f := &fakeRunner{revListCount: "0"}
	w := newWorktree(f)
	merged, err := w.MergedToBase("s1", 42)
	if err != nil {
		t.Fatal(err)
	}
	if !merged {
		t.Error("rev-list 0 (contained in origin/dev) → merged")
	}
	if !f.called("fetch origin dev") {
		t.Error("MergedToBase must fetch the base ref first (local dev is stale)")
	}
	if !f.called("rev-list --count origin/dev..delivery/s1/42") {
		t.Errorf("must compare the branch against origin/dev, not local dev: %v", f.calls)
	}

	f2 := &fakeRunner{revListCount: "2"}
	if merged, _ := newWorktree(f2).MergedToBase("s1", 42); merged {
		t.Error("rev-list >0 (commits not on origin/dev) → NOT merged")
	}
}

// A persisted quarantine from a prior run (same fixed suffix) must not make the
// next quarantine's `branch -m` collide + abort the sprint — the suffix bumps.
func TestQuarantineCollisionBumpsSuffix(t *testing.T) {
	f := &fakeRunner{
		revListCount: "3",                                            // unmerged → preserve path
		branches:     map[string]bool{"delivery/s1/42-stall0": true}, // a prior run's quarantine persists
	}
	w := newWorktree(f)
	w.Root = t.TempDir()
	orphan, err := w.Quarantine("s1", 42, "stall0")
	if err != nil {
		t.Fatal(err)
	}
	if orphan.Branch != "delivery/s1/42-stall0-2" {
		t.Errorf("a colliding suffix should bump; got %q", orphan.Branch)
	}
	if !f.called("branch -m delivery/s1/42 delivery/s1/42-stall0-2") {
		t.Errorf("rename should target the free bumped name: %v", f.calls)
	}
}

func TestQuarantineReclaimsClean(t *testing.T) {
	f := &fakeRunner{revListCount: "0"} // no commits beyond dev, clean tree
	w := newWorktree(f)
	orphan, err := w.Quarantine("s1", 42, "stall0")
	if err != nil {
		t.Fatal(err)
	}
	if orphan.Unmerged {
		t.Error("a clean unit should be reclaimed, not preserved")
	}
	if !f.called("worktree remove --force") || !f.called("branch -D delivery/s1/42") {
		t.Errorf("clean quarantine should reclaim the worktree + branch: %v", f.calls)
	}
	if f.called("worktree move") || f.called("branch -m") {
		t.Error("a clean quarantine must not move/rename — nothing to preserve")
	}
	// #158: classify against the freshly-fetched origin/dev, never the stale local ref.
	if !f.called("fetch origin dev") {
		t.Error("Quarantine must fetch the base before classifying (#158)")
	}
	if !f.called("rev-list --count origin/dev..delivery/s1/42") {
		t.Errorf("Quarantine must classify against origin/dev, not local dev (#158): %v", f.calls)
	}
}

// The load-bearing safety assertion (argv level): a unit with unmerged commits is
// PRESERVED via move+rename, never force-deleted — the retry gets the canonical
// name back while the work survives.
func TestQuarantinePreservesUnmerged(t *testing.T) {
	f := &fakeRunner{revListCount: "3"} // 3 commits not in dev → real work
	w := newWorktree(f)
	w.Root = t.TempDir() // preserve really mkdir's the quarantine parent
	orphan, err := w.Quarantine("s1", 77, "stall0")
	if err != nil {
		t.Fatal(err)
	}
	if !orphan.Unmerged {
		t.Fatal("an unmerged unit must be preserved")
	}
	if f.called("branch -D") || f.called("worktree remove") {
		t.Error("SAFETY VIOLATION: quarantine deleted an unmerged worktree/branch")
	}
	if !f.called("worktree move") || !f.called("branch -m delivery/s1/77 delivery/s1/77-stall0") {
		t.Errorf("unmerged quarantine should move the worktree + rename the branch: %v", f.calls)
	}
	if orphan.Branch != "delivery/s1/77-stall0" {
		t.Errorf("preserved branch = %q, want the suffixed name", orphan.Branch)
	}
}

func TestQuarantinePreservesDirty(t *testing.T) {
	f := &fakeRunner{revListCount: "0", statusPorcelain: " M feature.go\n"} // clean history, dirty tree
	w := newWorktree(f)
	w.Root = t.TempDir() // preserve really mkdir's the quarantine parent
	orphan, err := w.Quarantine("s1", 50, "stall0")
	if err != nil {
		t.Fatal(err)
	}
	if !orphan.Unmerged {
		t.Fatal("a dirty worktree must be preserved as unmerged")
	}
	if f.called("branch -D") || f.called("worktree remove") {
		t.Error("SAFETY VIOLATION: quarantine destroyed a worktree with uncommitted work")
	}
	if !f.called("worktree move") {
		t.Errorf("dirty quarantine should move the worktree aside: %v", f.calls)
	}
}

// Real-git proof that `git worktree move` preserves a DIRTY working tree — the
// branch-hygiene guarantee the fakeRunner argv tests can only assert at the
// command level. A worker killed mid-implement (uncommitted work) must survive
// quarantine, with the canonical delivery/<slug>/<id> name freed for the retry.
func TestQuarantineRealGitPreservesDirtyWorktree(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "dev")
	git("config", "user.email", "e2e@atl.local")
	git("config", "user.name", "atl")
	if err := os.WriteFile(filepath.Join(repo, "seed.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "seed")

	// Give the repo an origin so Quarantine's fetch (the #158 stale-ref fix) works,
	// and classification runs against the freshly-fetched origin/dev — the same
	// real-remote shape production always has.
	origin := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", "--bare", origin).CombinedOutput(); err != nil {
		t.Fatalf("bare origin init: %v\n%s", err, out)
	}
	git("remote", "add", "origin", origin)
	git("push", "-q", "origin", "dev")

	root := filepath.Join(repo, ".delivery", "worktrees")
	if err := os.MkdirAll(filepath.Join(root, "s1"), 0o755); err != nil {
		t.Fatal(err)
	}
	wtPath := filepath.Join(root, "s1", "42")
	git("worktree", "add", "-q", "-b", "delivery/s1/42", wtPath, "dev")
	// A worker killed mid-implement: uncommitted (untracked) work in the tree.
	if err := os.WriteFile(filepath.Join(wtPath, "feature.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := &Worktree{RepoDir: repo, Root: root, BaseRef: "dev", RemoteRef: "origin/dev", Run: ExecRunner}
	orphan, err := w.Quarantine("s1", 42, "stall0")
	if err != nil {
		t.Fatalf("Quarantine on a dirty worktree failed (git worktree move refused a dirty tree?): %v", err)
	}
	if !orphan.Unmerged {
		t.Fatal("the dirty worktree must be preserved as unmerged")
	}
	// The uncommitted work survived at the quarantine location.
	moved := filepath.Join(root, ".quarantine", "s1", "42-stall0", "feature.go")
	if _, err := os.Stat(moved); err != nil {
		t.Errorf("uncommitted work not preserved at quarantine: %v", err)
	}
	// The canonical worktree path is freed for the retry.
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("canonical worktree path should be freed after quarantine")
	}
	// The canonical branch name is free; the quarantined branch carries the work.
	if out, _ := exec.Command("git", "-C", repo, "branch", "--list", "delivery/s1/42").CombinedOutput(); strings.TrimSpace(string(out)) != "" {
		t.Errorf("canonical branch should be renamed away, still present: %s", out)
	}
	if out, _ := exec.Command("git", "-C", repo, "branch", "--list", "delivery/s1/42-stall0").CombinedOutput(); strings.TrimSpace(string(out)) == "" {
		t.Error("the quarantined branch delivery/s1/42-stall0 should exist")
	}
}

func TestReconcileSkipsActive(t *testing.T) {
	f := &fakeRunner{
		worktreeList: "worktree /repo/.delivery/worktrees/s1/5\nHEAD b\nbranch refs/heads/delivery/s1/5\n",
	}
	w := newWorktree(f)
	orphans, err := w.Reconcile(map[string]bool{"delivery/s1/5": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 0 {
		t.Errorf("active branch should be skipped, got %+v", orphans)
	}
	if f.called("rev-list") {
		t.Error("active branch should not even be classified")
	}
}
