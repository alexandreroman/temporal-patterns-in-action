---
name: "Casper info panel mirrors make endpoints"
description: "make endpoints prints the stack's addresses as Markdown; guarded defines mirror it into the workspace info panel, and every target that raises or drops those addresses updates it."
type: feedback
---

# Casper info panel mirrors make endpoints

`make endpoints` prints this worktree's published
addresses (App, Temporal Web UI, Codec Server) as
Markdown on stdout, built from `FRONTEND_PORT`,
`TEMPORAL_UI_PORT` and `CODEC_SERVER_PORT` — the same
`CASPER_PORT` offsets the `compose-override` target
uses, see [[project_casper_compose_port_isolation]].

The Makefile's only Casper-aware part is the guarded
pair that publishes and clears the workspace info
panel from that output:

- `in-casper-workspace` tests both
  `$CASPER_WORKSPACE_ID` and the presence of the
  `casper` CLI (the CLI is on PATH outside a workspace
  too), and `|| true` keeps a panel update from ever
  failing the target that asked for it.
- `publish-endpoints` runs in `app-up` and `dev`,
  before the command that takes over the terminal.
- `clear-endpoints` runs in `app-down`, `infra-down`
  (it stops the Temporal server, so the Web UI address
  the panel advertises stops answering) and `teardown`.
- Targets in between — `infra-up`, `compose-override`,
  `bootstrap`, `app-logs` — touch neither the panel nor
  `endpoints`.

**Why:** the demo has to read as an ordinary Temporal
demo to someone who has never heard of the workspace
tool, and the panel is worth trusting only if it tells
the truth whatever command was typed. The panel is a
display surface, not storage: it holds one Markdown
message, is replaced by each `casper info set`, and is
lost when Casper restarts — so nothing may live only
there.

**How to apply:** workspace-specific wiring goes in
`.casper.json`; the repository's own files stay
tool-agnostic apart from those guarded lines. A new
target that brings the published endpoints up or takes
them down publishes or clears the panel too, and a
pattern that publishes a new host port gets a row in
`endpoints` alongside its entry in the offset scheme.
