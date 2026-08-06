package commands

import (
	"encoding/json"
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
		asJSON, _ := cmd.Flags().GetBool("json")
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
		var entries []listEntry
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
			if !asJSON {
				fmt.Printf("%s:\n", sc.label)
			}
			for _, m := range ms {
				if asJSON {
					entries = append(entries, listEntry{
						Handle: m.Handle, Name: m.Name, Version: m.Version,
						Scope: sc.label, Reviewer: m.Reviewer,
					})
				} else {
					line := fmt.Sprintf("  %s/%s@%s", m.Handle, m.Name, m.Version)
					if m.Reviewer != "" {
						line += fmt.Sprintf("  (review: %s)", m.Reviewer)
					}
					fmt.Println(line)
				}
				total++
			}
		}
		if asJSON {
			// An empty install set prints [], not null — a consumer piping this
			// into jq must not have to special-case "no teams".
			if entries == nil {
				entries = []listEntry{}
			}
			out, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		}
		if total == 0 {
			fmt.Println("atl list: no teams installed")
		}
		return nil
	},
}

// listEntry is the machine-readable shape of an installed team. Deliberately not
// the manifest itself: that carries a per-file SHA map running to hundreds of
// entries, and a consumer asking "which teams are installed, and what does each
// declare?" should not have to wade through it.
type listEntry struct {
	Handle   string `json:"handle"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Scope    string `json:"scope"`
	Reviewer string `json:"reviewer,omitempty"`
}

func init() {
	listCmd.Flags().Bool("json", false, "emit the installed teams as JSON, including each team's declared review agent")
}
