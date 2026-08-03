# `*.tasks.yaml` schema (version 1)

Reference for interpreting validation errors and reviewing diffs. **You still do
not hand-edit these files** — `pm` writes them. See `../SKILL.md`.

## File role and name

One project per file, one project file per directory, named
`<project-id>.tasks.yaml`. The filename stem must equal the project's `id`.

## YAML profile (restricted)

Files use a deliberately narrow subset of YAML. `pm` rejects anything outside it
rather than guessing:

- UTF-8, LF line endings, two-space indentation. **Tabs are invalid.**
- Exactly one YAML document, whose root is a mapping.
- **No** anchors, aliases, merge keys, explicit tags, or duplicate mapping keys.
- **No comments.** Durable context belongs in the project definition; task detail
  belongs in `description`.
- Every date is quoted, so it stays a string (`created: "2026-07-31"`).
- Unknown fields are rejected, never silently preserved or dropped.
- Fields appear in canonical order (below). `pm projects format` enforces it.

## Structure

Three top-level fields, in this order: `schema-version`, `project`, `tasks`.

```yaml
schema-version: 1

project:
  id: example-project
  title: Example Project
  task-id-prefix: ex
  status: in-progress
  priority: 1
  areas:
    - knowledge-work
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: ex-001
    title: Do the thing
    description: |
      Scope, constraints, or acceptance detail.
    status: in-progress
    priority: 1
    created: "2026-07-31"
    started: "2026-07-31"
    tags:
      - tooling

  - id: ex-002
    title: Do the other thing
    status: backlog
    created: "2026-07-31"
```

`schema-version` is a required integer. Version 1 is the only supported value;
an unsupported version is an error, never a migration attempt.

## Project fields

Canonical order: `id`, `title`, `task-id-prefix`, `status`, `priority`, `areas`,
`created`, `started`, `due`, `blocked`, `cancellation`, `completed`.

| field | required | notes |
|-------|----------|-------|
| `id` | yes | stable lowercase kebab-case; unique per discovery root; must equal the filename stem. A title change never changes it. |
| `title` | yes | nonempty, single-line |
| `task-id-prefix` | yes | unique, **immutable** lowercase kebab-case; surviving renames of the project, ID, or directory |
| `status` | yes | `idea`, `todo`, `in-progress`, `blocked`, `cancelled`, `done` — independent of task statuses |
| `priority` | no | positive integer, `1` highest; ties allowed; unset sorts last |
| `areas` | no | unique lowercase kebab-case slugs, each resolving to `areas/<slug>.md` |
| `created` | yes | when the record was created |
| `started` | no | first entry into `in-progress`; never overwritten afterward |
| `due` | no | a real deadline only |
| `blocked` | if `blocked` | mapping: `reason`, `since` (+ optional `tasks`) |
| `cancellation` | if `cancelled` | mapping: `reason`, `date` |
| `completed` | if `done` | date |

## Task fields

Canonical order: `id`, `title`, `description`, `status`, `priority`, `parent`,
`created`, `started`, `due`, `tags`, `blocked`, `cancellation`, `completed`.

Required: `id`, `title`, `status`, `created`.

| field | notes |
|-------|-------|
| `id` | `<task-id-prefix>-<sequence>`, sequence ≥ 3 digits. Unique per discovery root, immutable, **never reused**. |
| `title` | nonempty, single-line, describing an actionable outcome |
| `description` | optional Markdown as a YAML literal block (`\|`). Scope and acceptance detail — not a progress log, not duplicated project context. |
| `status` | `backlog`, `todo`, `in-progress`, `in-review`, `cancelled`, `done` |
| `priority` | positive integer, `1` highest; ties allowed; unset sorts last |
| `parent` | task ID in the same project; cycles rejected |
| `tags` | unique lowercase kebab-case slugs |
| `blocked` | mapping: `reason`, `since`, optional `tasks` (blocking task IDs). Invalid on `backlog`, `cancelled`, `done`. |
| `cancellation` | required when `cancelled`: `reason`, `date` |
| `completed` | required when `done` |

### Status semantics

- `backlog` — speculative or under-defined work, not accepted into scope.
- `todo` — accepted work, ready or expected to become ready.
- `in-progress` — started. **A blocked task stays `in-progress`** if work had
  already begun; blocking is a separate record, not a status.
- `in-review` — finished, awaiting validation or acceptance.
- `cancelled` — deliberately closed without completion.
- `done` — completed and accepted.

## Task IDs

`pm` allocates `max(existing sequence) + 1` for the prefix, padded to at least
three digits, scanning `done` and `cancelled` tasks too. IDs never encode state
and never change when a task moves, is renamed, gains a parent, or the directory
is renamed. Nothing is hard-deleted — that is what keeps references honest.

A task without an ID is only ever an incomplete draft: validation fails until
`pm` assigns one.

## Ordering

`priority` is the primary signal (`1` highest, absent last). **File order is the
secondary signal**: within equal or absent priority in the same status and parent
group, earlier means higher priority. Root tasks order against root tasks,
children against siblings. Filtering preserves relative order.

The formatter writes tasks in tree preorder — a parent, then its descendants,
then the next sibling — and never reorders by priority, ID, or title. File order
is the stored order.

## What validation checks

`pm projects validate` reports every problem in one run:

- supported schema version and the restricted YAML profile
- required, unknown, duplicate, and canonically ordered fields
- project ID syntax, uniqueness, and agreement with the filename
- one project task file per directory
- project and task status values; priority type and range
- task ID format, prefix, uniqueness, and references
- date syntax and lifecycle consistency; terminal status requirements
- blocking rules and blocker cycles
- parent existence, ordering, and cycles
- tag syntax and uniqueness
- area file existence (`areas/<slug>.md`)
