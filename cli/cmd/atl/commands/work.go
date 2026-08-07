package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/agentteamland/atl/cli/internal/dispatch"
	"github.com/agentteamland/atl/cli/internal/promotiongate"
	"github.com/spf13/cobra"
)

// workCmd groups the delivery-team orchestration commands. It is HIDDEN by design:
// `atl work dispatch` is an internal engine invoked by the delivery-team's ceremony
// skills (/sprint-start), never typed by users — so it stays out of `atl --help`
// and out of the docs-coverage gate (commandDocs skips Hidden commands — see
// docs.go); the delivery-team docs page documents the engine's behavior instead.
var workCmd = &cobra.Command{
	Use:   "work",
	Short: "Delivery-team work orchestration (internal)",
	Long: "Orchestration engine for the delivery-team: schedule a sprint's work-units\n" +
		"across isolated `claude -p` workers in git worktrees. Invoked by the ceremony\n" +
		"skills, not run by hand.",
	Hidden: true,
}

var workDispatchCmd = &cobra.Command{
	Use:   "dispatch",
	Short: "Run a sprint's work-unit DAG across a pool of isolated workers",
	Long: "Read the materialized plan (.delivery/plan.json) a /sprint-start ceremony\n" +
		"wrote, then schedule its work-units over a fixed-cap pool of `claude -p`\n" +
		"workers — each in its own git worktree off `dev`, observed through the\n" +
		"status.json each worker writes. Deterministic supervisor: no LLM context,\n" +
		"no Azure calls (the workers own the Azure MCP surface).",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			return err
		}

		planPath := dispatch.PlanPath(root)
		plan, err := dispatch.Load(planPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("no plan at %s — run /sprint-start first", planPath)
			}
			return err
		}
		if err := dispatch.Validate(plan); err != nil {
			// A cycle or a dangling predecessor is a planning error to fix, not to
			// silently break — refuse to start and surface it (#15 step 2).
			return fmt.Errorf("plan is not schedulable: %w", err)
		}

		// Derive the integration branch from .delivery/config.json's branchPair
		// (default "dev"), which /delivery-init lets the project override — the
		// engine must fork/merge against the branch the project actually uses, not
		// a hardcoded literal. A missing config → nil → the "dev" default.
		cfg, cerr := dispatch.LoadDeliveryConfig(root)
		if cerr != nil {
			return cerr
		}
		dev := cfg.DevBranch()

		wt := &dispatch.Worktree{
			RepoDir:   root,
			Root:      filepath.Join(root, ".delivery", "worktrees"),
			BaseRef:   dev,
			RemoteRef: "origin/" + dev,
			Run:       dispatch.ExecRunner,
		}
		sched := dispatch.NewScheduler(plan, root, wt, dispatch.NewSpawner(), dispatch.Config{Cap: workDispatchCap})
		out := cmd.OutOrStdout()
		sched.Log = func(line string) { fmt.Fprintln(out, line) }

		fmt.Fprintf(out, "dispatching sprint %q — %d units, cap %d\n", plan.SprintSlug, len(plan.Units), sched.Cfg.Cap)

		// Translate a terminal interrupt (Ctrl-C / SIGTERM) into a context cancel so
		// the scheduler tears its worker pool down instead of leaving orphaned
		// `claude -p` processes running detached.
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		sum, err := sched.RunContext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				fmt.Fprintln(out, "\ndispatch interrupted — running workers were aborted; re-run to resume from durable state")
				return nil
			}
			return err
		}

		fmt.Fprintf(out, "\ndispatch complete: %d done, %d blocked, %d skipped\n",
			len(sum.Done), len(sum.Blocked), len(sum.SkippedByDep))
		if len(sum.Blocked) > 0 {
			fmt.Fprintf(out, "  blocked: %v — reports in %s (a ceremony reflects them to Azure)\n", sum.Blocked, dispatch.BlockedDir(root))
		}
		if len(sum.SkippedByDep) > 0 {
			fmt.Fprintf(out, "  skipped (blocked predecessor): %v\n", sum.SkippedByDep)
		}
		return nil
	},
}

// workPromoteRunner is the exec seam `work promote` shells gh through — the real
// binary in production, a fake in tests, so every hold path and the promote path
// are covered without a network.
var workPromoteRunner = promotiongate.ExecRunner

var workPromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Verify the commit-bound promotion approval and merge the dev→release PR",
	Long: "Read the `**[Promotion Approval]**` record the human product owner posted on\n" +
		"the dev→release PR, compare the commit it names against that PR's CURRENT head,\n" +
		"and merge only on an exact match — every other outcome holds and merges nothing.\n" +
		"Verification and merge are one step on purpose: a separate merge is a step a\n" +
		"ceremony can reach without the check. Invoked by /sprint-review's gate.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			return err
		}
		cfg, err := dispatch.LoadDeliveryConfig(root)
		if err != nil {
			return err
		}
		if cfg == nil {
			return fmt.Errorf("no %s — run /delivery-init first", dispatch.DeliveryConfigPath(root))
		}

		var res promotiongate.Result
		switch backend := cfg.ActiveBackend(); backend {
		case "github":
			if cfg.Owner == "" || cfg.Repo == "" {
				return fmt.Errorf("%s is missing owner/repo — the github backend needs both", dispatch.DeliveryConfigPath(root))
			}
			res = promotiongate.Gate{
				Owner:   cfg.Owner,
				Repo:    cfg.Repo,
				Dev:     cfg.DevBranch(),
				Release: cfg.ReleaseBranch(),
				PR:      workPromotePR,
				Run:     workPromoteRunner,
			}.Verify()
		default:
			// Only the record leg of concept #16 is bound outside GitHub; the
			// head-commit read is not, so the comparison cannot run at all.
			res = promotiongate.HoldBackendUnbound(backend)
		}

		out := cmd.OutOrStdout()
		if workPromoteJSON {
			b, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(b))
		} else {
			fmt.Fprintln(out, res.Message)
		}
		if res.Verdict != promotiongate.Promoted {
			// Non-zero so a caller that ignores the message still cannot read a hold
			// as a promotion. The detail is already on stdout (the `atl skills check`
			// shape); this line is the one-sentence summary.
			return fmt.Errorf("promotion held (%s) — nothing was merged", res.Reason)
		}
		return nil
	},
}

var (
	workDispatchCap int
	workPromotePR   int
	workPromoteJSON bool
)

func init() {
	workDispatchCmd.Flags().IntVar(&workDispatchCap, "cap", dispatch.DefaultCap, "max concurrent workers")
	workPromoteCmd.Flags().IntVar(&workPromotePR, "pr", 0, "promotion PR number (default: the open dev→release PR)")
	workPromoteCmd.Flags().BoolVar(&workPromoteJSON, "json", false, "emit the machine-readable verdict instead of the message")
	workCmd.AddCommand(workDispatchCmd, workPromoteCmd, workSprintScanCmd)
}
