package dispatch

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, root, body string) {
	t.Helper()
	dir := filepath.Join(root, ".delivery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDeliveryConfig(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `{"org":"mkurak","project":"ATL-TEST","repo":"ATL-TEST","pat":{"ref":"AZURE_DEVOPS_PAT"}}`)
		cfg, err := LoadDeliveryConfig(root)
		if err != nil {
			t.Fatal(err)
		}
		if cfg == nil || cfg.Org != "mkurak" || cfg.Project != "ATL-TEST" || cfg.patRef() != "AZURE_DEVOPS_PAT" {
			t.Fatalf("got %+v", cfg)
		}
	})

	t.Run("missing → nil,nil (degrade to inheritance)", func(t *testing.T) {
		cfg, err := LoadDeliveryConfig(t.TempDir())
		if err != nil || cfg != nil {
			t.Fatalf("want (nil,nil) for a missing config, got (%v,%v)", cfg, err)
		}
	})

	t.Run("malformed → error", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `{not json`)
		if _, err := LoadDeliveryConfig(root); err == nil {
			t.Fatal("want an error for a malformed config")
		}
	})

	t.Run("patRef defaults when absent", func(t *testing.T) {
		var c *DeliveryConfig
		if c.patRef() != "AZURE_DEVOPS_PAT" {
			t.Errorf("nil cfg patRef = %q, want the default", c.patRef())
		}
		c2 := &DeliveryConfig{}
		if c2.patRef() != "AZURE_DEVOPS_PAT" {
			t.Errorf("empty patRef = %q, want the default", c2.patRef())
		}
	})
}

func TestMCPConfigJSON(t *testing.T) {
	b, err := mcpConfigJSON("mkurak")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, b)
	}
	ado, ok := doc.MCPServers["azureDevOps"]
	if !ok {
		t.Fatalf("no azureDevOps server in %s", b)
	}
	if ado.Command != "npx" {
		t.Errorf("command = %q, want npx", ado.Command)
	}
	joined := strings.Join(ado.Args, " ")
	if !strings.Contains(joined, "@azure-devops/mcp") || !strings.Contains(joined, "mkurak") || !strings.Contains(joined, "--authentication pat") {
		t.Errorf("args = %v, want the official launcher scoped to the org", ado.Args)
	}
	// The safety invariant: no secret is ever written to the MCP config file.
	if strings.Contains(strings.ToLower(string(b)), "pat=") || strings.Contains(string(b), "PERSONAL_ACCESS_TOKEN") {
		t.Errorf("mcp config must not carry a secret:\n%s", b)
	}
}

