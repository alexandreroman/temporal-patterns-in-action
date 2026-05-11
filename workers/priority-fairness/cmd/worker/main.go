// Package main runs the priority-fairness pattern worker. The worker caps
// MaxConcurrentActivityExecutionSize at MaxConcurrentActivities — the demo's
// "4 agents" — so the activity backlog actually exercises Temporal's task
// queue priority and fairness ordering.
package main

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
	pf "github.com/alexandreroman/temporal-patterns-in-action/workers/priority-fairness"
)

func main() {
	events.RunWorker(pf.Pattern, pf.TaskQueue, func(w worker.Worker, pub events.Publisher) {
		w.RegisterWorkflow(pf.HelpdeskRunWorkflow)
		w.RegisterWorkflow(pf.ResolveTicketWorkflow)

		a := &pf.Activities{Publisher: pub}
		w.RegisterActivityWithOptions(a.AnnounceRunSeeded,
			activity.RegisterOptions{Name: "announce-run-seeded"})
		w.RegisterActivityWithOptions(a.AnnounceBurstExecuted,
			activity.RegisterOptions{Name: "announce-burst-executed"})
		w.RegisterActivityWithOptions(a.AnnounceIncidentInjected,
			activity.RegisterOptions{Name: "announce-incident-injected"})
		w.RegisterActivityWithOptions(a.ResolveTicket,
			activity.RegisterOptions{Name: "resolve-ticket"})
	}, worker.Options{
		MaxConcurrentActivityExecutionSize: pf.MaxConcurrentActivities,
	})
}
