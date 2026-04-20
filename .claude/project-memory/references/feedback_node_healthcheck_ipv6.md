---
name: "Node healthcheck: use 127.0.0.1, not localhost"
description: "Compose healthchecks probing the Nuxt frontend from inside the container must use 127.0.0.1, because busybox wget resolves localhost to ::1 first and Node listens on IPv4 only."
type: feedback
---

# Node healthcheck: use 127.0.0.1, not localhost

Compose healthchecks that probe the Nuxt frontend
from inside its container must target
`http://127.0.0.1:3000/...`, not
`http://localhost:3000/...`.

**Why:** the frontend image is `node:22-alpine`,
whose busybox `wget` resolves `localhost` to `::1`
first. Nuxt/nitro binds to `0.0.0.0` (IPv4 only)
by default, so the probe gets `Connection
refused` and the container is marked unhealthy
even though the endpoint is reachable on
`127.0.0.1` and via the published port on the
host. This was observed when adding
`/api/health` — the first attempt used
`localhost` and failed; switching to `127.0.0.1`
made the container healthy immediately.

**How to apply:** when adding a compose
healthcheck (or any in-container probe) that
hits the Nuxt frontend, always write
`http://127.0.0.1:<port>/...`. Same rule for any
future Node-based service in this repo unless it
is explicitly configured to listen on `[::]`.
