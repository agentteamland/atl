package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/sprintsignal"
	"github.com/agentteamland/atl/cli/internal/throttle"
	"github.com/agentteamland/atl/cli/internal/transcript"
)

// deliveryProject writes a .delivery/ pair and returns the project root.
func deliveryProject(t *testing.T, config, methodology string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
		t.Fatal(err)
	}
	if config != "" {
		if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if methodology != "" {
		if err := os.WriteFile(filepath.Join(root, ".delivery", "methodology.json"), []byte(methodology), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const (
	githubFlowConfig = `{"owner":"acme-org","repo":"widget-repo","backend":"github"}`
	flowMethodology  = `{"id":"scrum","mode":"flow"}`
)

// seedVerdict writes a cached verdict for project under the test's HOME.
func seedVerdict(t *testing.T, project string, v sprintsignal.Verdict) {
	t.Helper()
	path, err := sprintVerdictPath(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := sprintsignal.Save(path, v); err != nil {
		t.Fatal(err)
	}
}

// stubSpawn replaces the detached scan with a counter so no test forks a process.
func stubSpawn(t *testing.T) *int {
	t.Helper()
	var n int
	orig := sprintScanSpawn
	sprintScanSpawn = func() error { n++; return nil }
	t.Cleanup(func() { sprintScanSpawn = orig })
	return &n
}

func TestSprintSessionSignal_FiresWithOpenWorkAndNoActiveSprint(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubSpawn(t)
	root := deliveryProject(t, githubFlowConfig, flowMethodology)
	seedVerdict(t, root, sprintsignal.Verdict{OpenItems: 12, HighestSprint: 0, ScannedAt: time.Now().UTC()})

	out := captureStdout(t, func() { sprintSessionSignal(root) })

	if !strings.Contains(out, "12 open item(s)") {
		t.Errorf("the signal must quote the open count, got %q", out)
	}
	if !strings.Contains(out, "no active sprint") {
		t.Errorf("the signal must say what it found, got %q", out)
	}
	// It names the action…
	if !strings.Contains(out, "/sprint-plan") {
		t.Errorf("the signal must name the action that opens a sprint, got %q", out)
	}
	// …and says who acts. This is a report, not a dispatch: an agent that reads it
	// as a task to background would be making the product owner's decision.
	if !strings.Contains(out, "product owner") || !strings.Contains(out, "report, not a task to dispatch") {
		t.Errorf("the signal must route the decision to a person, got %q", out)
	}
	if strings.Contains(out, "background subagent") {
		t.Errorf("a report must not carry the sweep-dispatch instruction, got %q", out)
	}
}

func TestSprintSessionSignal_FiresWhenTheHighestSprintIsReviewed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubSpawn(t)
	root := deliveryProject(t, githubFlowConfig, flowMethodology)
	if err := os.MkdirAll(filepath.Join(root, "docs", "sprints"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sprintsignal.ReviewPagePath(root, 4), []byte("# review"), 0o644); err != nil {
		t.Fatal(err)
	}
	seedVerdict(t, root, sprintsignal.Verdict{OpenItems: 3, HighestSprint: 4, ScannedAt: time.Now().UTC()})

	out := captureStdout(t, func() { sprintSessionSignal(root) })
	if !strings.Contains(out, "sprint:4 is already reviewed") {
		t.Errorf("expected the reviewed-sprint wording, got %q", out)
	}
}

// Every silence the acceptance criteria name, in one table. They must all produce
// exactly the same thing — nothing — because the whole mechanism is opt-in on
// being able to answer.
func TestSprintSessionSignal_Silences(t *testing.T) {
	live := func() sprintsignal.Verdict {
		return sprintsignal.Verdict{OpenItems: 9, HighestSprint: 0, ScannedAt: time.Now().UTC()}
	}
	cases := []struct {
		name    string
		config  string
		method  string
		verdict *sprintsignal.Verdict
		why     string
	}{
		{
			name: "no .delivery/config.json", config: "", method: flowMethodology, verdict: ptr(live()),
			why: "a project with no board backend has no board to have drifted",
		},
		{
			name: "azure backend", config: `{"org":"o","project":"p","backend":"azure"}`, method: flowMethodology, verdict: ptr(live()),
			why: "the Azure board is an MCP surface Go cannot read — documented non-coverage",
		},
		{
			name: "backend defaulted to azure", config: `{"org":"o","project":"p"}`, method: flowMethodology, verdict: ptr(live()),
			why: "an absent backend field is azure, and azure is not covered",
		},
		{
			name: "scrum mode", config: githubFlowConfig, method: `{"id":"scrum","mode":"scrum"}`, verdict: ptr(live()),
			why: "under scrum the carrier is the Iteration field; the label predicate would false-fire on every session",
		},
		{
			name: "mode field absent", config: githubFlowConfig, method: `{"id":"scrum"}`, verdict: ptr(live()),
			why: "a methodology.json predating the mode discriminator is a scrum project",
		},
		{
			name: "no methodology.json", config: githubFlowConfig, method: "", verdict: ptr(live()),
			why: "nothing says this is a flow project",
		},
		{
			name: "malformed config", config: `{not json`, method: flowMethodology, verdict: ptr(live()),
			why: "a config that cannot be parsed cannot be acted on",
		},
		{
			name: "no cached verdict", config: githubFlowConfig, method: flowMethodology, verdict: nil,
			why: "no board read has ever succeeded here",
		},
		{
			name: "stale verdict", config: githubFlowConfig, method: flowMethodology,
			verdict: &sprintsignal.Verdict{OpenItems: 9, ScannedAt: time.Now().UTC().Add(-30 * 24 * time.Hour)},
			why:     "a board read that stopped working must stop speaking",
		},
		{
			name: "no open work", config: githubFlowConfig, method: flowMethodology,
			verdict: &sprintsignal.Verdict{OpenItems: 0, HighestSprint: 0, ScannedAt: time.Now().UTC()},
			why:     "an empty board is not drifting",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			stubSpawn(t)
			root := deliveryProject(t, c.config, c.method)
			if c.verdict != nil {
				seedVerdict(t, root, *c.verdict)
			}
			if out := captureStdout(t, func() { sprintSessionSignal(root) }); out != "" {
				t.Errorf("must be silent (%s), got %q", c.why, out)
			}
		})
	}
}

// A sprint IS active: the highest ordinal carries no review page. This is the one
// silence that depends on the working tree rather than on configuration, and it
// is read at print time so a review page written today takes effect today.
func TestSprintSessionSignal_SilentWhileASprintIsActive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubSpawn(t)
	root := deliveryProject(t, githubFlowConfig, flowMethodology)
	seedVerdict(t, root, sprintsignal.Verdict{OpenItems: 7, HighestSprint: 4, ScannedAt: time.Now().UTC()})

	if out := captureStdout(t, func() { sprintSessionSignal(root) }); out != "" {
		t.Fatalf("an unreviewed sprint 4 is still current — must be silent, got %q", out)
	}

	// Review it, and the very next session speaks, with no new board read.
	if err := os.MkdirAll(filepath.Join(root, "docs", "sprints"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sprintsignal.ReviewPagePath(root, 4), []byte("# review"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := captureStdout(t, func() { sprintSessionSignal(root) }); out == "" {
		t.Fatal("once sprint 4 is reviewed the signal must fire without waiting for a fresh scan")
	}
}

// A git worktree is silent and schedules nothing. This is the case that bites in
// production rather than in a test: .delivery/ is committed, so every worktree
// `atl work dispatch` cuts for a work-unit is a fully-configured board project,
// and without this each of N workers would print the report into an autonomous
// context AND schedule its own board read (a fresh path means a fresh throttle).
func TestSprintSessionSignal_SilentInAGitWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("HOME", t.TempDir())
	spawns := stubSpawn(t)

	// A real repo whose .delivery/ pair is COMMITTED, exactly as this one is.
	main := deliveryProject(t, githubFlowConfig, flowMethodology)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", main}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("add", ".delivery")
	git("commit", "-q", "-m", "delivery config")

	wt := filepath.Join(t.TempDir(), "wt")
	git("worktree", "add", "-q", "--detach", wt)

	// The worktree really did receive the committed configuration…
	if _, err := os.Stat(filepath.Join(wt, ".delivery", "config.json")); err != nil {
		t.Fatalf("the worktree must carry the committed config, else this test proves nothing: %v", err)
	}
	seedVerdict(t, wt, sprintsignal.Verdict{OpenItems: 9, HighestSprint: 0, ScannedAt: time.Now().UTC()})

	// …and is still silent, and still spawns nothing.
	if out := captureStdout(t, func() { sprintSessionSignal(wt) }); out != "" {
		t.Errorf("a worktree must be silent, got %q", out)
	}
	if *spawns != 0 {
		t.Errorf("a worktree must not schedule a board read, got %d", *spawns)
	}

	// The main checkout of the same repo still reports.
	seedVerdict(t, main, sprintsignal.Verdict{OpenItems: 9, HighestSprint: 0, ScannedAt: time.Now().UTC()})
	if out := captureStdout(t, func() { sprintSessionSignal(main) }); out == "" {
		t.Error("the main checkout must still report — the guard is about worktrees, not about git")
	}
}

func TestSprintSessionSignal_EmptyProjectRootIsSilent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	spawns := stubSpawn(t)
	if out := captureStdout(t, func() { sprintSessionSignal("") }); out != "" {
		t.Errorf("an unresolved project root must be silent, got %q", out)
	}
	if *spawns != 0 {
		t.Errorf("an unresolved project root must not schedule a scan, got %d", *spawns)
	}
}

// The brake stops both halves: the printed report and the network work behind it.
func TestSprintSessionSignal_BrakeStopsReportAndScan(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(envNoSprintSignal, "1")
	spawns := stubSpawn(t)
	root := deliveryProject(t, githubFlowConfig, flowMethodology)
	seedVerdict(t, root, sprintsignal.Verdict{OpenItems: 9, ScannedAt: time.Now().UTC()})

	if out := captureStdout(t, func() { sprintSessionSignal(root) }); out != "" {
		t.Errorf("the brake must silence the report, got %q", out)
	}
	if *spawns != 0 {
		t.Errorf("the brake must also stop the detached board read, got %d spawn(s)", *spawns)
	}
}

// The scan is scheduled once per interval per project, and the stamp is touched
// before the spawn so a failing scan retries tomorrow rather than every session.
func TestScheduleSprintScan_ThrottledPerProject(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	spawns := stubSpawn(t)
	const project = "/tmp/board/project"

	scheduleSprintScan(project)
	if *spawns != 1 {
		t.Fatalf("first run should spawn once, got %d", *spawns)
	}
	stamp, err := throttle.StampPath("sprint-scan-" + transcript.SlugForPath(project))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stamp); err != nil {
		t.Fatalf("the stamp must be recorded before the spawn: %v", err)
	}
	scheduleSprintScan(project)
	if *spawns != 1 {
		t.Fatalf("a second run inside the interval must not spawn, got %d", *spawns)
	}
	scheduleSprintScan("/tmp/other/project")
	if *spawns != 2 {
		t.Fatalf("a different project has its own throttle, got %d", *spawns)
	}
}

// A covered project schedules its refresh even when it has nothing to report —
// otherwise the first verdict could never be written.
func TestSprintSessionSignal_SchedulesTheScanWithNoCacheYet(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	spawns := stubSpawn(t)
	root := deliveryProject(t, githubFlowConfig, flowMethodology)

	out := captureStdout(t, func() { sprintSessionSignal(root) })
	if out != "" {
		t.Errorf("no verdict yet → silent, got %q", out)
	}
	if *spawns != 1 {
		t.Errorf("a covered project must schedule its first board read, got %d", *spawns)
	}
}

// An uncovered project must not spawn either: scheduling a scan that can only
// refuse burns a process every day forever.
func TestSprintSessionSignal_UncoveredProjectSchedulesNothing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	spawns := stubSpawn(t)
	root := deliveryProject(t, `{"org":"o","project":"p","backend":"azure"}`, flowMethodology)

	captureStdout(t, func() { sprintSessionSignal(root) })
	if *spawns != 0 {
		t.Errorf("an uncovered backend must not schedule a board read, got %d", *spawns)
	}
}

func TestDeliveryMode(t *testing.T) {
	cases := []struct{ file, want string }{
		{`{"mode":"flow"}`, "flow"},
		{`{"mode":"scrum"}`, "scrum"},
		{`{"id":"scrum"}`, ""},
		{`{not json`, ""},
	}
	for _, c := range cases {
		root := deliveryProject(t, githubFlowConfig, c.file)
		if got := deliveryMode(root); got != c.want {
			t.Errorf("deliveryMode(%s) = %q, want %q", c.file, got, c.want)
		}
	}
	if got := deliveryMode(t.TempDir()); got != "" {
		t.Errorf("deliveryMode with no methodology.json = %q, want \"\"", got)
	}
}

// The refresh command: the argv it issues, the number of calls it makes, and the
// verdict it leaves behind.
func TestRunSprintScan_WritesTheVerdict(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := deliveryProject(t, githubFlowConfig, flowMethodology)

	var calls [][]string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return []byte(`[{"number":1,"state":"OPEN","labels":[]},
		                {"number":2,"state":"CLOSED","labels":[{"name":"sprint:2"}]}]`), nil
	}
	if err := runSprintScan(root, run); err != nil {
		t.Fatalf("runSprintScan: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 board call, got %d: %v", len(calls), calls)
	}
	// The coordinates come from THIS project's config, not from a default.
	want := append([]string{"gh"}, sprintsignal.ScanArgs("acme-org", "widget-repo")...)
	if !reflect.DeepEqual(calls[0], want) {
		t.Errorf("argv drifted\n got: %v\nwant: %v", calls[0], want)
	}

	path, err := sprintVerdictPath(root)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := sprintsignal.Load(path, time.Hour, time.Now())
	if !ok {
		t.Fatal("the scan must leave a loadable verdict behind")
	}
	if v.OpenItems != 1 || v.HighestSprint != 2 {
		t.Errorf("verdict = %+v, want OpenItems 1 / HighestSprint 2", v)
	}
}

