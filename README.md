# Plaintext Projects (`pm`)

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

## Usage

Full command reference and examples: [`docs/usage.md`](docs/usage.md).

```sh
export PM_ROOT=/path/to/tasks          # optional default discovery root
pm projects list                       # overview with progress bars
pm tasks list --project website        # open tasks for one project
pm tasks add --project website --title "Design the header" --priority 1
pm tasks status web-001 in-progress
pm --json tasks list --status todo     # machine-readable output for agents
```

## Specifications (normative, frozen)

The behavior contract lives in the Plaintext Brain knowledge base, project
`plaintext-projects`, and is **not** duplicated here:

- **Project task format** — schema version 1 (`project-task-format.md`).
- **CLI specification** — commands, output, mutation safety, exit codes
  (`cli-specification.md`).

Our build plan lives in this repo: [`implementation-plan.md`](implementation-plan.md).

## Install

`pm` is a single self-contained binary with no runtime dependencies.

**Prebuilt binary** — download the archive for your OS/arch from the
[releases page](https://github.com/TheDivic/plaintext-projects/releases), extract
it, and put `pm` on your `PATH`.

**With Go** (1.26+):

```sh
go install github.com/TheDivic/plaintext-projects/cmd/pm@latest
```

**From source:**

```sh
git clone https://github.com/TheDivic/plaintext-projects
cd plaintext-projects
make build            # produces ./bin/pm with version metadata baked in
```

Verify the install and set a default discovery root so you can omit `--root`:

```sh
pm version
export PM_ROOT=/path/to/your/tasks   # optional; --root overrides it
```

## Agent use

Pass `--json` to any command for structured output and rely on the exit codes;
`pm` never prompts. To teach an LLM agent to drive `pm` without reading this
repository, hand it the portable skill in [`skills/`](skills/README.md) — it
carries the full command surface, the lifecycle rules, and the file schema.

## Requirements

- To build from source: Go 1.26 or newer. Prebuilt binaries need no runtime.

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
