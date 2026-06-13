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
		"recently? The same checks run automatically every session and self-heal\n" +
		"what they safely can; this command is the manual diagnostic surface.\n\n" +
		"Deterministic fixes (hook re-bind, fan-out retry) land as their\n" +
		"dependencies arrive. Queue items need an LLM to process, so the doctor\n" +
		"signals a backlog rather than draining it itself.",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			fmt.Printf("%-5s queue — cannot open: %v\n", doctor.Fail, err)
			fmt.Println("\ndoctor: failures above")
			return nil
		}
		defer st.Close()

		results := doctor.Run(doctor.QueueChecks(st, project, time.Now()))
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
		default:
			fmt.Println("\ndoctor: failures above")
		}
		return nil
	},
}
