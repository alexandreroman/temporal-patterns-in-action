// Package multiagent implements the multi-agent deep-research pattern: a
// parent workflow fans out to N child "research agent" workflows (one per
// subtopic), collects their results — tolerating per-search failures — and
// synthesises a final report. The four on-screen phases are Plan →
// Query generation → Research (fan-out) → Synthesis.
package multiagent

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// standardActivityOptions is the retry/timeout policy shared by the three
// parent-side "LLM" activities (plan, queries, synthesis) and the
// child-side web searches.
func standardActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			BackoffCoefficient: 1.5,
			MaximumAttempts:    3,
		},
	}
}

// fastActivityOptions is used for the two marker activities
// (announce-fanout, record-child-outcome). These are pure event-publish
// side effects — a NATS hiccup must not add seconds of retry latency to
// the visible phase transitions.
func fastActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	}
}

// DeepResearchWorkflow is the parent: plan → queries → fan-out → synthesise.
// Per-child failures are tolerated; the workflow only fails if all children
// fail (which the scripted demo never does).
func DeepResearchWorkflow(ctx workflow.Context, req DeepResearchRequest) (Report, error) {
	logger := workflow.GetLogger(ctx)
	parentExec := workflow.GetInfo(ctx).WorkflowExecution
	parentID, parentRunID := parentExec.ID, parentExec.RunID

	progress := Progress{Phase: PhaseIdle}
	if err := workflow.SetQueryHandler(ctx, "getProgress", func() (Progress, error) {
		return progress, nil
	}); err != nil {
		return Report{}, err
	}

	var a *Activities
	stdCtx := workflow.WithActivityOptions(ctx, standardActivityOptions())
	fastCtx := workflow.WithActivityOptions(ctx, fastActivityOptions())

	// Phase 1 — Plan.
	progress.Phase = PhasePlanning
	var plan ResearchPlan
	if err := workflow.ExecuteActivity(stdCtx, a.PlanResearch, req).Get(stdCtx, &plan); err != nil {
		return Report{}, err
	}
	progress.LLMCalls++

	// Phase 2 — Query generation.
	progress.Phase = PhaseQueries
	var queries ResearchQueries
	if err := workflow.ExecuteActivity(stdCtx, a.GenerateQueries, plan).Get(stdCtx, &queries); err != nil {
		return Report{}, err
	}
	progress.LLMCalls++

	// Phase 3 — Fan-out to child research-agent workflows.
	progress.Phase = PhaseResearch
	if err := workflow.ExecuteActivity(fastCtx, a.AnnounceFanout, len(queries.Topics)).Get(fastCtx, nil); err != nil {
		logger.Warn("announce-fanout failed", "error", err)
	}

	futures := make([]workflow.Future, len(queries.Topics))
	topicNames := make([]string, len(queries.Topics))
	for i, tq := range queries.Topics {
		topicNames[i] = tq.TopicName
		childCtx := workflow.WithChildOptions(stdCtx, workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s-agent-%d", parentID, tq.TopicIndex),
		})
		childInput := ResearchAgentInput{
			ParentWorkflowID: parentID,
			ParentRunID:      parentRunID,
			Scenario:         req.Scenario,
			TopicIndex:       tq.TopicIndex,
			TopicName:        tq.TopicName,
			Queries:          tq.Queries,
		}
		futures[i] = workflow.ExecuteChildWorkflow(childCtx, ResearchAgentWorkflow, childInput)
	}

	// Fan-in — tolerate per-child failures. One RecordChildOutcome call per
	// settled child keeps event emission out of workflow scope.
	successful := make([]ResearchResult, 0, len(futures))
	for i, f := range futures {
		var result ResearchResult
		if err := f.Get(stdCtx, &result); err != nil {
			logger.Warn("child workflow failed", "topicIndex", i, "error", err)
			if rerr := workflow.ExecuteActivity(fastCtx, a.RecordChildOutcome, ChildOutcomeInput{
				TopicIndex: i,
				TopicName:  topicNames[i],
				Failed:     true,
				Error:      err.Error(),
			}).Get(fastCtx, nil); rerr != nil {
				logger.Warn("record-child-outcome failed", "error", rerr)
			}
			continue
		}
		successful = append(successful, result)
		if rerr := workflow.ExecuteActivity(fastCtx, a.RecordChildOutcome, ChildOutcomeInput{
			TopicIndex: result.TopicIndex,
			TopicName:  result.TopicName,
			Sources:    len(result.Sources),
			Partial:    result.Partial,
		}).Get(fastCtx, nil); rerr != nil {
			logger.Warn("record-child-outcome failed", "error", rerr)
		}
	}

	if len(successful) == 0 {
		return Report{}, temporal.NewNonRetryableApplicationError(
			"every research agent failed", "AllChildrenFailed", nil)
	}

	// Phase 4 — Synthesis.
	progress.Phase = PhaseSynthesis
	var report Report
	if err := workflow.ExecuteActivity(stdCtx, a.SynthesizeReport, SynthesisInput{
		Prompt:  req.Prompt,
		Results: successful,
	}).Get(stdCtx, &report); err != nil {
		return Report{}, err
	}
	progress.LLMCalls++
	progress.Phase = PhaseCompleted
	progress.Completed = true

	return report, nil
}

// ResearchAgentWorkflow is a child: it runs every WebSearch for one
// subtopic and collects the sources. A child is only failed when every
// query fails; any successful query keeps the child alive and the result
// is flagged Partial for the parent.
func ResearchAgentWorkflow(ctx workflow.Context, in ResearchAgentInput) (ResearchResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, standardActivityOptions())

	result := ResearchResult{
		TopicIndex: in.TopicIndex,
		TopicName:  in.TopicName,
	}

	var a *Activities
	failures := 0
	for qi, q := range in.Queries {
		var sr SearchResult
		err := workflow.ExecuteActivity(ctx, a.WebSearch, SearchInput{
			ParentWorkflowID: in.ParentWorkflowID,
			ParentRunID:      in.ParentRunID,
			Scenario:         in.Scenario,
			TopicIndex:       in.TopicIndex,
			TopicName:        in.TopicName,
			QueryIndex:       qi,
			Query:            q,
		}).Get(ctx, &sr)
		if err != nil {
			logger.Warn("web-search failed", "topic", in.TopicName, "queryIndex", qi, "error", err)
			failures++
			continue
		}
		result.Sources = append(result.Sources, sr.Sources...)
	}

	if failures == len(in.Queries) {
		return ResearchResult{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("all %d searches failed for topic %q", failures, in.TopicName),
			"AllSearchesFailed", nil)
	}
	result.Partial = failures > 0
	return result, nil
}
