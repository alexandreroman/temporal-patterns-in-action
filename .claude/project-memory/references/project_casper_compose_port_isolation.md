---
name: "Casper compose/dev port isolation"
description: "make compose-override writes the gitignored compose.override.yaml from CASPER_PORT; the Makefile exports matching dev-process env."
type: project
---

# Casper compose/dev port isolation

Each worktree publishes on a `CASPER_PORT` block: frontend `+0`,
Temporal gRPC `+1`, Web UI `+2`, NATS `+3`, Codec Server `+4`.
`make compose-override` (a prerequisite of `bootstrap`, `infra-up`,
`app-up`) writes the gitignored `compose.override.yaml` and no-ops
when `CASPER_PORT` is unset or non-numeric; the Makefile's
`ifneq ($(CASPER_PORT),)` block exports `PORT`, `TEMPORAL_ADDRESS`
and `NATS_URL` on the same offsets for `make dev`. `.casper.json`
maps setup/run/dev/stop/teardown to `make` targets.

**Why:** Compose appends `ports:` lists across files, so the
override uses `!override` to replace them; that tag needs
compose-go (`docker-compose`, or `podman compose` delegating to
it), not the Python `podman-compose`. The codec server's
`UI_ORIGIN` must follow `+2` or `/decode` fails CORS.

**How to apply:** a new host port gets an offset in
`compose-override`, a row in `make endpoints`
([[feedback_casper_info_panel]]) and a matching export. Never
commit the override.
