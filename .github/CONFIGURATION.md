# GitHub Configuration

This directory contains GitHub-specific configuration files for the oastools repository.

## Files

### `AGENTS.md` (Repository Root)

Quick reference guide for AI coding agents, following the [AGENTS.md open format](https://agents.md). This file provides:

- Concise project overview and quick commands
- Essential dev environment setup (golangci-lint v2)
- Testing requirements and patterns
- Common code style patterns and pitfalls
- Clear boundaries and acceptance criteria
- Cross-references to detailed instructions

This file is optimized for agent consumption with actionable, focused guidance. For comprehensive details, agents should consult `.github/copilot-instructions.md`.

### `copilot-instructions.md`

Comprehensive repository-wide instructions for GitHub Copilot coding agent. This file provides:

- Detailed project overview and architecture
- Development commands and workflows
- Extensive code style guidelines with real examples
- Testing and benchmark requirements
- OpenAPI Specification concepts and version-specific features
- Security considerations and common vulnerability patterns
- Acceptance criteria for task completion
- Boundaries and exclusions (protected files/directories)
- Public API structure and usage examples

These instructions help GitHub Copilot understand the project deeply and make informed decisions when generating or modifying code.

#### Best Practices Followed

Both instruction files follow [GitHub's best practices](https://docs.github.com/en/copilot/how-tos/configure-custom-instructions/add-repository-instructions) for Copilot custom instructions:

- ✅ Clear project purpose and overview
- ✅ Specific build, test, and lint commands
- ✅ Detailed tech stack information (Go 1.24, frameworks, dependencies)
- ✅ Code style guidelines with real examples
- ✅ Explicit acceptance criteria for task completion
- ✅ Clear boundaries (files/directories that should not be modified)
- ✅ Security guidelines and common vulnerability patterns
- ✅ Quick reference format (AGENTS.md) + comprehensive guide (copilot-instructions.md)

#### Future Enhancements

If needed, additional instruction files can be added:

- **Path-specific instructions** (`.github/instructions/NAME.instructions.md`): For specialized rules in specific directories
- **Agent personas** (`.github/agents/AGENT_NAME.md`): For specialized tasks like documentation generation or test writing

### `workflows/`

GitHub Actions workflow definitions:

- `go.yml` - Main CI workflow (build, test, coverage)
- `go-race.yml` - Race condition detection tests
- `golangci-lint.yml` - Code quality linting
- `claude.yml` - Claude Code integration
- `release.yml` - Release automation with GoReleaser

See individual workflow files for details.

## References

- [GitHub Copilot Custom Instructions](https://docs.github.com/en/copilot/how-tos/configure-custom-instructions)
- [AGENTS.md Format Specification](https://agents.md) - Open format for guiding coding agents
- [GitHub Copilot AGENTS.md Support](https://github.blog/changelog/2025-08-28-copilot-coding-agent-now-supports-agents-md-custom-instructions/)
- [Best Practices for Copilot in Repositories](https://github.blog/ai-and-ml/github-copilot/how-to-write-a-great-agents-md-lessons-from-over-2500-repositories/)
- [Best Practices for Using Copilot to Work on Tasks](https://docs.github.com/en/copilot/tutorials/coding-agent/get-the-best-results)
- [Repository Context for AI Tools](https://www.nathannellans.com/post/all-about-github-copilot-custom-instructions)
