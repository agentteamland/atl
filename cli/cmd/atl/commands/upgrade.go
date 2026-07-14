package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/agentteamland/atl/cli/internal/buildinfo"
	"github.com/agentteamland/atl/cli/internal/selfupdate"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Update the atl binary to the latest stable release",
	Long: "Resolve the latest stable atl release, and if it is newer than the running\n" +
		"build, download it, verify its checksum, and atomically replace this binary.\n" +
		"Only ever upgrades (never downgrades); a dev build is left untouched; set\n" +
		"ATL_NO_SELF_UPDATE to disable. On Windows it reports the new version instead\n" +
		"(a running .exe can't self-replace — rerun the install script).",
	// Hidden until the feature's docs land (the docs correction is sequenced last,
	// per the atl-binary-self-update decision) — this keeps it out of the docs
	// coverage gate. Unhide + add cli/upgrade.md in the docs PR. Invocation still
	// works while hidden (the session-start auto-apply spawns it as a subprocess).
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		st, err := selfupdate.Check(ctx, buildinfo.Version)
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}
		if !st.Upgrade {
			if st.Reason != "" {
				fmt.Printf("atl %s — %s\n", st.Current, st.Reason)
			} else {
				fmt.Printf("atl %s is already the latest.\n", st.Current)
			}
			return nil
		}

		fmt.Printf("Upgrading atl %s → %s …\n", st.Current, st.Latest)
		if err := selfupdate.Apply(ctx, st.Latest); err != nil {
			if errors.Is(err, selfupdate.ErrWindowsManual) {
				fmt.Printf("A newer atl (%s) is available. On Windows, rerun the install script to upgrade.\n", st.Latest)
				return nil
			}
			return fmt.Errorf("applying update: %w", err)
		}
		fmt.Printf("✓ atl upgraded to %s\n", st.Latest)
		return nil
	},
}
