package dispatch

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DeliveryConfig is the subset of .delivery/config.json the zero-Azure engine reads
// to wire a spawned worker's Azure access (D3): org/project for the worker env +
// the REST carve-out, and pat.ref to resolve the PAT from the engine's own
// environment. Reading a local JSON file is not an Azure call, so the engine stays
// zero-Azure — it never touches the MCP; it only prepares the env the LLM worker's
// MCP + az-attach.sh will use.
type DeliveryConfig struct {
	Org     string `json:"org"`
	Project string `json:"project"`
	Repo    string `json:"repo"`
	PAT     struct {
		Ref string `json:"ref"`
	} `json:"pat"`
}

// DeliveryConfigPath is <root>/.delivery/config.json (the file /delivery-init writes).
func DeliveryConfigPath(root string) string {
	return filepath.Join(root, ".delivery", "config.json")
}

// LoadDeliveryConfig reads .delivery/config.json. A MISSING file returns (nil, nil):
// the caller degrades to the pre-#8 inheritance behavior, so a plan-only test harness
// with no config (e.g. the Layer-A work-dispatch blueprint) still runs. A present but
// malformed file is a real error, surfaced.
func LoadDeliveryConfig(root string) (*DeliveryConfig, error) {
	b, err := os.ReadFile(DeliveryConfigPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var c DeliveryConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", DeliveryConfigPath(root), err)
	}
	return &c, nil
}

// patRef is the env-var name the raw PAT is read from — config.pat.ref, defaulting to
// AZURE_DEVOPS_PAT (matching /delivery-init + azure-adapter §1).
func (c *DeliveryConfig) patRef() string {
	if c != nil && c.PAT.Ref != "" {
		return c.PAT.Ref
	}
	return "AZURE_DEVOPS_PAT"
}

// mcpConfigJSON is the .mcp.json binding the `azureDevOps` server to the official
// @azure-devops/mcp launcher scoped to THIS project's org (D3). Generated from
// config.json's org so a spawned worker ALWAYS gets the project's target org
// explicitly via --mcp-config — never the operator's global (e.g. employer)
// azureDevOps MCP by ambient inheritance (the safety gate). The PAT is NOT written
// here; it rides the worker env (the MCP server, a child of `claude`, inherits it),
// so no secret ever lands on disk.
func mcpConfigJSON(org string) ([]byte, error) {
	type server struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	doc := map[string]any{
		"mcpServers": map[string]any{
			"azureDevOps": server{
				Command: "npx",
				Args:    []string{"-y", "@azure-devops/mcp", org, "--authentication", "pat"},
			},
		},
	}
	return json.MarshalIndent(doc, "", "  ")
}

// deliveryWorkerEnv builds a worker's extra env (D3 / the F8-Go side): the org +
// project the az-attach.sh REST carve-out requires (the zero-Azure engine supplies
// them from config, not from Azure), plus the two PAT forms the two consumers need —
//
//   - AZURE_DEVOPS_PAT     = the RAW PAT, for az-attach.sh (F8: never the base64 form,
//     which curl -u would double-encode to a 401).
//   - PERSONAL_ACCESS_TOKEN = base64(":"+rawPAT), the Basic credential the
//     @azure-devops/mcp server decodes (it takes everything after the first ':').
//
// The raw PAT is read from the ENGINE's own env under config.pat.ref; if it is unset,
// no PAT vars are added and the worker blocks honestly at its first Azure call — never
// a silent green. (The exact PAT format the official server expects is what the real
// Azure #17 Layer-B run validates; this wiring follows the observed launcher contract.)
func deliveryWorkerEnv(cfg *DeliveryConfig) []string {
	if cfg == nil {
		return nil
	}
	env := []string{
		"AZURE_DEVOPS_ORG=" + cfg.Org,
		"AZURE_DEVOPS_PROJECT=" + cfg.Project,
	}
	if raw := os.Getenv(cfg.patRef()); raw != "" {
		env = append(env,
			"AZURE_DEVOPS_PAT="+raw,
			"PERSONAL_ACCESS_TOKEN="+base64.StdEncoding.EncodeToString([]byte(":"+raw)),
		)
	}
	return env
}

// writeMCPConfig writes the generated .mcp.json to a transient path under
// <root>/.delivery/mcp/ — a sibling of runstate.json / worktrees/, deliberately NOT
// inside the worker's worktree, so the worker can never commit it — and returns the
// path for the worker's --mcp-config.
func writeMCPConfig(root, org string, id int) (string, error) {
	body, err := mcpConfigJSON(org)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, ".delivery", "mcp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d.json", id))
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
