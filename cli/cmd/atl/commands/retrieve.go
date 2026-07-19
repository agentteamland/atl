package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/agentteamland/atl/cli/internal/retrieve"
	"github.com/spf13/cobra"
)

var retrieveCmd = &cobra.Command{
	Use:    "retrieve",
	Short:  "Per-prompt knowledge retrieval (hybrid lexical + semantic)",
	Hidden: true, // internal — wired as a UserPromptSubmit hook, not typed by hand
	Long: "Per-prompt hybrid retrieval over the project's knowledge base — the read\n" +
		"side of ATL's knowledge loop (the write side is learning-capture + /drain).\n\n" +
		"The semantic half runs a small local ONNX embedder (all-MiniLM-L6-v2 int8)\n" +
		"through the pure-Go gomlx backend, so the binary stays CGO-free. `warm`\n" +
		"downloads and verifies that model and proves the pipeline loads.",
}

// retrieveWarmCmd downloads + verifies the embedding model and runs one embed to
// prove the stack. Useful as a one-time prefetch and to validate the pipeline on
// a given machine.
var retrieveWarmCmd = &cobra.Command{
	Use:          "warm",
	Short:        "Download the embedding model and warm the pipeline",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		timeout, _ := cmd.Flags().GetDuration("timeout")
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()
		out := cmd.OutOrStdout()

		t0 := time.Now()
		dir, err := retrieve.EnsureModel(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "model ready: %s (%.1fs)\n", dir, time.Since(t0).Seconds())

		t1 := time.Now()
		emb, err := retrieve.NewEmbedder(ctx, dir)
		if err != nil {
			return err
		}
		defer emb.Close()
		vecs, err := emb.Embed(ctx, []string{"how does the dispatch merge-verify work"})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "embedded ok: dim=%d cold=%dms\n", len(vecs[0]), time.Since(t1).Milliseconds())
		return nil
	},
}

func init() {
	retrieveWarmCmd.Flags().Duration("timeout", 5*time.Minute, "overall deadline for the model download + warm")
	retrieveCmd.AddCommand(retrieveWarmCmd)
}
