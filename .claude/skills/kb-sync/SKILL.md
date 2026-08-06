---
name: kb-sync
description: Mirror progress, decisions, and spec changes from this repository into the maintainer's private knowledge-base checkout (a directory of *.tasks.yaml projects and notes), where this project's task state and decision record live. Use after completing or starting a piece of work, after making a decision, when CLI behavior changes, or when committing to the knowledge-base repo.
---

# Keeping the knowledge base in sync

This project's **task state and decision record** live in the maintainer's private
knowledge base, a separate Git checkout of `*.tasks.yaml` projects and Markdown
notes. Work is not finished until it is reflected there.

If that checkout is not present, this skill does not apply — skip it.

## Locating the checkout

The path is machine-specific. Set it once and derive everything from it:

```sh
export PM_BRAIN=/path/to/knowledge-base        # the maintainer's is <home>/plaintext-brain
export PM_ROOT="$PM_BRAIN"
```

**In a Docker sandbox, do not resolve this through `~`.** The sandbox home is not
the host home; the host home is mounted at its real absolute path (for example
`/Users/<name>/…`). `ls ~` will report the knowledge base as missing when it is
in fact mounted. Locate it by its absolute path.

The project directory is `$PM_BRAIN/projects/pm-cli/`:

| file | holds | how to change it |
|------|-------|------------------|
| `pm-cli.tasks.yaml` | task state and progress | **`pm` only** — never hand-edit |
| `pm-cli.md` | decisions (Decisions section), open questions, resources | edit directly |
| `pm-cli.notes.md` | live, provisional, or unresolved observations — *not* task state | edit directly |

## Progress: use `pm`, not an editor

The task file is a `*.tasks.yaml` file, so the `pm-cli` skill's one rule
applies — hand-editing bypasses validation, breaks canonical formatting, and
corrupts ID allocation. Dogfood the tool:

```sh
pm tasks add -p pm-cli -t "<what the work is>"
pm tasks status pt-0NN in-progress          # when starting
pm tasks status pt-0NN done                 # when finished
pm tasks status pt-0NN cancelled -r "<why>" # when superseded — reason required
```

Record work that got **superseded** rather than deleting or retitling it: cancel
the old task with a reason naming the replacement, and add a new one for what
actually shipped. That keeps the reversal legible instead of rewriting history.

## Decisions and spec changes

- A real decision goes in the Decisions section of `pm-cli.md`, as one
  sentence stating what was decided and why.
- **The normative specs are `docs/spec/` in this repository**, not in the
  knowledge base. If a change alters documented CLI behavior, amend
  `docs/spec/cli-specification.md` in the same pass — letting code and spec
  diverge is worse than editing a frozen file. The user asking for the behavior
  change is the authorization, but say plainly in your report that you touched a
  frozen spec. If the knowledge base carries a mirror copy, update it too.
- Provisional or unresolved things go in `pm-cli.notes.md`, not into task
  state and not into Decisions.

## Committing — a different style from this repo

The knowledge base is the **direct-to-main lane**: commit and push straight to
`main`, no branch, no PR, no need to ask. But the commit style is *not* this
repo's:

- **Plain-language, sentence-case, past-tense titles stating the outcome.**
- **No conventional-commit type or scope prefixes** — that style is the CLI repository only.

```
Excluded cancelled tasks from the project progress bar instead of backlog ones
Added a portable agent skill for the pm CLI
```

## Push safely

```sh
cd "$PM_BRAIN"
git add <only your intended paths>       # never -A, never `git commit -a`
git status --short                       # confirm nothing unrelated is staged
git fetch -q origin
git merge-base --is-ancestor origin/main HEAD   # must pass before pushing
git commit -m "..." -m "..."
git push origin main
```

**Preserve unrelated dirty files.** The checkout routinely carries edits that are
not yours — other projects' task files and the maintainer's own notes. Leave them
unstaged and never bundle them into a commit.

When your change and the user's share a file you cannot split (both editing
`pm-cli.tasks.yaml`, say), it is fine to carry theirs along — but note it
in the commit body and in your report to the user, rather than doing it silently.

Never force-push. Report the commit hash and push result when done.

## Reporting

State what you changed in the knowledge base alongside what you changed in the
code — the user tracks both. Name the task IDs you touched and their new status.
