# Project task format

This document defines schema version 1 of `*.tasks.yaml`. It is the normative reference for PM CLI files, migrations, validation, and CLI behavior.

## File role and name

Each project stores execution state in one file named `<project-id>.tasks.yaml`. The filename stem must match `project.id`, and a directory cannot contain more than one project task file. The file is the only source for project status, tasks, task order, blockers, hierarchy, and lifecycle dates.

The project definition and notes must not duplicate this state. Git records changes over time, so the task file does not contain an activity log or last-updated timestamp.

## YAML profile

Task files use a restricted YAML 1.2 profile:

- Encode files as UTF-8 with LF line endings.
- Use spaces for indentation, with two spaces at each level. Tabs are invalid.
- Store exactly one YAML document.
- Quote every date so parsers keep it as a string.
- Do not use anchors, aliases, merge keys, explicit tags, or duplicate mapping keys.
- Do not use comments. Put task-specific detail in `description` and durable project context in the project definition.
- Reject unknown fields instead of silently preserving or discarding them.
- Write fields in the canonical order defined below.

An empty project writes `tasks: []` rather than omitting `tasks` or using a null value.

## Top-level structure

The document has three top-level fields in this order:

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

tasks: []
```

`schema-version` is a required integer. A reader must reject unsupported versions and must not guess how to migrate them.

## Project record

The `project` mapping uses these fields in canonical order:

1. `id`
2. `title`
3. `task-id-prefix`
4. `status`
5. `priority`
6. `areas`
7. `created`
8. `started`
9. `due`
10. `blocked`
11. `cancellation`
12. `completed`

### Project fields

`id` is required. It is a stable lowercase kebab-case identifier used by CLI commands and must be unique within a discovery root. It must match the task filename before `.tasks.yaml`; changing the project title does not change the ID.

`title` is required. It is the nonempty, single-line project name shown in human-readable output and may be changed without changing the project ID.

`task-id-prefix` is required. It is a unique, immutable lowercase kebab-case prefix used to form task IDs. Renaming the project, directory, or project ID does not change the prefix or existing task IDs.

`status` is required and accepts `idea`, `todo`, `in-progress`, `in-review`, `blocked`, `cancelled`, or `done`. Project status is independent of individual task status. `in-review` means the project's work is finished and awaiting validation or acceptance, matching the task status of the same name; it carries no dates of its own.

`priority` is optional and accepts a positive integer. Lower numbers have higher priority, so `1` is highest. Equal values are allowed. Projects without a priority sort after prioritized projects.

`areas` is an optional list of unique lowercase kebab-case slugs. Each slug must resolve to `areas/<slug>.md`. A project may belong to more than one area, and the list may change as its responsibilities change.

`created` is required and records when the project execution record was created. `started` records the first date the project entered `in-progress` and remains unchanged afterward. `due` is optional and is used only for a real project deadline.

A project with status `blocked` requires a `blocked` mapping containing `reason` and `since`. A cancelled project requires `cancellation`. A done project requires `completed`.

## Task records

`tasks` is a flat sequence. Every task is a mapping with fields in this canonical order:

1. `id`
2. `title`
3. `description`
4. `status`
5. `priority`
6. `parent`
7. `created`
8. `started`
9. `due`
10. `tags`
11. `blocked`
12. `cancellation`
13. `completed`

### Required fields

Every task requires `id`, `title`, `status`, and `created`.

`id` must match `<task-id-prefix>-<sequence>`. The sequence contains at least three decimal digits. IDs are unique within a discovery root, immutable, and never reused.

`title` is a nonempty, single-line string that describes an actionable outcome. The CLI quotes it when YAML plain-scalar rules require quoting.

`status` accepts `backlog`, `todo`, `in-progress`, `in-review`, `cancelled`, or `done`.

`priority` is optional and accepts a positive integer, where `1` is the highest priority. Equal values are allowed. Tasks without a priority sort after prioritized tasks. Priority uses the same semantics as project priority. Within an equal or absent priority, file order within the same status and parent group breaks the tie and defines execution priority.

`created` records the date the task record entered the task system. For migrated tasks, it may be the migration date when the original creation date cannot be recovered reliably.

### Description

`description` is optional Markdown stored as a YAML literal block. Use it for scope, constraints, or acceptance detail that does not fit in the title. Do not use it as a progress log or duplicate project context.

```yaml
description: |
  Explain the required outcome in ordinary prose.

  Markdown lists and links are allowed:

  - preserve plaintext readability;
  - keep the detail relevant to this task.
