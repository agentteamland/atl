package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/teampkg"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh installed teams",
	Long: "Refresh installed teams. Runs automatically in the background (the\n" +
		"three-speed cadence: per-prompt local fan-out + throttled network update);\n" +
		"this command is the manual surface. It pulls a fresh index, upgrades any\n" +
		"team with a newer published version, and fans global gains out to project\n" +
		"copies — always preserving files you modified locally (pull, never push).",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		// Best-effort network refresh of the index cache; Resolve falls back to
		// cache/embedded on failure, so being offline is fine.
		_ = index.RefreshCache(index.RawURL)

		upgraded, err := updateTeams(projectRoot)
		if err != nil {
			return err
		}
		refreshed, err := fanOut(projectRoot)
		if err != nil {
			return err
		}

		switch {
		case upgraded > 0 && refreshed > 0:
			fmt.Printf("atl update: upgraded %d team(s), refreshed %d file(s) from global\n", upgraded, refreshed)
		case upgraded > 0:
			fmt.Printf("atl update: upgraded %d team(s)\n", upgraded)
		case refreshed > 0:
			fmt.Printf("atl update: refreshed %d file(s) from the global layer\n", refreshed)
		default:
			fmt.Println("atl update: everything up to date")
		}
		return nil
	},
}

// fanOut performs the local fan-out (decision doc 5.5): for every team installed
// at BOTH the global and project layers, unmodified project files refresh from
// the global copy while user-modified files are preserved. Pull, never push —
// "modified" is judged against the project manifest's install-time baseline, so
// a local edit is never silently overwritten.
func fanOut(projectRoot string) (refreshed int, err error) {
	globalLayer, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return 0, err
	}
	projLayer, err := scope.LayerDir(scope.Project, projectRoot)
	if err != nil {
		return 0, err
	}
	globalClaude, err := scope.ClaudeDir(scope.Global, "")
	if err != nil {
		return 0, err
	}
	projClaude, err := scope.ClaudeDir(scope.Project, projectRoot)
	if err != nil {
		return 0, err
	}

	projManifests, err := manifest.List(projLayer)
	if err != nil {
		return 0, err
	}
	for _, pm := range projManifests {
		// Only teams that also live in the global layer have an upstream to pull
		// from. Project-only teams refresh via the network path (updateTeams).
		if _, rerr := manifest.Read(globalLayer, pm.Handle, pm.Name); rerr != nil {
			continue
		}
		changed := false
		for file, baseline := range pm.Files {
			upstream, e := fanout.HashFile(filepath.Join(globalClaude, file))
			if e != nil {
				return refreshed, e
			}
			if upstream == "" {
				continue // not present in the global copy → nothing to fan out
			}
			local, e := fanout.HashFile(filepath.Join(projClaude, file))
			if e != nil {
				return refreshed, e
			}
			if fanout.Decide(baseline, local, upstream) != fanout.Refresh {
				continue // Preserve (user-modified) or UpToDate
			}
			if e := teampkg.CopyFile(filepath.Join(globalClaude, file), filepath.Join(projClaude, file)); e != nil {
				return refreshed, e
			}
			pm.Files[file] = upstream // advance the baseline to what we just took
			changed = true
			refreshed++
		}
		if changed {
			if e := pm.Write(projLayer); e != nil {
				return refreshed, e
			}
		}
	}
	return refreshed, nil
}
