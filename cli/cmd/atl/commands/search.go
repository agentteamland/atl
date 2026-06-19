package commands

import (
	"fmt"
	"strings"

	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [keyword]",
	Short: "Search the team catalog",
	Long: "Search the AgentTeamLand team catalog by handle, name, description, or\n" +
		"keyword. Resolves the same index `atl install` uses — offline-first: the\n" +
		"network-refreshed cache when present, otherwise the embedded seed. Run\n" +
		"without a keyword to browse the whole catalog.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ix, err := index.Resolve()
		if err != nil {
			return err
		}
		query := ""
		if len(args) == 1 {
			query = args[0]
		}
		hits := ix.Search(query)
		switch {
		case len(hits) == 0:
			fmt.Printf("atl search: no teams matching %q\n", query)
			return nil
		case query == "":
			fmt.Printf("%d team(s) in the catalog:\n\n", len(hits))
		default:
			fmt.Printf("%d team(s) matching %q:\n\n", len(hits), query)
		}
		for _, e := range hits {
			badge := ""
			if e.Verified {
				badge = " [verified]"
			}
			fmt.Printf("  %s@%s%s\n", e.Ref(), e.Version, badge)
			if e.Description != "" {
				fmt.Printf("    %s\n", e.Description)
			}
			if len(e.Keywords) > 0 {
				fmt.Printf("    keywords: %s\n", strings.Join(e.Keywords, ", "))
			}
			fmt.Printf("    install: atl install %s\n", e.Ref())
			fmt.Println()
		}
		return nil
	},
}
