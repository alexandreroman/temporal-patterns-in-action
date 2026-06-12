---
name: "Nuxt server env vars: use process.env, not runtimeConfig"
description: "In the Nuxt frontend, read server-side env vars via process.env.* at runtime; runtimeConfig bakes defaults at build time unless the var is prefixed NUXT_*."
type: feedback
---

# Nuxt server env vars: use process.env, not runtimeConfig

In the Nuxt frontend, server-side code must read env
vars via `process.env.FOO ?? "<default>"` directly at
the call site — not through `useRuntimeConfig()` /
`runtimeConfig` in `nuxt.config.ts`.

**Why:** Nuxt bakes `runtimeConfig` defaults at build
time. Setting a plain `FOO` env var in the container
does NOT override the baked value — only the
`NUXT_FOO` prefix does. Via `runtimeConfig`, a
`NATS_URL=nats://nats:4222` is ignored, the frontend
falls back to `nats://localhost:4222` →
`ECONNREFUSED ::1:4222`, and the UI appears frozen
("no workflow is launching") even though workflows
run fine on Temporal — only the SSE/NATS event
stream is broken. Reading `process.env.*` at the
call site avoids this. The env var stays named
`NATS_URL` (consistent with the Go workers), not
`NUXT_NATS_URL`.

**How to apply:** when adding a new server-side env
var (service URL, API key, …), follow the pattern in
`server/utils/temporal.ts` and `server/utils/nats.ts`
— read `process.env.*` at the call site. Do not
reintroduce a `runtimeConfig` block for server-only
values. Troubleshooting hint: if a container has the
right env var set but behaves as if it is missing,
check whether the Nuxt bundle is reading it via
`runtimeConfig` (build-time baked).
