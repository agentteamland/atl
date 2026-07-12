package dispatch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Scheduler is the deterministic run loop of `atl work dispatch` (#15): it admits
// ready units up to a fixed concurrency cap over the plan's dependency DAG,
// refills on merge-gated completion, and runs the #12 recovery ladder
// (reclaim → retry-once → mark-blocked) on stalls and crashes. It holds ZERO LLM
// context and makes ZERO Azure calls — it observes workers only through the
// status.json each writes and the process handle. Every external effect is an
// injected seam (Spawn, the git Runner inside Worktree, ReadStatus, the clock),
// so the whole loop is deterministic under injection and Layer-A testable with no
// real workers.
type Scheduler struct {
	Plan        *Plan
	ProjectRoot string
	Worktree    *Worktree
	Spawn       Spawner
	Cfg         Config

	// Seams — NewScheduler defaults each to its real implementation; tests inject
	// fakes to make the loop deterministic.
	ReadStatus func(worktreePath string) (*Status, error)
	Now        func() time.Time
	Sleep      func(time.Duration)
	BuildSpec  func(u WorkUnit, stage Stage, worktreeDir string) WorkerSpec
	Log        func(string)

	// deliveryCfg is the project's .delivery/config.json, loaded once at Run()
	// start; nil when no config is present (a plan-only harness) → workers fall
	// back to ambient inheritance (pre-#8 behavior). Used to wire each worker's
	// azureDevOps MCP (scoped to the config's org, D3) + PAT env (#17 / F8-Go).
	deliveryCfg *DeliveryConfig

	units map[int]*unitSched
}

// Config tunes the scheduler. Cap is the concurrency limit (keystone #4, ~4–6);
// PollInterval spaces the supervisor's status/liveness sweep; TermGrace is how
// long a SIGTERM'd worker gets before SIGKILL in the reclaim step; Stall maps a
// worker phase to its (phase-aware) breach thresholds.
type Config struct {
	Cap          int
	PollInterval time.Duration
	TermGrace    time.Duration
	Stall        func(phase string) StallConfig
}

// Defaults for an unset Config field.
const (
	DefaultCap          = 4
	DefaultPollInterval = 5 * time.Second
	DefaultTermGrace    = 30 * time.Second
	pollKillInterval    = 500 * time.Millisecond
	// mergeGraceWindow is how long a unit whose tech-lead exited 0 may sit
	// awaiting-merge before complete() gives up and marks it blocked. Azure's PR
	// autoComplete is asynchronous (the team docs put it at "within ~2 min"), so a
	// single immediate merge-check would false-block a unit that IS about to merge.
	mergeGraceWindow = 2 * time.Minute
)

// unitState is a work-unit's scheduler lifecycle.
type unitState int

const (
	statePending unitState = iota // not yet admitted (waiting on predecessors or a slot)
	stateRunning                  // a worker is live in its worktree
	stateDone                     // merged to dev, worktree reclaimed, slot freed
	stateBlocked                  // terminal failure, surfaced + reported, slot freed
)

// unitSched is the scheduler's per-unit bookkeeping. It is distinct from the
// durable UnitRunState (which mirrors only running units, for restart
// reconciliation); this struct also carries the in-memory handle + retry count.
type unitSched struct {
	unit           WorkUnit
	state          unitState
	stageIdx       int // index into deliveryPipeline of the CURRENTLY running stage (0 = developer)
	handle         Handle
	rs             *UnitRunState // non-nil only while running
	retries        int           // reclaim-and-retry count for the unit (0, then at most 1 per #12)
	mergeWaitUntil time.Time     // grace deadline while awaiting the tech-lead's async merge to land (see complete)
}

// Summary is the terminal outcome of a dispatch run, partitioned by unit fate.
// SkippedByDep are pending units that never became ready because a predecessor
// was blocked — surfaced so no work silently disappears.
type Summary struct {
	Done         []int
	Blocked      []int
	SkippedByDep []int
}

// NewScheduler builds a scheduler over plan, defaulting every seam + Config field
// to its real implementation. The caller wires the real Worktree + Spawner (or
// injects fakes in a test).
func NewScheduler(plan *Plan, projectRoot string, wt *Worktree, spawn Spawner, cfg Config) *Scheduler {
	if cfg.Cap <= 0 {
		cfg.Cap = DefaultCap
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultPollInterval
	}
	if cfg.TermGrace <= 0 {
		cfg.TermGrace = DefaultTermGrace
	}
	if cfg.Stall == nil {
		cfg.Stall = DefaultStallConfig
	}
	s := &Scheduler{
		Plan:        plan,
		ProjectRoot: projectRoot,
		Worktree:    wt,
		Spawn:       spawn,
		Cfg:         cfg,
		ReadStatus:  ReadStatus,
		Now:         time.Now,
		Sleep:       time.Sleep,
		BuildSpec:   DeliveryWorkerSpec(projectRoot),
		Log:         func(string) {},
		units:       make(map[int]*unitSched, len(plan.Units)),
	}
	for _, u := range plan.Units {
		s.units[u.ID] = &unitSched{unit: u, state: statePending}
	}
	return s
}

// Run drives the scheduler to completion under a background context (no
// cancellation). Most callers use this; a caller that wants Ctrl-C / SIGTERM to
// tear the worker pool down cleanly passes a cancellable context to RunContext.
func (s *Scheduler) Run() (Summary, error) {
	return s.RunContext(context.Background())
}

// RunContext drives the scheduler to completion: reconcile leftover worktrees
// from a prior (crashed) run, then loop step() on the poll interval until the
// sprint is terminal or ctx is cancelled, then return the outcome summary.
// Resumability rests on Azure state + branches (#11) — a fresh run owns no live
// process handles, so every leftover delivery/* worktree is reconciled (clean
// reclaimed, unmerged preserved) and the plan is re-driven from durable state
// (workers are idempotent per #16).
//
// ctx cancellation (a terminal Ctrl-C / SIGTERM the CLI translates into a cancel)
// makes the loop return promptly; the deferred abortRunning then SIGKILLs the
// whole worker process-group, so an interrupted supervisor never leaves detached
// `claude -p` workers burning budget with no one watching them.
func (s *Scheduler) RunContext(ctx context.Context) (Summary, error) {
	// On any exit — a cancel, an aborting error, OR normal completion — make sure no
	// worker is left running detached. On normal completion nothing is running, so
	// this is a no-op.
	defer s.abortRunning()

	// Single-instance guard: a second concurrent dispatch for the same project would
	// reconcile with an empty active set and quarantine the live run's worktrees out
	// from under its workers. Fail fast instead.
	lock, err := acquireRunLock(s.ProjectRoot)
	if err != nil {
		return Summary{}, err
	}
	defer lock.Release()

	// Load the project's Azure coordinates once (org/project/pat.ref) so each worker
	// is spawned with an azureDevOps MCP scoped to THIS project's org (D3) + its PAT
	// env (#17). A missing config → nil → workers fall back to ambient inheritance
	// (the Layer-A harness has no config); a malformed config is a real error.
	cfg, err := LoadDeliveryConfig(s.ProjectRoot)
	if err != nil {
		return Summary{}, err
	}
	s.deliveryCfg = cfg

	// Respect a prior run's give-up decisions: a durable BlockedReport means the
	// #12 ladder already exhausted its retries, so skip re-admitting that unit
	// (which would reset its retry budget across the restart).
	s.markPreviouslyBlocked()

	orphans, err := s.Worktree.Reconcile(map[string]bool{})
	if err != nil {
		return Summary{}, err
	}
	for _, o := range orphans {
		s.Log("reconcile: " + o.Branch + " — " + o.Detail)
	}

	for {
		if ctx.Err() != nil {
			// Interrupted — return what we have; the deferred abortRunning tears
			// down every still-running worker so nothing is orphaned.
			s.Log("dispatch interrupted — aborting running workers")
			return s.summary(), ctx.Err()
		}
		terminal, err := s.step()
		if err != nil {
			return Summary{}, err
		}
		if terminal {
			break
		}
		s.Sleep(s.Cfg.PollInterval)
	}
	return s.summary(), nil
}

// step runs one scheduling pass: poll every running worker and act on
// completion / crash / stall, persist the run-state mirror, then refill ready
// units up to the cap. Deterministic given the injected clock + seams. Returns
// true when the sprint is terminal — nothing is running after admission, so any
// still-pending unit is permanently stranded behind a blocked predecessor.
func (s *Scheduler) step() (bool, error) {
	now := s.Now()

	for _, id := range s.sortedUnitIDs() {
		u := s.units[id]
		if u.state != stateRunning {
			continue
		}
		if err := s.poll(u, now); err != nil {
			return false, err
		}
	}

	if err := s.persist(now); err != nil {
		return false, err
	}

	if err := s.admit(now); err != nil {
		return false, err
	}

	return s.runningCount() == 0, nil
}

// poll observes one running unit and dispatches on its terminal/liveness state.
func (s *Scheduler) poll(u *unitSched, now time.Time) error {
	status, statusErr := s.ReadStatus(u.rs.WorktreePath)
	if statusErr == nil {
		u.rs.ObservePhase(status.Phase, status.HeartbeatTS, now)
	}
	exited, code := u.handle.Exited()

	switch {
	case exited && code == 0 && (statusErr != nil || status.Blocker == ""):
		// A pipeline stage worker finished cleanly. Advance to the next stage
		// (developer→tester→tech-lead) — or, after the FINAL stage, run merge-gated
		// completion (Resolution #8): the supervisor verifies the tech-lead's merge
		// landed on origin/dev before tearing down. A non-final stage never triggers
		// the merge-verify (only the tech-lead merges), which is exactly the seam
		// §7 deferred to this window.
		return s.advanceOrComplete(u, now)

	case exited && statusErr == nil && status.Blocker != "":
		// The worker deliberately reported a blocker (#5 real-test-failure /
		// azure-auth row) and exited — honour it, no retry.
		return s.block(u, now, "worker-reported: "+status.Blocker, status.Phase, status.LastOutputSummary, u.handle.StderrTail())

	case exited:
		// Hard crash: non-zero exit, no terminal status → reclaim + retry-once.
		return s.recover(u, now, fmt.Sprintf("worker crashed (exit %d)", code))

	default:
		// Still running — liveness (#12).
		if statusErr != nil {
			if os.IsNotExist(statusErr) {
				// No status.json yet. Before the first heartbeat this is expected, but
				// a worker that never produces one within the heartbeat window is wedged.
				if now.Sub(u.rs.CreatedAt) > s.Cfg.Stall("").HeartbeatThreshold {
					return s.recover(u, now, "stalled: no first heartbeat")
				}
				return nil
			}
			// A torn/partial read of an EXISTING status.json (a write caught mid-flight):
			// the file exists, so the worker just wrote it — mtime moved, which IS the
			// heartbeat. Treat it as alive this poll and re-read next time; never conflate
			// a transient parse error with a missing first heartbeat (which would kill a
			// healthy long-running worker).
			return nil
		}
		if breach := Classify(status, u.rs.PhaseEnteredAt, now, s.Cfg.Stall(status.Phase)); breach != Alive {
			return s.recover(u, now, "stalled: "+breach.String())
		}
		return nil
	}
}

// advanceOrComplete routes a stage worker's clean exit: advance to the next
// pipeline stage if any remain, else finalize the unit (merge-verify + teardown)
// after the tech-lead stage. Only the final stage runs complete(), so a developer
// or tester exit-0 — neither of which merges — never mark-blocks on an unmerged
// branch (the §7 pitfall the single-worker model would have hit).
func (s *Scheduler) advanceOrComplete(u *unitSched, now time.Time) error {
	if u.stageIdx < len(deliveryPipeline)-1 {
		return s.advanceStage(u, now)
	}
	return s.complete(u, now)
}

// complete finalizes a worker that exited 0. It does NOT trust the exit code as
// proof of merge — the PR-A branch-hygiene lesson applied to the merge itself:
// the worker is a non-deterministic LLM, so the supervisor VERIFIES against the
// durable git state (MergedToBase) before it force-deletes anything.
//
//   - branch confirmed contained in origin/dev → tear down, mark done.
//   - branch NOT merged (exited 0 without completing merge-to-dev) → preserve +
//     mark-BLOCKED (never force-deleted, never counted done), so no unmerged work
//     is lost and no successor is admitted onto work that isn't on dev (Res #8).
//   - merged but the worktree still holds uncommitted leftovers → done, but the
//     leftover is preserved in place, never force-removed.
func (s *Scheduler) complete(u *unitSched, now time.Time) error {
	merged, err := s.Worktree.MergedToBase(s.Plan.SprintSlug, u.unit.ID)
	if err != nil {
		return err
	}
	if !merged {
		// The merge is asynchronous (the tech-lead set PR autoComplete, which lands
		// out-of-band), so a not-yet-merged branch here is usually about-to-merge, not
		// a real failure. Hold the unit in an awaiting-merge grace — it stays running
		// (its slot held, so no successor is admitted onto un-integrated work) and the
		// NEXT poll re-checks MergedToBase — and only mark it blocked once the grace
		// window elapses without the merge landing (a recoverable false-block at worst).
		if u.mergeWaitUntil.IsZero() {
			u.mergeWaitUntil = now.Add(mergeGraceWindow)
			s.Log(fmt.Sprintf("unit %d exited 0; awaiting async merge to %s (grace %s)", u.unit.ID, s.Worktree.BaseRef, mergeGraceWindow))
			return nil
		}
		if now.Before(u.mergeWaitUntil) {
			return nil // still within grace — re-check next poll
		}
		phase, summary := u.rs.Phase, ""
		if st, e := s.ReadStatus(u.rs.WorktreePath); e == nil {
			phase, summary = st.Phase, st.LastOutputSummary
		}
		return s.block(u, now, "worker exited 0 but its branch is not merged to "+s.Worktree.BaseRef+" after the merge-grace window",
			phase, summary, u.handle.StderrTail())
	}

	dirty, derr := s.Worktree.hasUncommittedWork(u.rs.WorktreePath)
	if derr != nil {
		dirty = true // a bad read is not a licence to force-delete
	}
	if dirty {
		s.Log(fmt.Sprintf("unit %d done (merged to %s) but left uncommitted changes at %s — preserved; inspect + remove when reviewed", u.unit.ID, s.Worktree.BaseRef, u.rs.WorktreePath))
		u.state = stateDone
		s.finishRunning(u)
		return nil
	}
	if err := s.Worktree.Teardown(s.Plan.SprintSlug, u.unit.ID); err != nil {
		return err
	}
	s.Log(fmt.Sprintf("unit %d done — merged to %s, worktree reclaimed", u.unit.ID, s.Worktree.BaseRef))
	u.state = stateDone
	s.finishRunning(u)
	return nil
}

// recover runs the #12 ladder for a stalled or crashed worker: kill it, then
// either reclaim-and-retry-once (first failure) or mark-blocked (second).
func (s *Scheduler) recover(u *unitSched, now time.Time, reason string) error {
	stderrTail := u.handle.StderrTail()
	s.kill(u)

	// Snapshot the last status BEFORE Quarantine may move the worktree.
	phase, summary := "", ""
	if st, e := s.ReadStatus(u.rs.WorktreePath); e == nil {
		phase, summary = st.Phase, st.LastOutputSummary
	}

	if u.retries == 0 {
		orphan, err := s.Worktree.Quarantine(s.Plan.SprintSlug, u.unit.ID, "stall0")
		if err != nil {
			return err
		}
		s.Log(fmt.Sprintf("unit %d %s — reclaimed (%s); retrying once", u.unit.ID, reason, orphan.Detail))
		u.retries = 1
		u.state = statePending // re-admitted this same step onto a fresh worktree off dev
		s.finishRunning(u)
		return nil
	}
	return s.block(u, now, reason+" (retry also failed)", phase, summary, stderrTail)
}

// block is the terminal mark-blocked step: quarantine-or-reclaim the worktree
// (branch-hygiene), write the durable BlockedReport for a ceremony skill to
// reflect to Azure, free the slot, and surface it. The worker process is already
// dead here (it exited, or recover killed it).
func (s *Scheduler) block(u *unitSched, now time.Time, reason, phase, summary, stderrTail string) error {
	orphan, err := s.Worktree.Quarantine(s.Plan.SprintSlug, u.unit.ID, fmt.Sprintf("blocked%d", u.retries))
	if err != nil {
		return err
	}
	report := &BlockedReport{
		ID:          u.unit.ID,
		Branch:      orphan.Branch,
		Reason:      reason,
		Phase:       phase,
		LastSummary: summary,
		StderrTail:  stderrTail,
		Preserved:   orphan.Unmerged,
		BlockedAt:   now,
	}
	if orphan.Unmerged {
		report.WorktreePath = orphan.Path
	}
	if err := WriteBlockedReport(s.ProjectRoot, report); err != nil {
		return err
	}
	s.Log(fmt.Sprintf("unit %d BLOCKED — %s (%s)", u.unit.ID, reason, orphan.Detail))
	u.state = stateBlocked
	s.finishRunning(u)
	return nil
}

// kill sends SIGTERM, waits up to TermGrace for the process to exit, then SIGKILL
// + reaps — so the worktree move that follows never races a live process. The
// wait is a bounded poll (deterministic under an injected clock + Sleep).
func (s *Scheduler) kill(u *unitSched) {
	if u.handle == nil {
		return
	}
	if exited, _ := u.handle.Exited(); exited {
		return // already dead (the crash path)
	}
	_ = u.handle.Signal(syscall.SIGTERM)
	polls := int(s.Cfg.TermGrace/pollKillInterval) + 1
	for i := 0; i < polls; i++ {
		if exited, _ := u.handle.Exited(); exited {
			return
		}
		s.Sleep(pollKillInterval)
	}
	_ = u.handle.Signal(syscall.SIGKILL)
	_ = u.handle.Wait()
}

// admit fills free concurrency slots with the highest-ranked ready units. Ready()
// gives the DAG frontier (all predecessors done) ordered by StackRank; the
// scheduler layers the cap + the running/blocked filter on top (#15 step 3/4).
func (s *Scheduler) admit(now time.Time) error {
	slots := s.Cfg.Cap - s.runningCount()
	if slots <= 0 {
		return nil
	}
	done := s.doneSet()
	for _, u := range Ready(s.Plan, done) {
		if slots == 0 {
			break
		}
		us := s.units[u.ID]
		if us.state != statePending {
			continue // Ready still lists running/blocked units (not done) — skip them
		}
		if err := s.spawnUnit(us, now); err != nil {
			return err
		}
		slots--
	}
	return nil
}

// spawnUnit admits a pending unit: it creates the unit's worktree off dev ONCE and
// starts the FIRST pipeline stage (the developer). The tester + tech-lead stages
// reuse this same worktree via advanceStage — they work over the developer's branch,
// not a fresh checkout (pr-and-review §1). Re-admission after a #12 retry re-enters
// here, resetting the pipeline to stage 0 on a fresh worktree.
func (s *Scheduler) spawnUnit(us *unitSched, now time.Time) error {
	path, err := s.Worktree.Create(s.Plan.SprintSlug, us.unit.ID)
	if err != nil {
		return err
	}
	us.stageIdx = 0 // (re-)admission always re-drives the pipeline from the developer
	if err := s.spawnStage(us, path, now); err != nil {
		return err
	}
	s.Log(fmt.Sprintf("unit %d admitted — %s worker pid %d in %s", us.unit.ID, deliveryPipeline[0], us.handle.PID(), path))
	return nil
}

// advanceStage moves a unit to its next pipeline stage after the current stage's
// worker exited 0. It spawns the next stage's worker in the SAME worktree (no new
// Create — the tester/tech-lead need the developer's commits in place), gives the
// fresh stage its own liveness clock, and clears the finished stage's status.json so
// the next stage's first-heartbeat window starts clean. The #12 retry budget is NOT
// reset here: it is a unit-level budget (a crash re-drives the WHOLE pipeline from the
// developer on a fresh worktree, since recover can't preserve mid-pipeline work), so
// resetting per stage would let a unit retry indefinitely.
func (s *Scheduler) advanceStage(us *unitSched, now time.Time) error {
	worktreePath := us.rs.WorktreePath
	us.stageIdx++
	// The worktree is reused, so the prior stage's telemetry would otherwise linger
	// and mislead the new stage's liveness read; drop it (best-effort).
	_ = os.Remove(filepath.Join(worktreePath, StatusFileName))
	if err := s.spawnStage(us, worktreePath, now); err != nil {
		return err
	}
	s.Log(fmt.Sprintf("unit %d → %s stage — worker pid %d (same worktree %s)", us.unit.ID, deliveryPipeline[us.stageIdx], us.handle.PID(), worktreePath))
	return nil
}

// spawnStage builds + starts the current stage's worker in worktreePath and records
// its run-state. Shared by spawnUnit (stage 0, fresh worktree) and advanceStage
// (later stages, reused worktree). The per-worker Azure wiring (D3 / #17) is applied
// here so EVERY stage's worker — not just the developer — gets the project-scoped
// azureDevOps MCP + PAT env: the tester attaches evidence and the tech-lead completes
// the PR + sets Done, both over Azure. Skipped when there's no config (ambient
// inheritance, the pre-#8 / Layer-A path).
func (s *Scheduler) spawnStage(us *unitSched, worktreePath string, now time.Time) error {
	id := us.unit.ID
	stage := deliveryPipeline[us.stageIdx]
	spec := s.BuildSpec(us.unit, stage, worktreePath)
	if s.deliveryCfg != nil {
		mcpPath, mErr := writeMCPConfig(s.ProjectRoot, s.deliveryCfg.Org, id)
		if mErr != nil {
			return mErr
		}
		spec.MCPConfigPath = mcpPath
		spec.ExtraEnv = append(spec.ExtraEnv, deliveryWorkerEnv(s.deliveryCfg)...)
	}
	handle, err := s.Spawn(spec)
	if err != nil {
		return err
	}
	us.handle = handle
	us.state = stateRunning
	us.rs = &UnitRunState{
		ID:             id,
		Branch:         BranchName(s.Plan.SprintSlug, id),
		WorktreePath:   worktreePath,
		Stage:          string(stage),
		PID:            handle.PID(),
		CreatedAt:      now,
		PhaseEnteredAt: now,
	}
	return nil
}

// persist writes the durable run-state mirror of the currently-running units
// (restart reconciliation, #11/#12).
func (s *Scheduler) persist(now time.Time) error {
	rs := &RunState{SprintSlug: s.Plan.SprintSlug, Units: make(map[int]*UnitRunState)}
	for id, u := range s.units {
		if u.state == stateRunning && u.rs != nil {
			rs.Units[id] = u.rs
		}
	}
	return WriteRunState(RunStatePath(s.ProjectRoot), rs, now)
}

func (s *Scheduler) finishRunning(u *unitSched) {
	u.handle = nil
	u.rs = nil
}

// abortRunning SIGKILLs (process-group, so grandchildren die too) and reaps every
// worker still marked running. Deferred by Run() so an aborting error never leaves
// detached claude -p workers burning budget with no supervisor. On normal
// completion nothing is running, so it is a no-op.
func (s *Scheduler) abortRunning() {
	for _, id := range s.sortedUnitIDs() {
		u := s.units[id]
		if u.state == stateRunning && u.handle != nil {
			_ = u.handle.Signal(syscall.SIGKILL)
			_ = u.handle.Wait() // WaitDelay-bounded, so this never hangs the abort
		}
	}
}

// markPreviouslyBlocked marks any unit with a durable BlockedReport from a prior
// run as blocked, so a restart does not re-admit a unit the #12 ladder already
// exhausted (which would reset its retry budget). The report persists until a
// ceremony skill reflects it to Azure and clears it.
func (s *Scheduler) markPreviouslyBlocked() {
	entries, err := os.ReadDir(BlockedDir(s.ProjectRoot))
	if err != nil {
		return // no blocked dir yet — nothing to skip
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		if u, ok := s.units[id]; ok && u.state == statePending {
			u.state = stateBlocked
			s.Log(fmt.Sprintf("unit %d was blocked in a prior run (%s) — skipping; clear the report to retry",
				id, filepath.Join(BlockedDir(s.ProjectRoot), e.Name())))
		}
	}
}

func (s *Scheduler) runningCount() int {
	n := 0
	for _, u := range s.units {
		if u.state == stateRunning {
			n++
		}
	}
	return n
}

func (s *Scheduler) doneSet() map[int]bool {
	d := make(map[int]bool, len(s.units))
	for id, u := range s.units {
		if u.state == stateDone {
			d[id] = true
		}
	}
	return d
}

func (s *Scheduler) sortedUnitIDs() []int {
	ids := make([]int, 0, len(s.units))
	for id := range s.units {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func (s *Scheduler) summary() Summary {
	var sum Summary
	for _, id := range s.sortedUnitIDs() {
		switch s.units[id].state {
		case stateDone:
			sum.Done = append(sum.Done, id)
		case stateBlocked:
			sum.Blocked = append(sum.Blocked, id)
		default: // pending at terminal = never ready (a blocked predecessor stranded it)
			sum.SkippedByDep = append(sum.SkippedByDep, id)
		}
	}
	return sum
}

// Stage is one step of a work-unit's delivery pipeline. Each unit runs the stages
// in order, each a fresh `claude -p` worker in the SAME worktree (pr-and-review §1):
// the developer implements + opens the PR, the tester runs independent Level-2
// verification, and the tech-lead reviews + — on green — completes the Azure PR
// (= the merge to dev) + sets Done. The engine advances stage on a worker's exit-0
// and only runs the merge-verify (complete()) after the final stage.
type Stage string

const (
	StageDeveloper Stage = "developer"
	StageTester    Stage = "tester"
	StageTechLead  Stage = "tech-lead"
)

// deliveryPipeline is the fixed ordered pipeline every work unit runs. D5 (v1): a
// failure at ANY stage → mark-blocked; the stage-level rework loop is deferred.
var deliveryPipeline = []Stage{StageDeveloper, StageTester, StageTechLead}

// DeliveryWorkerSpec builds a unit's worker invocation for a given pipeline stage: a
// `claude -p` worker (developer | tester | tech-lead) in the unit's worktree, running
// that stage's slice of the delivery micro-loop. It closes over the project root so
// each stage's prompt can point at its reflected role-agent + packs by absolute path
// (the worktree lives under root, so a root-anchored path is readable regardless of the
// worktree's own checkout).
//
// Fork A (worker-fetches-from-Azure): the plan carries only the work-item id, so each
// assembled prompt directs the worker to fetch the rest — brief, area tag, pack, wiki,
// PR — from Azure over its MCP at runtime. This keeps the engine zero-Azure and the
// plan contract minimal. This builder returns the base spec (prompt + worktree); the
// scheduler's spawnStage augments MCPConfigPath + ExtraEnv per-worker from
// .delivery/config.json (D3 / #17 — an azureDevOps MCP scoped to the project's org so a
// worker can't inherit the operator's global MCP, plus the PAT env the MCP server +
// az-attach.sh consume). The PAT never enters the argv (spawn.go), so it can't leak
// into logs. The real runtime wiring (MCP reachable, PAT format accepted by the live
// server) is what the stone-#9 Layer-B / #17 run validates; this assembles it.
func DeliveryWorkerSpec(root string) func(u WorkUnit, stage Stage, worktreeDir string) WorkerSpec {
	return func(u WorkUnit, stage Stage, worktreeDir string) WorkerSpec {
		return WorkerSpec{
			Prompt:      deliveryStagePrompt(u, root, stage),
			WorktreeDir: worktreeDir,
		}
	}
}

// deliveryStagePrompt assembles the headless prompt for one pipeline stage. Each
// stage points the worker at its authoritative role-agent (developer/tester/tech-lead)
// and inlines that stage's load-bearing invariants as a self-documenting safety net —
// it never duplicates the agent's role-craft, which would drift. The stage's role name
// ("delivery-team developer|tester|tech-lead") in the opening line is also the token
// the Layer-A fake worker keys off to simulate each stage.
func deliveryStagePrompt(u WorkUnit, root string, stage Stage) string {
	switch stage {
	case StageTester:
		return deliveryTesterPrompt(u, root)
	case StageTechLead:
		return deliveryTechLeadPrompt(u, root)
	default:
		return deliveryDeveloperPrompt(u, root)
	}
}

// deliveryDeveloperPrompt is the stage-1 prompt: the developer agent (+ its children/)
// is the authoritative manual, so the prompt points the worker at it, names the one
// work-item, and inlines the load-bearing invariants (fetch-from-Azure, the six phases,
// block-never-fake, job-ends-at-PR) as a safety net.
func deliveryDeveloperPrompt(u WorkUnit, root string) string {
	agentDir := filepath.Join(root, ".claude", "agents", "developer")
	configPath := filepath.Join(root, ".delivery", "config.json")
	packsDir := filepath.Join(root, ".claude", "packs")
	return fmt.Sprintf(`You are the delivery-team developer — a fresh, isolated worker spawned for exactly one Azure work-item. Your complete operating manual is the developer agent at %[2]s/agent.md and every file under %[2]s/children/. Read them first and follow them exactly; they are authoritative over the summary below.

Assignment: Azure work-item #%[1]d — %[4]q. Run your micro-loop to turn it into a green, review-ready pull request in this worktree, then stop.

Ground rules (your agent manual holds the full detail):
- Read %[3]s for the Azure org/project/repo, branchPair, and pat.ref. Reach Azure ONLY through the azureDevOps MCP — never a raw call, never an invented tool name, and never a hardcoded state literal (resolve state and type at runtime via wit_get_work_item_type).
- The engine handed you only this work-item's id; fetch everything else from Azure at runtime: claim the item (transition it to the runtime-resolved in-progress state plus a claim comment), then read the work-item, its **[Technical Analysis]** sentinel comment, and the tech-lead's **[Canonical Brief]** sentinel comment (both matched by their exact first-line sentinel via wit_list_work_item_comments, never "the newest comment"); resolve your area:<name> tag and load ONLY that area's pack under %[5]s/<area>/; read the brief-named Architecture/ and Conventions/ wiki pages.
- Before anything else, write status.json with a starting phase and a fresh heartbeat — a worker that writes no status.json within a few minutes is reclaimed as stalled. Then run your six phases, in order: claim -> plan -> implement -> self-test -> comment -> pr, writing a fresh phase + heartbeat to status.json on each and at least every couple of minutes.
- Self-test every surface the unit touches and attach evidence via scripts/az-attach.sh. A surface you could not run is UNVERIFIED — set status.json blocker and stop; never fake a green.
- Your job ENDS at the pull request: do NOT review your own PR, do NOT merge, and do NOT set the work-item Done — the tester verifies next, then the tech-lead reviews and, on green, completes the Azure PR (= the merge to dev) and sets Done; the engine only verifies the merge landed.`,
		u.ID, agentDir, configPath, u.Title, packsDir)
}

// deliveryTesterPrompt is the stage-2 prompt: independent Level-2 verification over the
// developer's branch. It points at the tester agent (children/verification-blueprint.md
// is the operative file) and inlines the re-derive-intent-fresh, evidence-attach, and
// hard-boundary invariants (owns tests, not code/review/state).
func deliveryTesterPrompt(u WorkUnit, root string) string {
	agentDir := filepath.Join(root, ".claude", "agents", "tester")
	configPath := filepath.Join(root, ".delivery", "config.json")
	return fmt.Sprintf(`You are the delivery-team tester — a fresh, isolated worker spawned to independently verify one Azure work-item that the developer has just turned into a pull request in this worktree. Your complete operating manual is the tester agent at %[2]s/agent.md and every file under %[2]s/children/ (start with children/verification-blueprint.md). Read them first and follow them exactly; they are authoritative over the summary below.

Assignment: Azure work-item #%[1]d — %[4]q. Run your Level-2 verification micro-loop over the developer's branch in this worktree, then stop.

Ground rules (your agent manual holds the full detail):
- Read %[3]s for the Azure org/project/repo and pat.ref. Reach Azure ONLY through the azureDevOps MCP — never a raw call, never an invented tool name, and never a hardcoded state literal.
- Re-derive intent FRESH from Azure — never inherit the developer's: read the work-item (System.Description acceptance criteria = the spec), its **[Technical Analysis]** sentinel comment, and the tech-lead's **[Canonical Brief]** sentinel comment (both matched by their exact first-line sentinel via wit_list_work_item_comments, never "the newest comment"); resolve the area:<name> tag.
- Build a risk-ranked strategy, then run the test-gates on the right surface (code; web via the preview MCP; or — only after acquiring the serialized single-slot emulator lease with a bootability preflight — mobile) and hunt edges + regression.
- Attach every piece of evidence to the work-item via scripts/az-attach.sh, then emit ONE verdict comment (pass/fail + criteria covered + edges probed + evidence pointers). A surface you could NOT run is UNVERIFIED — set status.json blocker and stop; never fake a green.
- Your boundaries: do NOT write or fix implementation code, do NOT judge code quality or architecture (that is the tech-lead), do NOT transition the work-item state, do NOT open or merge a PR. You own the test half of green = tests then review; the tech-lead reviews next and, on green, completes the PR (= the merge to dev) and sets Done; the engine only verifies the merge.
- Write status.json with a starting phase and a fresh heartbeat as your FIRST action — before you read your manual or re-derive intent — because a worker that writes no status.json within a few minutes is reclaimed as stalled; then keep it fresh on every step, at least every couple of minutes.`,
		u.ID, agentDir, configPath, u.Title)
}

// deliveryTechLeadPrompt is the stage-3 prompt: the single review gate + the closer.
// It points at the tech-lead agent (children/review-craft.md is the operative file) and
// inlines the ordered test-gate-first, the delivery-native-on-the-Azure-PR review with
// refute-to-keep, and the on-green complete-then-Done contract (§1/§4/§5).
func deliveryTechLeadPrompt(u WorkUnit, root string) string {
	agentDir := filepath.Join(root, ".claude", "agents", "tech-lead")
	configPath := filepath.Join(root, ".delivery", "config.json")
	return fmt.Sprintf(`You are the delivery-team tech-lead — a fresh, isolated worker spawned to review one Azure work-item's pull request and, on green, land it. Your complete operating manual is the tech-lead agent at %[2]s/agent.md and every file under %[2]s/children/ (the operative file for THIS stage is children/review-craft.md). Read them first and follow them exactly; they are authoritative over the summary below.

Assignment: Azure work-item #%[1]d — %[4]q. Review its pull request to dev, then either return it with findings or land it, then stop.

Ground rules (your agent manual holds the full detail):
- Read %[3]s for the Azure org/project/repo and pat.ref. Reach Azure ONLY through the azureDevOps MCP — never a raw call, never an invented tool name, and never a hardcoded state literal (resolve state and type at runtime via wit_get_work_item_type).
- Test-gate FIRST (green = tests then review, in that order): confirm the required evidence is ATTACHED to the work-item — code, web if a web surface, and the mobile-emulator run if a mobile surface (a MUST; missing = NOT green, full stop). Read it back via wit_get_work_item_attachment and confirm it matches the changed surfaces. Evidence missing → return the unit via a PR thread; do NOT weigh the diff.
- Then run the delivery-native review ON THE AZURE PR (never /create-pr): a generic baseline read + your tech-lead specialist read (against the Architecture/ boundaries, Conventions/, ADRs, and the AC + Scope you own) + a refute-to-keep pass — every finding needs a file:line / grep / failing-test anchor or it is DROPPED; each survivor is actively refuted and kept only if refutation fails. Raise surviving findings as PR threads (repo_create_pull_request_thread / repo_reply_to_comment).
- On green: vote (repo_vote_pull_request), then COMPLETE the Azure PR (repo_update_pull_request with autoComplete + mergeStrategy NoFastForward — never Rebase or Squash: both rewrite the branch's commit SHAs, so the supervisor's deterministic merge-verify (branch reachable from dev) would not recognize the landed merge and would false-block the unit — and transitionWorkItems:false); completing the PR IS the merge to dev. Then set the work-item to the runtime-resolved Done (wit_get_work_item_type, never a literal). Merge first, then Done.
- If the review is not green, hand the findings back as PR threads and set a status.json blocker rather than merging — never fake a green. The engine only VERIFIES your merge landed on dev; it never merges for you.
- Write status.json with a starting phase and a fresh heartbeat as your FIRST action — before you read your manual or re-derive intent — because a worker that writes no status.json within a few minutes is reclaimed as stalled; then keep it fresh on every step, at least every couple of minutes.`,
		u.ID, agentDir, configPath, u.Title)
}
