// Package main runs the encryption pattern worker. Two workers share the
// binary: one on a clear task queue (default converter — Temporal sees
// plaintext) and one on an encrypted queue whose client is wired with the
// EncryptionCodec (Temporal sees ciphertext).
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/encryption"
	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

func main() {
	address := getenv("TEMPORAL_ADDRESS", "localhost:7233")
	natsURL := getenv("NATS_URL", "nats://localhost:4222")

	publisher, err := events.NewPublisher(natsURL)
	if err != nil {
		log.Printf("nats unavailable at %s (%v) — running without event publishing", natsURL, err)
		publisher = events.NopPublisher{}
	}
	defer publisher.Close()

	// Two clients: one with the codec (server sees ciphertext), one plain
	// (server sees raw JSON). Each drives its own task queue.
	plainClient, err := client.Dial(client.Options{HostPort: address})
	if err != nil {
		log.Fatalf("unable to dial temporal (clear): %v", err)
	}
	defer plainClient.Close()

	encClient, err := client.Dial(client.Options{
		HostPort: address,
		DataConverter: converter.NewCodecDataConverter(
			converter.GetDefaultDataConverter(),
			&encryption.EncryptionCodec{Key: encryption.DemoKey},
		),
	})
	if err != nil {
		log.Fatalf("unable to dial temporal (encrypted): %v", err)
	}
	defer encClient.Close()

	interc := []interceptor.WorkerInterceptor{events.NewInterceptor(publisher, encryption.Pattern)}
	a := &encryption.Activities{Publisher: publisher}

	register := func(w worker.Worker) {
		w.RegisterWorkflow(encryption.ProcessSensitiveOrderWorkflow)
		w.RegisterActivityWithOptions(a.ValidateOrder, activity.RegisterOptions{Name: "validate-order"})
		w.RegisterActivityWithOptions(a.ChargeCard, activity.RegisterOptions{Name: "charge-card"})
		w.RegisterActivityWithOptions(a.ShipOrder, activity.RegisterOptions{Name: "ship-order"})
		w.RegisterActivityWithOptions(a.SendReceipt, activity.RegisterOptions{Name: "send-receipt"})
	}

	wClear := worker.New(plainClient, encryption.TaskQueueClear, worker.Options{Interceptors: interc})
	register(wClear)
	wEnc := worker.New(encClient, encryption.TaskQueueEncrypted, worker.Options{Interceptors: interc})
	register(wEnc)

	log.Printf("encryption worker connected to %s — listening on %s and %s",
		address, encryption.TaskQueueClear, encryption.TaskQueueEncrypted)

	// Shared interrupt channel so Ctrl-C stops both workers together.
	interruptCh := worker.InterruptCh()
	go func() {
		if err := wClear.Run(interruptCh); err != nil {
			log.Fatalf("clear worker stopped: %v", err)
		}
	}()
	if err := wEnc.Run(interruptCh); err != nil {
		log.Fatalf("encrypted worker stopped: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
