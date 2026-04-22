package multiagent

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Animation delays tuned so each phase transition is visible on screen
// without making the demo feel sluggish.
const (
	planThinkTime      = 1200 * time.Millisecond
	queriesThinkTime   = 900 * time.Millisecond
	searchMinTime      = 700 * time.Millisecond
	searchMaxTime      = 1000 * time.Millisecond
	synthesisThinkTime = 1500 * time.Millisecond
)

// Activities groups the multi-agent pattern activities. Fields can be used
// for dependency injection (LLM client, search client, event publisher, ...).
// The demo implementation is fully scripted to keep every run reproducible.
type Activities struct {
	Publisher events.Publisher
	// FastMode skips the simulated think/search sleeps so tests finish quickly.
	FastMode bool
}

func (a *Activities) pause(d time.Duration) {
	if a.FastMode {
		return
	}
	time.Sleep(d)
}

// PlanResearch simulates a planning LLM call. On the first attempt it also
// emits the multi-agent.user.prompt event (re-emitting on retry is harmless
// since consumers dedupe by envelope id, but the request is deterministic).
func (a *Activities) PlanResearch(ctx context.Context, req DeepResearchRequest) (ResearchPlan, error) {
	attempt := int(activity.GetInfo(ctx).Attempt)
	activity.GetLogger(ctx).Info("planning research",
		"scenario", req.Scenario, "attempt", attempt)

	if attempt == 1 {
		events.PublishBusiness(ctx, a.Publisher, Pattern, TypeUserPrompt, map[string]any{
			"prompt":   req.Prompt,
			"scenario": string(req.Scenario),
		})
	}

	a.pause(planThinkTime)

	plan := ResearchPlan{
		Prompt:    req.Prompt,
		Scenario:  req.Scenario,
		Subtopics: make([]Subtopic, len(demoSubtopics)),
	}
	for i, name := range demoSubtopics {
		plan.Subtopics[i] = Subtopic{Index: i, Name: name}
	}

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypePlanReady, map[string]any{
		"subtopics": plan.Subtopics,
		"tokens":    scriptedPlanTokens,
	})
	return plan, nil
}

// GenerateQueries simulates a query-generation LLM call. Produces 2 queries
// per subtopic from the scripted table.
func (a *Activities) GenerateQueries(ctx context.Context, plan ResearchPlan) (ResearchQueries, error) {
	activity.GetLogger(ctx).Info("generating queries", "topics", len(plan.Subtopics))

	a.pause(queriesThinkTime)

	out := ResearchQueries{
		Scenario: plan.Scenario,
		Topics:   make([]TopicQueries, len(plan.Subtopics)),
	}
	for i, sub := range plan.Subtopics {
		out.Topics[i] = TopicQueries{
			TopicIndex: sub.Index,
			TopicName:  sub.Name,
			Queries:    demoQueries[sub.Name],
		}
	}

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeQueriesReady, map[string]any{
		"queries": out.Topics,
		"tokens":  scriptedQueriesTokens,
	})
	return out, nil
}

// AnnounceFanout is a fast marker activity so the UI can flip the phase to
// "research" before the children start. No sleep — the audience should see
// the transition as soon as the parent reaches the fan-out step.
func (a *Activities) AnnounceFanout(ctx context.Context, agents int) error {
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeFanoutStarted, map[string]any{
		"agents": agents,
	})
	return nil
}

// shouldFailSearch picks the two (topic, query) pairs that fail in the
// "partial" scenario — kept central so the workflow test can reason about
// which queries succeed without duplicating the table.
func shouldFailSearch(scenario Scenario, topicIndex, queryIndex int) bool {
	if scenario != ScenarioPartial {
		return false
	}
	// Every topic keeps at least one successful search, so the fan-in sees
	// three results and two of them are marked Partial.
	switch {
	case topicIndex == 0 && queryIndex == 0:
		return true
	case topicIndex == 2 && queryIndex == 1:
		return true
	}
	return false
}

