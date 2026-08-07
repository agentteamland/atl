package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/agentteamland/atl/cli/internal/detach"
	"github.com/agentteamland/atl/cli/internal/dispatch"
	"github.com/agentteamland/atl/cli/internal/sprintsignal"
	"github.com/agentteamland/atl/cli/internal/throttle"
	"github.com/agentteamland/atl/cli/internal/transcript"
	"github.com/spf13/cobra"
)

const (
	// envNoSprintSignal turns the whole mechanism off — both the printed report
	// and the detached board read behind it. Every other automatic user-facing
	// behaviour here carries a brake (ATL_NO_SELF_UPDATE, ATL_NO_TEAM_UPDATE,
	// ATL_NO_RETRIEVE_INDEX, ATL_NO_CAPTURE_WATCHDOG, ATL_NO_STORE_GIT,
	// ATL_NO_SWEEP_DISPATCH); this one spawns a process that hits the network on
	// someone else's board, so it ships with one from the start rather than
	// acquiring it later the way the sweep signal did.
	envNoSprintSignal = "ATL_NO_SPRINT_SIGNAL"

	// sprintScanInterval bounds how often the board read fires, matching the
	// binary self-update and the team update. A sprint boundary is a
	// once-per-week-ish event; a day-old answer is a good one.
	sprintScanInterval = 24 * time.Hour

	// sprintVerdictMaxAge is how long a cached verdict may still be printed from.
	// Deliberately much larger than the scan interval — a machine offline for a
	// few days should keep its answer — and deliberately finite, so a board read
	// that has silently stopped working stops speaking instead of asserting a
	// months-old board state forever.
	sprintVerdictMaxAge = 7 * 24 * time.Hour

	// sprintScanTimeout bounds the detached gh call. Nothing waits on it, which
	// is exactly why it needs a bound: a hung read would otherwise hold the
	// throttle's promise (a scan ran today) without ever writing a verdict.
	sprintScanTimeout = 2 * time.Minute
)

// sprintScanSpawn detaches `atl work sprint-scan`. A package var so tests can
// observe the throttle/brake decision without forking a real process — the same
// seam teamUpdateSpawn uses.
var sprintScanSpawn = func() error { return detach.Spawn("work", "sprint-scan") }

// sprintSessionSignal reports, at session start, that a board-backed project has
// open work and no active sprint.
//
// It is a REPORT, not a dispatch, and the wording says so. The sweep signals name
// an action for an agent to take in a background subagent because a sweep can be
// finished from durable state alone; this one cannot. Which work enters a sprint
// is the product owner's call, so the whole remedy is that a person hears about
// it — an agent that "handled" this by opening a sprint would have made the
// decision the signal exists to hand over.
//
// Shape is the one session-start already uses for network work: the printing path
// touches only the local cache and the working tree, and the board read runs
// detached behind a throttle. A signal has to reach the CURRENT session, so the
// network can never be on its path.
func sprintSessionSignal(project string) {
	if project == "" || os.Getenv(envNoSprintSignal) != "" {
		return
	}
	cfg, err := dispatch.LoadDeliveryConfig(project)
	if err != nil || cfg == nil {
		return // no board backend here, or a malformed config — stay silent
	}
	if !sprintSignalCovers(cfg, project) {
		return
	}
	// Skip git worktrees, and note this is not cosmetic: .delivery/config.json and
	// methodology.json are COMMITTED, so every worktree `atl work dispatch` cuts
	// for a work-unit carries them and every worker's session-start would land
	// here. That is wrong twice over — N workers would each schedule their own
	// board read against one board (each worktree is a new path, so the
	// per-project throttle does not merge them), and the report would print into
	// an autonomous worker's context, where there is no product owner to raise it
	// with and the worker has exactly one unit to build. The signal belongs in the
	// session a person is sitting in. The git exec runs only after the cheap file
	// reads above, same ordering as autoIndexRetrieval.
	if inGitWorktree(project) {
		return
	}

	// Print first, from the cache a previous session's scan left behind.
	if path, perr := sprintVerdictPath(project); perr == nil {
		if v, ok := sprintsignal.Load(path, sprintVerdictMaxAge, time.Now()); ok &&
			v.OpenItems > 0 && v.NoActiveSprint(project) {
			fmt.Println(sprintSignalNotice(v.OpenItems, v.HighestSprint))
		}
	}
	// Then refresh for the next one.
	scheduleSprintScan(project)
}

// sprintSignalNotice is the reported line.
//
// It names the ceremony that opens a sprint, says whose decision that is, and
// says out loud that it is not a task to dispatch — because the addressee is what
// decides who acts on a signal, and an agent reading the sweep-dispatch rule has
// a standing habit of turning a session-start line into a background subagent.
func sprintSignalNotice(open, highest int) string {
	last := "none has been opened yet"
	if highest > 0 {
		last = fmt.Sprintf("sprint:%d is already reviewed", highest)
	}
	return fmt.Sprintf("atl: this project's board has %d open item(s) and no active sprint (%s) — "+
		"/sprint-plan opens the next one, and which work enters it is the product owner's call. "+
		"Raise it with them: this is a report, not a task to dispatch in the background.", open, last)
}

