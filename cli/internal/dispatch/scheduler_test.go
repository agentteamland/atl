package dispatch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fakeHandle is a test-controlled worker process: the test flips `exited`/`code`
// between steps to drive completion, crash, and stall paths deterministically.
type fakeHandle struct {
	pid    int
	exited bool
	code   int
	stderr string
	sigs   []os.Signal
	waited bool
}

func (h *fakeHandle) Wait() error { h.waited = true; return nil }
func (h *fakeHandle) Exited() (bool, int) {
	if h.exited {
		return true, h.code
	}
	return false, -1
}
func (h *fakeHandle) Signal(s os.Signal) error { h.sigs = append(h.sigs, s); return nil }
func (h *fakeHandle) StderrTail() string       { return h.stderr }
func (h *fakeHandle) PID() int                 { return h.pid }
func (h *fakeHandle) ExitCode() int {
	if h.exited {
		return h.code
	}
	return -1
}

// fakeWorld is the injected environment for a scheduler test: it manufactures a
// fakeHandle per admitted unit (keyed by the unit id parsed from the worktree
// path), serves a controllable status.json per unit, and holds the injected
// clock. The test mutates handles + statuses between step() calls.
type fakeWorld struct {
	root     string
	slug     string
	handles  map[int]*fakeHandle
	statuses map[int]*Status
	fr       *fakeRunner // the injected git seam; tests flip revListCount to drive merge-verify
	now      time.Time
	nextPID  int
	torn     map[int]bool // ids whose status.json read returns a torn/parse error (exists but unparseable)
}

func idFromPath(p string) int {
	id, _ := strconv.Atoi(filepath.Base(p))
	return id
}

func (w *fakeWorld) spawner(spec WorkerSpec) (Handle, error) {
	id := idFromPath(spec.WorktreeDir)
	w.nextPID++
	h := &fakeHandle{pid: w.nextPID}
	w.handles[id] = h
	return h, nil
}

func (w *fakeWorld) readStatus(path string) (*Status, error) {
	id := idFromPath(path)
	if w.torn[id] {
		return nil, errors.New("parse status: unexpected end of JSON input") // exists but unparseable — NOT IsNotExist
	}
	if st, ok := w.statuses[id]; ok && st != nil {
		return st, nil
	}
	return nil, os.ErrNotExist
}

func newTestScheduler(t *testing.T, plan *Plan, cap int) (*Scheduler, *fakeWorld) {
	t.Helper()
	base := t.TempDir()
	fr := &fakeRunner{revListCount: "0"} // clean git: no orphans, nothing unmerged
	wt := &Worktree{
		RepoDir:   base,
		Root:      filepath.Join(base, ".delivery", "worktrees"),
		BaseRef:   "dev",
		RemoteRef: "origin/dev",
		Run:       fr.run,
	}
	world := &fakeWorld{
		root:     wt.Root,
		slug:     plan.SprintSlug,
		handles:  map[int]*fakeHandle{},
		statuses: map[int]*Status{},
		fr:       fr,
		now:      time.Unix(1_000_000, 0),
		torn:     map[int]bool{},
	}
	s := NewScheduler(plan, base, wt, world.spawner, Config{Cap: cap, PollInterval: time.Second, TermGrace: time.Second})
	s.ReadStatus = world.readStatus
	s.Now = func() time.Time { return world.now }
	s.Sleep = func(time.Duration) {}
	return s, world
}

func planOf(slug string, units ...WorkUnit) *Plan {
	return &Plan{SprintSlug: slug, Granularity: GranularityPBI, Units: units}
}

func (s *Scheduler) stateOf(id int) unitState { return s.units[id].state }

// driveUnitDone exits the unit's current-stage worker 0 and steps until the unit
// leaves stateRunning — walking it through the whole developer→tester→tech-lead
// pipeline (each step advances one stage; the final step merge-verifies + completes).
// The spawner installs a fresh handle per stage under the same id, so each iteration
// re-exits the current stage's handle. The completing step also runs admit(), so a
// refilled successor is already running when this returns.
func driveUnitDone(t *testing.T, s *Scheduler, w *fakeWorld, id int) {
	t.Helper()
	for i := 0; i <= len(deliveryPipeline); i++ {
		h := w.handles[id]
		if h == nil {
			t.Fatalf("unit %d has no live handle at stage %d", id, i)
		}
		h.exited = true
		h.code = 0
		if _, err := s.step(); err != nil {
			t.Fatal(err)
		}
		if s.stateOf(id) != stateRunning {
			return
		}
	}
	t.Fatalf("unit %d never left running after %d stage exits", id, len(deliveryPipeline)+1)
}

