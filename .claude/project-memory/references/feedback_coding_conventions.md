---
name: "Coding conventions"
description: "Line length limits, markdown style, LTS/stable dependency rule, the no-compound-shell-commands Bash rule, and the docker-compose hyphenated form for committed files."
type: feedback
---

# Coding conventions

- **Line length:** Markdown/text ≤80 columns,
  code ≤120 columns.
- **Markdown:** blank line before and after
  headings and lists; fenced code blocks must
  carry a language tag.
- **Dependency versions:** always pick the
  latest LTS or stable release. Verify against
  official docs or via context7 before choosing
  a version.
- **No compound shell commands in Bash calls:**
  never chain `&&`, `;`, or `|` inside a single
  Bash tool call. One command per call. Common
  violations to watch for: `cd dir && make run`,
  `curl ... | grep ...`,
  `go test 2>&1 | tail -20`.
- **Compose invocation in committed files:** use
  `docker-compose` (hyphenated, standalone
  binary) — never `docker compose` (space, v2
  plugin form). Reason: the hyphenated form
  works with both Docker and `podman-compose`,
  so contributors running Podman keep the
  Makefile/README examples usable. Applies to
  Makefile targets, README snippets, CI jobs,
  and any committed script. Complements the
  user-level auto-memory rule (substitute
  `podman` for `docker` in commands I *run or
  suggest* on this machine, but leave committed
  docs alone).

**Why:** The user restates these rules often —
especially the compound-command one — so they
are load-bearing preferences rather than style
suggestions.

**How to apply:** Apply to every file written
and every Bash call made in this repo. Split
pipelines and chains into separate Bash calls
and correlate results in the conversation
rather than relying on shell composition.
