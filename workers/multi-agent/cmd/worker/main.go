// Package main runs the multi-agent deep-research pattern worker.
package main

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
	multiagent "github.com/alexandreroman/temporal-patterns-in-action/workers/multi-agent"
)

func main() {
	events.RunWorker(multiagent.Pattern, multiagent.TaskQueue, func(w worker.Worker, pub events.Publisher) {
		w.RegisterWorkflow(multiagent.DeepResearchWorkflow)
		w.RegisterWorkflow(multiagent.ResearchAgentWorkflow)

		a := &multiagent.Activities{Publisher: pub}
		w.RegisterActivityWithOptions(a.PlanResearch, activity.RegisterOptions{Name: "plan-research"})
		w.RegisterActivityWithOptions(a.GenerateQueries, activity.RegisterOptions{Name: "generate-queries"})
		w.RegisterActivityWithOptions(a.AnnounceFanout, activity.RegisterOptions{Name: "announce-fanout"})
		w.RegisterActivityWithOptions(a.WebSearch, activity.RegisterOptions{Name: "web-search"})
		w.RegisterActivityWithOptions(a.RecordChildOutcome, activity.RegisterOptions{Name: "record-child-outcome"})
		w.RegisterActivityWithOptions(a.SynthesizeReport, activity.RegisterOptions{Name: "synthesize-report"})
	})
}