// --- admission: cap + stack-rank order ---------------------------------------

func TestAdmitRespectsCapAndStackRank(t *testing.T) {
	// Three independent units; the two LOWEST stack-ranks (higher priority) fill
	// the cap-2 pool first; the highest-rank waits.
	plan := planOf("s1",
		WorkUnit{ID: 1, StackRank: 30},
		WorkUnit{ID: 2, StackRank: 10},
		WorkUnit{ID: 3, StackRank: 20},
	)
	s, _ := newTestScheduler(t, plan, 2)

	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(2) != stateRunning || s.stateOf(3) != stateRunning {
		t.Errorf("cap-2 should admit ranks 10 (id2) + 20 (id3); got 2=%v 3=%v", s.stateOf(2), s.stateOf(3))
	}
	if s.stateOf(1) != statePending {
		t.Errorf("rank 30 (id1) should wait for a slot; got %v", s.stateOf(1))
	}
	if s.runningCount() != 2 {
		t.Errorf("exactly cap=2 workers should run, got %d", s.runningCount())
	}
}

// --- merge-gated refill: a successor waits for its predecessor to complete -----

func TestRefillGatesSuccessorOnPredecessorDone(t *testing.T) {
	// B depends on A. Even with free slots, B must not start until A is done —
	// Resolution #8: a successor always branches from a dev containing A's merge.
	plan := planOf("s1",
		WorkUnit{ID: 1, StackRank: 1},                         // A
		WorkUnit{ID: 2, StackRank: 2, Predecessors: []int{1}}, // B ⟵ A
	)
	s, w := newTestScheduler(t, plan, 4)

	// Step 1: only A is admissible; B is gated by its predecessor.
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(1) != stateRunning {
		t.Fatalf("A should be running, got %v", s.stateOf(1))
	}
	if s.stateOf(2) != statePending {
		t.Fatalf("B must NOT start before A is done (merge-gate), got %v", s.stateOf(2))
	}

	// A runs its full pipeline (developer→tester→tech-lead); the completing step
	// refills B. Only A's tech-lead merge makes it done — and only then may B start.
	driveUnitDone(t, s, w, 1)
	if s.stateOf(1) != stateDone {
		t.Errorf("A should be done after its pipeline completes, got %v", s.stateOf(1))
	}
	if s.stateOf(2) != stateRunning {
		t.Errorf("B should be admitted once A is done (refill), got %v", s.stateOf(2))
	}
}

// --- completion tears down the merged worktree --------------------------------

func TestCompleteTearsDownAndFreesSlot(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 7})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil { // admit → developer stage
		t.Fatal(err)
	}
	// The developer + tester exit-0 advance the pipeline WITHOUT completing — neither
	// merges, so the unit stays running and rolls to the next stage.
	for _, wantStage := range []Stage{StageTester, StageTechLead} {
		w.handles[7].exited = true
		w.handles[7].code = 0
		if _, err := s.step(); err != nil {
			t.Fatal(err)
		}
		if s.stateOf(7) != stateRunning {
			t.Fatalf("a non-final stage exit-0 must keep the unit running, got %v", s.stateOf(7))
		}
		if got := deliveryPipeline[s.units[7].stageIdx]; got != wantStage {
			t.Fatalf("expected the %s stage next, got %s", wantStage, got)
		}
	}
	// The tech-lead exit-0 → merge verified → done + terminal.
	w.handles[7].exited = true
	w.handles[7].code = 0
	term, err := s.step()
	if err != nil {
		t.Fatal(err)
	}
	if s.stateOf(7) != stateDone {
		t.Errorf("unit should be done after the tech-lead stage, got %v", s.stateOf(7))
	}
	if !term {
		t.Error("the only unit is done → the run should be terminal")
	}
}

// --- merge verification: exit-0 without a confirmed merge blocks, never done ---

