# Plaintext Tasks (`pm`)

`pm` is a standalone command-line tool for managing projects and tasks stored as
readable `*.tasks.yaml` files. There is no hosted application and no database:
plain files are the source of truth and Git is the change history. `pm` is built
for both people (concise terminal output) and LLM agents (structured `--json`).

> **Status:** in active development. The task format and CLI contract are frozen;
> this repository implements the `pm` executable against them.

## What it does

- Discovers `*.tasks.yaml` files recursively, honoring `.gitignore` semantics
  without requiring the `git` executable.
- Validates files against schema version 1 with strict, actionable errors.
- Lists, filters, and inspects projects and tasks.
- Safely creates and edits projects and tasks, and drives their lifecycles,
  with atomic writes that leave the original file untouched on any failure.
- Emits a deterministic canonical format so `format` is idempotent.

## Specifications (normative, frozen)

The behavior contract lives in the Plaintext Brain knowledge base, project
`plaintext-tasks`, and is **not** duplicated here:

- **Project task format** — schema version 1 (`project-task-format.md`).
- **CLI specification** — commands, output, mutation safety, exit codes
  (`cli-specification.md`).

Our build plan lives in this repo: [`implementation-plan.md`](implementation-plan.md).

## Requirements

- Go 1.26 or newer.

## Development

```sh
make check   # gofmt check, go vet, golangci-lint, go test -race  (the CI gate)
make build   # builds ./bin/pm
make fmt     # rewrite files to canonical formatting
```

Enable the local push guard once after cloning:

```sh
git config core.hooksPath .githooks
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the full workflow, commit
conventions, and quality gates.
