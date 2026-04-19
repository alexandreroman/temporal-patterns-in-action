package batch

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// recordingPublisher captures every business event published by the activities
// so tests can assert that the expected types were emitted.
type recordingPublisher struct {
	mu    sync.Mutex
	types []string
}

func (p *recordingPublisher) Publish(_ context.Context, _ string, env events.Envelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.types = append(p.types, env.Type)
	return nil
}

func (*recordingPublisher) Close() error { return nil }

func (p *recordingPublisher) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.types))
	copy(out, p.types)
	return out
}

// flakyActivities wraps Activities so the first attempt of every item fails
// deterministically — independent of FailureRate's random draw. Used by the
// retry test to assert that bounded retries carry a transient failure to
// success.
type flakyActivities struct {
	Activities
}

func (a *flakyActivities) ProcessImage(ctx context.Context, item ImageItem) error {
	if int(activity.GetInfo(ctx).Attempt) == 1 {
		a.publishBusiness(ctx, TypeItemAttemptFailed, map[string]any{
			"index":   item.Index,
			"service": item.Service,
			"attempt": 1,
			"error":   "forced first-attempt failure",
		})
		return &retryable{msg: "forced first-attempt failure"}
	}
	return a.Activities.ProcessImage(ctx, item)
}

type retryable struct{ msg string }

func (e *retryable) Error() string { return e.msg }

func TestBatchProcessingWorkflow_HappyPath(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	a := &Activities{Publisher: events.NopPublisher{}}
	env.RegisterActivityWithOptions(a.ProcessImage, activity.RegisterOptions{Name: "process-image"})
	env.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})

	env.ExecuteWorkflow(BatchProcessingWorkflow, BatchInput{
		BatchID:     "batch-happy",
		Total:       8,
		Parallelism: 2,
		FailureRate: 0,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result BatchResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 8, result.Total)
	require.Equal(t, 8, result.Processed)
	require.Equal(t, 0, result.Failed)
}

func TestBatchProcessingWorkflow_RetriesSucceed(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Speed retries up so the test does not sleep through the production
	// backoff schedule.
	env.SetTestTimeout(30 * time.Second)

	a := &flakyActivities{Activities: Activities{Publisher: events.NopPublisher{}}}
	env.RegisterActivityWithOptions(a.ProcessImage, activity.RegisterOptions{Name: "process-image"})
	env.RegisterActivityWithOptions(a.Activities.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})

	env.ExecuteWorkflow(BatchProcessingWorkflow, BatchInput{
		BatchID:     "batch-retry",
		Total:       4,
		Parallelism: 2,
		FailureRate: 0,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result BatchResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 4, result.Total)
	require.Equal(t, 4, result.Processed, "bounded retries must carry first-attempt failures to success")
	require.Equal(t, 0, result.Failed)
}

func TestBatchProcessingWorkflow_PublishesBusinessEvents(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pub := &recordingPublisher{}
	a := &Activities{Publisher: pub}
	env.RegisterActivityWithOptions(a.ProcessImage, activity.RegisterOptions{Name: "process-image"})
	env.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})

	env.ExecuteWorkflow(BatchProcessingWorkflow, BatchInput{
		BatchID:     "batch-events",
		Total:       4,
		Parallelism: 2,
		FailureRate: 0,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	published := pub.snapshot()
	require.Contains(t, published, TypeItemStarted)
	require.Contains(t, published, TypeItemCompleted)
	require.Contains(t, published, TypeSummaryReported)
}