func TestCompleteBlocksWhenNotMerged(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 6})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil { // admit → developer stage
		t.Fatal(err)
	}
	// The branch is unmerged for the WHOLE run (commits beyond origin/dev). The
	// developer + tester exit-0 must still ADVANCE — they never merge, so the engine
	// must NOT merge-verify at a non-final stage (that would falsely block every unit).
	w.fr.revListCount = "2"
	for i := 0; i < 2; i++ {
		w.handles[6].exited = true
		w.handles[6].code = 0
		if _, err := s.step(); err != nil {
			t.Fatal(err)
		}
		if s.stateOf(6) != stateRunning {
			t.Fatalf("a non-final stage must advance even on an unmerged branch (no merge-check yet), got %v", s.stateOf(6))
		}
	}
	// Only the tech-lead exit-0 triggers the merge-verify — and its branch is NOT in
	// origin/dev (it exited 0 without completing the PR). Because Azure's autoComplete
	// is asynchronous, the first check enters an awaiting-merge GRACE (still running,
	// slot held) rather than blocking immediately.
	w.handles[6].exited = true
	w.handles[6].code = 0
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(6) != stateRunning {
		t.Fatalf("tech-lead exit-0 on an unmerged branch should enter merge-grace (still running), got %v", s.stateOf(6))
	}
	// The merge never lands; once the grace window elapses the unverified merge is
	// terminal — the unit blocks (never force-deleted + marked done).
	w.now = w.now.Add(mergeGraceWindow + time.Second)
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(6) != stateBlocked {
		t.Fatalf("tech-lead exit-0 without a verified merge (after grace) must block; got %v", s.stateOf(6))
	}
	assertBlockedReport(t, s.ProjectRoot, 6, "not merged")
}

// TestCompleteMergeGraceThenMerges: a unit that isn't merged at the first check
// but merges within the grace window completes normally (no false block).
func TestCompleteMergeGraceThenMerges(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 8})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil { // admit → developer
		t.Fatal(err)
	}
	// Walk developer → tester → tech-lead; the branch is unmerged the whole time.
	w.fr.revListCount = "2"
	for i := 0; i < len(deliveryPipeline)-1; i++ {
		w.handles[8].exited = true
		w.handles[8].code = 0
		if _, err := s.step(); err != nil {
			t.Fatal(err)
		}
	}
	// tech-lead exits 0, branch not yet merged → grace.
	w.handles[8].exited = true
	w.handles[8].code = 0
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(8) != stateRunning {
		t.Fatalf("should be awaiting merge, got %v", s.stateOf(8))
	}
	// The async merge lands within the window (rev-list now reports 0) → done.
	w.fr.revListCount = "0"
	w.now = w.now.Add(30 * time.Second) // still inside the grace window
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(8) != stateDone {
		t.Fatalf("a merge that lands within grace must complete the unit, got %v", s.stateOf(8))
	}
}

// --- pipeline: the three stages share ONE worktree ----------------------------

func TestPipelineReusesOneWorktreeAcrossStages(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 3})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil { // admit → developer (the ONE Create)
		t.Fatal(err)
	}
	worktreePath := s.units[3].rs.WorktreePath
	// Advance developer→tester→tech-lead; every stage must reuse the same worktree
	// (the tester + tech-lead need the developer's commits), never a fresh checkout.
	for i := 0; i < len(deliveryPipeline)-1; i++ {
		w.handles[3].exited = true
		w.handles[3].code = 0
		if _, err := s.step(); err != nil {
			t.Fatal(err)
		}
		if s.units[3].rs.WorktreePath != worktreePath {
			t.Fatalf("stage %d changed the worktree: got %q want %q", i+1, s.units[3].rs.WorktreePath, worktreePath)
		}
	}
	adds := 0
	for _, c := range w.fr.calls {
		if strings.Contains(strings.Join(c, " "), "worktree add") {
			adds++
		}
	}
	if adds != 1 {
		t.Errorf("the 3-stage pipeline must create the worktree ONCE and reuse it; got %d `worktree add` calls", adds)
	}
}

// --- pipeline: a failure at ANY stage marks the unit blocked (D5) --------------