func TestDeliveryWorkerEnv(t *testing.T) {
	cfg := &DeliveryConfig{Org: "mkurak", Project: "ATL-TEST"}
	cfg.PAT.Ref = "AZURE_DEVOPS_PAT"

	t.Run("with a raw PAT in the env", func(t *testing.T) {
		t.Setenv("AZURE_DEVOPS_PAT", "rawtoken123")
		env := deliveryWorkerEnv(cfg)
		m := envMap(env)
		if m["AZURE_DEVOPS_ORG"] != "mkurak" || m["AZURE_DEVOPS_PROJECT"] != "ATL-TEST" {
			t.Errorf("org/project not set: %v", m)
		}
		// az-attach.sh needs the RAW pat (F8).
		if m["AZURE_DEVOPS_PAT"] != "rawtoken123" {
			t.Errorf("AZURE_DEVOPS_PAT = %q, want the raw pat", m["AZURE_DEVOPS_PAT"])
		}
		// the MCP server decodes PERSONAL_ACCESS_TOKEN as base64("user:PAT").
		dec, err := base64.StdEncoding.DecodeString(m["PERSONAL_ACCESS_TOKEN"])
		if err != nil {
			t.Fatalf("PERSONAL_ACCESS_TOKEN not base64: %v", err)
		}
		if string(dec) != ":rawtoken123" {
			t.Errorf("decoded PERSONAL_ACCESS_TOKEN = %q, want \":rawtoken123\"", dec)
		}
	})

	t.Run("no PAT in env → no PAT vars (worker blocks honestly)", func(t *testing.T) {
		t.Setenv("AZURE_DEVOPS_PAT", "")
		m := envMap(deliveryWorkerEnv(cfg))
		if _, ok := m["AZURE_DEVOPS_PAT"]; ok {
			t.Error("AZURE_DEVOPS_PAT should be absent when the engine env has no pat")
		}
		if _, ok := m["PERSONAL_ACCESS_TOKEN"]; ok {
			t.Error("PERSONAL_ACCESS_TOKEN should be absent when the engine env has no pat")
		}
		// org/project are still supplied.
		if m["AZURE_DEVOPS_ORG"] != "mkurak" {
			t.Error("org should be set even without a pat")
		}
	})

	t.Run("nil cfg → nil env", func(t *testing.T) {
		if deliveryWorkerEnv(nil) != nil {
			t.Error("nil cfg should yield nil env")
		}
	})

	t.Run("github backend → GH_TOKEN, no Azure vars", func(t *testing.T) {
		gh := &DeliveryConfig{Backend: "github", Org: "ignored"}
		t.Setenv("GH_TOKEN", "ghtok999")
		m := envMap(deliveryWorkerEnv(gh))
		if m["GH_TOKEN"] != "ghtok999" {
			t.Errorf("GH_TOKEN = %q, want the token", m["GH_TOKEN"])
		}
		for _, k := range []string{"AZURE_DEVOPS_ORG", "AZURE_DEVOPS_PAT", "PERSONAL_ACCESS_TOKEN"} {
			if _, ok := m[k]; ok {
				t.Errorf("github worker env must not carry %s", k)
			}
		}
	})

	t.Run("github backend, no GH_TOKEN → empty (worker blocks honestly)", func(t *testing.T) {
		gh := &DeliveryConfig{Backend: "github"}
		t.Setenv("GH_TOKEN", "")
		if env := deliveryWorkerEnv(gh); len(env) != 0 {
			t.Errorf("no GH_TOKEN should yield no env vars, got %v", env)
		}
	})

	t.Run("github credential.ref names a custom source env var → injected as GH_TOKEN", func(t *testing.T) {
		var gh DeliveryConfig
		if err := json.Unmarshal([]byte(`{"backend":"github","credential":{"ref":"MY_GH_TOKEN"}}`), &gh); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GH_TOKEN", "")            // the default source is empty ...
		t.Setenv("MY_GH_TOKEN", "customtok") // ... but the configured source carries the value
		m := envMap(deliveryWorkerEnv(&gh))
		if m["GH_TOKEN"] != "customtok" {
			t.Errorf("GH_TOKEN = %q, want the value read from credential.ref (MY_GH_TOKEN) — the field is no longer inert", m["GH_TOKEN"])
		}
	})

	t.Run("githubTokenRef defaults to GH_TOKEN", func(t *testing.T) {
		if (*DeliveryConfig)(nil).githubTokenRef() != "GH_TOKEN" {
			t.Error("nil cfg githubTokenRef should default to GH_TOKEN")
		}
		if (&DeliveryConfig{}).githubTokenRef() != "GH_TOKEN" {
			t.Error("empty cfg githubTokenRef should default to GH_TOKEN")
		}
	})
}

func TestActiveBackend(t *testing.T) {
	// nil + empty → the "azure" default (backward-compat for pre-field configs).
	if got := (*DeliveryConfig)(nil).activeBackend(); got != "azure" {
		t.Errorf("nil config activeBackend = %q, want azure", got)
	}
	if got := (&DeliveryConfig{}).activeBackend(); got != "azure" {
		t.Errorf("empty config activeBackend = %q, want azure", got)
	}
	// an explicit backend is honored (parsed from JSON).
	var c DeliveryConfig
	if err := json.Unmarshal([]byte(`{"org":"o","backend":"github"}`), &c); err != nil {
		t.Fatal(err)
	}
	if got := c.activeBackend(); got != "github" {
		t.Errorf("explicit activeBackend = %q, want github", got)
	}
}

func TestWriteMCPConfig(t *testing.T) {
	root := t.TempDir()
	path, err := writeMCPConfig(root, "mkurak", 42)
	if err != nil {
		t.Fatal(err)
	}
	// It lands under .delivery/mcp/, a sibling of worktrees/ — NOT inside a worktree,
	// so a worker can never commit it.
	want := filepath.Join(root, ".delivery", "mcp", "42.json")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if strings.Contains(path, filepath.Join(".delivery", "worktrees")) {
		t.Errorf("mcp config must not live inside a worktree: %q", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "@azure-devops/mcp") || !strings.Contains(string(b), "mkurak") {
		t.Errorf("written config missing the launcher/org:\n%s", b)
	}
}

func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			m[kv[:i]] = kv[i+1:]
		}
	}
	return m
}

func TestDevBranchDefaultAndOverride(t *testing.T) {
	// nil config → the "dev" default.
	if got := (*DeliveryConfig)(nil).DevBranch(); got != "dev" {
		t.Errorf("nil config DevBranch = %q, want dev", got)
	}
	// config with no branchPair → still "dev".
	if got := (&DeliveryConfig{Org: "o"}).DevBranch(); got != "dev" {
		t.Errorf("empty branchPair DevBranch = %q, want dev", got)
	}
	// an overridden dev branch is honored (parsed from JSON).
	var c DeliveryConfig
	if err := json.Unmarshal([]byte(`{"org":"o","branchPair":{"dev":"main","release":"prod"}}`), &c); err != nil {
		t.Fatal(err)
	}
	if got := c.DevBranch(); got != "main" {
		t.Errorf("overridden DevBranch = %q, want main", got)
	}
}
