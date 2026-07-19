// Package settings installs ATL's automation hooks into Claude Code's
// settings.json idempotently.
//
// v2 makes automation mandatory (not opt-in), so this is the wiring that turns
// the three-speed cadence on: SessionStart and UserPromptSubmit invoke atl
// commands. Existing atl-owned hook groups are replaced on each install (so
// re-running never duplicates); any hooks the user added themselves are
// preserved untouched.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Hook is an event→command pair to install. Matcher is an optional tool-name
// matcher for the tool events (PreToolUse / PostToolUse) — empty means the hook
// fires on every tool call; for the non-tool events (SessionStart,
// UserPromptSubmit) it is left empty and no matcher key is emitted.
type Hook struct {
	Event   string
	Matcher string
	Command string
}

// Path returns ~/.claude/settings.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// InstallHooks idempotently merges the given atl-owned hooks into the Claude
// settings file and returns the path written. Atomic write.
func InstallHooks(hooks []Hook) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}

	obj := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		// An empty file is fine (start fresh), but a NON-empty file that won't parse
		// must not be silently overwritten — doing so would wipe every non-hook key
		// the user has (permissions, env, statusline, ...). Refuse and surface it.
		if len(strings.TrimSpace(string(b))) > 0 {
			if uerr := json.Unmarshal(b, &obj); uerr != nil {
				return "", fmt.Errorf("%s exists but is not valid JSON (%v); refusing to overwrite it — fix or remove the file, then re-run", path, uerr)
			}
		}
	}
	hooksMap, _ := obj["hooks"].(map[string]any)
	if hooksMap == nil {
		hooksMap = map[string]any{}
	}

	// Group the wanted hooks by event, preserving first-seen order. Multiple atl
	// hooks can share an event (e.g. two UserPromptSubmit hooks): filter that
	// event's existing atl groups exactly once, then append every wanted group.
	// Filtering per-hook instead would make the second hook's pass see the first
	// hook's freshly-added group as an existing atl group and drop it.
	byEvent := map[string][]Hook{}
	var order []string
	for _, h := range hooks {
		if _, seen := byEvent[h.Event]; !seen {
			order = append(order, h.Event)
		}
		byEvent[h.Event] = append(byEvent[h.Event], h)
	}
	for _, event := range order {
		groups, _ := hooksMap[event].([]any)
		kept := make([]any, 0, len(groups)+len(byEvent[event]))
		for _, g := range groups {
			if !isAtlGroup(g) { // preserve the user's own hooks
				kept = append(kept, g)
			}
		}
		for _, h := range byEvent[event] {
			group := map[string]any{
				"hooks": []any{
					map[string]any{"type": "command", "command": h.Command},
				},
			}
			if h.Matcher != "" {
				group["matcher"] = h.Matcher
			}
			kept = append(kept, group)
		}
		hooksMap[event] = kept
	}
	obj["hooks"] = hooksMap

	if err := writeJSONAtomic(path, obj); err != nil {
		return "", err
	}
	return path, nil
}

// isAtlGroup reports whether a hook group is one atl owns (any command starts
// with "atl ").
func isAtlGroup(g any) bool {
	gm, ok := g.(map[string]any)
	if !ok {
		return false
	}
	hs, ok := gm["hooks"].([]any)
	if !ok {
		return false
	}
	for _, h := range hs {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		if cmd, _ := hm["command"].(string); strings.HasPrefix(cmd, "atl ") {
			return true
		}
	}
	return false
}

func writeJSONAtomic(path string, obj any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
