// Package skillcheck holds the deterministic content-quality checks for the
// platform's own skills, agents, and team manifests — the sibling of docscheck.
//
// docscheck validates the docs *site* against the code (docs-drift); skillcheck
// validates the *assets themselves*: does every skill/agent carry a valid
// frontmatter, does each team.json match what's on disk, does every declared
// capture channel name assets the team actually ships, does every agent-KB
// child declare its summary, and does any skill's executable shell body carry
// one of two known shell-fragile constructs. Every check is LLM-free. The four
// structural checks are zero-false-positive by construction — the finding is a
// fact about the file; the shell check is a deliberately narrow pattern match
// over named constructs, so see ShellBodies for what it does and does NOT cover.
// The judgment half — obedience, redundancy, principle-worthiness —
// lives in the /skill-stocktake skill (the CLI/Skill boundary), not here.
package skillcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agentteamland/atl/cli/internal/manifest"
)

// Severity ranks a finding. Fail breaks the CI gate; Warn is surfaced only.
type Severity string

const (
	Fail Severity = "fail"
	Warn Severity = "warn"
)

// Finding is a single content-quality problem.
type Finding struct {
	Check    string // "frontmatter" | "manifest" | "channel" | "children" | "shell"
	Severity Severity
	Path     string // asset path relative to the repo root
	Detail   string
}

// Input bundles the roots the checks walk.
type Input struct {
	CoreDir  string // <repo>/core   ("" to skip core skills)
	TeamsDir string // <repo>/teams  ("" to skip teams)
}

// RunAll runs every deterministic check.
func RunAll(in Input) []Finding {
	var f []Finding
	f = append(f, Frontmatter(in.CoreDir, in.TeamsDir)...)
	f = append(f, TeamManifest(in.TeamsDir)...)
	f = append(f, Channels(in.TeamsDir)...)
	f = append(f, Children(in.TeamsDir)...)
	f = append(f, ShellBodies(in.CoreDir, in.TeamsDir)...)
	return f
}

// Frontmatter checks that every skill's SKILL.md and every agent's agent.md
// carries a name + description frontmatter. Fail-level.
func Frontmatter(coreDir, teamsDir string) []Finding {
	var f []Finding
	if coreDir != "" {
		f = append(f, checkSkillDir(filepath.Join(coreDir, "skills"), "core/skills")...)
	}
	for _, team := range teamNames(teamsDir) {
		base := filepath.Join(teamsDir, team)
		f = append(f, checkSkillDir(filepath.Join(base, "skills"), "teams/"+team+"/skills")...)
		f = append(f, checkAgentDir(filepath.Join(base, "agents"), "teams/"+team+"/agents")...)
	}
	return f
}

func checkSkillDir(dir, rel string) []Finding {
	var f []Finding
	for _, name := range subdirs(dir) {
		md, mdRel := skillFile(filepath.Join(dir, name), filepath.Join(rel, name))
		f = append(f, requireFrontmatter(md, mdRel, "skill")...)
	}
	return f
}

// skillFile resolves a skill's markdown file. The correct, discoverable name is
// SKILL.md (uppercase) — a lowercase skill.md is NOT found as a /slash command by
// Claude Code on case-sensitive Linux. This still tolerates the lowercase form so
// the check itself doesn't crash on a mis-named skill, but authors must ship
// SKILL.md (the docs teach that). Resolution is case-sensitive (correct on Linux
// CI, not just case-insensitive macOS); falls back to SKILL.md for the "missing"
// error path.
func skillFile(dir, rel string) (path, relPath string) {
	names, _ := os.ReadDir(dir)
	for _, e := range names {
		if n := e.Name(); n == "SKILL.md" || n == "skill.md" {
			return filepath.Join(dir, n), filepath.ToSlash(filepath.Join(rel, n))
		}
	}
	return filepath.Join(dir, "SKILL.md"), filepath.ToSlash(filepath.Join(rel, "SKILL.md"))
}

func checkAgentDir(dir, rel string) []Finding {
	var f []Finding
	for _, name := range subdirs(dir) {
		md := filepath.Join(dir, name, "agent.md")
		f = append(f, requireFrontmatter(md, filepath.ToSlash(filepath.Join(rel, name, "agent.md")), "agent")...)
	}
	return f
}

func requireFrontmatter(path, relPath, kind string) []Finding {
	kv, ok := frontmatter(path)
	if !ok {
		if _, err := os.Stat(path); err != nil {
			return []Finding{{"frontmatter", Fail, relPath, kind + " directory has no " + filepath.Base(path)}}
		}
		return []Finding{{"frontmatter", Fail, relPath, kind + " is missing its `---` frontmatter block"}}
	}
	var f []Finding
	if strings.TrimSpace(kv["name"]) == "" {
		f = append(f, Finding{"frontmatter", Fail, relPath, kind + " frontmatter is missing `name`"})
	}
	if strings.TrimSpace(kv["description"]) == "" {
		f = append(f, Finding{"frontmatter", Fail, relPath, kind + " frontmatter is missing `description`"})
	}
	return f
}

