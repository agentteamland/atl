package commands

import "github.com/spf13/cobra"

var installCmd = &cobra.Command{
	Use:   "install <handle>/<team>",
	Short: "Install a team from the registry index",
	Long: "Resolve a team from the GitHub-backed index and install it.\n\n" +
		"Scope follows the v2 axis: the publisher declares a default; --global or\n" +
		"--project overrides it. Project installs shadow global on conflict.",
	Args: cobra.MaximumNArgs(1),
	RunE: stub("install"),
}

func init() {
	installCmd.Flags().Bool("global", false, "install at user-global scope")
	installCmd.Flags().Bool("project", false, "install at project scope")
}
