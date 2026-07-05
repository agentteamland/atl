package dispatch

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestDeliveryWorkerPrompt_Invariants locks the load-bearing contract the assembled
// worker prompt must carry (Fork A: the plan gives only the id, so the prompt directs
// the worker to fetch the rest from Azure). These are the invariants a drifting edit
// would silently drop — the six phases, the fetch-from-Azure sequence, the
// runtime-state-resolution rule, and the hard job-ends-at-PR boundary.
func TestDeliveryWorkerPrompt_Invariants(t *testing.T) {
	root := "/proj"
	u := WorkUnit{ID: 4242, Title: "Add credential validation"}
	got := deliveryWorkerPrompt(u, root)

	agentDir := filepath.Join(root, ".claude", "agents", "developer")
	packsDir := filepath.Join(root, ".claude", "packs")
	configPath := filepath.Join(root, ".delivery", "config.json")

	must := []string{
		"#4242",                        // the one work-item id the engine knows
		"Add credential validation",    // its title
		agentDir + "/agent.md",         // points at the developer agent as operating manual
		agentDir + "/children/",        // and its children/
		configPath,                     // read .delivery/config.json for coordinates + pat.ref
		packsDir + "/<area>/",          // load only the tagged area's pack
		"azureDevOps MCP",              // MCP-first transport (#17)
		"wit_get_work_item_type",       // runtime state/type resolution — never a literal
		"**[Technical Analysis]**",     // the sentinel comment it must locate
		"canonical brief",              // the tech-lead artifact it fetches
		"area:<name>",                  // the pack-binding tag it resolves
		"claim -> plan -> implement -> self-test -> comment -> pr", // the six phases, in order
		"status.json",                  // the only channel back to the supervisor
		"never fake a green",           // block-never-silent-pass
		"do NOT merge",                 // job ends at PR
		"do NOT set the work-item Done", // the tech-lead completes the PR + sets Done; the engine only verifies
	}
	for _, want := range must {
		if !strings.Contains(got, want) {
			t.Errorf("worker prompt missing invariant %q\n--- prompt ---\n%s", want, got)
		}
	}
}

// TestDeliveryWorkerSpec_Fields asserts the spec wiring: the worktree is the cwd, and
// the MCP config + PAT env are left to inheritance on purpose (empty MCPConfigPath →
// inherit the worktree's .mcp.json; nil ExtraEnv → the PAT rides the inherited process
// env, never the argv). A regression that started pinning a per-unit MCP path or
// injecting the token into the spec would trip this.
func TestDeliveryWorkerSpec_Fields(t *testing.T) {
	build := DeliveryWorkerSpec("/proj")
	spec := build(WorkUnit{ID: 7, Title: "x"}, "/proj/.delivery/worktrees/s1/7")

	if spec.WorktreeDir != "/proj/.delivery/worktrees/s1/7" {
		t.Errorf("WorktreeDir = %q, want the passed worktree dir", spec.WorktreeDir)
	}
	if spec.MCPConfigPath != "" {
		t.Errorf("MCPConfigPath = %q, want empty (inherit the worktree .mcp.json)", spec.MCPConfigPath)
	}
	if spec.ExtraEnv != nil {
		t.Errorf("ExtraEnv = %v, want nil (PAT inherits from the process env, never the argv)", spec.ExtraEnv)
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		t.Error("Prompt is empty")
	}
}