// sprintSignalCovers reports whether this project's delivery configuration is one
// a deterministic Go scan can actually read. Both refusals are deliberate,
// stated, and narrower than the mechanism — the card's acceptance allows not
// covering a backend as long as the code says which and why.
func sprintSignalCovers(cfg *dispatch.DeliveryConfig, projectRoot string) bool {
	// AZURE IS NOT COVERED. Its board is reached through an MCP surface — an
	// LLM-side tool with no Go client, and nothing to shell out to the way GitHub
	// has gh (the only provider CLI this binary invokes). Answering the predicate
	// there needs an Azure REST client, which is separate scope; guessing at it
	// from local state is worse than silence, because the local proxies do not
	// carry the answer (plan.json's sprintSlug records that an autonomous plan was
	// materialised, not that a sprint is open).
	if cfg.ActiveBackend() != "github" {
		return false
	}
	// SCRUM IS NOT COVERED. Under mode "scrum" the sprint carrier is the Projects
	// v2 Iteration field, not a sprint:<n> label, and "active" is a date window
	// rather than an unwritten review page — a different read answering a
	// different question. Running the label predicate over a scrum board would
	// find no sprint: label anywhere and report "no active sprint" on every
	// session of every scrum project: a false positive by construction, on the
	// mode that is still the delivery-team's default.
	//
	// An absent mode field reads as not-flow on purpose: flow mode is the newer
	// half, so a methodology.json written before it is a scrum project.
	if deliveryMode(projectRoot) != "flow" {
		return false
	}
	return true
}

// deliveryMode reads methodology.mode from .delivery/methodology.json — the
// discriminator the ceremony-adoption decision added. Parsed and failed the same
// way boardTrackedSignal parses config.json: missing or malformed yields "", and
// the caller stays silent.
func deliveryMode(projectRoot string) string {
	b, err := os.ReadFile(filepath.Join(projectRoot, ".delivery", "methodology.json"))
	if err != nil {
		return ""
	}
	var m struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	return m.Mode
}

// sprintVerdictPath is the cached verdict for one project, under ~/.atl/cache
// beside the throttle stamps. Kept out of the project's own .delivery/ because a
// machine-local cache is not project state, and .delivery/ is committed.
func sprintVerdictPath(project string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".atl", "cache",
		"sprint-verdict-"+transcript.SlugForPath(project)+".json"), nil
}

// scheduleSprintScan spawns the detached board read, at most once per interval
// per project.
//
// The stamp is touched BEFORE the spawn, like autoUpdateTeams: a scan that fails
// (no gh, an expired token, a renamed repo) then retries tomorrow rather than
// forking a doomed process on every single session.
func scheduleSprintScan(project string) {
	stamp, err := throttle.StampPath("sprint-scan-" + transcript.SlugForPath(project))
	if err != nil || !throttle.Gate(stamp, sprintScanInterval) {
		return
	}
	_ = throttle.Touch(stamp)
	_ = sprintScanSpawn() // best-effort; the next session prints the result
}

// workSprintScanCmd refreshes the cached verdict. Internal: session-start spawns
// it detached, and it is under the already-Hidden `work` group so it stays out of
// `atl --help` and out of the docs-coverage gate.
var workSprintScanCmd = &cobra.Command{
	Use:   "sprint-scan",
	Short: "Refresh this project's cached sprint verdict from the board (internal)",
	Long: "Read the board once and cache whether this project has open work and\n" +
		"which sprint ordinal is highest, so session-start can report a board that\n" +
		"has drifted out of a sprint without doing network work on the hook's path.\n" +
		"Spawned detached by session-start; running it by hand just refreshes the\n" +
		"cache now.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			return err
		}
		return runSprintScan(root, sprintScanRunner)
	},
}

// runSprintScan is the command body with the shell-out injected, so the coverage
// gate, the config refusals, and the cache write are all testable without gh.
func runSprintScan(root string, run sprintsignal.Runner) error {
	cfg, err := dispatch.LoadDeliveryConfig(root)
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("no .delivery/config.json at %s — this project has no board backend", root)
	}
	if !sprintSignalCovers(cfg, root) {
		return fmt.Errorf("the deterministic sprint scan covers the github backend under mode flow only; "+
			"this project is %s/%s", cfg.ActiveBackend(), deliveryMode(root))
	}
	v, err := sprintsignal.Scan(run, cfg.Owner, cfg.Repo)
	if err != nil {
		return err
	}
	path, err := sprintVerdictPath(root)
	if err != nil {
		return err
	}
	return sprintsignal.Save(path, v)
}

// sprintScanRunner runs the real gh binary under a hard timeout. Output(), not
// CombinedOutput(): the result is parsed as JSON, so stderr must not be mixed in.
func sprintScanRunner(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sprintScanTimeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}
