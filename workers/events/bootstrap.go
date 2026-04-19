package events

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
)

// RunWorker wires the boilerplate shared by every pattern's cmd/worker/main.go:
// resolves TEMPORAL_ADDRESS / NATS_URL from the environment, dials Temporal,
// constructs a worker bound to taskQueue with the progress-event interceptor
// wired for `pattern`, then invokes `register` so the caller can add its own
// workflows and activities. An optional worker.Options value may be supplied
// to override tuning fields (e.g. MaxConcurrentActivityExecutionSize); the
// interceptor list is always set by this function. Returns only on failure
// or interrupt.
func RunWorker(pattern, taskQueue string, register func(w worker.Worker, publisher Publisher), extra ...worker.Options) {
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

	log.Printf("%s worker connected to %s — listening on task queue: %s", pattern, address, taskQueue)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}