func TestBlockerAtNonFinalStageBlocksUnit(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 2})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil { // admit → developer
		t.Fatal(err)
	}
	// developer exits clean → advance to the tester stage.
	w.handles[2].exited = true
	w.handles[2].code = 0
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if deliveryPipeline[s.units[2].stageIdx] != StageTester {
		t.Fatalf("expected the tester stage after the developer, got %s", deliveryPipeline[s.units[2].stageIdx])
	}
	// The tester reports a blocker (an un-runnable surface) and exits → mark-blocked
	// (D5: v1 is fail-at-any-stage, no stage-level rework loop).
	w.handles[2].exited = true
	w.handles[2].code = 1
	w.statuses[2] = &Status{Phase: "verify", Blocker: "mobile emulator would not boot"}
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(2) != stateBlocked {
		t.Fatalf("a blocker at the tester stage must block the unit (D5), got %v", s.stateOf(2))
	}
	if s.units[2].retries != 0 {
		t.Error("a worker-reported blocker must not be retried, even mid-pipeline")
	}
	assertBlockedReport(t, s.ProjectRoot, 2, "worker-reported")
}

// --- restart respects a prior run's give-up (durable BlockedReport) ------------

func TestMarkPreviouslyBlockedSkipsUnitOnRestart(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 8})
	s, _ := newTestScheduler(t, plan, 1)
	// A prior run exhausted the ladder and left a durable report on disk.
	if err := WriteBlockedReport(s.ProjectRoot, &BlockedReport{ID: 8, Reason: "prior run gave up"}); err != nil {
		t.Fatal(err)
	}
	sum, err := s.Run()
	if err != nil {
		t.Fatal(err)
	}
	if s.stateOf(8) != stateBlocked {
		t.Fatalf("a unit with a durable BlockedReport should not be re-admitted; got %v", s.stateOf(8))
	}
	if len(sum.Blocked) != 1 || sum.Blocked[0] != 8 {
		t.Errorf("the previously-blocked unit should surface as blocked, got %+v", sum)
	}
}

func TestMarkPreviouslyDoneSkipsUnitOnRestart(t *testing.T) {
	// A prior run completed unit 1 (its branch/worktree torn down, no BlockedReport); the
	// durable run-state checkpointed it done. On restart the engine must NOT re-admit it —
	// a re-run would re-open a Done work-item and open a duplicate/empty PR. RED before
	// markPreviouslyDone existed (unit 1 was re-spawned because done was never reconstructed).
	plan := planOf("s1", WorkUnit{ID: 1})
	s, w := newTestScheduler(t, plan, 1)
	if err := WriteRunState(RunStatePath(s.ProjectRoot), &RunState{SprintSlug: "s1", Done: []int{1}}, w.now); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sum, err := s.RunContext(ctx) // ctx-bounded so a regression (unit re-admitted) fails fast, never hangs the suite
	if err != nil {
		t.Fatal(err)
	}
	if s.stateOf(1) != stateDone {
		t.Fatalf("unit 1 was done in the prior run; restart must seed it done, got %v", s.stateOf(1))
	}
	if _, spawned := w.handles[1]; spawned {
		t.Fatal("unit 1 must NOT be re-admitted / re-spawned on restart")
	}
	if len(sum.Done) != 1 || sum.Done[0] != 1 {
		t.Errorf("the previously-done unit should surface as done, got %+v", sum)
	}
}

func TestMarkPreviouslyDoneGuardsOnSprintSlug(t *testing.T) {
	// A run-state left by a DIFFERENT sprint must never skip this sprint's work.
	plan := planOf("s2", WorkUnit{ID: 1})
	s, w := newTestScheduler(t, plan, 1)
	if err := WriteRunState(RunStatePath(s.ProjectRoot), &RunState{SprintSlug: "s1", Done: []int{1}}, w.now); err != nil {
		t.Fatal(err)
	}
	s.markPreviouslyDone()
	if s.stateOf(1) != statePending {
		t.Fatalf("a run-state from sprint s1 must not seed done in sprint s2; unit 1 should stay pending, got %v", s.stateOf(1))
	}
	// the matching-sprint case DOES seed done.
	plan2 := planOf("s1", WorkUnit{ID: 1})
	s2, w2 := newTestScheduler(t, plan2, 1)
	if err := WriteRunState(RunStatePath(s2.ProjectRoot), &RunState{SprintSlug: "s1", Done: []int{1}}, w2.now); err != nil {
		t.Fatal(err)
	}
	s2.markPreviouslyDone()
	if s2.stateOf(1) != stateDone {
		t.Fatalf("a matching-sprint run-state must seed done; got %v", s2.stateOf(1))
	}
}

// --- abort cleanup: an aborting Run kills every still-running worker -----------

