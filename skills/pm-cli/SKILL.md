---
name: pm-cli
description: Manage projects and tasks stored as plaintext *.tasks.yaml files using the `pm` CLI. Use whenever the user asks to list, filter, inspect, add, edit, complete, cancel, block, or reprioritize tasks or projects, asks what to work on next or how a project is progressing, or when the working tree contains *.tasks.yaml files. Covers the full command surface, JSON output for automation, and lifecycle rules — never hand-edit a *.tasks.yaml file.
---

# PM CLI (`pm`)

`pm` manages projects and tasks stored as human-readable `*.tasks.yaml` files. No
database, no server: the files are the source of truth and Git is the history.

## The one rule

**Never hand-edit a `*.tasks.yaml` file. Never write one with `Write`/`Edit`, and
never patch one with `sed`.** Always go through `pm`.

Every write goes through a safety envelope — lock, parse, validate, apply one
change, re-validate, emit canonical bytes, atomic rename — so a failure leaves
the original bytes untouched. Hand-editing bypasses all of that: it silently
breaks canonical formatting (making the next `pm projects format` produce a large
spurious diff), skips validation, and can corrupt task-ID allocation. `pm` also
manages lifecycle dates for you; setting them by hand gets them wrong.

If `pm` cannot express the change you need, say so rather than editing the file
directly.

## Preflight

```sh
pm version                             # confirm pm is installed and on PATH
```

If `pm` is missing, stop and tell the user — do not fall back to editing files by
hand. It is a single self-contained binary installed from its own repository (a
released archive, `go install`, or `make build`), not a `pip`/`npm` package.

Set the discovery root once, then omit it:

```sh
export PM_ROOT=/path/to/tasks          # or pass --root <path> per command
```

Resolution order is `--root`, then `$PM_ROOT`, then the current directory. `pm`
finds every `*.tasks.yaml` beneath the root, honors `.gitignore` (use
`--no-ignore` to override), skips `.git`, and never follows directory symlinks.

**Task IDs are globally unique and self-routing.** A task ID carries its
project's prefix (`web-001` → project with prefix `web`), so task commands never
need a project or a path — just the ID.

## Global flags

| flag | effect |
|------|--------|
| `--root <path>` | discovery root (default `$PM_ROOT`, then cwd) |
| `--json` | machine-readable output; **use this whenever you will parse the result** |
| `--no-ignore` | include paths `.gitignore` would exclude |
| `--color <auto\|always\|never>` | override terminal-color detection; `always` keeps `projects doc` output styled through a pipe into a pager (`pm projects doc <id> --color always \| less -R`); default `auto`, or `$PM_COLOR` |

## Reading

```sh
pm projects list [-a] [-s <status>]... [--priority <n>]... [--area <a>]...
pm projects show <project-id>
pm projects doc <project-id>
pm projects validate [<project-id> | --all]
pm tasks list [-a] [-p <project>]... [-s <status>]... [-g <tag>]... [--area <a>]... \
              [--parent <task-id>] [--blocked] [--due-before <date>] [--due-on <date>]
pm tasks show <task-id>
pm tags
```

- `tasks list` **hides `done` and `cancelled` by default** so the view is
  actionable. `-a`/`--all` includes them; an explicit `-s` overrides the default.
- Repeated values of one filter are OR'd; different filters are AND'd. So
  `-s todo -s in-progress -p web` means "(todo or in-progress) and project web".
- Results arrive **pre-ranked** — status group (in-review, in-progress, todo,
  backlog, done, cancelled), then priority (1 is highest, unset last), then file
  order. To answer "what should I work on next?", run `pm tasks list` and take
  the top row. Do not re-sort.
- `projects list` **hides `done` and `cancelled` projects by default** (`-a`
  includes them; an explicit `-s` overrides). It orders in-review projects
  first, then in-progress, then the rest — and within each group by priority,
  creation date, ID.
- `projects show` reports a `Doc`/`doc_path` pointer when the project has a
  Markdown document, but not its content. Use `projects doc <project-id>` for
  that — rendered for a terminal by default, or the raw source under `doc` in
  `--json`. Read it that way instead of opening the file yourself.
- `pm tags` lists the tag vocabulary with usage counts — check it before
  inventing a new tag.

Progress bars count tasks that can still be finished: `backlog` counts toward the
denominator, `cancelled` does not.

## Writing

```sh
pm tasks add [-p <project-id>] -t <title> [-s <status>] [--priority <n>] \
             [--due <YYYY-MM-DD>] [-g <tag>]... [--parent <task-id>] \
             [--description-file <file|->]
pm tasks edit <task-id>... [-t <title>] [--priority <n> | --clear-priority] \
             [--due <date> | --clear-due] [--add-tag <t>]... [--remove-tag <t>]... \
             [--parent <id> | --clear-parent] [--description-file <file|->]
pm tasks status <task-id>... <status> [-r <reason>]
pm tasks block <task-id>... -r <reason> [--task <blocking-task-id>]...
pm tasks unblock <task-id>...
pm tasks delete <task-id>... [--cascade]

pm projects create --id <id> -t <title> --task-id-prefix <prefix> \
             [-s <status>] [--priority <n>] [--due <date>] [--area <a>]... [--path <file>]
pm projects edit <project-id> [-t <title>] [--priority <n> | --clear-priority] \
             [--due <date> | --clear-due] [--add-area <a>]... [--remove-area <a>]...
pm projects status <project-id> <status> [-r <reason>]
pm projects format [<project-id> | --all]
```

