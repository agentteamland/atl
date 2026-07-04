package dispatch

import (
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
