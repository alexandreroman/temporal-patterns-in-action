// Package main runs the saga pattern worker.
package main

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
	"github.com/alexandreroman/temporal-patterns-in-action/workers/saga"
)

func main() {
	events.RunWorker(saga.Pattern, saga.TaskQueue, func(w worker.Worker, pub events.Publisher) {
		w.RegisterWorkflow(saga.OrderProcessingWorkflow)

		// Register each saga activity under its canonical kebab-case name so the
		// NATS event interceptor emits progress.step.* events matching the step IDs
		// used by the workflow and the frontend pipeline.
		a := &saga.Activities{Publisher: pub, TimeoutChance: 0.3}
		w.RegisterActivityWithOptions(a.ReserveInventory, activity.RegisterOptions{Name: "reserve-inventory"})
		w.RegisterActivityWithOptions(a.ReleaseInventory, activity.RegisterOptions{Name: "release-inventory"})
		w.RegisterActivityWithOptions(a.ChargePayment, activity.RegisterOptions{Name: "charge-payment"})
		w.RegisterActivityWithOptions(a.RefundPayment, activity.RegisterOptions{Name: "refund-payment"})
		w.RegisterActivityWithOptions(a.ShipOrder, activity.RegisterOptions{Name: "ship-order"})
		w.RegisterActivityWithOptions(a.CancelShipment, activity.RegisterOptions{Name: "cancel-shipment"})
		w.RegisterActivityWithOptions(a.SendConfirmation, activity.RegisterOptions{Name: "send-confirmation"})
	})
}
