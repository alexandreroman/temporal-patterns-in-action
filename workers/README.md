# Workers

Go workers hosting Temporal workflows for each pattern demo.
Each pattern ships as its own dedicated binary sharing a
single `go.mod`.

## Layout

```
workers/
├── go.mod                   # shared module for all patterns
└── saga/                    # Saga pattern (trip booking with compensations)
    ├── activities.go
    ├── types.go
    ├── workflow.go
    ├── workflow_test.go
    └── cmd/
        └── worker/
            └── main.go      # saga worker entry point
```

Each pattern exposes its own `TaskQueue` constant and
provides its own `cmd/worker` binary. Patterns do not
depend on each other.

## Running

```bash
make tidy        # download dependencies
make run-saga    # start the saga worker (connects to localhost:7233)
make test        # run workflow tests across all patterns
make check       # vet + lint + test
make build       # build all pattern binaries into bin/
```

`make run` on its own lists the available per-pattern
targets. Use `make dev-saga` to run the saga worker with
hot-reload (requires [Air](https://github.com/air-verse/air)).

Set `TEMPORAL_ADDRESS` to target a different Temporal
frontend (default `localhost:7233`).
