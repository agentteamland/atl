package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/agentteamland/atl/cli/internal/gc"
	"github.com/spf13/cobra"
)

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Reclaim orphaned assets — the reversible inverse of install",
	Long: "Reclaim orphaned assets across the global and project layers — files under\n" +
		".claude/agents|skills|rules that no install manifest owns (a file dropped\n" +
		"upstream on update, a learning-loop gain left after a team was removed, or a\n" +
		"hand-made dir), plus promote conflict archives older than 30 days.\n\n" +
		"gc is the reversible inverse of install. doctor heals (restores manifest-listed\n" +
		"files); gc prunes (removes files no manifest owns) — and never irreversibly:\n" +
		"  atl gc                  report only (dry run — touches nothing)\n" +
		"  atl gc --apply          soft-delete to ~/.atl/gc-trash (reversible)\n" +
		"  atl gc --apply --include-gains  also reclaim gains beside installed units\n" +
		"  atl gc --undo           restore the most recent soft-delete batch\n" +
		"  atl gc --purge          hard-delete expired trash batches (the only real delete)\n\n" +
		"A file that sits beside an installed unit but no manifest lists (a learning-loop\n" +
		"gain a /drain grew, or a hand edit) is RETAINED by --apply unless you pass\n" +
		"--include-gains — so the automatic gc pass can never sweep accumulated learning.",
	RunE: func(cmd *cobra.Command, args []string) error {
		apply, _ := cmd.Flags().GetBool("apply")
		undo, _ := cmd.Flags().GetBool("undo")
		purge, _ := cmd.Flags().GetBool("purge")
		includeGains, _ := cmd.Flags().GetBool("include-gains")

		if err := gcFlagConflict(apply, undo, purge, includeGains); err != nil {
			return err
		}

		trash, err := gc.TrashRoot()
		if err != nil {
			return err
		}

		switch {
		case undo:
			n, err := gc.Undo(trash)
			if err != nil {
				return err
			}
			if n == 0 {
				fmt.Println("gc: nothing to undo (trash is empty)")
			} else {
				fmt.Printf("gc: restored %d item(s) from the most recent batch\n", n)
			}
			return nil
		case purge:
			n, err := gc.Purge(trash, gc.HistoryMaxAge, time.Now())
			if err != nil {
				return err
			}
			fmt.Printf("gc: purged %d expired trash batch(es) for good\n", n)
			return nil
		}

		project, err := os.Getwd()
		if err != nil {
			return err
		}
		orphans, err := gc.Scan(project, time.Now())
		if err != nil {
			return err
		}
		if len(orphans) == 0 {
			fmt.Println("gc: nothing to reclaim — no orphaned assets")
			return nil
		}

		// Partition: a gain (a file beside an installed unit, owned by no manifest)
		// is retained by default so the automatic pass never sweeps learning; only
		// wholly-unowned units + expired history are swept unless --include-gains.
		var sweepable, gains []gc.Orphan
		for _, o := range orphans {
			if o.Owned && !includeGains {
				gains = append(gains, o)
			} else {
				sweepable = append(sweepable, o)
			}
		}

		var bytes int64
		for _, o := range sweepable {
			bytes += o.Size
		}

		if !apply {
			if len(sweepable) > 0 {
				fmt.Printf("gc: %d orphaned item(s) (%s) — dry run, nothing removed:\n", len(sweepable), humanBytes(bytes))
				for _, o := range sweepable {
					fmt.Printf("  [%-7s] %s  (%s)\n", o.Scope, o.Rel, o.Origin())
				}
				fmt.Println("\nRun `atl gc --apply` to soft-delete these to ~/.atl/gc-trash — reversible with `atl gc --undo`.")
			} else {
				fmt.Println("gc: nothing to reclaim — no unowned orphans")
			}
			reportRetainedGains(gains)
			return nil
		}

		if len(sweepable) == 0 {
			fmt.Println("gc: nothing to reclaim — no unowned orphans")
			reportRetainedGains(gains)
			return nil
		}

		stamp := time.Now().UTC().Format("20060102-150405")
		batch, err := gc.SoftDelete(sweepable, trash, stamp)
		if err != nil {
			return err
		}
		fmt.Printf("gc: soft-deleted %d item(s) (%s) to %s\n", len(sweepable), humanBytes(bytes), batch)
		fmt.Println("Reversible: `atl gc --undo` restores them; `atl gc --purge` clears expired trash for good.")
		reportRetainedGains(gains)
		return nil
	},
}

// gcFlagConflict rejects selecting more than one mutually exclusive gc mode.
// gc has three modes — reclaim (--apply, optionally narrowed by its
// --include-gains modifier), --undo, and --purge — dispatched by the RunE
// switch. Selecting more than one is contradictory intent the switch would
// otherwise resolve silently by precedence (e.g. `--purge --apply` runs the
// irreversible purge and drops --apply). --include-gains only shapes the
// reclaim path, so `--apply --include-gains` is a single mode; pairing
// --include-gains with --undo or --purge is rejected like any other cross-mode
// combination rather than being silently ignored.
func gcFlagConflict(apply, undo, purge, includeGains bool) error {
	if boolCount(apply || includeGains, undo, purge) > 1 {
		return fmt.Errorf("conflicting gc modes: --apply (optionally with --include-gains), --undo and --purge are mutually exclusive")
	}
	return nil
}

// reportRetainedGains notes the gains gc kept (files beside an installed unit
// that no manifest owns — learning-loop growth or hand edits), so the retention
// is visible and the escape hatch is discoverable.
func reportRetainedGains(gains []gc.Orphan) {
	if len(gains) == 0 {
		return
	}
	fmt.Printf("gc: retained %d gain(s) beside installed units (learning-loop growth or hand edits):\n", len(gains))
	for _, o := range gains {
		fmt.Printf("  [%-7s] %s  (%s)\n", o.Scope, o.Rel, o.Origin())
	}
	fmt.Println("Pass `atl gc --apply --include-gains` to reclaim these too.")
}

// humanBytes formats a byte count for the gc report.
func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGT"[exp])
}

func init() {
	gcCmd.Flags().Bool("apply", false, "soft-delete orphans to ~/.atl/gc-trash (reversible)")
	gcCmd.Flags().Bool("include-gains", false, "also reclaim gains beside installed units (off by default — protects learning-loop growth)")
	gcCmd.Flags().Bool("undo", false, "restore the most recent soft-delete batch")
	gcCmd.Flags().Bool("purge", false, "hard-delete expired trash batches (irreversible)")
}
