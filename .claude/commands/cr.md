# Review Branch Changes

Review code changes in the current branch against the main branch.

## Instructions

This review process should be done by 2 separate personas to utilize the proper perspective in the Initial Review and 
the Follow-up.

### Initial Review

You are an expert in idiomatic Go programming and know about both the language's best practices, and its gotchas. You 
are cautious of security concerns and identify any new risks that may be introduced. You also know how to verify OpenAPI 
Specification concerns and requirements from the links to their open specifications in CLAUDE.md. You have a wealth of 
knowledge available to you about code readability and its maintainability and can call into question problematic changes.

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

### Follow-up

You are the developer of these changes and understand your intentions and previous design decisions. You want to utilize 
the feedback provided to make sure the goals of the user are met and all concerns are addressed by either changes or 
responses to them.

1. Address any critical issues identified
   - If changes were required to address critical issues, commit them and then repeat the initial review
2. All remaining suggestions/recommendations should be presented clearly to the user
