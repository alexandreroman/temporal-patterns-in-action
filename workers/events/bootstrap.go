package events

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
)

const healthPort = 8080

// RunWorker wires the boilerplate shared by every pattern's cmd/worker/main.go:
// resolves TEMPORAL_ADDRESS / NATS_URL from the environment, dials Temporal,
// constructs a worker bound to taskQueue with the progress-event interceptor
// wired for `pattern`, then invokes `register` so the caller can add its own
// workflows and activities. An optional worker.Options value may be supplied
// to override tuning fields (e.g. MaxConcurrentActivityExecutionSize); the
// interceptor list is always set by this function. Returns only on failure
// or interrupt.
func RunWorker(pattern, taskQueue string, register func(w worker.Worker, publisher Publisher), extra ...worker.Options) {
	// The distroless runtime image has no shell, wget, or curl, so Compose
	// healthchecks re-exec the worker binary with `-healthcheck` to probe the
	// in-process health server.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		runHealthcheck()
		return
	}

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
	defer publisher.Close()

	c, err := client.Dial(client.Options{HostPort: address})
	if err != nil {
		log.Fatalf("unable to create temporal client: %v", err)
	}
	defer c.Close()

	opts := worker.Options{}
	if len(extra) > 0 {
		opts = extra[0]
	}
	opts.Interceptors = []interceptor.WorkerInterceptor{NewInterceptor(publisher, pattern)}

	w := worker.New(c, taskQueue, opts)
	register(w, publisher)

	go serveHealth()

	log.Printf("%s worker connected to %s — listening on task queue: %s", pattern, address, taskQueue)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}

func serveHealth() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	addr := fmt.Sprintf(":%d", healthPort)
	log.Printf("health server listening on %s/healthz", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("health server stopped: %v", err)
	}
}

func runHealthcheck() {
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get(fmt.Sprintf("http://localhost:%d/healthz", healthPort))
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}
