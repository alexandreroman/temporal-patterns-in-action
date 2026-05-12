package events

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
)

// healthPort reads HEALTH_PORT from the environment. Zero (the default
// when the variable is unset or invalid) disables the health server, so
// running multiple workers on a shared host (e.g. `make dev`) doesn't
// cause `:8080` collisions. Compose sets HEALTH_PORT=8080 per worker
// service to enable the in-container probe.
func healthPort() int {
	v := os.Getenv("HEALTH_PORT")
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 65535 {
		log.Printf("invalid HEALTH_PORT=%q, disabling health server", v)
		return 0
	}
	return n
}

// HandleHealthcheck probes the in-process health server and exits if the binary
// was invoked with the "-healthcheck" arg. Otherwise returns immediately. Call
// this at the top of any cmd/worker/main.go that builds its own worker setup
// instead of using RunWorker. Compose runs the worker binary with this arg
// every 10 s to drive its container healthcheck.
func HandleHealthcheck() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		runHealthcheck()
	}
}

// ServeHealth starts the /healthz server in a goroutine when HEALTH_PORT is
// set. Safe to call multiple times — no-op when the env var is empty.
func ServeHealth() {
	if port := healthPort(); port > 0 {
		go serveHealth(port)
	}
}

// RunWorker wires the boilerplate shared by every pattern's cmd/worker/main.go:
// resolves TEMPORAL_ADDRESS / NATS_URL from the environment, dials Temporal,
// constructs a worker bound to taskQueue with the progress-event interceptor
// wired for `pattern`, then invokes `register` so the caller can add its own
// workflows and activities. The `opts` value tunes the worker (e.g.
// MaxConcurrentActivityExecutionSize); pass worker.Options{} for defaults.
// The interceptor list is always set by this function. Returns only on
// failure or interrupt.
//
// Callers register every activity under a kebab-case name (e.g. "call-llm")
// so the event interceptor can emit progress.step.* events matching the step
// IDs used by the frontend.
func RunWorker(pattern, taskQueue string, register func(w worker.Worker, publisher Publisher), opts worker.Options) {
	// The distroless runtime image has no shell, wget, or curl, so Compose
	// healthchecks re-exec the worker binary with `-healthcheck` to probe the
	// in-process health server.
	HandleHealthcheck()

	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		address = "localhost:7233"
	}

	natsURL, ok := os.LookupEnv("NATS_URL")
	if !ok {
		natsURL = "nats://localhost:4222"
	}

	publisher, err := NewPublisher(natsURL)
	if err != nil {
		log.Printf("nats unavailable at %s (%v) — running without event publishing", natsURL, err)
		publisher = NopPublisher{}
	}
	defer func() { _ = publisher.Close() }()

	c, err := client.Dial(client.Options{HostPort: address})
	if err != nil {
		log.Fatalf("unable to create temporal client: %v", err)
	}
	defer c.Close()

	opts.Interceptors = []interceptor.WorkerInterceptor{NewInterceptor(publisher, pattern)}

	w := worker.New(c, taskQueue, opts)
	register(w, publisher)

	ServeHealth()

	log.Printf("%s worker connected to %s — listening on task queue: %s", pattern, address, taskQueue)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}

func serveHealth(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	addr := fmt.Sprintf(":%d", port)
	log.Printf("health server listening on %s/healthz", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("health server stopped: %v", err)
	}
}

func runHealthcheck() {
	port := healthPort()
	if port == 0 {
		os.Exit(1)
	}
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get(fmt.Sprintf("http://localhost:%d/healthz", port))
	if err != nil {
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}
