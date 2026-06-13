package commands

import (
	"fmt"

	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <handle>/<team>",
	Short: "Install a team from the registry index",
	Long: "Resolve a team from the GitHub-backed index and install it.\n\n" +
		"Scope follows the v2 axis: the publisher declares a default; --global or\n" +
		"--project overrides it. Project installs shadow global on conflict.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, _ := cmd.Flags().GetBool("global")
		p, _ := cmd.Flags().GetBool("project")
		if g && p {
			return fmt.Errorf("--global and --project are mutually exclusive")
		}
		override := scope.NoOverride
		switch {
		case g:
			override = scope.ForceGlobal
		case p:
			override = scope.ForceProject
		}
		// The publisher's declared default will come from the team manifest;
		// until install is real, assume the project default.
		declared := scope.Project
		eff := scope.Resolve(declared, override)

		name := "<handle>/<team>"
		if len(args) > 0 {
			name = args[0]
		}
		fmt.Printf("atl install %s — v2 scaffold; would install at %s scope "+
			"(publisher default: %s, your override: %s)\n",
			name, eff, declared, override)
		return nil
	},
}

func init() {
	installCmd.Flags().Bool("global", false, "install at user-global scope")
	installCmd.Flags().Bool("project", false, "install at project scope")
}
