package dispatch

import (
	"path/filepath"
	"strings"
	"testing"
)

func assertContainsAll(t *testing.T, got string, must []string) {
	t.Helper()
	for _, want := range must {
		if !strings.Contains(got, want) {
			t.Errorf("prompt missing invariant %q\n--- prompt ---\n%s", want, got)
		}
	}
}

// assertContainsNone locks the backend-neutralization: no worker prompt may hardcode a
// backend's concrete tool binding — the concrete tools live in the active backend's
// adapter, so a regression that re-inlined an Azure (or GitHub) tool name into the engine
// prompt would re-couple the engine to one backend and trip this.
func assertContainsNone(t *testing.T, got string, banned []string) {
	t.Helper()
	for _, bad := range banned {
		if strings.Contains(got, bad) {
			t.Errorf("prompt must not hardcode a backend tool/term %q (it belongs in the adapter)\n--- prompt ---\n%s", bad, got)
		}
	}
}

// backendHardcoded are concrete backend tool names / field literals that must NEVER appear
// in any engine worker prompt — the prompt defers all tool binding to the active backend's
// adapter (backends/<backend>/adapter.md), so any of these tokens signals a re-coupling.
var backendHardcoded = []string{
	"azureDevOps MCP", "Azure work-item", "az-attach.sh", "pat.ref", "System.Description",
	"wit_get_work_item_type", "wit_list_work_item_comments", "wit_get_work_item_attachment",
	"repo_vote_pull_request", "repo_update_pull_request", "repo_create_pull_request_thread",
	"autoComplete", "NoFastForward", "transitionWorkItems",
}

// TestDeliveryStagePrompt_DeveloperInvariants locks the load-bearing contract the
// developer stage prompt must carry (Fork A: the plan gives only the id, so the prompt
// directs the worker to fetch the rest from the tracker through the active backend's
// adapter). These are the invariants a drifting edit would silently drop — the role
// token, the six phases, the fetch-from-the-tracker sequence, the resolve-at-runtime rule
// (deferred to the adapter, never a hardcoded literal), and the job-ends-at-PR boundary.
func TestDeliveryStagePrompt_DeveloperInvariants(t *testing.T) {
	root := "/proj"
	u := WorkUnit{ID: 4242, Title: "Add credential validation"}
	got := deliveryStagePrompt(u, root, StageDeveloper)

	agentDir := filepath.Join(root, ".claude", "agents", "developer")
	packsDir := filepath.Join(root, ".claude", "packs")
	backendsDir := filepath.Join(root, ".claude", "backends")
	configPath := filepath.Join(root, ".delivery", "config.json")

	assertContainsAll(t, got, []string{
		"delivery-team developer",   // the role token the Layer-A fake keys off
		"#4242",                     // the one work-item id the engine knows
		"Add credential validation", // its title
		agentDir + "/agent.md",      // points at the developer agent as operating manual
		agentDir + "/children/",     // and its children/
		configPath,                  // read .delivery/config.json for the backend + coordinates
		packsDir + "/<area>/",       // load only the tagged area's pack
		backendsDir + "/<backend>/adapter.md", // the active backend's adapter binds the tools
		"config.backend",                      // the worker resolves which backend from config
		"**[Technical Analysis]**",            // the analysis sentinel comment it locates
		"**[Canonical Brief]**",               // the tech-lead brief, located by its sentinel
		"area:<name>",                         // the pack-binding tag it resolves
		"claim -> plan -> implement -> self-test -> comment -> pr", // the six phases, in order
		"status.json",                   // the only channel back to the supervisor
		"reclaimed as stalled",          // the early-heartbeat instruction (write status.json first)
		"never fake a green",            // block-never-silent-pass
		"do NOT merge",                  // job ends at PR
		"do NOT set the work-item Done", // the tech-lead completes the PR + sets Done
	})
	assertContainsNone(t, got, backendHardcoded)
}

