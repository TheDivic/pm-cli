# Contributing

This document is the authoritative record of the development process and
conventions for the `pm` CLI. Agents and people follow the same rules.

## Branching and merging

- `main` is the primary branch and is **protected**: never push to it directly.
  Enforcement is twofold — a GitHub branch-protection rule on the remote and a
  local `pre-push` hook in `.githooks/` (enable with
  `git config core.hooksPath .githooks`).
- Do all work on a **feature branch**, named `type/short-slug` where `type`
  matches the commit type below — for example `feat/yaml-parser`,
  `fix/emit-idempotence`, `chore/ci`.
- Land each feature by opening a **pull request** and using **squash merge**, so
  every feature becomes exactly one Conventional Commit on `main` and history
  stays linear and readable.
- Pull requests open as **draft** and are merged only on explicit maintainer
  authorization.

## Commit messages

- Use [Conventional Commits](https://www.conventionalcommits.org/).
- **Scope is the package name**, e.g. `feat(parse): reject merge keys`,
  `fix(emit): stabilize blank line between tasks`, `test(discover): nested ignore`.
- Allowed types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`,
  `build`. Use `!` (or a `BREAKING CHANGE:` footer) for breaking changes.
- Keep commit messages plain. **Do not add** `Co-Authored-By` trailers or any
  other AI-attribution footers.

## Quality gates

Every pull request must be green before merge. `make check` runs the same gate
locally that CI runs:

- **Formatting** — `gofmt` clean (and `goimports` when available).
- **Vet** — `go vet ./...`.
- **Lint** — `golangci-lint run` (config in `.golangci.yml`).
- **Tests** — `go test -race ./...`.
- **Build** — `go build ./...`.

Other standing rules:

- Keep `go mod tidy` clean and commit `go.sum`. No vendoring.
- The `go` directive and toolchain are pinned in `go.mod` (Go 1.26).
- Document exported identifiers with Go doc comments.

## Testing

- Table-driven tests live beside the code they cover; a new package ships with
  its tests in the same pull request.
- Golden fixtures under `testdata/` verify canonical formatting and its
  idempotence (emit twice, compare bytes).
- Mutation tests operate on copies in a temporary discovery root. Live knowledge
  base task files may be used as **read-only** inputs for parse, validate, and
  discovery acceptance only — never mutated by tests.
- Lifecycle dates come from an injectable clock; tests never depend on the
  machine date.

## Continuous integration

`.github/workflows/ci.yml` runs the formatting check, `go vet`,
`golangci-lint`, `go test -race`, and `go build` on every pull request and on
pushes to `main`. Because the workflow file is executable automation, changes to
it go through the pull-request lane like any other code.

## Repository layout and plan

The package layout and the ordered task breakdown (T1–T13) are described in
[`implementation-plan.md`](implementation-plan.md). The normative format and CLI
specifications are frozen in [`docs/spec/`](docs/spec/); treat them as the
contract, and amend them only deliberately, in the same change as the code.
