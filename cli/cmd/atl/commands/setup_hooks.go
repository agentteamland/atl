package commands

import (
	"fmt"
	"time"

	"github.com/agentteamland/atl/cli/internal/settings"
	"github.com/spf13/cobra"
)

var setupHooksCmd = &cobra.Command{
	Use:   "setup-hooks",
	Short: "Install the ATL automation hooks into Claude Code",
	Long: "Install the automation hooks into ~/.claude/settings.json. In v2 this is\n" +
		"a mandatory part of install — automation is on by default, not opt-in.\n\n" +
		"  SessionStart     → atl session-start   (drain previous session + doctor)\n" +
		"  UserPromptSubmit → atl tick --throttle  (in-session drain every interval)\n" +
		"  UserPromptSubmit → atl retrieve         (surface relevant knowledge pages)\n" +
		"  PreToolUse       → atl guard            (block irreversible ops + grep-before-edit nudge)\n\n" +
		"Idempotent: re-running replaces the atl hooks without duplicating, and\n" +
		"leaves any hooks you added yourself untouched.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		interval, _ := cmd.Flags().GetString("throttle")
		// Validate the throttle up front: an unparsable value would be baked into
		// settings.json and make every hooked `atl tick` fail, silently stopping the
		// whole automation loop.
		if _, perr := time.ParseDuration(interval); perr != nil {
			return fmt.Errorf("invalid --throttle %q: %v (use a duration like 10m or 30s)", interval, perr)
		}
		path, err := settings.InstallHooks([]settings.Hook{
			{Event: "SessionStart", Command: "atl session-start"},
			{Event: "UserPromptSubmit", Command: "atl tick --throttle=" + interval},
			{Event: "UserPromptSubmit", Command: "atl retrieve"},
			{Event: "PreToolUse", Matcher: "Bash|Edit|Write", Command: "atl guard"},
		})
		if err != nil {
			return err
		}
		fmt.Printf("atl: hooks installed into %s\n", path)
		fmt.Println("  SessionStart     → atl session-start")
		fmt.Printf("  UserPromptSubmit → atl tick --throttle=%s\n", interval)
		fmt.Println("  UserPromptSubmit → atl retrieve")
		fmt.Println("  PreToolUse       → atl guard")
		return nil
	},
}

func init() {
	setupHooksCmd.Flags().String("throttle", "10m", "tick throttle interval for the UserPromptSubmit hook")
}
