#!/usr/bin/env bash
#
# Seeds a throwaway workspace for the README demo recording and prints its path.
#
# The demo must never touch real task files, so everything lives in a fresh
# temporary directory that the recording cds into. Run it through `make demo`,
# which puts the freshly built ./bin/pm on PATH first.
#
# Usage: root=$(scripts/demo-setup.sh)

set -euo pipefail

pm="${PM:-pm}"
command -v "$pm" >/dev/null 2>&1 || {
	echo "demo-setup: '$pm' is not on PATH; run 'make demo' or 'make build' first" >&2
	exit 1
}

root="$(mktemp -d "${TMPDIR:-/tmp}/pm-demo.XXXXXX")"
run() { "$pm" --root "$root" "$@" >/dev/null; }

# Two projects with contrasting shapes: one mid-flight with a mixed task board,
# one still an idea. Enough to show ordering and progress without a wall of rows.
run projects create --id website --title "Website relaunch" \
	--task-id-prefix web --status in-progress --priority 1
run projects create --id newsletter --title "Monthly newsletter" \
	--task-id-prefix news --status idea --priority 2

# Tasks are created and then transitioned rather than created in a terminal
# status, because pm owns the lifecycle dates that those statuses require.
run tasks add -p website -t "Audit the current information architecture" -s todo
run tasks add -p website -t "Design the new header" -s todo --priority 1 -g design
run tasks add -p website -t "Rewrite the landing copy" -s todo --priority 2 -g copy
run tasks add -p website -t "Migrate the blog archive" -s todo -g migration
run tasks add -p website -t "Dark mode" -g design

run tasks add -p newsletter -t "Pick a sending platform" -s todo --priority 1
run tasks add -p newsletter -t "Draft the first issue" -g copy

run tasks status web-001 done
run tasks status web-002 in-progress

# Blocking is orthogonal to status: the task keeps its real state.
run tasks block web-003 -r "waiting on the brand guidelines" --task web-002

# A project document, so `projects show` has something to render.
cat >"$root/website/website.md" <<'EOF'
# Website relaunch

Rebuild the marketing site on the new design system, without a content freeze.

## Decisions

- Ship page by page; the old site stays live until the last page moves.
- **Copy is the long pole** — design can iterate, copy needs sign-off.

## Success

- [x] Information architecture agreed
- [ ] Every page passes an accessibility audit
EOF

echo "$root"
