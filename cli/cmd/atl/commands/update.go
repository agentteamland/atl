package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/pin"
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
		// cache/embedded on failure, so being offline is fine. Capture the error
		// so the "nothing changed" message can be honest about an offline run.
		refreshErr := index.RefreshCache(index.RawURL)

		upgraded, err := updateTeams(projectRoot)
		if err != nil {
			return err
		}
		refreshed, err := fanOut(projectRoot)
		if err != nil {
			return err
		}

		// Core ships in the binary; refresh it into the global layer so a binary
		// upgrade propagates the latest rules + skills. Best-effort.
		if coreN, _ := reflectCore(); coreN > 0 {
			fmt.Printf("atl update: refreshed %d core file(s)\n", coreN)
		}

		switch {
		case upgraded > 0 && refreshed > 0:
			fmt.Printf("atl update: upgraded %d team(s), refreshed %d file(s) from global\n", upgraded, refreshed)
		case upgraded > 0:
			fmt.Printf("atl update: upgraded %d team(s)\n", upgraded)
		case refreshed > 0:
			fmt.Printf("atl update: refreshed %d file(s) from the global layer\n", refreshed)
		default:
			fmt.Println(upToDateMessage(refreshErr != nil))
		}

		// F4: this is the throttled network pass, which already re-fetched above,
		// so surface any team whose global gains aren't upstream yet. Suggestion
		// only — publishing stays an explicit, consent-gated act. Best-effort.
		suggestPublishable()
		return nil
	},
}

// upToDateMessage picks the "nothing changed" line for `atl update`. When the
// network refresh could not run (offline / fetch failed), Resolve fell back to
// the cached/embedded index — so asserting "everything up to date" would falsely
// imply the check reached the network. Say so honestly instead.
func upToDateMessage(offline bool) string {
	if offline {
		return "atl update: up to date (offline — using cached index)"
	}
	return "atl update: everything up to date"
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

	pins, err := pin.Load(projLayer)
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
		gm, rerr := manifest.Read(globalLayer, pm.Handle, pm.Name)
		if rerr != nil {
			continue
		}
		// Fan out the UNION of the project baseline and the global manifest: an
		// existing project file refreshes against its recorded baseline, while a
		// file the global layer gained (a promoted gain the project doesn't have
		// yet) has no local baseline — it copies in fresh. Without the union half,
		// promote's ring 2->3 is broken: additive gains never reach sibling projects.
		rels := map[string]string{} // rel -> project baseline ("" = global-only, no local baseline)
		for file, baseline := range pm.Files {
			rels[file] = baseline
		}
		globalOnly := map[string]bool{}
		for file := range gm.Files {
			if _, ok := pm.Files[file]; !ok {
				rels[file] = ""
				globalOnly[file] = true
			}
		}
		changed := false
		for file, baseline := range rels {
			// A global-only gain the project pinned is deliberately kept project-free.
			if globalOnly[file] && pins.Pinned(file) {
				continue
			}
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
			// baseline "" + local "" -> Refresh (copy the new gain); baseline "" +
			// an independently-created local file -> Preserve (keep the user's).
			if fanout.Decide(baseline, local, upstream) != fanout.Refresh {
				continue // Preserve (user-modified) or UpToDate
			}
			if e := teampkg.CopyFile(filepath.Join(globalClaude, file), filepath.Join(projClaude, file)); e != nil {
				return refreshed, e
			}
			pm.Files[file] = upstream // record/advance the baseline to what we just took
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