Shorthands: `-p` project, `-t` title, `-s` status, `-r` reason, `-g` tag, `-a` all.

- `tasks add` assigns the next task ID and today's `created` date. **Default
  status is `backlog`** — pass `-s todo` for work that is actually queued.
- **`-p` is optional: without it the task goes to the `inbox` project**, created
  under the root on first use. Use this when the user hands you work that does
  not clearly belong to a project — capture it rather than guessing a project or
  stopping to ask. Say where it landed, and file it later with `tasks edit`.
- **One command changes one thing.** There is no combined "add and start"; run
  `tasks add` then `tasks status`.
- **`edit`, `status`, `block`, and `unblock` take several task IDs at once** and
  apply the same change to each: `pm tasks status web-001 web-002 done`. For
  `status` the last argument is the status; everything before it is an ID. A bad
  ID aborts before anything is written. IDs may span projects, but each file is
  written separately, so a mid-batch failure can leave earlier files changed —
  the committed IDs are named on stderr. `-t/--title` stays single-task.
- **Prefer cancelling to deleting.** `tasks status <id> cancelled -r "<why>"`
  keeps the outcome on the record; `tasks delete` destroys it and is recoverable
  only through Git. Do not delete tasks to "clean up" — cancel them. Delete is
  for records that should never have existed (a duplicate, a mistaken entry).
  It refuses when another task points at the target as a parent or blocker;
  `--cascade` removes the subtree and drops those references.
- Multi-line descriptions come from a file or stdin, never an argument:
  `printf '%s\n' "..." | pm tasks add -p web -t "Audit" --description-file -`

### Lifecycle

Statuses — tasks: `backlog`, `todo`, `in-progress`, `in-review`, `done`,
`cancelled`. Projects: `idea`, `todo`, `in-progress`, `in-review`, `blocked`,
`done`, `cancelled` (projects have no `backlog`; tasks have no `blocked` status
— see below).

`pm` owns every date; never pass or set one:

| transition | effect |
|-----------|--------|
| → `in-progress` | sets `started` **only if absent** (re-entering never overwrites it) |
| → `done` | sets `completed` |
| → `cancelled` | **requires `-r`**; records reason + date |
| → anything else | clears the terminal record it is leaving |

Blocking is **orthogonal to status**: `tasks block` records a blocker without
changing the status, so a blocked task keeps its real state. It is rejected on
`backlog`, `done`, and `cancelled` tasks. Projects differ — they have a real
`blocked` status, set via `projects status <id> blocked -r "<why>"`.

Set `--parent` to nest a task under another in the same project; parent and
blocker cycles are rejected by validation.

## Automation and errors

Pass `--json` whenever you will parse the output. Success goes to stdout; errors
go to **stderr** as a single object with a stable `code` and `message`, plus
`file`, `project`, `task`, and `field` when known:

```json
{"code":"usage","message":"no task with id \"dm-999\" under the discovery root","task":"dm-999"}
```

| exit | code | meaning | what to do |
|------|------|---------|-----------|
| 0 | — | success | continue |
| 1 | `validation` | bad task data, references, or transition | fix the data; run `pm projects validate` for the full report |
| 2 | `usage` | bad command syntax or arguments | fix the command; check `--help` |
| 3 | `io` / `internal` | I/O, locking, or unexpected failure | do not retry blindly; report it |

`pm` is strictly non-interactive — it never prompts, so it is safe in scripts and
pipelines. Every command takes what it needs from arguments, flags, or stdin.

`projects validate` reports **every** problem in one run (exit 1 if any), so run
it once and fix the whole list rather than iterating. `projects format` rewrites
files to canonical form and is idempotent; run it after bulk changes, and expect
no diff if `pm` made all the edits.

JSON field names: tasks carry `id`, `project`, `status`, `title`, and — when set
— `priority`, `parent`, `due`, `tags`, `blocked`. `projects show --json` and
`projects list --json` include `task_counts` with raw per-status counts plus
`total`, so derive any ratio you need from those rather than scraping the bar.

## Recipes

```sh
# What should I work on next?
pm tasks list                                    # top row of the top group

# Start, finish, retire
pm tasks status web-001 in-progress
pm tasks status web-001 done
pm tasks status web-004 cancelled -r "superseded by web-001"

# Queue a new piece of work (add is backlog by default)
pm tasks add -p website -t "Design the header" -s todo --priority 1 -g design

# Everything overdue, machine-readable
pm --json tasks list --due-before 2026-01-01

# Where is this project at?
pm projects show website

# Record that something is stuck, without losing its real status
pm tasks block web-002 -r "waiting on brand assets" --task web-001

# Health check across all files
pm projects validate --all
```

## The file format

You should not need it — `pm` reads and writes these files for you. If you must
understand a validation error or review a diff, read `references/format.md` for
the schema: field order, the restricted YAML profile, ID rules, and what
validation enforces.