// WebSearch simulates an external search call. Runs inside a child workflow,
// so business events MUST route to the parent's subject via PublishBusinessAs.
// Failures are non-retryable so the partial-scenario failure injection does
// not turn into a retry storm.
func (a *Activities) WebSearch(ctx context.Context, in SearchInput) (SearchResult, error) {
	attempt := int(activity.GetInfo(ctx).Attempt)
	activity.GetLogger(ctx).Info("web search",
		"topic", in.TopicName, "query", in.Query, "attempt", attempt)

	events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.ParentWorkflowID, in.ParentRunID,
		TypeSearchStarted, map[string]any{
			"topicIndex": in.TopicIndex,
			"topicName":  in.TopicName,
			"queryIndex": in.QueryIndex,
			"query":      in.Query,
			"attempt":    attempt,
		})

	// Alternate the 700/1000ms extremes so the fan-out on screen has a
	// staggered feel without introducing real randomness.
	dur := searchMinTime
	if (in.TopicIndex+in.QueryIndex)%2 == 0 {
		dur = searchMaxTime
	}
	a.pause(dur)

	if shouldFailSearch(in.Scenario, in.TopicIndex, in.QueryIndex) {
		msg := fmt.Sprintf("search provider rejected query %q", in.Query)
		events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.ParentWorkflowID, in.ParentRunID,
			TypeSearchFailed, map[string]any{
				"topicIndex": in.TopicIndex,
				"queryIndex": in.QueryIndex,
				"error":      msg,
			})
		return SearchResult{}, temporal.NewNonRetryableApplicationError(msg, "SearchRejected", nil)
	}

	sources := scriptedSources(in.TopicName, in.QueryIndex)
	events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.ParentWorkflowID, in.ParentRunID,
		TypeSearchComplete, map[string]any{
			"topicIndex":   in.TopicIndex,
			"queryIndex":   in.QueryIndex,
			"sourcesFound": len(sources),
			"tokens":       scriptedSearchTokens(in.TopicIndex, in.QueryIndex),
		})

	return SearchResult{
		TopicIndex: in.TopicIndex,
		QueryIndex: in.QueryIndex,
		Sources:    sources,
	}, nil
}

// RecordChildOutcome is a fast parent-side activity — it keeps the parent
// workflow free of event-emission side effects while still emitting one
// event per settled child.
func (a *Activities) RecordChildOutcome(ctx context.Context, in ChildOutcomeInput) error {
	if in.Failed {
		events.PublishBusiness(ctx, a.Publisher, Pattern, TypeChildFailed, map[string]any{
			"topicIndex": in.TopicIndex,
			"topicName":  in.TopicName,
			"error":      in.Error,
		})
		return nil
	}
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeChildCompleted, map[string]any{
		"topicIndex":   in.TopicIndex,
		"topicName":    in.TopicName,
		"sourcesFound": in.Sources,
		"partial":      in.Partial,
	})
	return nil
}

// SynthesizeReport simulates the final report-writing LLM call. It tallies
// citations/sources across the successful children and emits the closing
// business event.
func (a *Activities) SynthesizeReport(ctx context.Context, in SynthesisInput) (Report, error) {
	activity.GetLogger(ctx).Info("synthesizing report", "results", len(in.Results))

	a.pause(synthesisThinkTime)

	sourcesUsed := 0
	partialCount := 0
	for _, r := range in.Results {
		sourcesUsed += len(r.Sources)
		if r.Partial {
			partialCount++
		}
	}

	report := Report{
		Summary: fmt.Sprintf(
			"Synthesised report from %d topics (%d sources, %d partial).",
			len(in.Results), sourcesUsed, partialCount),
		Sections:     len(in.Results),
		SourcesUsed:  sourcesUsed,
		PartialCount: partialCount,
	}

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeReportReady, map[string]any{
		"summary":      report.Summary,
		"sections":     report.Sections,
		"sourcesUsed":  report.SourcesUsed,
		"partialCount": report.PartialCount,
		"tokens":       scriptedSynthesisTokens(sourcesUsed),
	})
	return report, nil
}

// Realistic-looking (non-round) token counts emitted per LLM call so the UI
// counter ticks through believable numbers instead of round multiples.
// Plan/Queries/Synthesis are parent-side LLM calls; the search table covers
// each (topicIndex, queryIndex) pair for the three scripted subtopics.
const (
	scriptedPlanTokens    = 612
	scriptedQueriesTokens = 847
)

var scriptedSearchTokensTable = [3][2]int{
	{186, 213}, // Job displacement
	{174, 241}, // New job creation
	{197, 228}, // Policy & regulation
}

func scriptedSearchTokens(topicIndex, queryIndex int) int {
	return scriptedSearchTokensTable[topicIndex][queryIndex]
}

// scriptedSynthesisTokens scales with the number of sources the synthesis
// call folds together so partial runs report a visibly lower total than the
// happy path.
func scriptedSynthesisTokens(sourcesUsed int) int {
	return 1423 + sourcesUsed*89
}

// scriptedSources returns a small, stable set of fake sources per (topic,
// queryIndex). Two per query keeps the UI dense without overwhelming it.
func scriptedSources(topicName string, queryIndex int) []Source {
	base := fmt.Sprintf("%s-q%d", topicName, queryIndex)
	return []Source{
		{
			URL:     fmt.Sprintf("https://example.com/%s/a", base),
			Title:   fmt.Sprintf("%s: analysis", topicName),
			Snippet: fmt.Sprintf("Key findings for %q.", topicName),
		},
		{
			URL:     fmt.Sprintf("https://example.com/%s/b", base),
			Title:   fmt.Sprintf("%s: policy brief", topicName),
			Snippet: fmt.Sprintf("Policy context around %q.", topicName),
		},
	}
}
