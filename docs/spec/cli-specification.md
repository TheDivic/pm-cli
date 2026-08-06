# PM CLI specification

PM CLI reads, validates, queries, formats, and updates `*.tasks.yaml` files. People use concise terminal output; LLM agents and repository automation can request JSON.

## Entry point

Install and invoke the standalone executable directly:

```text
pm [--root <path>] [--json] [--no-ignore] <resource> <command> [arguments]
```

`--root` selects the discovery root explicitly. Without it, `pm` uses the `PM_ROOT` environment variable when set, and otherwise the current working directory. The precedence is `--root`, then `PM_ROOT`, then the working directory. The executable does not depend on a repository-specific launcher, a particular knowledge base, or an agent runtime.

## Discovery

Walk the discovery root recursively and treat every file ending in `.tasks.yaml` as a project task file. This supports invocation from an individual project directory, a directory containing several projects, or a higher parent directory.

Respect `.gitignore` files found at the selected root and below it. Apply nested rules with Git ignore semantics, including negated patterns and directory patterns. Exclude matching files and directories before task-file discovery. Ignore rules above the selected root do not apply.

The global `--no-ignore` option disables `.gitignore` filtering. It does not allow traversal into `.git` or following directory symlinks. Discovery implements ignore matching without invoking or requiring the Git executable.

Always skip `.git` and do not follow directory symlinks. Sort discovered projects by project ID so repeated commands produce stable output.

Each discovered file declares `project.id`, `project.title`, and `project.task-id-prefix`. Project IDs and task-ID prefixes must be unique within the discovery root. Duplicate values stop the command and report every conflicting path.

Commands accept a project ID or a globally unique task ID. A task ID resolves through its prefix without requiring the caller to supply a project path.

## Commands

The version 1 interface provides these command groups:

```text
pm projects list [-a|--all] [filters]
pm projects show <project-id>
pm projects doc <project-id>
pm projects create --id <project-id> --title <text> --task-id-prefix <prefix> [options]
pm projects edit <project-id> [options]
pm projects status <project-id> <status> [--reason <sentence>]
pm projects validate [<project-id> | --all]
pm projects format [<project-id> | --all]

pm tasks list [filters]
pm tasks show <task-id>
pm tasks add [--project <project-id>] --title <text> [options]
pm tasks edit <task-id>... [options]
pm tasks delete <task-id>... [--cascade]
pm tasks status <task-id>... <status> [--reason <sentence>]
pm tasks block <task-id>... --reason <sentence> [--task <task-id> ...]
pm tasks unblock <task-id>...

pm tags
pm completion <bash|zsh|fish|powershell>
```

`completion` writes a shell completion script to standard output. Completion is dynamic: task IDs, project IDs, tags, and area slugs are resolved from the discovery root at the moment of the keystroke, and statuses come from the lifecycle vocabulary for the resource being completed. `tasks status` completes task IDs in every position and adds the task statuses from the second argument onward, matching its `<task-id>... <status>` grammar. IDs already present on the command line are not offered again. A completion request never reports an error or writes diagnostics: an unreadable or invalid file yields no suggestions rather than noise in the user's prompt.

`tags` lists the distinct tags in use across all discovered tasks with a per-tag usage count, most-used first, in human-readable and JSON form. It counts every task, including terminal ones, so it reflects the full tag vocabulary.

