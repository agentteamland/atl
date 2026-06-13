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
	"os"
	"path/filepath"
	"strings"
)

// Hook is an event→command pair to install.
type Hook struct {
	Event   string
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
		_ = json.Unmarshal(b, &obj) // tolerate empty/corrupt; we only own the hooks key
	}
	hooksMap, _ := obj["hooks"].(map[string]any)
	if hooksMap == nil {
		hooksMap = map[string]any{}
	}

	for _, h := range hooks {
		groups, _ := hooksMap[h.Event].([]any)
		kept := make([]any, 0, len(groups)+1)
		for _, g := range groups {
			if !isAtlGroup(g) { // preserve the user's own hooks
				kept = append(kept, g)
			}
		}
		kept = append(kept, map[string]any{
			"hooks": []any{
				map[string]any{"type": "command", "command": h.Command},
			},
		})
		hooksMap[h.Event] = kept
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
