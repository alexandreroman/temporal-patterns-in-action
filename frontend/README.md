# Frontend

Nuxt 4 + Vue 3 + Tailwind CSS 4 UI that triggers and observes the Temporal
pattern demos. Nuxt server routes hold the Temporal client — the browser
never talks to Temporal directly.

## Prerequisites

- Node.js ≥ 22 LTS
- pnpm (enabled via `corepack enable`)
- Temporal running on `localhost:7233` — see root `compose.yaml`
- The Go worker from `../workers` running on task queue `patterns-saga`

## Running

```bash
make install   # pnpm install
make dev       # Nuxt dev server on http://localhost:3000
```

## Configuration

Environment variables (read by `nuxt.config.ts`):

| Variable             | Default                 | Purpose                        |
| -------------------- | ----------------------- | ------------------------------ |
| `TEMPORAL_ADDRESS`   | `localhost:7233`        | Temporal frontend gRPC address |
| `TEMPORAL_NAMESPACE` | `default`               | Target Temporal namespace      |
| `NATS_URL`           | `nats://localhost:4222` | NATS event bus URL             |

## Layout

```
frontend/
├── app/                  # Vue pages, components, assets
│   ├── app.vue
│   ├── assets/
│   ├── components/       # Pattern-specific UI (timeline, pipeline, …)
│   ├── composables/      # usePatternStream — SSE client
│   └── pages/
│       ├── index.vue
│       └── patterns/
│           └── saga.vue
├── server/               # Nitro server (Temporal client lives here)
│   ├── api/
│   │   ├── saga/                           # saga-specific actions
│   │   └── patterns/[pattern]/[id]/        # generic SSE event stream
│   └── utils/
│       ├── temporal.ts
│       └── nats.ts       # per-pattern NATS subscription helper
└── shared/               # Types shared between server and client
    ├── events.ts         # event envelope + category
    └── types.ts
```