// TestDeliveryStagePrompt_TesterInvariants locks the tester stage prompt: independent
// Level-2 verification over the developer's branch, re-deriving intent fresh, attaching
// evidence per the active adapter, and the hard boundaries (owns tests — not code, not
// review, not state).
func TestDeliveryStagePrompt_TesterInvariants(t *testing.T) {
	root := "/proj"
	u := WorkUnit{ID: 51, Title: "Login screen"}
	got := deliveryStagePrompt(u, root, StageTester)

	agentDir := filepath.Join(root, ".claude", "agents", "tester")
	backendsDir := filepath.Join(root, ".claude", "backends")

	assertContainsAll(t, got, []string{
		"delivery-team tester",   // the role token the Layer-A fake keys off
		"#51", "Login screen",    // the assignment
		agentDir + "/agent.md",   // points at the tester agent
		agentDir + "/children/",  // and its children/
		"verification-blueprint", // the operative child file
		backendsDir + "/<backend>/adapter.md", // the active backend's adapter binds the tools
		"config.backend",                      // resolve the backend from config
		"**[Technical Analysis]**",            // re-derive intent fresh from the sentinel
		"**[Canonical Brief]**",               // the brief, located by its sentinel
		"acceptance criteria = the spec",      // AC drives the strategy
		"per the active adapter's evidence mechanism", // evidence attach — adapter-bound
		"do NOT write or fix implementation code",     // hard boundary: not the developer
		"do NOT judge code quality or architecture",   // hard boundary: not the tech-lead
		"do NOT transition the work-item state",       // hard boundary: not the state owner
		"never fake a green",
		"status.json",
		"reclaimed as stalled", // the early-heartbeat instruction
	})
	assertContainsNone(t, got, backendHardcoded)
}

// TestDeliveryStagePrompt_TechLeadInvariants locks the tech-lead stage prompt: the
// single review gate + closer — test-gate first, delivery-native review on the PR with
// refute-to-keep, and on green vote → land (a real merge commit, never squash/rebase) →
// Done, all deferred to the active backend's adapter for the concrete tools.
func TestDeliveryStagePrompt_TechLeadInvariants(t *testing.T) {
	root := "/proj"
	u := WorkUnit{ID: 77, Title: "Payment flow"}
	got := deliveryStagePrompt(u, root, StageTechLead)

	agentDir := filepath.Join(root, ".claude", "agents", "tech-lead")
	backendsDir := filepath.Join(root, ".claude", "backends")

	assertContainsAll(t, got, []string{
		"delivery-team tech-lead", // the role token the Layer-A fake keys off
		"#77", "Payment flow",     // the assignment
		agentDir + "/agent.md",    // points at the tech-lead agent
		agentDir + "/children/",   // and its children/
		"review-craft",            // the operative child file for this stage
		backendsDir + "/<backend>/adapter.md", // the active backend's adapter binds the tools
		"config.backend",                      // resolve the backend from config
		"Test-gate FIRST",                     // tests before review
		"refute-to-keep",                      // the review pattern
		"real merge commit",                   // the SHA-preserving merge the verify depends on
		"never squash or rebase",              // both rewrite SHAs → would false-block merge-verify
		"false-block",                         // the reason the merge strategy is pinned
		"runtime-resolved Done",               // sets Done after the merge, resolved at runtime
		"Merge first, then Done",              // the order
		"never fake a green",
		"status.json",
		"reclaimed as stalled", // the early-heartbeat instruction
	})
	assertContainsNone(t, got, backendHardcoded)
}

// TestDeliveryWorkerSpec_Fields asserts the base spec wiring per stage: the worktree is
// the cwd, and the MCP config + credential env are left empty in the base spec on purpose
// — spawnStage augments them per-worker (empty MCPConfigPath here → the scheduler injects
// the per-backend --mcp-config for Azure; nil ExtraEnv here → the scheduler appends the
// credential env, which never enters the argv). A regression that started pinning the MCP
// path or injecting the token into the base spec would trip this.
func TestDeliveryWorkerSpec_Fields(t *testing.T) {
	build := DeliveryWorkerSpec("/proj")
	for _, stage := range deliveryPipeline {
		spec := build(WorkUnit{ID: 7, Title: "x"}, stage, "/proj/.delivery/worktrees/s1/7")

		if spec.WorktreeDir != "/proj/.delivery/worktrees/s1/7" {
			t.Errorf("[%s] WorktreeDir = %q, want the passed worktree dir", stage, spec.WorktreeDir)
		}
		if spec.MCPConfigPath != "" {
			t.Errorf("[%s] MCPConfigPath = %q, want empty (spawnStage injects it)", stage, spec.MCPConfigPath)
		}
		if spec.ExtraEnv != nil {
			t.Errorf("[%s] ExtraEnv = %v, want nil (spawnStage appends the credential env)", stage, spec.ExtraEnv)
		}
		if strings.TrimSpace(spec.Prompt) == "" {
			t.Errorf("[%s] Prompt is empty", stage)
		}
	}
}
