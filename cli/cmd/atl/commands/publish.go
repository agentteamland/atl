package commands

import "github.com/spf13/cobra"

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish your team to the index, or propose upstream",
	Long: "Publish an update to a team you own (re-publish to your repo + index), or\n" +
		"prepare a best-effort upstream contribution for a team you don't own\n" +
		"(ring 2→3). Your own local + global gains never block on acceptance.",
	RunE: stub("publish"),
}