`projects list` supports `-a`/`--all`, `--status`, `--priority`, and `--area`. By default it shows only projects that are not finished (`idea`, `todo`, `in-progress`, `in-review`, `blocked`), because completed and abandoned projects accumulate without bound and would bury the work in flight. The `-a`/`--all` flag includes `done` and `cancelled` projects, and an explicit `--status` filter overrides the default — the same rule `tasks list` follows. By default it lists in-review projects first, then in-progress projects (in-review leads because it is closer to completion, matching the task list's grouping), then all other projects. Within each group it sorts positive integer priorities from lowest to highest, puts projects without a priority last, and breaks ties by creation date (oldest first) and then project ID. Human-readable output lists, in column order, the project ID, title, status, creation date, and a compact completion progress bar; priority is not shown as a column since it already governs the sort order, but it is still reported by `projects show`. In JSON mode each project includes per-status task counts. The task-file path is not shown in the list; `projects show` reports it in project details.

`projects show` reports a project's stored fields, its task-file path, and a task summary: a completion progress bar (tasks done out of the countable total, with a percentage) and a per-status breakdown in lifecycle order so project progress is visible. In JSON mode the same information is available as task counts. It does not render the project's Markdown document; `projects doc` does. When the project has a document, which by convention is `<project-id>.md` beside the task file, `show` reports its path as `Doc` in human-readable output and `doc_path` in JSON, so a caller knows it exists without reading it. A project without a document reports neither.

`projects doc <project-id>` reports the project's Markdown document in full. Human-readable output renders it for the terminal: headings, list bullets and task checkboxes, block quotes, thematic breaks, tables, and fenced code blocks (with syntax highlighting) are formatted, emphasis, code span, and link markup is consumed rather than printed, and prose wraps to the terminal width with a hanging indent instead of running to the edge unbroken. Rendering uses ANSI styling only when writing to a terminal without `NO_COLOR`; otherwise the same layout is rendered in plain text with the styling removed. Wrap width tracks the terminal's column width, within a readable minimum and maximum, and falls back to a fixed width for non-terminal output. JSON mode carries `project_id`, `doc_path`, and the document's Markdown source in `doc`, unrendered, since a consumer that wants Markdown wants the source. A project without a document, or an unknown project ID, is a usage error.

Completion progress measures the work that can still be finished. The denominator is the countable total: every task except those in `cancelled`. Backlog tasks count, because every task is expected to reach either `done` or `cancelled` and unfinished work is unfinished wherever it sits. Cancelled tasks do not, because work that will never be finished would otherwise hold a project below 100% permanently. Human-readable output states the excluded cancelled count alongside the ratio, and a project whose tasks are all cancelled reports that nothing is countable rather than a zero-percent bar. JSON output carries the raw per-status counts and the cancellation-inclusive `total`, so any other ratio can be derived from it.

`projects create` accepts optional `--path`, `--status`, `--priority`, `--due`, and repeated `--area` flags. Its default status is `idea`, and its default path is `<root>/<project-id>/<project-id>.tasks.yaml`. It creates the target directory when needed but refuses to replace an existing task file.

`projects edit` can replace the title, set or clear the priority and due date, and add or remove areas. It does not change the project ID, task-ID prefix, status, or lifecycle dates.

`projects status` applies a valid lifecycle transition and manages its dates. Entering `in-progress` sets `started` only when absent. Entering `done` sets `completed`. Entering `blocked` or `cancelled` requires `--reason` and records the corresponding date. Leaving `blocked` removes the blocking record.

`tasks list` supports `--project`, `--status`, `--area`, `--tag`, `--parent`, `--blocked`, `--due-before`, and `--due-on`. Multiple filters combine with logical AND. Repeated values for the same filter combine with logical OR. A `--project` value that matches no discovered project is a usage error, not an empty result, so a mistyped ID cannot be read as a project with no open tasks.

By default `tasks list` shows only open tasks (`backlog`, `todo`, `in-progress`, `in-review`) so the overview focuses on actionable work. The `-a`/`--all` flag includes `done` and `cancelled` tasks, and an explicit `--status` filter overrides the default.

Task list output is grouped by status for display: in-review first, then in-progress (in-review leads because it is closer to completion), then todo, backlog, done, and cancelled. Within each status group it sorts by task priority (lowest number first, tasks without a priority last), breaking ties by file order so the first ready task in a status group is next. Human-readable output shows the project ID and a priority column; JSON identifies the project by ID and includes the priority and tags.

`tasks add` accepts optional `--description-file`, `--status`, `--priority`, `--parent`, `--due`, and repeated `--tag` flags. Its default status is `backlog`, so newly captured work starts speculative until accepted into scope. The CLI assigns `created` and the next task ID. `--description-file -` reads the description from standard input.

`--project` is optional. Without it the task goes to the inbox: the project whose ID is `inbox`. When no such project exists under the discovery root, `tasks add` creates it at `<root>/inbox/inbox.tasks.yaml` with the title `Inbox`, the task-ID prefix `in`, and status `in-progress`, then adds the task. Capture must never be blocked on deciding where work belongs; filing it afterwards is an ordinary edit. Creating the inbox is reported as a note on standard error, never as a second record on standard output, so a command still emits exactly one result. If another project already holds the `inbox` ID or the `in` prefix, the existing project is used or the collision is reported, and no file is written.

Common flags provide single-letter shorthands: `-p` (`--project`), `-t` (`--title`), `-s` (`--status`), `-r` (`--reason`), `-g` (`--tag`), and `-a` (`--all`).

`tasks edit` can replace the title or description, set or clear the priority and due date, add or remove tags, and assign or clear a parent. It does not change status or lifecycle dates. It also accepts `--description-file -` for standard input.

`tasks status` applies a valid lifecycle transition and manages its dates. Entering `in-progress` sets `started` only when absent. Entering `done` sets `completed`. Entering `cancelled` requires `--reason` and records the cancellation date.

`tasks block` records a sentence, date, and optional repeated task references without changing the task status. `tasks unblock` removes the complete blocking record.

### Batch task mutations

`tasks edit`, `tasks status`, `tasks block`, and `tasks unblock` accept more than one task ID and apply the same change to each. For `tasks status` the final argument is the target status and every argument before it is a task ID, so the single-task form is unchanged. A repeated ID is applied once. `tasks edit --title` is restricted to a single task, because a title names one specific outcome.

Every ID is resolved before anything is written, so an unresolvable ID fails the command with no file modified. Tasks are then grouped by file and each file passes through the mutation envelope once: within a file the change is all-or-nothing, and one failing task leaves that entire file unchanged. Across files it is not atomic, because each file is its own atomic write; when a later file fails, the tasks already committed are named on standard error.

Human-readable output prints one line per changed task. In JSON mode a single task keeps the flat mutation result, and two or more return a batch object with `kind`, `status` when applicable, `count`, and a `tasks` array of `id` and `path`.

`tasks delete` removes tasks from the file. It is the one mutation that destroys history rather than recording an outcome, so cancelling remains the way to retire work whose outcome should stay legible; Git is the only recovery path for a delete.

A delete is refused, with nothing written, when a task outside the deletion points at one inside it — as a `parent` or in a `blocked.tasks` list. The refusal names every referring task and the relationship. `--cascade` widens the deletion to every descendant of a named task and strips the deleted IDs from the blocker lists of the tasks that remain; a blocking record whose list empties keeps its reason and date. Naming every task in a subtree explicitly deletes it without `--cascade`, because no dangling reference is left behind.

Delete accepts several task IDs and follows the batch rules above. Its output reports every task actually deleted, which under `--cascade` includes descendants the caller did not name.

## Dates

Commands that create lifecycle events use the current calendar date from the process's local timezone. Tests inject a clock and must not depend on the machine date.

`started` is set only when absent. Returning a task to `in-progress` does not replace it. `completed`, `cancellation.date`, and `blocked.since` record their corresponding current events.

## Output

Human-readable output is concise and stable enough for interactive use. The global `--json` option applies to every command and returns schema-defined values rather than formatted YAML. Relationships use project and task IDs.

Successful mutation output includes the changed project or task ID, resulting status when applicable, and task-file path. Validation output groups errors by file and returns all independent findings in one run.

In JSON mode, successful results go to standard output and errors go to standard error. Each error contains a stable code, message, file path, and the project ID, task ID, and field path when available.

All commands are non-interactive. When required information is missing, `pm` returns a usage error instead of opening a prompt or requiring a terminal session.

Do not use ANSI color when output is not a terminal or when `NO_COLOR` is set.

## Mutation safety

Every mutation follows this sequence:

1. Resolve and lock the exact target task file.
2. Read and parse the current bytes.
3. Validate the complete document and required repository references.
4. Apply one requested mutation in memory.
5. Validate the result.
6. Render the complete document in canonical form.
7. Write a temporary file in the target directory, flush it, and atomically rename it over the original.
8. Release the lock.

Any error before the rename leaves the original bytes unchanged. The tool must not follow a task-file symlink outside the selected task root.

Locking protects commands in one checkout. Git remains responsible for changes made in separate clones, branches, or remote sessions.

## Parsing and formatting

Use a YAML parser that exposes enough syntax information to reject duplicate keys, aliases, anchors, merge keys, explicit tags, and unsupported scalar forms. Decode into a typed model only after syntax-level checks pass.

The formatter writes the field order, indentation, blank lines, date quoting, description blocks, and tree preorder defined by the format specification. Running `format` twice without an intervening edit produces identical bytes.

`format` rejects invalid semantics instead of guessing repairs. ID assignment is part of `add`, not an implicit formatting side effect.

## Validation and exit behavior

Use these process exit codes:

- `0`: command succeeded and validation found no errors.
- `1`: task data, references, or a requested transition is invalid.
- `2`: command syntax or arguments are invalid.
- `3`: an I/O, locking, or unexpected internal error occurred.

Write normal output to standard output and diagnostics to standard error. Error messages include the affected path and field or task ID when available. Do not print stack traces unless an explicit debug option is active.

## Tests

Automated tests cover:

- valid minimal and complete files;
- every invalid field, status, date, and conditional requirement;
- recursive discovery from project and parent directories;
- root and nested `.gitignore` rules, including negated patterns and directory patterns;
- ignored directories and ignored task files;
- discovery with `--no-ignore` and without the Git executable;
- duplicate project IDs and task-ID prefixes across discovered files;
- duplicate IDs, prefixes, fields, tags, and references;
- missing areas, parents, and blocker tasks;
- parent and blocker cycles;
- ID allocation with gaps and more than three digits;
- lifecycle transitions and immutable `started` dates;
- project creation, metadata edits, and lifecycle transitions;
- project priority filtering and ordering;
- task priority ordering and validation;
- task ordering within status and parent groups;
- multiline Markdown descriptions;
- canonical formatting and idempotence;
- filtering and human-readable output;
- JSON success and error output for every command;
- operation without a terminal or interactive prompt;
- lock contention, interrupted writes, and unchanged files after failure; and
- root-boundary and symlink checks.

Read-only integration and acceptance checks may use real project task files as input. Formatting and mutation tests run against copies in a temporary discovery root and must not change live project files.

## Acceptance

The CLI is ready for user review when every task file under the intended discovery root validates, automated tests pass, documentation matches the shipped interface, and a fresh agent can complete common task operations without editing YAML. Delivery follows the executable pull-request lane in the repository Git workflow.
