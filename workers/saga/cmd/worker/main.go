// Package main runs the saga pattern worker.
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
	"github.com/alexandreroman/temporal-patterns-in-action/workers/saga"
)

func main() {
	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		address = "localhost:7233"
	}

	natsURL, ok := os.LookupEnv("NATS_URL")
	if !ok {
		natsURL = "nats://localhost:4222"
	}

	publisher, err := events.NewPublisher(natsURL)
	if err != nil {
		log.Printf("nats unavailable at %s (%v) — running without event publishing", natsURL, err)
		publisher = events.NopPublisher{}
	}
	defer publisher.Close()

	c, err := client.Dial(client.Options{HostPort: address})
	if err != nil {
		log.Fatalf("unable to create temporal client: %v", err)
	}
	defer c.Close()

	w := worker.New(c, saga.TaskQueue, worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{
			events.NewInterceptor(publisher, saga.Pattern),
		},
	})
	w.RegisterWorkflow(saga.OrderProcessingWorkflow)

	// Register each saga activity under its canonical kebab-case name so the
	// NATS event interceptor emits progress.step.* events matching the step IDs
	// used by the workflow and the frontend pipeline.
	a := &saga.Activities{Publisher: publisher, TimeoutChance: 0.3}
	w.RegisterActivityWithOptions(a.ReserveInventory, activity.RegisterOptions{Name: "reserve-inventory"})
	w.RegisterActivityWithOptions(a.ReleaseInventory, activity.RegisterOptions{Name: "release-inventory"})
	w.RegisterActivityWithOptions(a.ChargePayment, activity.RegisterOptions{Name: "charge-payment"})
	w.RegisterActivityWithOptions(a.RefundPayment, activity.RegisterOptions{Name: "refund-payment"})
	w.RegisterActivityWithOptions(a.ShipOrder, activity.RegisterOptions{Name: "ship-order"})
	w.RegisterActivityWithOptions(a.CancelShipment, activity.RegisterOptions{Name: "cancel-shipment"})
	w.RegisterActivityWithOptions(a.SendConfirmation, activity.RegisterOptions{Name: "send-confirmation"})
	w.RegisterActivityWithOptions(a.RetractEmail, activity.RegisterOptions{Name: "retract-email"})

	log.Printf("saga worker connected to %s — listening on task queue: %s", address, saga.TaskQueue)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}
