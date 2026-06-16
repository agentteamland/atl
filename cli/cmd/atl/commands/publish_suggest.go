package commands

import (
	"fmt"
	"os"

	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/publish"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/source"
)

// fetchFunc matches source.Fetch. Injecting it keeps the scan logic testable
// without hitting the network (the same convention updateTeams relies on for its
// own fetch, proven live in the e2e harness rather than unit-tested).
type fetchFunc func(repo, subpath, ref string) (dir string, err error)

// publishSuggestion is one globally-installed team whose global-layer copy has
// diverged from its published version — gains the user could contribute upstream.
type publishSuggestion struct {
	Ref   string // "<handle>/<name>"
	Count int    // number of publishable gains (differing files)
}

// detectPublishable scans every team installed at the global layer and reports
// those whose global copy differs from its published version — the same
// whole-file diff `atl publish` shows. This is the deterministic core of the F4
// suggestion. Best-effort: a team whose published source can't be fetched
// (offline, repo moved) or compared is skipped, never fatal — a suggestion must
// not break `atl update`. fetch is source.Fetch in production.
func detectPublishable(globalLayer, globalClaude string, fetch fetchFunc) ([]publishSuggestion, error) {
	manifests, err := manifest.List(globalLayer)
	if err != nil {
		return nil, err
	}
	var out []publishSuggestion
	for _, m := range manifests {
		dir, ferr := fetch(m.Source.Repo, m.Source.Subpath, m.Source.Ref)
		if ferr != nil {
			continue // can't fetch the published version → can't compare → skip
		}
		candidates := make([]string, 0, len(m.Files))
		for rel := range m.Files {
			candidates = append(candidates, rel)
		}
		changes, perr := publish.Plan(globalClaude, dir, candidates)
		os.RemoveAll(dir)
		if perr != nil {
			continue
		}
		if len(changes) > 0 {
			out = append(out, publishSuggestion{Ref: m.Handle + "/" + m.Name, Count: len(changes)})
		}
	}
	return out, nil
}

// suggestPublishable surfaces a doctor-style note for each globally-installed
// team with gains not yet upstream — the F4 "proactive suggestion" half of
// publish. The act stays explicit and consent-gated (it only points at
// `atl publish X`; nothing is published automatically). It runs from the
// throttled network pass (`atl update`), which already re-fetches; the tarball
// diff is too heavy for every tick. Best-effort and silent on any setup error.
func suggestPublishable() {
	globalLayer, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return
	}
	globalClaude, err := scope.ClaudeDir(scope.Global, "")
	if err != nil {
		return
	}
	suggestions, err := detectPublishable(globalLayer, globalClaude, source.Fetch)
	if err != nil {
		return
	}
	for _, s := range suggestions {
		fmt.Printf("atl update: gains in %s not yet upstream (%d file(s)) — run `atl publish %s` to contribute them\n",
			s.Ref, s.Count, s.Ref)
	}
}
