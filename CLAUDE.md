@AGENTS.md

## Claude Code notes

- Work in a git worktree for any change that edits files, e.g. `git worktree add -b <branch> .claude/worktree/<task> origin/main`. `.claude/worktree/` is gitignored.
- Verify with `make check` and report which checks ran. Do not claim a change works from reading the code alone.
- A change that touches Telegram behavior can't be tested without a live bot. Say so instead of implying it was tested, and don't start a second copy on a token that is already in use.
- The `Claude agent` workflow (`.github/workflows/agent.yml`) answers `@claude` comments from the repository owner only and needs the `CLAUDE_CODE_OAUTH_TOKEN` secret.
