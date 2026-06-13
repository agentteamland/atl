package commands

import "github.com/spf13/cobra"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh installed teams",
	Long: "Refresh installed teams. Runs automatically in the background (the\n" +
		"three-speed cadence: per-prompt local fan-out + throttled network update);\n" +
		"this command is the manual surface. Unmodified project copies refresh from\n" +
		"global; modified copies are preserved (pull, never push).",
	RunE: stub("update"),
}
