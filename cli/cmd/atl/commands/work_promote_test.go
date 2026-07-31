package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	promoteSHA    = "738cdb2a1f4e6b90c3d5827a4e1b06f9d3a7c215"
	promoteNewSHA = "1abfcf1e02d47b6a95c8130fe27b4a6d8091c3ee"
)

// fakeGH answers `gh` from canned stdout keyed by subcommand ("pr list", "pr
// view", "pr merge") and records every argv — the whole command runs with no
// network and no PR is ever really merged.
type fakeGH struct {
	calls [][]string
	out   map[string]string
}

func (f *fakeGH) runner(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	key := strings.Join(args[:min(2, len(args))], " ")
	return []byte(f.out[key]), nil
}

func (f *fakeGH) merged() bool {
	for _, c := range f.calls {
		if len(c) >= 3 && c[1] == "pr" && c[2] == "merge" {
			return true
		}
	}
	return false
}

func approvalComment(sha string) string {
	return `{"author":{"login":"po"},"createdAt":"2026-07-31T10:00:00Z",` +
		`"body":"**[Promotion Approval]**\n\n## Approved Commit\n` + sha + `\n"}`
}

// runPromote chdirs into a project with the given .delivery/config.json, drives
// `atl work promote` through fake, and returns its stdout plus the RunE error
// (non-nil = the non-zero exit).
func runPromote(t *testing.T, cfg string, fake *fakeGH, args ...string) (string, error) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
		t.Fatal(err)
	}
	if cfg != "" {
		if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)

	prev := workPromoteRunner
	t.Cleanup(func() {
		workPromoteRunner = prev
		workPromotePR, workPromoteJSON = 0, false
	})
	if fake != nil {
		workPromoteRunner = fake.runner
	}
	if err := workPromoteCmd.ParseFlags(args); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	workPromoteCmd.SetOut(&buf)
	err := workPromoteCmd.RunE(workPromoteCmd, nil)
	return buf.String(), err
}

const githubCfg = `{"owner":"acme","repo":"widgets","backend":"github",` +
	`"branchPair":{"dev":"dev","release":"release"}}`

// TestWorkPromoteMerges is the one path that promotes: an approval naming the
// PR's current head merges it, pinned to that commit, and exits 0.
func TestWorkPromoteMerges(t *testing.T) {
	f := &fakeGH{out: map[string]string{
		"pr list": `[{"number":42}]`,
		"pr view": `{"headRefOid":"` + promoteSHA + `","comments":[` + approvalComment(promoteSHA) + `]}`,
	}}
	out, err := runPromote(t, githubCfg, f)
	if err != nil {
		t.Fatalf("promote failed: %v\n%s", err, out)
	}
	if !f.merged() {
		t.Fatal("a verified approval did not merge")
	}
	want := "gh pr merge 42 --repo acme/widgets --merge --match-head-commit " + promoteSHA
	var got string
	for _, c := range f.calls {
		if len(c) >= 3 && c[2] == "merge" {
			got = strings.Join(c, " ")
		}
	}
	if got != want {
		t.Errorf("merge argv:\n got %q\nwant %q", got, want)
	}
	if !strings.Contains(out, "Promoted") {
		t.Errorf("stdout: %s", out)
	}
}

// TestWorkPromoteHolds is the regression this whole command exists for: the head
// moved past the approval, so the gate must hold, merge nothing, and exit
// non-zero — the turn-7/turn-8 inconsistency the skill prose could not hold.
func TestWorkPromoteHolds(t *testing.T) {
	f := &fakeGH{out: map[string]string{
		"pr list": `[{"number":42}]`,
		"pr view": `{"headRefOid":"` + promoteNewSHA + `","comments":[` + approvalComment(promoteSHA) + `]}`,
	}}
	out, err := runPromote(t, githubCfg, f)
	if err == nil {
		t.Fatal("a superseded approval exited 0 — a hold must be non-zero")
	}
	if f.merged() {
		t.Fatal("a superseded approval merged the PR")
	}
	if !strings.Contains(out, promoteSHA) || !strings.Contains(out, promoteNewSHA) {
		t.Errorf("the hold must name both commits:\n%s", out)
	}
}

