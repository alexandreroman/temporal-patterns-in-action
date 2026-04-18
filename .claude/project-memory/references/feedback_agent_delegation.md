---
name: "Agent and skill delegation"
description: "Required subagents (code-writer, code-reviewer) and the skill-temporal-developer skill for code, review, and Temporal-specific tasks."
type: feedback
---

# Agent and skill delegation

Use the [skillbox](https://github.com/alexandreroman/skillbox)
plugin's agents for all code work in this repo:

- **code-writer** — for ANY task that writes,
  modifies, or refactors code. Includes
  one-line fixes, import changes, visibility
  tweaks, and adding assertions. Never use
  `Edit` or `Write` directly on source files —
  always delegate.
- **code-reviewer** — for read-only review
  before merging or when investigating issues.

When touching Temporal workflow or activity
code, also load the **skill-temporal-developer**
skill to consult language-specific references
(determinism, patterns, testing, versioning).

**Why:** The user mandates delegation so that
code changes go through a specialist with the
right conventions loaded.

**How to apply:** Before editing any Go, TS,
Vue, or related source file, dispatch to
code-writer with a self-contained brief.
Configuration-as-code (Dockerfiles, compose,
Makefiles) also goes through code-writer.
CLAUDE.md and memory files are instruction
data, not code — those may be edited directly.
For Temporal workflow/activity changes, load
the skill in parallel with the agent call.
