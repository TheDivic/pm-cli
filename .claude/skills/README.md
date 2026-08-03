# Project-scoped skills

Skills for agents **working in this repository**. They load automatically for
anyone who opens this project in Claude Code, and they are committed so the
knowledge travels with the repo instead of living in one person's head.

| skill | covers |
|-------|--------|
| `sandbox-checks` | running `make check` and building `pm` correctly in a Docker sandbox |
| `pr-flow` | branch, commit, and pull-request conventions for landing a change |
| `kb-sync` | mirroring progress, decisions, and spec changes into the Plaintext Brain knowledge base |

Each exists because the knowledge was rediscovered the hard way — a `make check`
that fails for environment reasons, a `gh` subcommand that fails silently, a
knowledge base with a different commit style from this repo. If you hit something
non-obvious and work it out, add it here.

## Not to be confused with `/skills`

Two directories, two audiences:

- **`.claude/skills/`** (here) — how to *work on* `pm`. Internal, loads only in
  this project.
- **`/skills`** (repo root) — how to *use* `pm`. A portable skill handed to any
  LLM agent that needs to drive the CLI against its own task files, with no
  knowledge of this repository.

A change to the CLI's surface may need updating in both.

## Plugins

`settings.json` enables the [`gopls`](https://github.com/Piebald-AI/claude-code-lsps)
language-server plugin at project scope and declares the marketplace it comes
from, so a fresh clone needs no manual setup. It does need the binary:

```sh
go install golang.org/x/tools/gopls@latest      # then ensure $(go env GOPATH)/bin is on PATH
```

`gopls` gives real code intelligence — go-to-definition, find-references, and
module-wide rename — instead of grepping for symbols.
