# Plaintext Projects — implementation plan

Implementation path for the `pm` CLI. The format and CLI **contract** are frozen by the
accepted specs in the knowledge base and are not restated or changed here:

- `projects/plaintext-projects/project-task-format.md` — normative schema v1.
- `projects/plaintext-projects/cli-specification.md` — commands, output, mutation safety, exit codes.

This document owns only *how* we build it: module, dependencies, package layout, parser/emitter
strategy, and our own task breakdown.

## Toolchain and dependencies

- Language: Go 1.26, module `github.com/TheDivic/plaintext-projects`, binary `pm`.
- `github.com/spf13/cobra` — subcommand dispatch and help. Configured with `SilenceErrors` and
  `SilenceUsage`; `main` maps our structured errors to exit codes 0/1/2/3 and picks human vs JSON
  rendering. Cobra never prints our diagnostics.
- `gopkg.in/yaml.v3` — used **only** to parse to a `yaml.Node` tree. We never `Marshal` and never
  decode straight into structs.
- `github.com/go-git/go-git/v5/plumbing/format/gitignore` — pure-Go gitignore matching (nested
  files, negation, directory patterns). No `git` executable required at runtime.

## Core architectural stance

**1. Read and write are asymmetric — no marshaller round-trip.**

- Read: bytes → `yaml.Node` → restricted-profile enforcement on the tree (reject aliases, anchors,
  merge keys, explicit tags, duplicate keys, tabs, multiple documents, unsupported scalar forms) →
  decode the validated tree into typed structs → semantic validation.
- Write: a hand-written deterministic emitter serializes the typed model directly. This is the only
  way to guarantee the spec's exact canonical bytes — field order, blank line between tasks, quoted
  dates, `description: |` literal blocks, preorder task layout — and byte-identical `format`
  idempotence.

**2. One mutation = one pure transform.** Each mutating command is `model → model` in memory,
wrapped by a single shared safety envelope: resolve+lock target → read → validate → apply one
change → re-validate → emit canonical → temp-write + fsync → atomic rename → unlock. Commands never
touch the filesystem directly, so "failure leaves original bytes unchanged" is structural.

**3. Injectable clock.** All lifecycle dates come from a `clock` interface. Tests inject it; nothing
reads the machine date directly.

## Package layout

```
plaintext-projects/
  go.mod
  cmd/pm/main.go            thin: build cobra tree, dispatch, map errors -> exit codes
  internal/
    model/       typed structs, status enums, canonical field-order metadata
    yamlprofile/ yaml.Node loader + restricted-profile enforcement
    decode/      validated Node -> typed model
    validate/    semantic validation; collects ALL findings in one run
    emit/        canonical deterministic emitter (idempotent)
    query/       filtering + ordering (file order preserved after filter)
    mutate/      pure model transforms + task-ID allocation (max+1, >=3 digits)
    discover/    recursive walk, .git/symlink guards, project ID/prefix uniqueness
    ignore/      gitignore engine wrapper over go-git matcher
    fsatomic/    lockfile + temp-file + atomic rename
    clock/       injectable clock
    pmerr/       structured error: code, message, file, projectID, taskID, field
    cli/         command wiring, flag parsing, human + JSON renderers
  testdata/      golden fixtures for parse / validate / format / query
```

## Task breakdown

Vertical slices, each independently testable. Right column maps to the frozen knowledge-base
milestones (pt-011..pt-017) for traceability.

| #   | Task | KB milestone |
|-----|------|--------------|
| T1  | Scaffold module, `cmd/pm` cobra root, `pmerr` type + exit-code mapping, injectable clock | pt-011 |
| T2  | `model` structs + canonical field-order metadata | pt-011 |
| T3  | `yamlprofile` loader + restricted-profile checks → `decode` to model | pt-011 |
| T4  | `validate`: full semantic validation, all-errors collection | pt-011 |
| T5  | `emit`: canonical emitter + `format` idempotence golden tests | pt-011 |
| T6  | `ignore` gitignore engine + tests (negation / dir / nested) | pt-012 |
| T7  | `discover`: recursive walk, `.git`/symlink guards, ID/prefix uniqueness | pt-012 |
| T8  | `cli` plumbing: global `--root/--json/--no-ignore`, human/JSON renderers, `NO_COLOR` | pt-012 |
| T9  | Read-only commands: `projects list/show/validate/format`, `tasks list/show` (+ filters, ordering) | pt-012 |
| T10 | `fsatomic` lock + atomic write; wire the mutation safety envelope | pt-013 |
| T11 | Mutations: `projects create/edit/status`, `tasks add/edit/status/block/unblock` (+ ID allocation) | pt-013 |
| T12 | Acceptance: read-only run vs live brain `--root`; full spec test matrix; operator README | pt-015 |
| T13 | Package `pm` build; open draft PR | pt-014 / pt-016 |

## Testing approach

- Table-driven unit tests per package; `testdata/` golden files for canonical `format` idempotence
  (emit twice → identical bytes).
- Mutation tests operate on copies in a temp discovery root; live brain files are read-only inputs
  for parse/validate/discover acceptance only.
- Every spec bullet under "Tests" maps to at least one case: restricted-profile rejections, nested
  `.gitignore` semantics, duplicate IDs/prefixes, ref/parent/blocker cycles, ID allocation with gaps
  and >3 digits, immutable `started`, lock contention and interrupted writes, symlink/root-boundary
  guards, and JSON success/error shape for every command.

## Open implementation questions

None blocking. Revisit before T13: exact repo hosting (public vs private) and release/build packaging
(`go build` vs goreleaser) once the interface is validated.
