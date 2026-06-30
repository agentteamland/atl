package commands

import (
	"encoding/json"
	"io"
	"os"

	"github.com/agentteamland/atl/cli/internal/guard"
	"github.com/spf13/cobra"
)

var guardCmd = &cobra.Command{
	Use:    "guard",
	Short:  "PreToolUse enforcement hook (run by Claude Code)",
	Hidden: true, // an internal hook command, never typed by a user
	Long: "Run by the PreToolUse hook on Bash / Edit / Write tool calls. Promotes ATL's\n" +
		"prose discipline to deterministic enforcement, in two layers split by\n" +
		"reversibility:\n\n" +
		"  • catastrophe (deny) — irreversible Bash operations (force-push,\n" +
		"    reset --hard, force-clean, destructive SQL, --no-verify) are blocked.\n" +
		"  • quality (non-blocking) — the first edit of an existing file injects a\n" +
		"    grep-before-edit reminder as context, with no permission decision.\n\n" +
		"Reads the hook JSON on stdin and prints a decision on stdout. Never fails:\n" +
		"on any error it stays silent and lets the tool call proceed.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil // never block on guard's own failure
		}
		var in guard.Input
		if err := json.Unmarshal(data, &in); err != nil {
			return nil
		}
		res := guard.Decide(in, fileExists, guard.FirstEditFunc(in.SessionID))
		out := guardOutput(res)
		if out == nil {
			return nil // no-op: emit nothing, normal permission flow applies
		}
		_ = json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		return nil
	},
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// guardOutput maps a guard.Result to the Claude Code PreToolUse hookSpecificOutput
// JSON, or nil for a no-op. A deny sets permissionDecision; a context nudge omits
// it (injecting additionalContext without overriding the permission flow).
func guardOutput(res guard.Result) map[string]any {
	switch res.Action {
	case guard.Deny:
		return map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":            "PreToolUse",
				"permissionDecision":       "deny",
				"permissionDecisionReason": res.Reason,
			},
		}
	case guard.Context:
		return map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":     "PreToolUse",
				"additionalContext": res.Reason,
			},
		}
	default:
		return nil
	}
}
