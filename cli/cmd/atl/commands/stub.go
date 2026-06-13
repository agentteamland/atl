package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stub is the placeholder RunE for commands whose v2 logic lands in a later
// implementation step. It keeps the command surface real (and the binary
// buildable) while the deterministic plumbing behind each verb is filled in.
func stub(name string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("atl %s — v2 scaffold; implementation pending (see .atl/docs/atl-v2.md)\n", name)
		return nil
	}
}