```

### Status semantics

- `backlog` contains optional, speculative, or insufficiently defined work that has not been accepted into scope.
- `todo` contains accepted work that is ready or expected to become ready. File order is execution priority among tasks with the same status and parent.
- `in-progress` contains work that has started. A blocked task remains `in-progress` if work had already begun.
- `in-review` contains finished work awaiting validation or acceptance.
- `cancelled` contains work deliberately closed without completion.
- `done` contains completed and accepted work.

Status transitions may skip intermediate states when the work makes that accurate. A task blocked before it starts remains `todo`; blocking does not define a separate status.

### Dates

All dates are quoted strings in `YYYY-MM-DD` form and must be valid calendar dates.

`created` is required. `started` is added the first time work begins and is never replaced, even if the task later returns to `todo` or `in-progress`. Migrated historical tasks may omit `started` when no reliable date exists.

`due` is optional and means an actual deadline, not a preferred work date. Keep it after completion or cancellation so later reports can compare the deadline with the outcome.

Tasks with status `done` require `completed`. Tasks with status `cancelled` require `cancellation`. Other statuses must not contain those terminal fields.

### Cancellation

`cancellation` contains a required sentence explaining the decision and the date it occurred:

```yaml
cancellation:
  reason: The prototype no longer supports the accepted project direction.
  date: "2026-07-31"
```

The reason must be useful without consulting Git history. The CLI does not delete unfinished tasks; it cancels them.

### Blocking

`blocked` is optional on `todo`, `in-progress`, and `in-review` tasks. It is invalid on `backlog`, `cancelled`, and `done` tasks.

```yaml
blocked:
  reason: The format specification must receive acceptance before implementation begins.
  tasks:
    - ex-001
  since: "2026-07-31"
```

`reason` and `since` are required. `reason` is a concise sentence that explains what prevents progress and makes the resumption condition clear. `tasks` is an optional nonempty list of task IDs that currently cause the blockage.

Referenced blockers must exist in the same project. A task cannot block itself. Duplicate references and dependency cycles are invalid. Remove the complete `blocked` mapping when the condition is resolved; Git retains its history.

### Parent-child relationships

`parent` is optional and contains the ID of another task in the same project. The flat list remains the storage model; clients may render it as a tree.

Parent references must exist and cannot form self-references or cycles. Put children immediately after their parent and preserve sibling order. A parent cannot become `done` while a descendant is unfinished. Cancelling a parent requires cancelling or reparenting every unfinished descendant.

Task status, dates, tags, and blockers do not inherit from the parent.

### Tags

`tags` is an optional list of unique lowercase kebab-case values. Tags support cross-project queries such as `research` or `documentation`. They do not replace structured fields and should not repeat the project name, area, status, or blocker state.

The repository should maintain a controlled vocabulary once repeated use establishes useful tags. Schema version 1 does not require a global tag registry.

## Task IDs

The CLI allocates `max(existing sequence) + 1` for the project's prefix and pads the sequence to at least three digits. It scans completed and cancelled tasks as well as unfinished tasks.

IDs do not encode task state and do not change when a task moves, is renamed, gains a parent, or the project directory is renamed. Tasks and IDs are not hard-deleted. This rule prevents references from silently pointing at a different task later.

Manual edits may add a task without an ID only as an incomplete draft. Validation fails until the CLI assigns the ID. A mutation command must never leave an ID missing or duplicated.

## Ordering

The optional `priority` field is the primary priority signal (`1` highest, absent last). File order is the secondary signal: within an equal or absent priority in the same status and parent group, an earlier task has higher priority. Root tasks are ordered relative to other root tasks, and children are ordered relative to their siblings. When filtering tasks, preserve their relative order. The first ready `todo` task in the sorted order is next.

The formatter uses tree preorder: write a parent, then its descendants, before continuing with the next sibling. File order is the stored order and the formatter does not reorder tasks by priority, ID, or title.

## Validation rules

Validation covers the complete file and any referenced repository paths. At minimum it checks:

- the supported schema version and restricted YAML profile;
- required, unknown, duplicate, and canonically ordered fields;
- project ID syntax and uniqueness within the discovery root;
- project ID and task filename agreement;
- project and task priority type and range;
- project and task status values;
- task ID format, prefix, uniqueness, and references;
- area file existence;
- date syntax and lifecycle consistency;
- terminal status requirements;
- blocking rules and cycles;
- parent existence, ordering, and cycles;
- tag syntax and uniqueness; and
- one project task file per directory.

Validation reports every independent problem it can find in one run. Each error includes the file, task ID when available, field path, and a plain explanation.

## Canonical example

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
    title: Define the accepted task schema
    description: |
      Record fields, lifecycle rules, and validation behavior.
    status: in-progress
    priority: 1
    created: "2026-07-31"
    started: "2026-07-31"
    tags:
      - project-management

  - id: ex-002
    title: Implement the schema validator
    status: todo
    parent: ex-001
    created: "2026-07-31"
    blocked:
      reason: The schema must receive acceptance before implementation begins.
      tasks:
        - ex-001
      since: "2026-07-31"
```
