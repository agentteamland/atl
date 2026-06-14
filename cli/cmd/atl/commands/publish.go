package commands

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/publish"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/source"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish <handle>/<team>",
	Short: "Publish your global-layer gains — re-publish (own team) or propose upstream",
	Long: "Publish the gains accumulated in your global layer for a team — ring 2→3 of\n" +
		"gain circulation. It diffs your global copy against the team's published\n" +
		"version; the differing files are the publishable gains. If you own the\n" +
		"team's repo they re-publish to it; otherwise they're proposed upstream as a\n" +
		"best-effort contribution (a gh fork + PR) the owner can accept — your own\n" +
		"local + global gains never block on acceptance.\n\n" +
		"publish is deliberate by design (it crosses the author boundary); it never\n" +
		"runs automatically. This surfaces the plan; the apply step (re-publish /\n" +
		"propose-upstream) lands in a follow-up.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		handle, name, err := index.ParseRef(args[0])
		if err != nil {
			return err
		}
		globalLayer, err := scope.LayerDir(scope.Global, "")
		if err != nil {
			return err
		}
		globalClaude, err := scope.ClaudeDir(scope.Global, "")
		if err != nil {
			return err
		}
		m, err := manifest.Read(globalLayer, handle, name)
		if err != nil {
			return fmt.Errorf("%s/%s is not installed at the global layer (publish works from your global gains): %w", handle, name, err)
		}

		// Fetch the published version to diff the global gains against.
		srcDir, err := source.Fetch(m.Source.Repo, m.Source.Subpath, m.Source.Ref)
		if err != nil {
			return fmt.Errorf("fetch published version: %w", err)
		}
		defer os.RemoveAll(srcDir)

		candidates := make([]string, 0, len(m.Files))
		for rel := range m.Files {
			candidates = append(candidates, rel)
		}
		sort.Strings(candidates)

		changes, err := publish.Plan(globalClaude, srcDir, candidates)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			fmt.Printf("atl publish: nothing to publish — your global %s/%s matches the published %s\n", handle, name, m.Source.Ref)
			return nil
		}

		owns := publish.Owns(m.Source.Repo, ghLogin())
		fmt.Printf("atl publish: %d publishable gain(s) in %s/%s (vs published %s):\n", len(changes), handle, name, m.Source.Ref)
		for _, c := range changes {
			kind := "modified"
			if c.New {
				kind = "new"
			}
			fmt.Printf("  %-9s %s\n", kind, c.Rel)
		}
		if owns {
			fmt.Printf("\nYou own %s — these would re-publish to it (commit + version bump + tag).\n", m.Source.Repo)
		} else {
			fmt.Printf("\nYou don't own %s — these would be proposed upstream (a gh fork + PR; the /publish skill writes the PR body).\n", m.Source.Repo)
		}
		fmt.Println("(The apply step lands in a follow-up — this is the plan.)")
		return nil
	},
}

// ghLogin returns the authenticated GitHub login via gh, or "" if unavailable.
func ghLogin() string {
	out, err := exec.Command("gh", "api", "user", "-q", ".login").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
