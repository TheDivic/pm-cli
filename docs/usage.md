# `pm` usage guide

`pm` reads, validates, queries, formats, and safely mutates `*.tasks.yaml`
project files. It is non-interactive: every command takes what it needs from
arguments, flags, or standard input, and never opens a prompt.

The normative contract lives in [`docs/spec/`](spec/)
([format](spec/project-task-format.md), [CLI](spec/cli-specification.md)). This
guide is a practical reference.

## Global options

```
pm [--root <path>] [--json] [--no-ignore] <command> [arguments]
```

- `--root <path>` — discovery root. If omitted, `pm` uses `PM_ROOT`, then the
  current working directory. Set `export PM_ROOT=/path/to/tasks` to drop the
  flag.
- `--json` — machine-readable output for agents and automation. Successful
  results go to stdout; errors go to stderr.
- `--no-ignore` — include files that `.gitignore` rules would exclude.

Common flags have shorthands: `-p` (`--project`), `-t` (`--title`), `-s`
(`--status`), `-r` (`--reason`), `-g` (`--tag`), and `-a` (`--all`).

## Discovery

`pm` finds every `*.tasks.yaml` file beneath the root. It honors `.gitignore`
rules from the root down (nested, negated, and directory patterns), always skips
`.git`, and never follows directory symlinks. Project IDs and task-ID prefixes
must be unique across the root. A task ID resolves to its project through its
prefix, so most commands accept a task ID without a project path.

## Exit codes

| code | meaning |
|------|---------|
| 0 | success |
| 1 | invalid task data, references, or a bad transition |
| 2 | invalid command syntax or arguments |
| 3 | I/O, locking, or unexpected internal error |

## Projects

```sh
pm projects list [--status s]... [--priority n]... [--area a]...
pm projects show <project-id>
pm projects validate [<project-id> | --all]
pm projects format [<project-id> | --all]
pm projects create --id <id> --title <text> --task-id-prefix <prefix> \
    [--path <file>] [--status <s>] [--priority <n>] [--due <date>] [--area <a>]...
pm projects edit <project-id> [--title <t>] [--priority <n> | --clear-priority] \
    [--due <date> | --clear-due] [--add-area <a>]... [--remove-area <a>]...
pm projects status <project-id> <status> [--reason <sentence>]
```

- **list** — in-review projects first, then in-progress, then others; within a
  group by priority (lowest first, unset last), creation date, then ID. Columns: ID,
  title, status, priority, creation date, and a completion progress bar. JSON
  adds per-status task counts.
- **show** — full detail including areas, dates, blocking/cancellation, the
  task-file path, and a per-status task breakdown.

Progress counts every task that can still be finished. Backlog tasks are part of
the denominator — unfinished work is unfinished wherever it sits — but cancelled
tasks are not, since work that will never be done would otherwise cap a project
below 100% forever. `projects show` names the excluded count
(`80% (8/10 done, 2 cancelled)`), and a project whose tasks are all cancelled
reports nothing countable instead of 0%. JSON output is unchanged — it reports
raw per-status counts, including `cancelled` and `total`, so consumers can
compute whatever ratio they need.
- **validate** — checks the file(s) and report every problem in one run.
- **format** — rewrite in canonical form (idempotent); rejects invalid files
  without writing. Requires a project ID or `--all`.
- **status** — `blocked` and `cancelled` require `--reason`; entering
  `in-progress` sets `started` once; `done` sets `completed`; leaving `blocked`
  clears the record.

```sh
pm projects create --id website --title "Website" --task-id-prefix web --status in-progress
pm projects list --status in-progress --priority 1
pm projects status website blocked --reason "waiting on brand assets"
```

## Tasks

```sh
pm tasks list [-a|--all] [--project p]... [--status s]... [--tag t]... [--area a]... \
    [--parent <task-id>] [--blocked] [--due-before <date>] [--due-on <date>]
pm tasks show <task-id>
pm tasks add [--project <project-id>] --title <text> \
    [--description-file <file|->] [--status <s>] [--priority <n>] \
    [--parent <task-id>] [--due <date>] [--tag <t>]...
pm tasks edit <task-id>... [--title <t>] [--description-file <file|->] \
    [--priority <n> | --clear-priority] [--due <date> | --clear-due] \
    [--add-tag <t>]... [--remove-tag <t>]... [--parent <id> | --clear-parent]
pm tasks status <task-id>... <status> [--reason <sentence>]
pm tasks block <task-id>... --reason <sentence> [--task <task-id>]...
pm tasks unblock <task-id>...
pm tasks delete <task-id>... [--cascade]
```

