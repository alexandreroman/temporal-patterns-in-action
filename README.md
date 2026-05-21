# Temporal Patterns In Action

[![CI](https://github.com/alexandreroman/temporal-patterns-in-action/actions/workflows/ci.yml/badge.svg)](https://github.com/alexandreroman/temporal-patterns-in-action/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Runnable demos of the core [Temporal](https://temporal.io) patterns —
saga, entity workflow, long-running batch, payload encryption,
durable AI agent, multi-agent deep research — with Go workers and a
Frontend to trigger and observe them.

![Temporal patterns in action](patterns.png)

## Prerequisites

- [Docker](https://www.docker.com/) or
  [Podman](https://podman.io/), with Compose.

## Getting Started

https://github.com/user-attachments/assets/01131162-3e34-4e5a-8bf6-3d16da19a930

Bring up the full stack — Temporal dev server, NATS, the Frontend,
and every worker:

```bash
docker-compose up -d --build
```

Then open:

- UI — <http://localhost:3000>
- Temporal Web UI — <http://localhost:8233>
- Codec Server — <http://localhost:8888> (opt-in,
  see [Payload Encryption](#payload-encryption))

Stop everything with `docker-compose down`.

## Configuration

| Variable                | Description                        | Default |
| ----------------------- | ---------------------------------- | ------- |
| `BATCH_WORKER_REPLICAS` | Number of `worker-batch` replicas  | `1`     |

The frontend and workers read `TEMPORAL_ADDRESS`, `TEMPORAL_NAMESPACE`,
and `NATS_URL`. Defaults wired in `compose.yaml` cover the
containerized stack; override them only when running outside compose.

## Local development

Prerequisites: Go 1.25+, Node.js 22 LTS, pnpm (via
`corepack enable`), and [Air](https://github.com/air-verse/air)
for worker hot-reload.

Launch Temporal + NATS in containers, then run the frontend and
all workers locally with hot-reload:

```bash
make infra-up
make dev
```

Or work on a single module at a time:

```bash
make frontend     # Nuxt dev server on :3000
make worker-saga  # also: worker-batch, worker-agent
```

Run all checks (lint, build, tests) across modules with
`make check`. Stop the infra with `make infra-down`.

After cloning, run `make setup` once to enable the versioned
git hooks in [`.githooks`](.githooks/) — `pre-push` runs `make
check` so CI failures are caught before they reach GitHub.
Use `git push --no-verify` to bypass for WIP branches.

## Architecture

```mermaid
graph LR
    User[Browser] --> Frontend
    Frontend -->|start / query| Temporal[(Temporal server)]
    Workers[Go workers] -->|poll task queue| Temporal
    Workers -->|publish| NATS[(NATS event bus)]
    NATS --> Frontend
    Frontend -->|SSE| User
```

| Module          | Description                                                 |
| --------------- | ----------------------------------------------------------- |
| `workers/`      | Go workers, one binary per pattern                          |
| `frontend/`     | Nuxt 4 + Vue 3 + Tailwind CSS 4 UI and API                  |
| `codec-server/` | Temporal Codec Server that decodes encrypted payloads on demand for the Temporal Web UI |

### How a run flows

1. The user picks a pattern in the UI and triggers a
   scenario. The Nuxt server route starts a Temporal
   workflow and immediately opens a Server-Sent Events
   (SSE) stream back to the browser.
2. The matching Go worker polls its task queue, runs
   the workflow, and executes activities. Temporal owns
   the durable state — retries, timers, history — so a
   worker crash is replayed, not lost.
3. Each activity publishes lifecycle events
   (`progress.step.started|completed|failed`) to NATS
   via a shared Temporal interceptor. Activities also
   emit business events (e.g.
   `saga.inventory.reserved`) where the pattern needs
   to show domain-level progress.
4. The Nuxt SSE endpoint subscribes to the relevant
   NATS subjects, forwards envelopes to the browser,
   and synthesises a terminal
   `progress.workflow.completed|failed` event from
   `handle.result()` once the workflow ends.

### Why NATS

Temporal is the source of truth for workflow state,
but polling its history from the browser to animate
a live timeline is awkward and lossy. NATS acts as
a **low-latency fan-out bus** between workers and the
frontend:

- **Push, not poll** — activities publish as they
  progress; the UI renders events as they arrive.
- **Subject hierarchy**
  `patterns.<pattern>.<workflowId>.<category>` lets
  the frontend filter per-run, per-pattern, or
  cluster-wide without parsing payloads.
- **Clean Temporal timeline** — publishing happens in
  activity scope (via an interceptor), never from
  workflow code, so the Temporal Web UI stays focused
  on the pattern's real activities with no
  `LocalActivityMarker` clutter.

### Why a frontend

The Temporal Web UI already shows workflow history,
but it is generic by design. This project ships a
purpose-built frontend because the patterns are
**pedagogical**: each one has a story to tell that a
raw event list cannot.

- **One page per pattern**, with a scenario selector
  that lets you pick happy path, partial failure, or
  compensation without editing code.
- **Live timeline** driven by the NATS event stream —
  steps light up in order, compensations highlight in
  a different style, retries are visible.
- **Pattern-specific panels** — saga compensation
  bracket, entity cart state, batch progress bar,
  agent reasoning / tool calls, multi-agent fan-out —
  render state that would be buried in a generic
  history view.
- **Side-by-side source viewer** pins the exact
  snippet responsible for the step currently running,
  with a language switcher (Go, Java, Python,
  TypeScript) so the UI doubles as a guided tour of
  the code in your SDK of choice.
- **Link back to Temporal Web UI** on every run for
  when you want to inspect raw history, retries, or
  the event payloads directly.

## Patterns

| Pattern                      | Package                     |
| ---------------------------- | --------------------------- |
| Saga                         | `workers/saga`              |
| Entity Workflow              | `workers/entity`            |
| Long-running batch           | `workers/batch`             |
| Payload Encryption           | `workers/encryption`        |
| Durable AI Agent             | `workers/agent`             |
| Multi-agent (deep research)  | `workers/multi-agent`       |
| Priority and Fairness        | `workers/priority-fairness` |

### Payload Encryption

The Encryption pattern wires an AES-256-GCM
`PayloadCodec` on the Temporal client and worker
so the server only ever stores ciphertext. Open
any workflow on the `patterns-encryption-encrypted`
task queue in the Temporal Web UI and you will see
opaque base64 — that's the point.

A companion **Codec Server** runs in the
`codec-server` service (port `8888`). It exposes
the same codec over HTTP so the Web UI can call
`/decode` and reveal the plaintext on demand. It
is **opt-in**: nothing is decoded automatically.
To enable it for your browser session:

1. Open an encrypted workflow in the Temporal Web
   UI and click the **glasses icon** in the top
   bar (next to the user avatar).
2. In the *Codec Server* modal, pick *"Use my
   browser setting and ignore Cluster-level
   setting"*.
3. Enter the codec server address —
   `http://localhost:8888` — and click *Apply*.

Toggling the option off restores the ciphertext
view.

The codec server reuses `workers/encryption`'s
`EncryptionCodec` via a `replace` directive in
its `go.mod`, so the encrypt/decrypt logic lives
in a single place.

## License

Apache-2.0 — see [LICENSE](LICENSE).
