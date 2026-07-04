package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func findChild(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func TestWorkCommandTree(t *testing.T) {
	work := findChild(rootCmd, "work")
	if work == nil {
		t.Fatal("`work` command not registered on root")
	}
	if !work.Hidden {
		t.Error("`work` must be Hidden — internal engine, delivery-team not shipped, kept out of the docs-coverage gate")
	}

	dispatch := findChild(work, "dispatch")
	if dispatch == nil {
		t.Fatal("`dispatch` subcommand not registered on `work`")
	}
	if dispatch.RunE == nil {
		t.Error("`work dispatch` should have a RunE")
	}
	if err := dispatch.Args(dispatch, []string{}); err != nil {
		t.Errorf("`work dispatch` should accept zero args: %v", err)
	}
}