func TestAbortRunningKillsLiveWorkers(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 1}, WorkUnit{ID: 2})
	s, w := newTestScheduler(t, plan, 2)
	if _, err := s.step(); err != nil { // admit both
		t.Fatal(err)
	}
	if s.runningCount() != 2 {
		t.Fatalf("both units should be running, got %d", s.runningCount())
	}
	s.abortRunning()
	for _, id := range []int{1, 2} {
		if !hasSignal(w.handles[id].sigs, syscall.SIGKILL) {
			t.Errorf("unit %d's worker should be SIGKILL'd on abort", id)
		}
		if !w.handles[id].waited {
			t.Errorf("unit %d's worker should be reaped (Wait) on abort", id)
		}
	}
}

func hasSignal(sigs []os.Signal, want os.Signal) bool {
	for _, s := range sigs {
		if s == want {
			return true
		}
	}
	return false
}

// --- worker-reported blocker: honour it, no retry -----------------------------

func TestWorkerReportedBlockerIsTerminalNoRetry(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 5})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	// The worker exits non-zero AND set a blocker → real failure, honour it.
	w.handles[5].exited = true
	w.handles[5].code = 1
	w.statuses[5] = &Status{Phase: "self-test", Blocker: "tests still red after 2 self-fix loops"}
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(5) != stateBlocked {
		t.Fatalf("a worker-reported blocker should mark the unit blocked, got %v", s.stateOf(5))
	}
	if s.units[5].retries != 0 {
		t.Error("a reported blocker must NOT be retried (#5) — retries should stay 0")
	}
	assertBlockedReport(t, s.ProjectRoot, 5, "worker-reported")
}

// --- hard crash: reclaim + retry once, then block -----------------------------

func TestHardCrashRetriesOnceThenBlocks(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 9})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	pid1 := w.handles[9].pid

	// First crash: non-zero exit, no terminal status → retry-once (fresh worker).
	w.handles[9].exited = true
	w.handles[9].code = 137
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(9) != stateRunning {
		t.Fatalf("after a first crash the unit should be retried (running again), got %v", s.stateOf(9))
	}
	if s.units[9].retries != 1 {
		t.Errorf("retry count should be 1, got %d", s.units[9].retries)
	}
	if w.handles[9].pid == pid1 {
		t.Error("the retry should be a FRESH worker (new pid)")
	}

	// Second crash → terminal, mark-blocked.
	w.handles[9].exited = true
	w.handles[9].code = 1
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(9) != stateBlocked {
		t.Fatalf("a second crash should mark the unit blocked, got %v", s.stateOf(9))
	}
	assertBlockedReport(t, s.ProjectRoot, 9, "retry also failed")
}

// --- heartbeat stall: no first heartbeat within the window → recover ----------

func TestNoFirstHeartbeatIsAStall(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 3})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil { // admit at t0
		t.Fatal(err)
	}
	if s.stateOf(3) != stateRunning {
		t.Fatal("unit should be running")
	}
	orig := w.handles[3] // the retry replaces the map entry, so capture the stalled worker now
	// No status.json ever written; advance well past the heartbeat threshold.
	w.now = w.now.Add(20 * time.Minute)
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	// The stalled worker was SIGTERM'd and the unit retried once onto a fresh worker.
	if len(orig.sigs) == 0 {
		t.Error("a stalled worker should be signalled (SIGTERM)")
	}
	if s.units[3].retries != 1 || s.stateOf(3) != stateRunning {
		t.Errorf("first stall → retry-once; got retries=%d state=%v", s.units[3].retries, s.stateOf(3))
	}
}

// --- phase-stall: fresh heartbeat but the phase never advances -----------------

func TestPhaseStallBreachRecovers(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 4})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil { // admit at t0
		t.Fatal(err)
	}
	// Enter the "implement" phase (records PhaseEnteredAt).
	w.now = w.now.Add(1 * time.Minute)
	w.statuses[4] = &Status{Phase: "implement", HeartbeatTS: w.now}
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	orig := w.handles[4] // captured before the retry replaces it

	// Keep the heartbeat FRESH (not a heartbeat breach) but pin the phase past the
	// phase-stall window (30 min general) — a worker looping in place.
	w.now = w.now.Add(35 * time.Minute)
	w.statuses[4] = &Status{Phase: "implement", HeartbeatTS: w.now}
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if len(orig.sigs) == 0 {
		t.Error("a phase-stalled worker should be signalled (SIGTERM), not left looping")
	}
	if s.units[4].retries != 1 {
		t.Errorf("phase-stall → retry-once; got retries=%d", s.units[4].retries)
	}
}

