---
name: "cmux compose port isolation"
description: "post-create.sh generates a gitignored compose.override.yaml remapping host ports from CMUX_PORT so parallel worktrees don't collide."
type: project
---

# cmux compose port isolation

`.cmux/post-create.sh` generates a per-worktree,
gitignored `compose.override.yaml` that remaps the
stack's **host-published** ports onto the workspace's
`CMUX_PORT` block, so several cmux worktrees can run
`docker-compose up` at once without fighting over the
same host ports.

Offset scheme (base = `$CMUX_PORT`):

- frontend (web UI) -> `CMUX_PORT` (container 3000)
- temporal gRPC     -> `CMUX_PORT+1` (container 7233)
- temporal web UI   -> `CMUX_PORT+2` (container 8233)
- nats client       -> `CMUX_PORT+3` (container 4222)
- codec server      -> `CMUX_PORT+4` (container 8888)

**Why:** Compose **concatenates** multi-value options
(`ports`, `expose`, ...) when merging files, so a
plain override would keep the base `3000/7233/8233/...`
bindings AND add the remapped ones — defeating the
isolation. The override therefore uses the `!override`
YAML tag to fully **replace** each `ports:` list. This
tag is honored by Compose v2 / compose-go (here
`podman compose` delegates to the `docker-compose`
provider, which supports it); `podman-compose` (Python)
would not.

**How to apply:**

- Only host-published ports change; container-internal
  addresses (`temporal:7233`, `nats://nats:4222`) stay
  fixed, so workers/frontend still reach the stack.
- The codec server's `UI_ORIGIN` CORS allowlist is also
  overridden to the remapped Temporal UI port
  (`CMUX_PORT+2`), or the codec `/decode` demo breaks.
  See [[project_codec_server_ui_endpoint]].
- `compose.override.yaml` / `.yml` are gitignored —
  never commit them; the main worktree keeps the stock
  3000/7233/8233 ports.
- The temporal service does not set `--ui-port`;
  `start-dev` defaults the UI to 8233 (gRPC+1000), which
  is the container-side port published in the override.
- Extend the offset scheme in `post-create.sh` whenever
  a new pattern adds a host-published port; see
  [[project_adding_new_pattern]].
