// Package main runs the entity pattern worker.
package main

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/entity"
	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

func main() {
	events.RunWorker(entity.Pattern, entity.TaskQueue, func(w worker.Worker, pub events.Publisher) {
		w.RegisterWorkflow(entity.ShoppingCartWorkflow)

		// Register each entity activity under its canonical kebab-case name so
		// the NATS event interceptor emits progress.step.* events matching the
		// step IDs used by the workflow and the frontend pipeline.
		a := &entity.Activities{Publisher: pub}
		w.RegisterActivityWithOptions(a.ValidateItem, activity.RegisterOptions{Name: "validate-item"})
		w.RegisterActivityWithOptions(a.PriceItem, activity.RegisterOptions{Name: "price-item"})
		w.RegisterActivityWithOptions(a.UpdateQty, activity.RegisterOptions{Name: "update-qty"})
		w.RegisterActivityWithOptions(a.RemoveItem, activity.RegisterOptions{Name: "remove-item"})
		w.RegisterActivityWithOptions(a.ProcessPayment, activity.RegisterOptions{Name: "process-payment"})
		w.RegisterActivityWithOptions(a.SendConfirmation, activity.RegisterOptions{Name: "send-confirmation"})
	})
}
