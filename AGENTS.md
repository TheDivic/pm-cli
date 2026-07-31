# Agent instructions

Instructions for AI agents working in this repository. The full rationale lives
in [`CONTRIBUTING.md`](CONTRIBUTING.md); this is the operational summary.

## Purpose

Implement `pm`, the Plaintext Tasks CLI, against the frozen format and CLI
specifications in the Plaintext Brain knowledge base (project `plaintext-tasks`).
Do not change the documented format or CLI behavior; only the implementation
path is ours to decide. See [`implementation-plan.md`](implementation-plan.md).

## Rules

- **Never push to `main`.** Work on a `type/short-slug` feature branch and open a
  **draft** pull request. Merge only with explicit maintainer authorization,
  using **squash merge**.
- Use **Conventional Commits** with the package name as scope (for example
  `feat(validate): check blocker cycles`). Allowed types: `feat`, `fix`, `docs`,
  `test`, `refactor`, `chore`, `ci`, `build`.
- **Do not add `Co-Authored-By` or any AI-attribution trailer** to commits.
- Before committing, run `make check` (gofmt, `go vet`, `golangci-lint`,
  `go test -race`). Keep `go mod tidy` clean and commit `go.sum`.
- Add or update tests in the same change that adds or changes behavior. Never
  mutate live knowledge base task files from tests; use temporary copies.
- Keep lifecycle dates behind the injectable clock; never read the machine date
  directly.
- **In a sandbox, build to a scratch path, not `./bin`.** When the working tree
  is a mounted copy of the host's (for example a Docker sandbox), a binary you
  compile is built for the sandbox OS/arch and will overwrite the host's
  `bin/pm` with something it cannot exec ("exec format error"). Use
  `go build -o /tmp/pm ./cmd/pm` or `go run ./cmd/pm ...` for smoke tests, and
  leave `bin/` for the host's own `make build`.

## Layout

Package layout and the ordered task breakdown (T1–T13) are in
[`implementation-plan.md`](implementation-plan.md).
