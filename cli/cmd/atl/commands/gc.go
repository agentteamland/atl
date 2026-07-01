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
		"  atl gc            report only (dry run — touches nothing)\n" +
		"  atl gc --apply    soft-delete to ~/.atl/gc-trash (reversible)\n" +
		"  atl gc --undo     restore the most recent soft-delete batch\n" +
		"  atl gc --purge    hard-delete expired trash batches (the only real delete)",
	RunE: func(cmd *cobra.Command, args []string) error {
		apply, _ := cmd.Flags().GetBool("apply")
		undo, _ := cmd.Flags().GetBool("undo")
		purge, _ := cmd.Flags().GetBool("purge")

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

		var bytes int64
		for _, o := range orphans {
			bytes += o.Size
		}

		if !apply {
			fmt.Printf("gc: %d orphaned item(s) (%s) — dry run, nothing removed:\n", len(orphans), humanBytes(bytes))
			for _, o := range orphans {
				fmt.Printf("  [%-7s] %s  (%s)\n", o.Scope, o.Rel, o.Origin())
			}
			fmt.Println("\nRun `atl gc --apply` to soft-delete these to ~/.atl/gc-trash — reversible with `atl gc --undo`.")
			return nil
		}

		stamp := time.Now().UTC().Format("20060102-150405")
		batch, err := gc.SoftDelete(orphans, trash, stamp)
		if err != nil {
			return err
		}
		fmt.Printf("gc: soft-deleted %d item(s) (%s) to %s\n", len(orphans), humanBytes(bytes), batch)
		fmt.Println("Reversible: `atl gc --undo` restores them; `atl gc --purge` clears expired trash for good.")
		return nil
	},
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
	gcCmd.Flags().Bool("undo", false, "restore the most recent soft-delete batch")
	gcCmd.Flags().Bool("purge", false, "hard-delete expired trash batches (irreversible)")
}
