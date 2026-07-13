package commands

import (
	"fmt"
	"time"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and self-heal the platform",
	Long: "On-demand health check: is the queue draining? did the loop tick\n" +
		"recently? are the automation hooks bound? The same checks run automatically\n" +
		"every session and self-heal what they safely can (restoring a missing\n" +
		"installed file, re-binding a dropped automation hook); this command is the\n" +
		"manual diagnostic surface. Queue items need an LLM to process, so the doctor\n" +
		"signals a backlog rather than draining it itself.\n\n" +
		"Exits non-zero when a check FAILs (warnings never fail), so it can gate a\n" +
		"script or CI step.",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			fmt.Printf("%-5s queue — cannot open: %v\n", doctor.Fail, err)
			fmt.Println("\ndoctor: failures above")
			return fmt.Errorf("doctor: cannot open the queue: %w", err)
		}
		defer st.Close()

		results := doctor.Run(append(doctor.QueueChecks(st, project, time.Now()), integrityCheck(project), hooksCheck()))
		for _, r := range results {
			healed := ""
			if r.Healed {
				healed = " (self-healed)"
			}
			fmt.Printf("%-5s %s — %s%s\n", r.Status, r.Name, r.Detail, healed)
		}
		switch doctor.Worst(results) {
		case doctor.OK:
			fmt.Println("\ndoctor: all healthy")
		case doctor.Warn:
			fmt.Println("\ndoctor: warnings above (not fatal)")
			return nil
		default:
			fmt.Println("\ndoctor: failures above")
			return fmt.Errorf("doctor: one or more checks failed")
		}
		return nil
	},
}
