# Workers

Go workers hosting Temporal workflows for each pattern demo.
Each pattern ships as its own dedicated binary sharing a
single `go.mod`.

## Layout

```text
workers/
├── go.mod          # shared module for all patterns
├── events/         # NATS publisher, activity interceptor, RunWorker
└── <pattern>/      # saga, entity, batch, encryption, agent, …
    ├── events.go   # Pattern constant + business event types
    ├── types.go
    ├── activities.go
    ├── workflow.go
    ├── workflow_test.go
    └── cmd/worker/main.go
```

Each pattern exposes its own `TaskQueue` constant and
provides its own `cmd/worker` binary. Patterns depend
only on the shared `events` package, never on each other.

## Running

```bash
make tidy        # download dependencies
make run-saga    # start the saga worker (connects to localhost:7233)
make test        # run workflow tests across all patterns
make check       # vet + lint + workflowcheck + test
make build       # build all pattern binaries into bin/
```

`make run` on its own lists the available per-pattern
targets. Use `make dev-saga` to run the saga worker with
hot-reload (requires [Air](https://github.com/air-verse/air)).

Set `TEMPORAL_ADDRESS` to target a different Temporal
frontend (default `localhost:7233`) and `NATS_URL` for a
different NATS server (default `nats://localhost:4222`).
