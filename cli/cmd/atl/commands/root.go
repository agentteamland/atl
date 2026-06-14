package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is the platform version. v2 is pre-release until feature-parity
// cutover (decision doc, migration step 5).
const version = "2.0.0-dev"

var rootCmd = &cobra.Command{
	Use:           "atl",
	Short:         "AgentTeamLand — the agent-team platform CLI",
	Long:          "atl is the AgentTeamLand platform CLI: install teams, keep them updated,\ncirculate the gains your agents learn, and let the platform run itself in the\nbackground so you can focus on your project.",
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "atl: "+err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		installCmd,
		updateCmd,
		promoteCmd,
		pinCmd,
		unpinCmd,
		publishCmd,
		learningsCmd,
		tickCmd,
		sessionStartCmd,
		setupHooksCmd,
		doctorCmd,
	)
}