// TeamManifest checks that each team.json's agents[]/skills[] names match the
// directories on disk, both directions. Fail-level.
func TeamManifest(teamsDir string) []Finding {
	var f []Finding
	for _, team := range teamNames(teamsDir) {
		base := filepath.Join(teamsDir, team)
		b, err := os.ReadFile(filepath.Join(base, "team.json"))
		if err != nil {
			f = append(f, Finding{"manifest", Fail, "teams/" + team + "/team.json", "team has no team.json"})
			continue
		}
		var meta struct {
			Agents []struct {
				Name string `json:"name"`
			} `json:"agents"`
			Skills []struct {
				Name string `json:"name"`
			} `json:"skills"`
		}
		if err := json.Unmarshal(b, &meta); err != nil {
			f = append(f, Finding{"manifest", Fail, "teams/" + team + "/team.json", "team.json is not valid JSON: " + err.Error()})
			continue
		}
		declaredAgents := make([]string, len(meta.Agents))
		for i, a := range meta.Agents {
			declaredAgents[i] = a.Name
		}
		declaredSkills := make([]string, len(meta.Skills))
		for i, s := range meta.Skills {
			declaredSkills[i] = s.Name
		}
		f = append(f, matchNames(base, team, "agents", declaredAgents)...)
		f = append(f, matchNames(base, team, "skills", declaredSkills)...)
	}
	return f
}

// matchNames compares team.json-declared names against the on-disk dirs of a kind
// (agents|skills), both directions.
func matchNames(base, team, kind string, declared []string) []Finding {
	var f []Finding
	rel := "teams/" + team + "/" + kind
	dcl := map[string]bool{}
	for _, n := range declared {
		dcl[n] = true
	}
	disk := map[string]bool{}
	for _, n := range subdirs(filepath.Join(base, kind)) {
		disk[n] = true
	}
	for n := range dcl {
		if !disk[n] {
			f = append(f, Finding{"manifest", Fail, rel, "team.json lists " + kind + " `" + n + "` but there is no " + kind + "/" + n + " dir"})
		}
	}
	for n := range disk {
		if !dcl[n] {
			f = append(f, Finding{"manifest", Fail, rel + "/" + n, kind + "/" + n + " exists on disk but is not declared in team.json"})
		}
	}
	return f
}

// Channels checks every capture channel a team declares under
// `capabilities.<name>.channel` — that the declaration carries every field a
// signal sentence needs, and that its `rule` and `drain` name assets the team
// actually ships. Fail-level.
//
// This is the deterministic gate for the one failure the runtime cannot catch.
// The platform assembles a channel's signals purely from the four declared
// words and stops there, so a declaration naming a rule the team does not ship
// is accepted, the channel goes active, and its markers are captured into the
// queue — while nothing ever tells an agent to drain them. They accumulate
// forever, and the only symptom is a backlog with no explanation. The install-
// time doctor check (`channelsCheck`) sees the same declaration but has no way
// to resolve it: it inspects an installed manifest, where the team's source
// tree is long gone. Here, in the monorepo, the assets are right there — so a
// first-party team's broken declaration fails CI instead of shipping.
//
// The reach is exactly first-party teams under teams/. A third-party team's
// declaration is caught by nothing until it is installed, where the doctor's
// runtime warning is the backstop for the field-presence half.
func Channels(teamsDir string) []Finding {
	var f []Finding
	for _, team := range teamNames(teamsDir) {
		base := filepath.Join(teamsDir, team)
		rel := "teams/" + team + "/team.json"
		b, err := os.ReadFile(filepath.Join(base, "team.json"))
		if err != nil {
			continue // TeamManifest already reports a missing/unparseable team.json
		}
		var meta struct {
			Capabilities map[string]json.RawMessage `json:"capabilities"`
		}
		if err := json.Unmarshal(b, &meta); err != nil {
			continue
		}
		for _, capName := range sortedCapabilities(meta.Capabilities) {
			var wrapper struct {
				Channel *manifest.Channel `json:"channel"`
			}
			// A capability whose value carries no `channel` — a bare string, an
			// object without the field — is not a declaration; skip it, the same
			// tolerance teampkg.DeclaredChannels applies.
			if err := json.Unmarshal(meta.Capabilities[capName], &wrapper); err != nil || wrapper.Channel == nil {
				continue
			}
			f = append(f, checkChannel(base, team, rel, capName, *wrapper.Channel)...)
		}
	}
	return f
}

