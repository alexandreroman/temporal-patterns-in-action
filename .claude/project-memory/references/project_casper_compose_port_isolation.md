---
name: "Casper compose/dev port isolation"
description: "compose-override target generates the gitignored compose.override.yaml from CASPER_PORT; infra-up, app-up and bootstrap all depend on it so make dev launches infra on matching ports."
type: project
---

# Casper compose/dev port isolation

Casper isolates each worktree's host ports via
`$CASPER_PORT` (offset scheme identical to cmux —
see [[project_cmux_compose_port_isolation]]:
frontend `+0`, temporal gRPC `+1`, temporal UI
`+2`, nats `+3`, codec `+4`). `.casper.json` maps
the lifecycle: `setup`→`make bootstrap`,
`run`→`make app-up`, `dev`→`make dev`,
`stop`/`teardown`→`make app-down`/`make teardown`.

Two independent mechanisms must agree:

- **Container host ports** come from the gitignored
  `compose.override.yaml` (uses `!override` to fully
  replace each `ports:` list).
- **Host-run dev processes** (`make dev`): the
  Makefile's top `ifneq ($(CASPER_PORT),)` block
  exports `PORT`, `TEMPORAL_ADDRESS=localhost:+1`,
  `NATS_URL=nats://localhost:+3` for the frontend
  and workers.

The Makefile has a dedicated `compose-override`
target that (re)generates `compose.override.yaml`
from `$CASPER_PORT` and no-ops when it is unset or
non-numeric (so the main worktree stays on stock
3000/7233/8233 ports). `bootstrap`, `infra-up`, and
`app-up` all declare it as a prerequisite. `make dev`
inherits it through its existing `infra-up` dep.

**Why:** originally the override was generated only
inside the `bootstrap` recipe, so `make dev` (which
only runs `infra-up`) brought infra up on the
default 7233/4222 while the exported dev-process env
pointed workers at `CASPER_PORT+1/+3` — they never
connected, and in a second worktree the default
ports collided with the main worktree's containers
so infra failed to start entirely. Making the
override a shared prerequisite keeps the container
ports and the dev-process targets consistent on
every launch path.

**How to apply:** never regenerate the override in
the main worktree (`compose.override.yaml` is
gitignored and must not be committed). When adding a
pattern that publishes a new host port, extend the
offset scheme in the `compose-override` target.
Keep the codec-server `UI_ORIGIN` override pointing
at `CASPER_PORT+2` or the codec `/decode` demo
breaks — see [[project_codec_server_ui_endpoint]].
