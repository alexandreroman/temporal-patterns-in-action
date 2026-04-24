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

		a := &saga.Activities{Publisher: pub, TimeoutChance: 0.3}
		w.RegisterActivityWithOptions(a.CheckFraud, activity.RegisterOptions{Name: "check-fraud"})
		w.RegisterActivityWithOptions(a.ReleaseFraudHold, activity.RegisterOptions{Name: "release-fraud-hold"})
		w.RegisterActivityWithOptions(a.PrepareShipment, activity.RegisterOptions{Name: "prepare-shipment"})
		w.RegisterActivityWithOptions(a.CancelShipment, activity.RegisterOptions{Name: "cancel-shipment"})
		w.RegisterActivityWithOptions(a.ChargeCustomer, activity.RegisterOptions{Name: "charge-customer"})
		w.RegisterActivityWithOptions(a.RefundCustomer, activity.RegisterOptions{Name: "refund-customer"})
		w.RegisterActivityWithOptions(a.SendConfirmation, activity.RegisterOptions{Name: "send-confirmation"})
	}, worker.Options{})
}
