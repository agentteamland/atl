// Package skillcheck holds the deterministic content-quality checks for the
// platform's own skills, agents, and team manifests — the sibling of docscheck.
//
// docscheck validates the docs *site* against the code (docs-drift); skillcheck
// validates the *assets themselves*: does every skill/agent carry a valid
// frontmatter, does each team.json match what's on disk, does every agent-KB
// child declare its summary. Every check is LLM-free and zero-false-positive by
// construction. The judgment half — obedience, redundancy, principle-worthiness —
// lives in the /skill-stocktake skill (the CLI/Skill boundary), not here.
package skillcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Severity ranks a finding. Fail breaks the CI gate; Warn is surfaced only.
type Severity string

const (
	Fail Severity = "fail"
	Warn Severity = "warn"
)

// Finding is a single content-quality problem.
type Finding struct {
	Check    string // "frontmatter" | "manifest" | "children"
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
	f = append(f, Children(in.TeamsDir)...)
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
		md := filepath.Join(dir, name, "SKILL.md")
		f = append(f, requireFrontmatter(md, filepath.ToSlash(filepath.Join(rel, name, "SKILL.md")), "skill")...)
	}
	return f
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

// Children checks every agent-KB child (agents/<x>/children/*.md) declares a
// non-empty knowledge-base-summary frontmatter — the KB-rebuild contract. Fail.
func Children(teamsDir string) []Finding {
	var f []Finding
	for _, team := range teamNames(teamsDir) {
		agentsDir := filepath.Join(teamsDir, team, "agents")
		for _, agent := range subdirs(agentsDir) {
			childrenDir := filepath.Join(agentsDir, agent, "children")
			kids, err := os.ReadDir(childrenDir)
			if err != nil {
				continue
			}
			for _, k := range kids {
				if k.IsDir() || !strings.HasSuffix(k.Name(), ".md") {
					continue
				}
				rel := filepath.ToSlash(filepath.Join("teams", team, "agents", agent, "children", k.Name()))
				kv, ok := frontmatter(filepath.Join(childrenDir, k.Name()))
				if !ok || strings.TrimSpace(kv["knowledge-base-summary"]) == "" {
					f = append(f, Finding{"children", Fail, rel, "agent-KB child is missing its `knowledge-base-summary` frontmatter"})
				}
			}
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
