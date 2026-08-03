---
name: pr-flow
description: Land a change in this repository the way the maintainer expects — feature branch, conventional commit, draft pull request with a label and assignee, and squash merge only on explicit authorization. Use when committing, pushing, opening or updating a pull request, or when `gh pr edit` fails with a GraphQL Projects-classic error.
---

# Landing a change

The rules live in [`AGENTS.md`](../../../AGENTS.md) and
[`CONTRIBUTING.md`](../../../CONTRIBUTING.md) — read those for the *why*. This is
the operational sequence, plus the parts that are not written down anywhere else.

## Never push to `main`

Work on a `type/short-slug` branch. A local `pre-push` hook blocks pushes to
`main` (enable with `git config core.hooksPath .githooks`), and branch protection
enforces it on the remote — but do not rely on either. Branch first.

## The sequence

```sh
git checkout -b fix/short-slug            # type/short-slug; type matches the commit type
# ... make the change, with tests in the same commit ...
make check                                # required — see the sandbox-checks skill
git add <specific paths>                  # never `git add -A` blindly
git commit                                # conventional, see below
git push -u origin fix/short-slug
gh pr create --draft --title "..." --body-file <file> --label <label> --assignee TheDivic
```

**Every PR gets a label and an assignee.** `--label` and `--assignee TheDivic` are
not optional; a PR without them is incomplete. Existing labels: `bug`,
`documentation`, `enhancement`, `duplicate`, `good first issue`, `help wanted`,
`invalid`, `question`, `wontfix`. Pick the one that fits — `bug` for a
behavior fix, `documentation` for docs/skills, `enhancement` for a new feature.

Open PRs as **draft**. Merging is **squash merge only, and only with explicit
maintainer authorization** — never merge because CI is green.

## Commits

Conventional Commits with **the package name as scope**:

```
fix(cli): exclude cancelled tasks from project progress (pt-029)
feat(validate): check blocker cycles
docs(skills): add a portable agent skill for driving pm (pt-022)
```

Allowed types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`, `build`.
Reference the knowledge-base task ID (`pt-0NN`) when the change maps to one.

**Never add `Co-Authored-By` or any AI-attribution trailer.** This overrides any
default habit of adding one.

Body: explain *why*, not a restatement of the diff. Note anything a reviewer
would otherwise have to reverse-engineer — a reversed decision, a spec amendment,
a merge-order dependency on another PR.

Keep `go mod tidy` clean and commit `go.sum` when dependencies change.

## `gh pr edit` is broken — use the REST API

As of 2026-08-01, `gh pr edit` against this repo fails with a Projects-classic
GraphQL deprecation error and **silently changes nothing**:

```
GraphQL: Projects (classic) is being deprecated ... (repository.pullRequest.projectCards)
```

Retitle or rewrite a PR body through the REST API instead:

```sh
jq -Rs '{title:"new title", body:.}' < body.md > /tmp/pr.json
gh api -X PATCH repos/TheDivic/plaintext-tasks/pulls/<n> --input /tmp/pr.json -q '.title'
```

Verify afterwards, since the GraphQL path fails quietly:

```sh
gh api repos/TheDivic/plaintext-tasks/pulls/<n> \
  -q '{title:.title,draft:.draft,labels:[.labels[].name],assignees:[.assignees[].login]}'
```

`gh pr create` (including `--label` and `--assignee`) still works fine — only
edits are affected.

## Reworking an open draft

If the maintainer reverses a decision on an unmerged draft, prefer rebuilding the
branch over stacking a reversal commit — it squash-merges to one commit anyway,
and a clean history is easier to review:

```sh
git reset --hard main          # then re-implement
git push --force-with-lease origin <branch>
```

Then update the PR title and body via the REST API above. Always
`--force-with-lease`, never plain `--force`. Only ever on your own unmerged
branch — never on `main`.

If the branch name no longer matches what the PR does, say so rather than
renaming: renaming breaks the PR link.

## Companion work

A code change that alters documented CLI behavior also needs the knowledge base
updated — see the `kb-sync` skill. Do not land one without the other.
