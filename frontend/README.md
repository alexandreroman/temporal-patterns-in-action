# Frontend

Nuxt 4 + Vue 3 + Tailwind CSS 4 UI that triggers and observes the
Temporal pattern demos. Nuxt server routes hold the Temporal client
and relay NATS events over SSE — the browser never talks to Temporal
or NATS directly.

## Prerequisites

- Node.js 24 LTS
- pnpm (enabled via `corepack enable`)
- Temporal, NATS, and the Go workers running — `make dev` at the
  repository root starts them all

## Running

```bash
make install   # pnpm install
make dev       # Nuxt dev server on http://localhost:3000
```

The server reads `TEMPORAL_ADDRESS` (default `localhost:7233`),
`TEMPORAL_NAMESPACE` (default `default`), and `NATS_URL` (default
`nats://localhost:4222`) from the environment at runtime.

See the [root README](../README.md) for the architecture and the
list of patterns.
