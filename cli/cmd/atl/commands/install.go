package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentteamland/atl/cli/internal/generation"
	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scaffold"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/settings"
	"github.com/agentteamland/atl/cli/internal/source"
	"github.com/agentteamland/atl/cli/internal/teampkg"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <handle>/<team>",
	Short: "Install a team (and its dependencies) from the index",
	Long: "Resolve a team from the GitHub-backed index and install it, together\n" +
		"with any teams it declares in team.json `dependencies` (transitively).\n\n" +
		"Scope follows the v2 axis: the publisher declares a default; --global or\n" +
		"--project overrides it. A 'both' default installs at both layers, and\n" +
		"project shadows global on conflict. A dependency installs at its OWN\n" +
		"declared scope (a global dependency stays global regardless of how the\n" +
		"consumer is installed). Assets are written into Claude Code's directory\n" +
		"directly (~/.claude or <project>/.claude); ATL records a manifest of\n" +
		"baseline hashes under the matching .atl directory. Automation hooks are\n" +
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

		projectRoot, err := os.Getwd()
		if err != nil {
			return err
		}

		// Install the team + its transitive dependencies. Fetch is the real
		// network fetcher here; tests inject a fake.
		visited := map[string]bool{}
		var installed []installedTeam
		if err := installWithDeps(ix, entry, override, projectRoot, source.Fetch, visited, &installed, false); err != nil {
			return err
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

		// Drop a project CLAUDE.md starter if this project has none — only-if-absent,
		// so a user's own file is never touched. Best-effort; never fail the install.
		if path, created, serr := scaffold.WriteIfAbsent(scaffold.Project, projectRoot, filepath.Base(projectRoot)); serr == nil && created {
			fmt.Printf("atl: created %s — a project CLAUDE.md starter; fill in the sections marked for you.\n", path)
		}
		// Also drop the .atl/ decision-state starters (backlog + tasks), only-if-absent.
		// Best-effort — a write failure must never fail the install.
		if statePaths, serr := scaffold.WriteStateFilesIfAbsent(projectRoot); serr == nil {
			for _, p := range statePaths {
				fmt.Printf("atl: created %s — project decision state (backlog + tasks).\n", p)
			}
		}

		for _, it := range installed {
			suffix := ""
			if it.dep {
				suffix = " (dependency)"
			}
			fmt.Printf("atl: installed %s@%s at %s scope%s\n", it.ref, it.version, it.scopeLabel, suffix)
		}
		return nil
	},
}

// installedTeam records one installed team for the end-of-run report.
type installedTeam struct {
	ref        string
	version    string
	scopeLabel string
	dep        bool
}

// installWithDeps installs entry across its target layers, then recursively
// installs the teams it declares in team.json `dependencies` — skipping "core"
// (the always-present platform core, reflected from the binary). A dependency
// installs at its OWN declared scope, never the consumer's override. visited
// (keyed by "<handle>/<name>") makes diamonds and cycles safe.
func installWithDeps(ix *index.Index, entry *index.Entry, override scope.Override, projectRoot string, fetch fetchFunc, visited map[string]bool, installed *[]installedTeam, isDep bool) error {
	if visited[entry.Ref()] {
		return nil
	}
	visited[entry.Ref()] = true

	tm, targets, err := installResolved(entry, override, projectRoot, fetch)
	if err != nil {
		return err
	}
	version := tm.Version
	if version == "" {
		version = entry.Version
	}
	*installed = append(*installed, installedTeam{ref: entry.Ref(), version: version, scopeLabel: scopeLabel(targets), dep: isDep})

	for dep := range tm.Dependencies {
		if dep == "core" {
			continue // the platform core is always present (reflected from the binary)
		}
		depEntry, derr := resolveDep(ix, dep)
		if derr != nil {
			fmt.Printf("atl: warning — dependency %q of %s is not in the index; skipping\n", dep, entry.Ref())
			continue
		}
		if err := installWithDeps(ix, depEntry, scope.NoOverride, projectRoot, fetch, visited, installed, true); err != nil {
			return err
		}
	}
	return nil
}

// installResolved fetches one entry and reflects it into each target layer,
// returning the parsed team.json (for dependency traversal) and the layers it
// wrote (for the report). It does no dependency work itself.
func installResolved(entry *index.Entry, override scope.Override, projectRoot string, fetch fetchFunc) (*teampkg.TeamManifest, []scope.Scope, error) {
	declared, err := scope.Parse(entry.Scope)
	if err != nil {
		return nil, nil, fmt.Errorf("team %s declares an %w", entry.Ref(), err)
	}
	eff := scope.Resolve(declared, override)

	srcDir, err := fetch(entry.Source.Repo, entry.Source.Subpath, entry.Source.Ref)
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(srcDir)

	tm, err := teampkg.ReadManifest(srcDir)
	if err != nil {
		return nil, nil, err
	}

	targets := installTargets(eff)
	for _, target := range targets {
		if err := installAt(target, projectRoot, entry.Handle, entry.Name, entry, tm, srcDir); err != nil {
			return nil, nil, err
		}
		if target == scope.Global {
			_ = generation.Bump() // global layer changed → other projects fan out
		}
	}
	return tm, targets, nil
}

// resolveDep resolves a dependency key: "<handle>/<name>" is looked up exactly;
// a bare name resolves by name (preferring a verified publisher).
func resolveDep(ix *index.Index, dep string) (*index.Entry, error) {
	if strings.Contains(dep, "/") {
		handle, name, err := index.ParseRef(dep)
		if err != nil {
			return nil, err
		}
		return ix.Lookup(handle, name)
	}
	return ix.LookupByName(dep)
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
//
// Re-installing an already-installed team — which happens on purpose (a repeat
// `atl install`) and, more often, automatically when a consumer team pulls in a
// dependency that is already installed — reflects under fan-out discipline against
// the existing baseline, so files the user (or the learning loop) modified are
// preserved and the baseline is carried forward. A first install is a clean copy.
// This restores v1's "re-install never clobbers local mutations" guarantee.
func installAt(target scope.Scope, projectRoot, handle, name string, entry *index.Entry, tm *teampkg.TeamManifest, srcDir string) error {
	claudeDir, err := scope.ClaudeDir(target, projectRoot)
	if err != nil {
		return err
	}
	layerDir, err := scope.LayerDir(target, projectRoot)
	if err != nil {
		return err
	}
	version := tm.Version
	if version == "" {
		version = entry.Version
	}
	src := manifest.Source{Repo: entry.Source.Repo, Subpath: entry.Source.Subpath, Ref: entry.Source.Ref}

	if existing, rerr := manifest.Read(layerDir, handle, name); rerr == nil {
		// Already installed here — reflect over the existing baseline so local
		// edits survive (pull, never clobber), then advance version + source.
		files, err := reflectWithFanout(srcDir, claudeDir, existing.Files)
		if err != nil {
			return err
		}
		existing.Version = version
		existing.Scope = target.String()
		existing.Source = src
		existing.Files = files
		return existing.Write(layerDir)
	}

	files, err := teampkg.CopyAssets(srcDir, claudeDir)
	if err != nil {
		return err
	}
	m := &manifest.Manifest{
		Handle:  handle,
		Name:    name,
		Version: version,
		Scope:   target.String(),
		Source:  src,
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
		{Event: "UserPromptSubmit", Command: "atl retrieve"},
		{Event: "PreToolUse", Matcher: "Bash|Edit|Write", Command: "atl guard"},
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
