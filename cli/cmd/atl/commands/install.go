package commands

import (
	"fmt"
	"os"

	"github.com/agentteamland/atl/cli/internal/generation"
	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/settings"
	"github.com/agentteamland/atl/cli/internal/source"
	"github.com/agentteamland/atl/cli/internal/teampkg"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <handle>/<team>",
	Short: "Install a team from the index",
	Long: "Resolve a team from the GitHub-backed index and install it.\n\n" +
		"Scope follows the v2 axis: the publisher declares a default; --global or\n" +
		"--project overrides it. A 'both' default installs at both layers, and\n" +
		"project shadows global on conflict. Assets are written into Claude Code's\n" +
		"directory directly (~/.claude or <project>/.claude); ATL records a manifest\n" +
		"of baseline hashes under the matching .atl directory. Automation hooks are\n" +
		"installed as a mandatory part of install — automation is on by default.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("usage: atl install <handle>/<team> [--global|--project]")
		}
		g, _ := cmd.Flags().GetBool("global")
		p, _ := cmd.Flags().GetBool("project")
		if g && p {
			return fmt.Errorf("--global and --project are mutually exclusive")
		}
		override := scope.NoOverride
		switch {
		case g:
			override = scope.ForceGlobal
		case p:
			override = scope.ForceProject
		}

		handle, name, err := index.ParseRef(args[0])
		if err != nil {
			return err
		}
		ix, err := index.Resolve()
		if err != nil {
			return err
		}
		entry, err := ix.Lookup(handle, name)
		if err != nil {
			return err
		}
		declared, err := scope.Parse(entry.Scope)
		if err != nil {
			return fmt.Errorf("team %s declares an %w", entry.Ref(), err)
		}
		eff := scope.Resolve(declared, override)

		projectRoot, err := os.Getwd()
		if err != nil {
			return err
		}

		// Fetch the source once; reflect into each target layer.
		srcDir, err := source.Fetch(entry.Source.Repo, entry.Source.Subpath, entry.Source.Ref)
		if err != nil {
			return err
		}
		defer os.RemoveAll(srcDir)

		tm, err := teampkg.ReadManifest(srcDir)
		if err != nil {
			return err
		}

		targets := installTargets(eff)
		for _, target := range targets {
			if err := installAt(target, projectRoot, handle, name, entry, tm, srcDir); err != nil {
				return err
			}
			if target == scope.Global {
				_ = generation.Bump() // global layer changed → other projects fan out
			}
		}

		// Automation is mandatory (decision doc D-3): bind the hooks on install.
		// A failure here shouldn't fail the install — surface it and move on.
		if _, herr := settings.InstallHooks(defaultHooks()); herr != nil {
			fmt.Printf("atl: warning — could not bind automation hooks: %v\n", herr)
		}

		// Reflect the platform core (rules + skills) into the global layer — it
		// ships in the binary and underpins every team. Best-effort.
		if _, cerr := reflectCore(); cerr != nil {
			fmt.Printf("atl: warning — could not reflect core: %v\n", cerr)
		}

		fmt.Printf("atl: installed %s@%s at %s scope\n", entry.Ref(), tm.Version, scopeLabel(targets))
		return nil
	},
}

// installTargets expands an effective scope into the concrete single layers to
// write. A "both" install writes to global and project.
func installTargets(eff scope.Scope) []scope.Scope {
	if eff == scope.Both {
		return []scope.Scope{scope.Global, scope.Project}
	}
	return []scope.Scope{eff}
}

// installAt reflects the fetched team (srcDir) into one layer and writes the
// install manifest. It does no network I/O, so it is the unit-testable core.
func installAt(target scope.Scope, projectRoot, handle, name string, entry *index.Entry, tm *teampkg.TeamManifest, srcDir string) error {
	claudeDir, err := scope.ClaudeDir(target, projectRoot)
	if err != nil {
		return err
	}
	layerDir, err := scope.LayerDir(target, projectRoot)
	if err != nil {
		return err
	}
	files, err := teampkg.CopyAssets(srcDir, claudeDir)
	if err != nil {
		return err
	}
	version := tm.Version
	if version == "" {
		version = entry.Version
	}
	m := &manifest.Manifest{
		Handle:  handle,
		Name:    name,
		Version: version,
		Scope:   target.String(),
		Source:  manifest.Source{Repo: entry.Source.Repo, Subpath: entry.Source.Subpath, Ref: entry.Source.Ref},
		Files:   files,
	}
	return m.Write(layerDir)
}

// defaultHooks are the automation hooks install binds (mandatory, D-3). The
// throttle mirrors setup-hooks' default.
func defaultHooks() []settings.Hook {
	return []settings.Hook{
		{Event: "SessionStart", Command: "atl session-start"},
		{Event: "UserPromptSubmit", Command: "atl tick --throttle=10m"},
	}
}

func scopeLabel(targets []scope.Scope) string {
	if len(targets) == 2 {
		return "both"
	}
	return targets[0].String()
}

func init() {
	installCmd.Flags().Bool("global", false, "install at user-global scope")
	installCmd.Flags().Bool("project", false, "install at project scope")
}
