# Review Branch Changes

Review code changes in the current branch against the main branch.

## Instructions

1. Get the default branch name (main or master) using `git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@'` or default to "main"
2. Get the current branch name using `git branch --show-current`
3. If on the default branch, review uncommitted changes with `git diff` instead
4. Otherwise, get the diff with `git diff <default-branch>...<current-branch>`
5. If there are no changes, inform the user
6. Otherwise, review the changes focusing on:
   - **Code Quality**: Bugs, logic errors, edge cases not handled
   - **Best Practices**: Go conventions, project patterns from CLAUDE.md
   - **Security**: Vulnerabilities, unsafe operations, secrets in code
   - **Performance**: Obvious performance issues
   - **Testing**: Test coverage, missing test cases
   - **Documentation**: Comments for complex logic, missing godoc

Be concise and actionable. If changes look good, say so briefly.
