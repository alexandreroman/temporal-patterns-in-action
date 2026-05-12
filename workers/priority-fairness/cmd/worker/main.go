// Package main runs the priority-fairness pattern worker. The worker caps
// MaxConcurrentActivityExecutionSize at MaxConcurrentActivities — the demo's
// "4 agents" — so the activity backlog actually exercises Temporal's task
// queue priority and fairness ordering.
//
// Unlike the other patterns, this worker dials its own Temporal client and
// hands it to Activities so the StartResolveTicket local activity can create
// new top-level ResolveTicketWorkflow executions (not ChildWorkflows). The
// worker that events.RunWorker stands up has its own internal client; the
// extra dial here is intentionally local so the shared events.RunWorker
// signature stays unchanged for the other patterns.
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
	pf "github.com/alexandreroman/temporal-patterns-in-action/workers/priority-fairness"
)

func main() {
	events.HandleHealthcheck()

	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		address = "localhost:7233"
	}
	c, err := client.Dial(client.Options{HostPort: address})
	if err != nil {
		log.Fatalf("priority-fairness: unable to dial temporal client for activities: %v", err)
	}
	defer c.Close()

	events.RunWorker(pf.Pattern, pf.TaskQueue, func(w worker.Worker, pub events.Publisher) {
		w.RegisterWorkflow(pf.HelpdeskRunWorkflow)
		w.RegisterWorkflow(pf.ResolveTicketWorkflow)

		a := &pf.Activities{Publisher: pub, Client: c}
		w.RegisterActivityWithOptions(a.AnnounceRunSeeded,
			activity.RegisterOptions{Name: "announce-run-seeded"})
		w.RegisterActivityWithOptions(a.AnnounceIncidentInjected,
			activity.RegisterOptions{Name: "announce-incident-injected"})
		w.RegisterActivityWithOptions(a.StartResolveTicket,
			activity.RegisterOptions{Name: "start-resolve-ticket"})
		w.RegisterActivityWithOptions(a.WaitTicketDone,
			activity.RegisterOptions{Name: "wait-ticket-done"})
		w.RegisterActivityWithOptions(a.ResolveTicket,
			activity.RegisterOptions{Name: "resolve-ticket"})
	}, worker.Options{
		MaxConcurrentActivityExecutionSize: pf.MaxConcurrentActivities,
	})
}
