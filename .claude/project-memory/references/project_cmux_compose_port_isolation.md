---
name: "cmux compose port isolation"
description: "cmux worktrees reuse the Casper port scheme: the Makefile defaults CASPER_PORT to CMUX_PORT, and post-create.sh runs make bootstrap."
type: project
---

# cmux compose port isolation

The root Makefile sets `CASPER_PORT ?= $(CMUX_PORT)`, so every
target in a cmux worktree (`bootstrap` from `.cmux/post-create.sh`,
then `infra-up`, `app-up`, `dev`, `endpoints`) uses the same
`compose.override.yaml` and offset scheme as Casper
([[project_casper_compose_port_isolation]]). `.cmux/pre-destroy.sh`
stops the worktree's stack and any host-side dev processes.

**Why:** one generator for the override file keeps the two
workspace tools from drifting apart.

**How to apply:** port changes go in the Makefile's
`compose-override` target only; never duplicate them in `.cmux/`.
