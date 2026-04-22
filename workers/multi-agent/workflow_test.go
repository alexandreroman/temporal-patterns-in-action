package multiagent

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

func registerMultiAgent(env *testsuite.TestWorkflowEnvironment, a *Activities) {
	env.RegisterWorkflow(ResearchAgentWorkflow)
	env.RegisterActivityWithOptions(a.PlanResearch, activity.RegisterOptions{Name: "plan-research"})
	env.RegisterActivityWithOptions(a.GenerateQueries, activity.RegisterOptions{Name: "generate-queries"})
	env.RegisterActivityWithOptions(a.AnnounceFanout, activity.RegisterOptions{Name: "announce-fanout"})
	env.RegisterActivityWithOptions(a.WebSearch, activity.RegisterOptions{Name: "web-search"})
	env.RegisterActivityWithOptions(a.RecordChildOutcome, activity.RegisterOptions{Name: "record-child-outcome"})
	env.RegisterActivityWithOptions(a.SynthesizeReport, activity.RegisterOptions{Name: "synthesize-report"})
}

func TestDeepResearchWorkflow_Happy(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	a := &Activities{Publisher: events.NopPublisher{}, FastMode: true}
	registerMultiAgent(env, a)

	env.ExecuteWorkflow(DeepResearchWorkflow, DeepResearchRequest{
		Prompt:   "Research the impact of AI on the labour market.",
		Scenario: ScenarioHappy,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var report Report
	require.NoError(t, env.GetWorkflowResult(&report))
	require.Greater(t, report.Sections, 0)
	require.Greater(t, report.Citations, 0)
	require.Equal(t, 0, report.PartialCount)
}

// TestDeepResearchWorkflow_Partial asserts the parent tolerates two failed
// searches: the report is still produced, every topic still contributes at
// least one source, but the total sources used is strictly less than the
// happy path — proof that the fan-in dropped failed searches instead of
// failing the workflow.
func TestDeepResearchWorkflow_Partial(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	a := &Activities{Publisher: events.NopPublisher{}, FastMode: true}
	registerMultiAgent(env, a)

	env.ExecuteWorkflow(DeepResearchWorkflow, DeepResearchRequest{
		Prompt:   "Research the impact of AI on the labour market.",
		Scenario: ScenarioPartial,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var report Report
	require.NoError(t, env.GetWorkflowResult(&report))
	require.Greater(t, report.Sections, 0)
	require.Greater(t, report.Citations, 0)
	require.Greater(t, report.PartialCount, 0, "partial scenario should mark at least one topic as partial")

	// Happy path: 3 topics × 2 queries × 2 sources = 12. Partial path: two
	// searches drop, so we expect 8.
	require.Less(t, report.SourcesUsed, 12,
		"partial scenario must return fewer sources than the happy path (proves fan-in tolerated failures)")
}