// A failing read writes nothing — so the previous verdict is neither replaced by
// a wrong one nor half-written, and the signal falls back to silence once the old
// one ages out.
func TestRunSprintScan_FailingReadWritesNoVerdict(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := deliveryProject(t, githubFlowConfig, flowMethodology)

	run := func(string, ...string) ([]byte, error) { return nil, fmt.Errorf("gh: exit 1") }
	if err := runSprintScan(root, run); err == nil {
		t.Fatal("a failing board read must surface an error")
	}
	path, err := sprintVerdictPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("a failing read must not write a verdict file")
	}
}

// The command refuses what it cannot answer, and refuses it BEFORE spending a
// board call.
func TestRunSprintScan_RefusesUncoveredProjects(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cases := []struct{ name, config, method string }{
		{"azure", `{"org":"o","project":"p","backend":"azure"}`, flowMethodology},
		{"scrum", githubFlowConfig, `{"mode":"scrum"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var calls int
			run := func(string, ...string) ([]byte, error) { calls++; return nil, nil }
			err := runSprintScan(deliveryProject(t, c.config, c.method), run)
			if err == nil {
				t.Fatal("an uncovered project must be refused")
			}
			if calls != 0 {
				t.Errorf("a refused project must make no board call, got %d", calls)
			}
		})
	}
	t.Run("no config", func(t *testing.T) {
		var calls int
		run := func(string, ...string) ([]byte, error) { calls++; return nil, nil }
		if err := runSprintScan(t.TempDir(), run); err == nil {
			t.Fatal("a project with no .delivery/config.json must be refused")
		}
		if calls != 0 {
			t.Errorf("expected no board call, got %d", calls)
		}
	})
}

// The verdict cache is machine-local, per project, and never inside the project.
func TestSprintVerdictPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a, err := sprintVerdictPath("/tmp/one")
	if err != nil {
		t.Fatal(err)
	}
	b, err := sprintVerdictPath("/tmp/two")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("two projects must not share one verdict")
	}
	if !strings.HasPrefix(a, filepath.Join(home, ".atl", "cache")) {
		t.Errorf("the verdict cache belongs under ~/.atl/cache, got %q", a)
	}
}

// The scan command is registered and reachable, under the Hidden `work` group so
// it stays out of `atl --help` and the docs-coverage gate.
func TestWorkSprintScanRegistered(t *testing.T) {
	var found bool
	for _, c := range workCmd.Commands() {
		if c.Name() == "sprint-scan" {
			found = true
		}
	}
	if !found {
		t.Fatal("`atl work sprint-scan` must be registered — session-start spawns it by name")
	}
	if !workCmd.Hidden {
		t.Error("the `work` group must stay Hidden — sprint-scan is internal")
	}
}

func ptr[T any](v T) *T { return &v }
