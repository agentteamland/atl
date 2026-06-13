package commands

import "github.com/spf13/cobra"

var promoteCmd = &cobra.Command{
	Use:   "promote <agent|skill>",
	Short: "Promote project-local gains to the user-global layer",
	Long: "Lift a project-local learning set into the global-layer copy of its team\n" +
		"(ring 1→2 of gain circulation). Additive merge; on true conflict the\n" +
		"project-local value wins, the prior global value is kept as history.",
	Args: cobra.MaximumNArgs(1),
	RunE: stub("promote"),
}
