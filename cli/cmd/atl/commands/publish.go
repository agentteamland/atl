package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path"
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
		"runs automatically. By default it shows the plan; --apply re-publishes to a\n" +
		"team you own (commit + version bump + tag) or forks + opens the PR\n" +
		"(propose-upstream) for one you don't.",
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

		// --only restricts to the skill's kept subset (its judgment step drops
		// project/user-specific gains before applying).
		if only, _ := cmd.Flags().GetStringSlice("only"); len(only) > 0 {
			keep := make(map[string]bool, len(only))
			for _, o := range only {
				keep[o] = true
			}
			filtered := changes[:0:0]
			for _, c := range changes {
				if keep[c.Rel] {
					filtered = append(filtered, c)
				}
			}
			changes = filtered
		}

		if len(changes) == 0 {
			fmt.Printf("atl publish: nothing to publish — your global %s/%s matches the published %s\n", handle, name, m.Source.Ref)
			return nil
		}

		owns := publish.Owns(m.Source.Repo, ghLogin())

		// Always show the plan first — publish is a deliberate, visible act.
		fmt.Printf("atl publish: %d publishable gain(s) in %s/%s (vs published %s):\n", len(changes), handle, name, m.Source.Ref)
		for _, c := range changes {
			kind := "modified"
			if c.New {
				kind = "new"
			}
			fmt.Printf("  %-9s %s\n", kind, c.Rel)
		}

		apply, _ := cmd.Flags().GetBool("apply")
		if !apply {
			if owns {
				fmt.Printf("\nYou own %s — these would re-publish to it (commit + version bump + tag).\n", m.Source.Repo)
			} else {
				fmt.Printf("\nYou don't own %s — these would be proposed upstream (a gh fork + PR).\n", m.Source.Repo)
			}
			fmt.Println("\nRe-run with --apply to act (the /publish skill authors the PR body and passes it via --body-file).")
			return nil
		}

		// --- apply ---
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		bodyFile, _ := cmd.Flags().GetString("body-file")

		if owns {
			if dryRun {
				fmt.Printf("\nDRY RUN — would re-publish to %s (you own it):\n", m.Source.Repo)
				fmt.Printf("  stage %d file(s) under %q, bump team.json version, commit + tag + push\n", len(changes), m.Source.Subpath)
				for _, c := range changes {
					fmt.Printf("    %s\n", path.Join(m.Source.Subpath, c.Rel))
				}
				fmt.Println("  ensure the atl-team topic so index CI reindexes")
				return nil
			}
			if err := requireBodyFile(bodyFile); err != nil {
				return err
			}
			tag, err := publish.RePublish(publish.NewGH(), m, globalClaude, bodyFile, changes)
			if err != nil {
				return err
			}
			fmt.Printf("\natl publish: re-published %s as %s\n", m.Source.Repo, tag)
			return nil
		}

		// not owned → propose upstream
		if dryRun {
			branch := publish.BranchName(handle, name)
			fmt.Printf("\nDRY RUN — would propose upstream to %s:\n", m.Source.Repo)
			fmt.Printf("  fork %s → your account, branch %s off the default branch\n", m.Source.Repo, branch)
			fmt.Printf("  stage %d file(s) under %q:\n", len(changes), m.Source.Subpath)
			for _, c := range changes {
				fmt.Printf("    %s\n", path.Join(m.Source.Subpath, c.Rel))
			}
			fmt.Println("  push to your fork, then open a PR against the source repo")
			return nil
		}
		if err := requireBodyFile(bodyFile); err != nil {
			return err
		}
		url, err := publish.ProposeUpstream(publish.NewGH(), m, globalClaude, bodyFile, changes)
		if err != nil {
			return err
		}
		fmt.Printf("\natl publish: opened %s\n", url)
		return nil
	},
}

// requireBodyFile validates the --body-file flag for an apply: it must be set
// (the /publish skill authors the PR body / commit message) and exist.
func requireBodyFile(bodyFile string) error {
	if bodyFile == "" {
		return fmt.Errorf("--apply needs --body-file (the /publish skill authors it); pass --dry-run to preview without it")
	}
	if _, err := os.Stat(bodyFile); err != nil {
		return fmt.Errorf("--body-file %q: %w", bodyFile, err)
	}
	return nil
}

func init() {
	publishCmd.Flags().Bool("apply", false, "apply the plan (fork + open the PR), not just show it")
	publishCmd.Flags().String("body-file", "", "file holding the PR body (the /publish skill authors it); required with --apply")
	publishCmd.Flags().StringSlice("only", nil, "restrict to these .claude-relative paths (the skill's kept subset)")
	publishCmd.Flags().Bool("dry-run", false, "with --apply, print what would happen without forking or pushing")
}

// ghLogin returns the authenticated GitHub login via gh, or "" if unavailable.
func ghLogin() string {
	out, err := exec.Command("gh", "api", "user", "-q", ".login").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
