# Claude Code Plugin

oastools provides a [Claude Code plugin](https://docs.anthropic.com/en/docs/claude-code/plugins) that gives Claude direct access to all OpenAPI tooling through the built-in [MCP server](mcp-server.md). Once installed, Claude can validate, fix, convert, diff, walk, and generate code from your OpenAPI specs without any manual setup.

## Prerequisites

The plugin runs `oastools mcp` under the hood, so the **oastools binary must be installed** and available on your `PATH` before the plugin will work.

Install via any of these methods:

**Homebrew (macOS/Linux):**

```bash
brew tap erraggy/oastools && brew install oastools
```

**Go install:**

```bash
# Requires Go 1.24+
go install github.com/erraggy/oastools/cmd/oastools@latest
```

**Binary download:**

Download a pre-built binary from the [Releases page](https://github.com/erraggy/oastools/releases/latest) and place it on your `PATH`.

Verify the binary is available:

```bash
oastools --version
```

---

## Installation

### Option 1: Plugin Marketplace (recommended)

This is the easiest method. Run these commands inside Claude Code (the `/plugin` commands are Claude Code slash commands, not shell commands):

```
/plugin marketplace add erraggy/oastools
/plugin install oastools
```

The first command registers the oastools marketplace. The second installs the plugin. You'll be prompted to choose an installation scope:

| Scope | Effect |
|-------|--------|
| **User** | Available in all your projects |
| **Project** | Shared with collaborators (saved to `.claude/settings.json`) |
| **Local** | Only for you, only in this repository |

### Option 2: Manual MCP Configuration

If you prefer not to use the plugin system, you can configure the MCP server directly. Add this to your project's `.mcp.json` (or create the file if it doesn't exist):

```json
{
  "mcpServers": {
    "oastools": {
      "type": "stdio",
      "command": "oastools",
      "args": ["mcp"]
    }
  }
}
```

!!! note
    The manual approach gives you the 15 MCP tools but **not** the guided skills or CLAUDE.md instructions that come with the plugin. For the full experience, use Option 1.

---

## Verifying the Installation

After installing, restart Claude Code (or start a new session) and check that the tools are available:

```
/mcp
```

You should see `oastools` listed as an MCP server with 15 tools. You can also ask Claude directly:

> "What oastools MCP tools do you have access to?"

---

## What the Plugin Provides

### 15 MCP Tools

The plugin configures the oastools MCP server, which exposes 15 tools that Claude can call directly:

**Core (9):** `validate`, `parse`, `fix`, `convert`, `diff`, `join`, `overlay_apply`, `overlay_validate`, `generate`

**Walk (6):** `walk_operations`, `walk_schemas`, `walk_parameters`, `walk_responses`, `walk_security`, `walk_paths`

See the [MCP Server reference](mcp-server.md) for detailed documentation of each tool.

### 5 Guided Skills

The plugin includes skills that teach Claude best-practice workflows for common tasks:

| Skill | What it does |
|-------|-------------|
| **validate-spec** | Validate a spec, explain errors in plain language, suggest or apply fixes |
| **fix-spec** | Preview fixes with dry run, apply with confirmation, re-validate |
| **explore-api** | Parse for an overview, walk endpoints and schemas, drill into specifics |
| **diff-specs** | Compare two versions, categorize by severity, suggest migration steps |
| **generate-code** | Parse the spec, generate Go code, review output, suggest integration |

### CLAUDE.md Instructions

The plugin includes a `CLAUDE.md` that teaches Claude:

- The input model (file vs URL vs inline content)
- When to prefer `file` over `content`
- Common workflows (validate-then-fix, explore-then-modify)
- Best practices (filter walk results, use dry-run for fix, check breaking changes)

---

## Usage Examples

Once the plugin is installed, you can ask Claude to work with your OpenAPI specs naturally:

**Validate a spec:**

> "Validate my openapi.yaml and explain any errors"

Claude will call the `validate` tool, interpret the results, and explain each error with suggested fixes.

**Explore an API:**

> "What endpoints does this API have? Show me the user-related ones."

Claude will `parse` for an overview, then `walk_operations` with a tag or path filter.

**Compare API versions:**

> "Compare api-v1.yaml and api-v2.yaml — are there any breaking changes?"

Claude will call `diff` with both specs and categorize changes by severity.

**Fix common issues:**

> "Fix the duplicate operationIds in my spec"

Claude will preview with `dry_run`, show you the planned fixes, then apply them.

**Generate a Go client:**

> "Generate a Go HTTP client from openapi.yaml into ./client"

Claude will call `generate` with `client: true` and report the generated files.

---

## Updating the Plugin

If you installed via the marketplace, Claude Code will check for updates automatically. You can also update manually:

```
/plugin update oastools
```

---

## Uninstalling

```
/plugin uninstall oastools
```

---

## Plugin Structure

```
plugin/
├── .claude-plugin/
│   └── plugin.json        # Plugin manifest (name, version, author)
├── .mcp.json              # MCP server configuration
├── CLAUDE.md              # Instructions for Claude (tools, best practices)
└── skills/
    ├── validate-spec.md   # Guided validation workflow
    ├── fix-spec.md        # Guided fix workflow
    ├── explore-api.md     # Guided exploration workflow
    ├── diff-specs.md      # Guided diff workflow
    └── generate-code.md   # Guided code generation workflow
```

---

## Troubleshooting

**"oastools: command not found"**

The MCP server requires the `oastools` binary on your PATH. See [Prerequisites](#prerequisites) above for installation options. After installing, you may need to restart your terminal or Claude Code session.

**Tools aren't appearing in Claude Code**

1. Verify the plugin is installed: `/plugin list`
2. Check that `oastools mcp` runs without errors: `echo '{}' | oastools mcp`
3. Restart Claude Code to pick up configuration changes

**"spec input is required" errors**

Every tool requires a spec input. Make sure you're providing exactly one of `file`, `url`, or `content` inside the `spec` object:

```json
{"spec": {"file": "openapi.yaml"}}
```
