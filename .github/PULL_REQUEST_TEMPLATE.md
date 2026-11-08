# Pull Request

## What

<!-- What does this PR change? Link related issues: Fixes #123 -->

## Why

<!-- Why is this change needed? -->

## Checklist

- [ ] `make check` passes (`go vet`, `go test -race`, `golangci-lint`, gofmt)
- [ ] New code has tests, a bug fix has a regression test
- [ ] Schema changes only add tables or columns (`CREATE TABLE IF NOT EXISTS`)
- [ ] User text reaches messages through `html.EscapeString`
- [ ] Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
- [ ] No tokens, `.env` files or database files included
