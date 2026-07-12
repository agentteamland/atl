package commands

import (
	"fmt"
	"os"

	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed teams",
	Long: "List the teams installed at each scope — global (`~/.claude`) and project\n" +
		"(`<cwd>/.claude`). A team present at both is shown under each.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		scopes := []struct {
			label string
			s     scope.Scope
			root  string
		}{
			{"global", scope.Global, ""},
			{"project", scope.Project, projectRoot},
		}
		total := 0
		for _, sc := range scopes {
			layer, err := scope.LayerDir(sc.s, sc.root)
			if err != nil {
				return err
			}
			ms, err := manifest.List(layer)
			if err != nil {
				return err
			}
			if len(ms) == 0 {
				continue
			}
			fmt.Printf("%s:\n", sc.label)
			for _, m := range ms {
				fmt.Printf("  %s/%s@%s\n", m.Handle, m.Name, m.Version)
				total++
			}
		}
		if total == 0 {
			fmt.Println("atl list: no teams installed")
		}
		return nil
	},
}
