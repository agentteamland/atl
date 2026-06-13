// Command atl is the AgentTeamLand platform CLI.
//
// v2 rebuild — this binary is the deterministic "plumbing" layer of the
// platform: install/update/promote/publish mechanics, the durable learning
// queue, scope management, and the self-healing doctor. LLM "intelligence"
// lives in skills, not here (see .atl/docs/atl-v2.md, the CLI/Skill boundary).
package main

import "github.com/agentteamland/atl/cli/cmd/atl/commands"

func main() {
	commands.Execute()
}
