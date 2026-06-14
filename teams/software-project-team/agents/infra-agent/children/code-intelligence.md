---
knowledge-base-summary: "Knowledge graph of entire codebase via MCP. 66 languages, zero dependencies, sub-ms queries. Solves context blindness in large projects — agent knows who calls whom, who implements what. Always install on projects with 50+ files. Auto-indexed, `.mcp.json` configured."
---
# Code Intelligence: codebase-memory-mcp Integration

## Purpose

Large projects (100k+ lines) exceed AI context windows. Without structural understanding, the AI guesses at dependencies and relationships. `codebase-memory-mcp` solves this by creating a persistent knowledge graph of the entire codebase — function definitions, call chains, class hierarchies, cross-file dependencies — queryable in milliseconds via MCP.

## What It Provides

- **Knowledge graph** — not just text search, but structural relationships (who calls whom, who implements what)
- **66 language support** — via tree-sitter AST parsing
- **Zero dependencies** — single static binary, no runtime requirements
- **Sub-millisecond queries** — SQLite-backed, instant answers
- **99%+ token reduction** — agent gets precise context instead of scanning entire files
- **MCP native** — 14 tools directly available to Claude Code

## Installation

### Automatic (via npm)

```bash
npx @anthropic-ai/codebase-memory install
```

### Manual (download binary)

```bash
# Download from GitHub releases
# https://github.com/DeusData/codebase-memory-mcp
curl -L -o codebase-memory https://github.com/DeusData/codebase-memory-mcp/releases/latest/download/codebase-memory-$(uname -s)-$(uname -m)
chmod +x codebase-memory
mv codebase-memory /usr/local/bin/
```

## Project Configuration

Add to project's `.mcp.json` at the repository root:

```json
{
  "mcpServers": {
    "codebase-memory": {
      "command": "codebase-memory",
      "args": ["--project-root", "."]
    }
  }
}
```

## Initial Indexing

First time or after major changes:

```bash
codebase-memory index --path .
```

For auto-reindexing on file changes (development):

```bash
codebase-memory index --path . --watch
```

## Available MCP Tools

| Tool | Purpose |
|------|---------|
| `search_code` | Find code by semantic meaning |
| `read_graph` | Query structural relationships (callers, callees, implementors) |
| `get_context` | Get rich context for a specific code location |
| + 11 more | Various code intelligence operations |

## When to Use

- **Always** on projects with 50+ files or 10k+ lines
- **Before** major refactoring (understand impact)
- **During** debugging (trace call chains)
- **For** onboarding new agents to existing projects

## Integration with /create-new-project

When `/create-new-project` scaffolds a new project, it should:
1. Add `.mcp.json` with codebase-memory configuration
2. Document the setup in CLAUDE.md
3. Note in README that code intelligence is available

## Docker Compose Integration

The binary runs on the host (not in Docker) since it needs access to source files. No Docker service needed.

## Important Rules

1. **Index after scaffold.** Run indexing after project is created so agents start with full codebase knowledge.
2. **Re-index after migrations.** Schema changes = new entity relationships. Re-index to keep graph current.
3. **Watch mode in development.** Auto-reindex catches changes as they happen.
4. **Not a replacement for reading code.** Knowledge graph helps find the right files — agent still reads and understands them.
