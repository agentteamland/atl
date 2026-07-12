package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/agentteamland/atl/cli/internal/gc"
	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <handle>/<team>",
	Short: "Remove an installed team",
	Long: "Remove a team's installed files and manifest at the chosen scope. Defaults\n" +
		"to the project scope; --global removes from the user-global layer. Only the\n" +
		"files recorded in the install manifest are removed; now-empty directories are\n" +
		"pruned. Removal is reversible: the files are soft-deleted to ~/.atl/gc-trash,\n" +
		"so `atl gc --undo` restores them (and `atl gc --purge` clears the trash for good).",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		handle, name, err := index.ParseRef(args[0])
		if err != nil {
			return err
		}
		root, err := os.Getwd()
		if err != nil {
			return err
		}
		sc := scope.Project
		if g, _ := cmd.Flags().GetBool("global"); g {
			sc, root = scope.Global, ""
		}
		n, err := removeTeam(handle, name, sc, root)
		if err != nil {
			return err
		}
		fmt.Printf("atl remove: removed %s/%s (%d files) from %s scope — reversible with `atl gc --undo`\n", handle, name, n, sc)
		return nil
	},
}

// removeTeam soft-deletes a team's manifest-recorded files to ~/.atl/gc-trash and
// drops the manifest at one scope, pruning directories it empties. Returns the
// count of files removed. Reversible (`atl gc --undo`) so a promoted gain that
// landed in the manifest is never hard-deleted without recourse. Pure of
// cwd/flags so it's unit-testable.
func removeTeam(handle, name string, sc scope.Scope, root string) (int, error) {
	layer, err := scope.LayerDir(sc, root)
	if err != nil {
		return 0, err
	}
	claudeDir, err := scope.ClaudeDir(sc, root)
	if err != nil {
		return 0, err
	}
	m, err := manifest.Read(layer, handle, name)
	if err != nil {
		return 0, fmt.Errorf("%s/%s is not installed at %s scope", handle, name, sc)
	}
	// Soft-delete the files that actually exist on disk (reversible move to trash).
	var orphans []gc.Orphan
	for rel := range m.Files {
		abs := filepath.Join(claudeDir, filepath.FromSlash(rel))
		if _, e := os.Stat(abs); e == nil {
			orphans = append(orphans, gc.Orphan{Scope: sc.String(), Rel: rel, Abs: abs})
		}
	}
	removed := 0
	if len(orphans) > 0 {
		trash, terr := gc.TrashRoot()
		if terr != nil {
			return 0, terr
		}
		stamp := time.Now().UTC().Format("20060102-150405")
		if _, serr := gc.SoftDelete(orphans, trash, stamp); serr != nil {
			return 0, serr
		}
		removed = len(orphans)
	}
	pruneEmptyDirs(claudeDir, m.Files)
	if err := manifest.Remove(layer, handle, name); err != nil {
		return removed, err
	}
	return removed, nil
}

// pruneEmptyDirs removes the directories that held the team's files, deepest
// first, skipping any that aren't empty (a sibling team or user file keeps them).
func pruneEmptyDirs(claudeDir string, files map[string]string) {
	dirs := map[string]bool{}
	for rel := range files {
		d := filepath.Dir(filepath.Join(claudeDir, filepath.FromSlash(rel)))
		for len(d) > len(claudeDir) {
			dirs[d] = true
			d = filepath.Dir(d)
		}
	}
	ordered := make([]string, 0, len(dirs))
	for d := range dirs {
		ordered = append(ordered, d)
	}
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	for _, d := range ordered {
		_ = os.Remove(d) // succeeds only if empty
	}
}

func init() {
	removeCmd.Flags().Bool("global", false, "remove from the global layer instead of the project")
}
