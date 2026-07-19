package commands

import (
	"fmt"
	"os"

	"github.com/agentteamland/atl/cli/internal/buildinfo"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "atl",
	Short:         "AgentTeamLand — the agent-team platform CLI",
	Long:          "atl is the AgentTeamLand platform CLI: install teams, keep them updated,\ncirculate the gains your agents learn, and let the platform run itself in the\nbackground so you can focus on your project.",
	Version:       buildinfo.Version,
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
		guardCmd,
		doctorCmd,
		listCmd,
		searchCmd,
		removeCmd,
		docsCmd,
		initCmd,
		gcCmd,
		skillsCmd,
		rulesCmd,
		workCmd,
		upgradeCmd,
		retrieveCmd,
	)
}
