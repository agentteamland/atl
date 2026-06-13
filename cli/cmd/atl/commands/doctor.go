package commands

import "github.com/spf13/cobra"

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and self-heal the platform",
	Long: "On-demand health check: are hooks bound? is the queue draining? did the\n" +
		"loop fire recently? The same checks run automatically every session inside\n" +
		"the session-start path and self-heal what they can (queue retry, fan-out\n" +
		"re-run, hook re-bind); this command is the manual diagnostic surface.",
	RunE: stub("doctor"),
}
