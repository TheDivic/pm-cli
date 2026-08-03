---
name: kb-sync
description: Mirror progress, decisions, and spec changes from this repository into the Plaintext Brain knowledge base at ~/plaintext-brain, which is the source of truth for the plaintext-projects project. Use after completing or starting a piece of work, after making a decision, when CLI behavior changes, or when committing to the knowledge base repo.
---

# Keeping the knowledge base in sync

The knowledge base is the **primary source of truth** for this project's progress
and specification. The code repo implements it; it does not define it. Work is not
finished until the KB reflects it.

Location: `/Users/divic/plaintext-brain/projects/plaintext-projects/`

| file | holds | how to change it |
|------|-------|------------------|
| `plaintext-projects.tasks.yaml` | task state and progress | **`pm` only** — never hand-edit |
| `plaintext-projects.md` | decisions (Decisions section), open questions, resources | edit directly |
| `plaintext-projects.notes.md` | live, provisional, or unresolved observations — *not* task state | edit directly |
| `project-task-format.md` | **frozen** normative schema | only on explicit authorization |
| `cli-specification.md` | **frozen** normative CLI contract | only on explicit authorization |

## Progress: use `pm`, not an editor

The task file is a `*.tasks.yaml` file, so the `plaintext-projects` skill's one rule
applies — hand-editing bypasses validation, breaks canonical formatting, and
corrupts ID allocation. Dogfood the tool:

```sh
export PM_ROOT=/Users/divic/plaintext-brain
pm tasks add -p plaintext-projects -t "<what the work is>"
pm tasks status pt-0NN in-progress          # when starting
pm tasks status pt-0NN done                 # when finished
pm tasks status pt-0NN cancelled -r "<why>" # when superseded — reason required
```

Record work that got **superseded** rather than deleting or retitling it: cancel
the old task with a reason naming the replacement, and add a new one for what
actually shipped. That keeps the reversal legible instead of rewriting history.

## Decisions and spec changes

- A real decision goes in the Decisions section of `plaintext-projects.md`, as one
  sentence stating what was decided and why.
- **If a change alters documented CLI behavior, the frozen
  `cli-specification.md` must be amended in the same pass.** Letting code and spec
  diverge is worse than editing a frozen file. The user asking for the behavior
  change is the authorization — but say plainly in your report that you touched a
  frozen spec.
- Provisional or unresolved things go in `plaintext-projects.notes.md`, not into task
  state and not into Decisions.

## Committing — a different style from this repo

The KB is the **direct-to-main lane**: commit and push straight to `main`, no
branch, no PR, no need to ask. But the commit style is *not* this repo's:

- **Plain-language, sentence-case, past-tense titles stating the outcome.**
- **No conventional-commit type or scope prefixes** — that style is the CLI repository only.

```
Excluded cancelled tasks from the project progress bar instead of backlog ones
Added a portable agent skill for the pm CLI
```

Full workflow reference: `projects/plaintext-brain/git-workflow.md`.

## Push safely

```sh
cd /Users/divic/plaintext-brain
git add <only your intended paths>       # never -A, never `git commit -a`
git status --short                       # confirm nothing unrelated is staged
git fetch -q origin
git merge-base --is-ancestor origin/main HEAD   # must pass before pushing
git commit -m "..." -m "..."
git push origin main
```

**Preserve unrelated dirty files.** The checkout routinely carries edits that are
not yours — `projects/plaintext-brain/plaintext-brain.notes.md` in particular is
the user's; leave it unstaged and never bundle it into a commit.

When your change and the user's share a file you cannot split (both editing
`plaintext-projects.tasks.yaml`, say), it is fine to carry theirs along — but note it
in the commit body and in your report to the user, rather than doing it silently.

Never force-push. Report the commit hash and push result when done.

## Reporting

State what you changed in the KB alongside what you changed in the code — the user
tracks both. Name the task IDs you touched and their new status.
