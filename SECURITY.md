# Security policy

## Supported versions

Only the latest release and `main` get security fixes.

| Version | Supported |
| ------- | --------- |
| latest release, `main` | ✅ |
| older releases | ❌ |

## Reporting a vulnerability

Please **don't** report vulnerabilities in public GitHub issues.

Use [GitHub private vulnerability reporting](https://github.com/belaytzev/pole-chudes-bot/security/advisories/new): open the repository's **Security** tab and click **Report a vulnerability**.

It helps if you include:

- what the vulnerability is and what an attacker could do with it
- steps to reproduce, or a proof of concept
- the affected component, for example `main.go`, the SQLite store or the Docker image
- a suggested fix, if you have one

Expect a first reply within a few days. Please give us time to fix the issue before you disclose it publicly.

## What to look at

- The bot token (`BOT_TOKEN`). It must never appear in logs, replies or the image.
- Chat data in the SQLite file: player names, user ids, birthdays, saved quotes.
- Text from users that the bot echoes back (names, quotes). It is sent with HTML parse mode and must stay escaped.
- The Docker deployment (`Dockerfile`, `docker-compose.yml`).
