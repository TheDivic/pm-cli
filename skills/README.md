# Agent skills

Portable skills that teach an LLM agent to drive `pm` without reading this
repository. Hand an agent the skill directory and it has the full command
surface, the lifecycle rules, and the file schema.

## `pm-cli`

```
skills/pm-cli/
  SKILL.md               command surface, lifecycle rules, JSON/exit-code contract
  references/format.md   the *.tasks.yaml schema, read on demand
```

`SKILL.md` is self-contained: an agent that loads it needs neither this
repository nor its specifications to use `pm` correctly.

### Install

The skill assumes `pm` is already on `PATH` (`make build`, then put `bin/pm`
somewhere on your path).

**Claude Code** — symlink so the skill tracks this repository:

```sh
# available in every project
ln -s "$PWD/skills/pm-cli" ~/.claude/skills/pm-cli

# or scoped to one project
ln -s /path/to/pm-cli/skills/pm-cli .claude/skills/pm-cli
```

Copy the directory instead of symlinking if the agent runs somewhere this
checkout is not present.

**Other agents** — most harnesses either read a Markdown instruction file
directly or accept a directory of them; point yours at `SKILL.md`. The YAML
frontmatter (`name`, `description`) is what Claude Code uses to decide when to
load the skill, and is harmless to a tool that ignores it.

### Keeping it accurate

The skill states `pm`'s flags and behavior verbatim. When a command's surface
changes, update `SKILL.md` in the same change — a skill that lies about the CLI
is worse than none, because the agent trusts it instead of checking.