- **list** — grouped by status (in-review, in-progress, todo, backlog, done,
  cancelled); within a group by priority then file order. Hides `done` and
  `cancelled` by default; `-a`/`--all` includes them, and an explicit `--status`
  overrides. Filters combine with AND; repeated values combine with OR.
- **add** — assigns the next task ID and the `created` date. Default status is
  `backlog`. `--description-file -` reads the description from standard input.
- **inbox** — omit `--project` and the task lands in the `inbox` project, created
  under the root the first time it is needed (`inbox/inbox.tasks.yaml`, prefix
  `in`, status `in-progress`). Capture first, file later:
  `pm tasks edit in-004 --parent web-001` or move it by hand once you know where
  it belongs.
- **status** — manages lifecycle dates and the mutually exclusive terminal and
  blocking fields. `cancelled` requires `--reason`.
- **block / unblock** — record or remove a blocking condition without changing
  the task status.
- **delete** — removes tasks outright. Refuses when another task points at one
  being deleted (as a parent or a blocker), naming the referrers; `--cascade`
  removes the subtree and drops those references. Prefer `status <id> cancelled`
  when the outcome should stay on the record — a delete is only recoverable
  through Git.
- **batch** — `edit`, `status`, `block`, and `unblock` take several task IDs and
  apply the same change to each. For `status` the last argument is the status and
  everything before it is an ID. Every ID is resolved before anything is written,
  so a typo changes nothing; tasks are then written one file at a time, so a
  file is all-or-nothing but a multi-file batch is not. Tasks already committed
  when a later file fails are named on stderr. `--title` stays single-task.

```sh
pm tasks add --project website --title "Design the header" --priority 1 --tag design
pm tasks add --title "Call the dentist"          # no project -> the inbox
echo "Acceptance: passes Lighthouse." | pm tasks add --project website --title "Audit perf" --description-file -
pm tasks status web-001 in-progress
pm tasks status web-001 web-002 web-003 done     # one status, several tasks
pm tasks edit web-001 web-002 --add-tag q3       # one edit, several tasks
pm tasks block web-002 --reason "depends on the header" --task web-001
pm tasks list --project website            # open work only
pm tasks list --project website --status done
```

## Tags

```sh
pm tags
```

Lists the distinct tags in use across all tasks with a per-tag usage count,
most-used first, in human-readable and `--json` form.

## Shell completion

```sh
pm completion bash > /etc/bash_completion.d/pm       # bash
pm completion zsh  > "${fpath[1]}/_pm"               # zsh
pm completion fish > ~/.config/fish/completions/pm.fish
pm completion powershell | Out-String | Invoke-Expression
```

Completion is dynamic — it reads the discovery root as you type, so task IDs
come annotated with status and title, project IDs with their titles, and `--tag`
and `--area` offer the vocabulary already in use. `pm tasks status <TAB>` offers
task IDs; from the second argument on it offers statuses too, and IDs you have
already typed drop out of the list. Set `PM_ROOT` so completion works from any
directory.

## Mutation safety

Every write follows a fixed sequence: lock the target file, read and parse it,
validate the document, apply one change, validate the result, render canonical
bytes, and atomically replace the file. Any error before the final write leaves
the original bytes unchanged, and a per-file lock prevents concurrent commands
in one checkout from colliding.

`tasks delete` is the one command that removes history rather than recording an
outcome. Cancelling (`tasks status <id> cancelled -r "<why>"`) remains the way
to retire work you want to stay legible; deletes are recoverable only through
Git.

## Agent and automation use

Pass `--json` to any command for structured output. Errors in JSON mode are a
single object on stderr with a stable `code`, `message`, and — when available —
`file`, `project`, `task`, and `field`. Combined with non-zero exit codes, this
lets an agent perform every operation and react to failures without parsing
human text.
