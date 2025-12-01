# GitHub Configuration

This directory contains GitHub-specific configuration files for the oastools repository.

## Files

### `copilot-instructions.md`

Repository-wide instructions for GitHub Copilot coding agent. This file provides:

- Project overview and architecture
- Development commands and workflows
- Code style guidelines and best practices
- Testing and benchmark requirements
- Security considerations
- Acceptance criteria for task completion
- Boundaries and exclusions (protected files/directories)

These instructions help GitHub Copilot understand the project and make informed decisions when generating or modifying code.

#### Best Practices Followed

The instructions file follows [GitHub's best practices](https://docs.github.com/en/copilot/how-tos/configure-custom-instructions/add-repository-instructions) for Copilot custom instructions:

- ✅ Clear project purpose and overview
- ✅ Specific build, test, and lint commands
- ✅ Detailed tech stack information (Go 1.24, frameworks, dependencies)
- ✅ Code style guidelines with real examples
- ✅ Explicit acceptance criteria for task completion
- ✅ Clear boundaries (files/directories that should not be modified)
- ✅ Security guidelines and common vulnerability patterns

#### Future Enhancements

If needed, additional instruction files can be added:

- **Path-specific instructions** (`.github/instructions/NAME.instructions.md`): For specialized rules in specific directories
- **Agent personas** (`.github/agents/AGENT_NAME.md`): For specialized tasks like documentation generation or test writing

Currently, the single `copilot-instructions.md` file is comprehensive and sufficient for this repository.

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
- [Best Practices for Copilot in Repositories](https://github.blog/ai-and-ml/github-copilot/how-to-write-a-great-agents-md-lessons-from-over-2500-repositories/)
- [Repository Context for AI Tools](https://www.nathannellans.com/post/all-about-github-copilot-custom-instructions)