// TestWorkPromoteJSON proves the machine-readable verdict the ceremony relays and
// the e2e asserts on.
func TestWorkPromoteJSON(t *testing.T) {
	f := &fakeGH{out: map[string]string{
		"pr list": `[{"number":42}]`,
		"pr view": `{"headRefOid":"` + promoteNewSHA + `","comments":[` + approvalComment(promoteSHA) + `]}`,
	}}
	out, err := runPromote(t, githubCfg, f, "--json")
	if err == nil {
		t.Fatal("hold must exit non-zero even with --json")
	}
	var res struct {
		Verdict  string `json:"verdict"`
		Reason   string `json:"reason"`
		Message  string `json:"message"`
		PR       int    `json:"pr"`
		Head     string `json:"head"`
		Approved string `json:"approved"`
		Records  []struct {
			Author    string `json:"author"`
			CreatedAt string `json:"createdAt"`
			Commit    string `json:"commit"`
		} `json:"records"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if res.Verdict != "hold" || res.Reason != "superseded" {
		t.Errorf("verdict=%q reason=%q", res.Verdict, res.Reason)
	}
	if res.PR != 42 || res.Head != promoteNewSHA || res.Approved != promoteSHA {
		t.Errorf("pr=%d head=%q approved=%q", res.PR, res.Head, res.Approved)
	}
	if len(res.Records) != 1 || res.Records[0].Author != "po" || res.Records[0].CreatedAt == "" {
		t.Errorf("records = %+v, want one record with author + timestamp", res.Records)
	}
	if res.Message == "" {
		t.Error("json must carry the PO-facing reason too")
	}
}

// TestWorkPromotePRFlag proves --pr overrides the branch-pair lookup.
func TestWorkPromotePRFlag(t *testing.T) {
	f := &fakeGH{out: map[string]string{
		"pr view": `{"headRefOid":"` + promoteSHA + `","comments":[` + approvalComment(promoteSHA) + `]}`,
	}}
	if _, err := runPromote(t, githubCfg, f, "--pr", "7"); err != nil {
		t.Fatalf("promote failed: %v", err)
	}
	for _, c := range f.calls {
		if len(c) >= 3 && c[2] == "list" {
			t.Error("--pr must skip the open-PR lookup")
		}
	}
	if len(f.calls) == 0 || !strings.Contains(strings.Join(f.calls[0], " "), "pr view 7") {
		t.Errorf("calls = %v, want the named PR read first", f.calls)
	}
}

// TestWorkPromoteAzureHolds proves the unbound backend holds instead of falling
// back to a conversational promotion — and makes no gh call at all.
func TestWorkPromoteAzureHolds(t *testing.T) {
	f := &fakeGH{}
	out, err := runPromote(t, `{"org":"acme","project":"widgets","backend":"azure"}`, f)
	if err == nil {
		t.Fatal("the azure backend must exit non-zero")
	}
	if len(f.calls) != 0 {
		t.Errorf("azure must not shell out: %v", f.calls)
	}
	if !strings.Contains(out, "cannot read the promotion PR's current head commit") {
		t.Errorf("stdout: %s", out)
	}
}

// TestWorkPromoteNoConfig — no .delivery/config.json means no coordinates to
// verify against; that is a setup error, surfaced, not a silent promotion.
func TestWorkPromoteNoConfig(t *testing.T) {
	if _, err := runPromote(t, "", nil); err == nil {
		t.Fatal("a project with no delivery config must fail")
	}
}

func TestWorkPromoteRegistered(t *testing.T) {
	work := findChild(rootCmd, "work")
	if work == nil {
		t.Fatal("`work` command not registered on root")
	}
	promote := findChild(work, "promote")
	if promote == nil {
		t.Fatal("`promote` subcommand not registered on `work`")
	}
	if promote.RunE == nil {
		t.Error("`work promote` should have a RunE")
	}
}