// --- skipped-by-dependency: a blocked unit strands its successors --------------

func TestBlockedPredecessorStrandsSuccessor(t *testing.T) {
	plan := planOf("s1",
		WorkUnit{ID: 1, StackRank: 1},
		WorkUnit{ID: 2, StackRank: 2, Predecessors: []int{1}},
	)
	s, w := newTestScheduler(t, plan, 4)
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	// A reports a blocker and exits → blocked (no retry). B can never become ready.
	w.handles[1].exited = true
	w.handles[1].code = 1
	w.statuses[1] = &Status{Phase: "self-test", Blocker: "unfixable"}
	term, err := s.step()
	if err != nil {
		t.Fatal(err)
	}
	if !term {
		t.Fatal("nothing can run (A blocked, B stranded) → terminal")
	}
	sum := s.summary()
	if len(sum.Blocked) != 1 || sum.Blocked[0] != 1 {
		t.Errorf("A should be blocked, got %v", sum.Blocked)
	}
	if len(sum.SkippedByDep) != 1 || sum.SkippedByDep[0] != 2 {
		t.Errorf("B should be surfaced as skipped-by-dependency, got %v", sum.SkippedByDep)
	}
}

// --- Run(): a full happy path converges to all-done ---------------------------

func TestRunHappyPathAllDone(t *testing.T) {
	plan := planOf("s1",
		WorkUnit{ID: 1, StackRank: 1},
		WorkUnit{ID: 2, StackRank: 2, Predecessors: []int{1}},
		WorkUnit{ID: 3, StackRank: 3},
	)
	s, w := newTestScheduler(t, plan, 2)
	// Every spawned worker exits 0 immediately, so Run converges deterministically.
	s.spawnHook(w)
	sum, err := s.Run()
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Done) != 3 || len(sum.Blocked) != 0 || len(sum.SkippedByDep) != 0 {
		t.Fatalf("all three units should complete, got %+v", sum)
	}
}

// spawnHook wraps the world spawner so every worker is born already-exited-0 —
// turning Run into a deterministic, sleep-free convergence test.
func (s *Scheduler) spawnHook(w *fakeWorld) {
	inner := w.spawner
	s.Spawn = func(spec WorkerSpec) (Handle, error) {
		h, err := inner(spec)
		if err != nil {
			return nil, err
		}
		fh := h.(*fakeHandle)
		fh.exited = true
		fh.code = 0
		return fh, nil
	}
}

func assertBlockedReport(t *testing.T, root string, id int, wantReason string) {
	t.Helper()
	path := filepath.Join(BlockedDir(root), strconv.Itoa(id)+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("blocked report not written for unit %d: %v", id, err)
	}
	if !strings.Contains(string(b), wantReason) {
		t.Errorf("blocked report %d should mention %q, got %s", id, wantReason, b)
	}
}

// --- torn status.json: a transient parse error must not kill a healthy worker --

func TestPollTornStatusDoesNotKill(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 9})
	s, w := newTestScheduler(t, plan, 1)
	if _, err := s.step(); err != nil { // admit → developer running
		t.Fatal(err)
	}
	// The worker's status.json exists but is caught mid-write (unparseable). Even
	// well past the first-heartbeat window, a torn read is a fresh write (mtime
	// moved), so it must NOT be reclaimed as "no first heartbeat".
	w.torn[9] = true
	w.now = w.now.Add(time.Hour)
	if _, err := s.step(); err != nil {
		t.Fatal(err)
	}
	if s.stateOf(9) != stateRunning {
		t.Fatalf("a torn/parse-error status read must keep the worker alive, got %v", s.stateOf(9))
	}
}

// --- ctx cancel: RunContext returns promptly and leaves nothing running --------

func TestRunContextCancelReturns(t *testing.T) {
	plan := planOf("s1", WorkUnit{ID: 1})
	s, _ := newTestScheduler(t, plan, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before Run
	_, err := s.RunContext(ctx)
	if err == nil {
		t.Fatal("a cancelled context should surface a non-nil error")
	}
	if s.runningCount() != 0 {
		t.Errorf("no worker should be left running after a cancel, got %d", s.runningCount())
	}
}
