# Contributing to Pole Chudes Bot

Thanks for your interest. Bug reports, feature ideas, new phrases, documentation fixes and code are all welcome.

## Ways to contribute

- Report a bug: open a [bug report](https://github.com/belaytzev/pole-chudes-bot/issues/new?template=bug_report.yml).
- Suggest a feature: open a [feature request](https://github.com/belaytzev/pole-chudes-bot/issues/new?template=feature_request.yml).
- Add phrases. The host's lines live in [`phrases.go`](phrases.go) and [`fun_phrases.go`](fun_phrases.go), one list per stage of the draw. A new line is a one-line change.
- Improve the documentation.
- Send code: fixes, new commands, new draw modes.

## Development setup

You need **Go 1.27+** (the exact version is in [`go.mod`](go.mod)) and [golangci-lint](https://golangci-lint.run/) v2.

```bash
git clone https://github.com/<your-username>/pole-chudes-bot.git
cd pole-chudes-bot
cp .env.example .env   # put a test bot token from @BotFather into BOT_TOKEN
```

Run the bot with a throwaway database:

```bash
BOT_TOKEN=... DB_PATH=/tmp/pole-test.db go run .
```

Use a separate test bot, not the one that serves a real chat: two copies on one token fight over the updates.

## Checks

CI runs the same checks, so run them before you push:

```bash
make check
```

That is `go vet`, `go test -race`, `golangci-lint` and a `gofmt` check. CI also runs `govulncheck` and CodeQL.

## Guidelines

- Add a test for new logic and a regression test for a bug fix. Store code is tested against an in-memory SQLite database, see `store_test.go`.
- The database schema may only grow: add tables with `CREATE TABLE IF NOT EXISTS`. Existing chats keep their data across upgrades.
- User-supplied text (names, quotes) must go through `html.EscapeString` before it reaches a message: the bot sends HTML.
- Keep comments for the non-obvious "why". Don't narrate what the code already says.
- New phrases are checked by `TestPhrasesRender`: add your pool to its list.

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Keep one change per pull request and describe what and why.
3. Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages (`feat:`, `fix:`, `docs:`, `ci:`).
4. Make sure CI is green. A maintainer merges with squash.

By contributing you agree that your contribution is licensed under the [MIT License](LICENSE).
