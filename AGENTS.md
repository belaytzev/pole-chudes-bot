# AGENTS.md

Guide for AI coding agents working in this repository. The README is the source of truth for users; this file covers what is easy to break.

## What this is

A Telegram bot in Go ([telebot v3](https://github.com/tucnak/telebot)) with SQLite storage ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), no CGO). Once a day it picks a random player per group chat and plays a game-show style animation. It also handles birthdays, greetings and a quote book.

## Layout

| Path | Role |
|---|---|
| `main.go` | bot setup, command handlers, the draw animation, graceful shutdown |
| `store.go` | storage core: `players`, `draws`, `Draw`/`DrawExcept` |
| `store_fun.go` | birthdays, quotes, greetings tables and queries |
| `fun.go` | `/dr`, `/citata`, the greeting loop |
| `phrases.go`, `fun_phrases.go` | all text the bot says; one pool (a `[]string`) per stage |
| `store_test.go`, `fun_test.go` | tests against an in-memory SQLite |
| `assets/` | picture, stickers, animation and `draw.py` that renders them |

## Commands

```sh
make check   # go vet, go test -race, golangci-lint, gofmt check
make test
go run .     # needs BOT_TOKEN; use a throwaway DB_PATH and a test bot
```

Run `make check` before declaring a change done. CI also runs `govulncheck` and CodeQL.

## Rules that are easy to break

- **Schema only grows.** Add tables with `CREATE TABLE IF NOT EXISTS`; never drop or rewrite existing ones. Live chats keep years of history.
- **One poller per token.** Never start a second instance with a real token (e.g. to test): Telegram answers `409` and the bots fight. Tests use `OpenStore(":memory:")`, not Telegram.
- **The bot sends HTML.** Every user-supplied string (names, quotes) goes through `html.EscapeString` or `esc(u)`. `TestPhrasesRender` fails when a phrase contains `{`, `}`, `<` or `&` after rendering.
- **New phrase pool?** Add it to the list in `TestPhrasesRender`.
- **SQLite runs with `SetMaxOpenConns(1)`.** Do not hold a `*sql.Rows` open while issuing another query on the same store: it deadlocks.
- **Read-only root filesystem.** The container writes only to `/data`. SQLite uses `temp_store(memory)` for that reason; keep temp files off disk.
- **Shutdown must finish running draws.** Handlers are tracked by a `WaitGroup` in `main`; keep new long-running handlers inside it.
- **The title is masked** (`Title` in `phrases.go`) on purpose; don't spell it out.

## Style

- Go standard formatting; `gofmt` is enforced.
- Comment only a non-obvious "why". No narration of what the code does.
- Conventional Commits (`feat:`, `fix:`, `docs:`, `ci:`). Squash merge.
- Never commit tokens, `.env`, or `*.db` files.

## Releases

Pushing a `v*` tag runs CI, publishes `ghcr.io/belaytzev/pole-chudes-bot:<version>` and creates a GitHub release. Merging to `main` publishes `latest`.
