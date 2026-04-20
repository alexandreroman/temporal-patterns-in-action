package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

func registerAgentActivities(env *testsuite.TestWorkflowEnvironment, a *Activities) {
	env.RegisterActivityWithOptions(a.CallLLM, activity.RegisterOptions{Name: "call-llm"})
	env.RegisterActivityWithOptions(a.ExecuteMCPTool, activity.RegisterOptions{Name: "execute-mcp-tool"})
	env.RegisterActivityWithOptions(a.RecordApproval, activity.RegisterOptions{Name: "record-approval"})
	env.RegisterActivityWithOptions(a.ReportPlan, activity.RegisterOptions{Name: "report-plan"})
}

func TestTravelAgentWorkflow_HappyPath(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	a := &Activities{Publisher: events.NopPublisher{}, FastMode: true}
	registerAgentActivities(env, a)

	env.ExecuteWorkflow(TravelAgentWorkflow, UserRequest{
		Prompt:   "Plan a 5-day trip to Tokyo in October.",
		Scenario: ScenarioHappy,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var plan Plan
	require.NoError(t, env.GetWorkflowResult(&plan))
	require.NotEmpty(t, plan.Summary)
}

// TestTravelAgentWorkflow_ApprovalResumes verifies the workflow suspends on
// the approval phase and resumes when the signal arrives — the defining
// property of the durable-agent pattern's human-in-the-loop branch.
func TestTravelAgentWorkflow_ApprovalResumes(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	a := &Activities{Publisher: events.NopPublisher{}, FastMode: true}
	registerAgentActivities(env, a)

	// Delay in simulated time; the test env fast-forwards through sleeps so
	// this fires once the workflow is blocked on approvalCh.Receive.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalApproval, ApprovalDecision{Approved: true})
	}, 30*time.Second)

	env.ExecuteWorkflow(TravelAgentWorkflow, UserRequest{
		Prompt:   "Plan a trip needing approval.",
		Scenario: ScenarioApproval,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var plan Plan
	require.NoError(t, env.GetWorkflowResult(&plan))
	require.NotEmpty(t, plan.Summary)
}
