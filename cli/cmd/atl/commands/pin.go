package commands

import (
	"fmt"
	"os"

	"github.com/agentteamland/atl/cli/internal/pin"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/spf13/cobra"
)

var pinCmd = &cobra.Command{
	Use:   "pin [path]",
	Short: "Keep a project-local path from being promoted to global",
	Long: "Mark a path (relative to this project's .claude dir) as project-only, so\n" +
		"`atl promote` never lifts it — or its subtree — to the global layer. Use it\n" +
		"for a deliberately project-specific agent/skill/rule customization you don't\n" +
		"want circulating to your other projects. With no argument, lists the current\n" +
		"pins.\n\n" +
		"Promotion is automatic by default; a pin is the declarative opt-out (it does\n" +
		"not stop fan-out, which already preserves any file you've changed locally).\n" +
		"Examples: `atl pin agents/api-agent`, `atl pin rules/house-style.md`.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		layer, err := projectAtlDir()
		if err != nil {
			return err
		}
		set, err := pin.Load(layer)
		if err != nil {
			return err
		}
		if len(args) == 0 {
			if len(set.Pins) == 0 {
				fmt.Println("atl pin: no pins — every gain promotes to global")
				return nil
			}
			fmt.Println("atl pin — project-only paths (never promoted):")
			for _, p := range set.Pins {
				fmt.Printf("  %s\n", p)
			}
			return nil
		}
		if set.Add(args[0]) {
			if err := set.Write(layer); err != nil {
				return err
			}
			fmt.Printf("atl pin: %s is now project-only (won't be promoted)\n", pin.Normalize(args[0]))
		} else {
			fmt.Printf("atl pin: %s is already pinned\n", pin.Normalize(args[0]))
		}
		return nil
	},
}

var unpinCmd = &cobra.Command{
	Use:   "unpin <path>",
	Short: "Allow a previously pinned path to be promoted again",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		layer, err := projectAtlDir()
		if err != nil {
			return err
		}
		set, err := pin.Load(layer)
		if err != nil {
			return err
		}
		if set.Remove(args[0]) {
			if err := set.Write(layer); err != nil {
				return err
			}
			fmt.Printf("atl unpin: %s will be promoted again\n", pin.Normalize(args[0]))
		} else {
			fmt.Printf("atl unpin: %s was not pinned\n", pin.Normalize(args[0]))
		}
		return nil
	},
}

// projectAtlDir returns the current project's .atl layer root (where pins live).
func projectAtlDir() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return scope.LayerDir(scope.Project, root)
}