// checkChannel validates one declaration: every field present, and `rule` +
// `drain` resolving to assets on disk.
func checkChannel(base, team, rel, capName string, ch manifest.Channel) []Finding {
	where := "capabilities." + capName + ".channel"

	// Field presence first — the same predicate the read path and the doctor
	// check use, so the three can never disagree about what "valid" means. A
	// declaration missing `name` is the sharpest case: the platform drops it
	// entirely, so it looks declared in team.json and is invisible everywhere else.
	if missing := ch.MissingFields(); len(missing) > 0 {
		return []Finding{{"channel", Fail, rel,
			where + " is missing `" + strings.Join(missing, "`, `") + "` — the platform refuses a declaration it cannot word a signal from"}}
	}

	var f []Finding
	if _, err := os.Stat(filepath.Join(base, "rules", ch.Rule+".md")); err != nil {
		f = append(f, Finding{"channel", Fail, rel,
			where + " names rule `" + ch.Rule + "` but there is no rules/" + ch.Rule + ".md — the signal would point at a rule the team never ships, so nothing would act on it"})
	}
	// The declared drain is a slash-command as the agent types it; the asset on
	// disk is the skill dir it resolves to. skillFile carries the case-sensitivity
	// rule (SKILL.md vs skill.md) so this does not re-derive it.
	drain := strings.TrimPrefix(ch.Drain, "/")
	skillPath, _ := skillFile(filepath.Join(base, "skills", drain), "")
	if _, err := os.Stat(skillPath); err != nil {
		f = append(f, Finding{"channel", Fail, rel,
			where + " names drain `" + ch.Drain + "` but there is no skills/" + drain + "/SKILL.md — the signal would tell an agent to spawn a skill the team never ships"})
	}
	return f
}

// sortedCapabilities orders capability names so findings are deterministic
// (map iteration is not).
func sortedCapabilities(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Children checks every agent-KB child (agents/<x>/children/*.md) declares a
// non-empty knowledge-base-summary frontmatter — the KB-rebuild contract. Fail.
func Children(teamsDir string) []Finding {
	var f []Finding
	for _, team := range teamNames(teamsDir) {
		agentsDir := filepath.Join(teamsDir, team, "agents")
		for _, agent := range subdirs(agentsDir) {
			rel := filepath.Join("teams", team, "agents", agent, "children")
			f = append(f, checkChildren(filepath.Join(agentsDir, agent, "children"), rel, Fail)...)
		}
	}
	return f
}

// InstalledChildren applies the same knowledge-base-summary contract to an
// INSTALLED layer — a .claude dir, ~/.claude or <project>/.claude.
//
// Children above walks the monorepo's authored copies, which arrive in a PR and
// are gated by CI. /drain writes only here, into
// <scope>/.claude/agents/<agent>/children/, so until this walk existed not one
// file the learning loop created was validated by anything.
//
// Warn-level, and deliberately NOT part of RunAll: `atl skills check` is the CI
// gate, and CI has no installed layer — gating on it would gate on something the
// runner cannot see. The one caller is the session-start signal.
//
// The honest limit: this REPORTS a child written without its summary; it cannot
// make /drain write one. The contract still lives in the skill's prose — this
// only stops a breach from being silent.
func InstalledChildren(claudeDir string) []Finding {
	if claudeDir == "" {
		return nil
	}
	var f []Finding
	agentsDir := filepath.Join(claudeDir, "agents")
	for _, agent := range subdirs(agentsDir) {
		// An ATL agent knowledge base is agent.md + children/. Requiring the
		// agent.md keeps a directory that merely happens to be named children/
		// from being read as one.
		if _, err := os.Stat(filepath.Join(agentsDir, agent, "agent.md")); err != nil {
			continue
		}
		rel := filepath.Join("agents", agent, "children")
		f = append(f, checkChildren(filepath.Join(agentsDir, agent, "children"), rel, Warn)...)
	}
	return f
}

// checkChildren validates one agents/<x>/children dir. rel is the reported path
// prefix; sev differs by surface — a shipped child fails the gate, an installed
// one only warns.
func checkChildren(childrenDir, rel string, sev Severity) []Finding {
	kids, err := os.ReadDir(childrenDir)
	if err != nil {
		return nil
	}
	var f []Finding
	for _, k := range kids {
		if k.IsDir() || !strings.HasSuffix(k.Name(), ".md") {
			continue
		}
		kv, ok := frontmatter(filepath.Join(childrenDir, k.Name()))
		if !ok || strings.TrimSpace(kv["knowledge-base-summary"]) == "" {
			path := filepath.ToSlash(filepath.Join(rel, k.Name()))
			f = append(f, Finding{"children", sev, path, "agent-KB child is missing its `knowledge-base-summary` frontmatter"})
		}
	}
	return f
}

// --- helpers ---

// teamNames returns the immediate subdirectory names under teamsDir (each a team).
func teamNames(teamsDir string) []string {
	if teamsDir == "" {
		return nil
	}
	return subdirs(teamsDir)
}

// subdirs returns the immediate subdirectory names of dir (sorted by ReadDir),
// or nil if dir is absent.
func subdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// frontmatter returns the key/value pairs in a leading `---` … `---` block.
// Values have surrounding quotes stripped. No leading block → (nil, false).
func frontmatter(path string) (map[string]string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	s := string(b)
	if !strings.HasPrefix(s, "---") {
		return nil, false
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, false
	}
	kv := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.Trim(strings.TrimSpace(line[i+1:]), `"'`)
		kv[key] = val
	}
	return kv, true
}
