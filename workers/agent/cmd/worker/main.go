// Package main runs the durable AI agent pattern worker.
package main

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/agent"
	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

func main() {
	events.RunWorker(agent.Pattern, agent.TaskQueue, func(w worker.Worker, pub events.Publisher) {
		w.RegisterWorkflow(agent.TravelAgentWorkflow)

		a := &agent.Activities{Publisher: pub}
		w.RegisterActivityWithOptions(a.CallLLM, activity.RegisterOptions{Name: "call-llm"})
		w.RegisterActivityWithOptions(a.ExecuteMCPTool, activity.RegisterOptions{Name: "execute-mcp-tool"})
		w.RegisterActivityWithOptions(a.RecordApproval, activity.RegisterOptions{Name: "record-approval"})
		w.RegisterActivityWithOptions(a.ReportPlan, activity.RegisterOptions{Name: "report-plan"})
	})
}
