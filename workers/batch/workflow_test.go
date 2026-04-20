package batch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// flakyActivities wraps Activities so the first attempt of every thumbnail
// stage fails deterministically — independent of FailureRate's random draw.
// Used by the retry test to assert that bounded retries carry a transient
// stage failure to success. Picking a middle stage (not the first or last)
// exercises the retry path without hiding in boundary behaviour.
type flakyActivities struct {
	Activities
}

func (a *flakyActivities) CreateThumbnail(ctx context.Context, in StageInput) error {
	if int(activity.GetInfo(ctx).Attempt) == 1 {
		runID := activity.GetInfo(ctx).WorkflowExecution.RunID
		events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.RootWorkflowID, runID, TypeItemAttemptFailed, map[string]any{
			"index":   in.Index,
			"service": in.Service,
			"attempt": 1,
			"error":   "forced first-attempt failure",
		})
		return &retryable{msg: "forced first-attempt failure"}
	}
	return a.Activities.CreateThumbnail(ctx, in)
}

type retryable struct{ msg string }

func (e *retryable) Error() string { return e.msg }

func TestBatchProcessingWorkflow_HappyPath(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	a := &Activities{Publisher: events.NopPublisher{}}
	env.RegisterWorkflow(ProcessImageWorkflow)
	env.RegisterActivityWithOptions(a.ResizeImage, activity.RegisterOptions{Name: "resize-image"})
	env.RegisterActivityWithOptions(a.CreateThumbnail, activity.RegisterOptions{Name: "create-thumbnail"})
	env.RegisterActivityWithOptions(a.UploadToCDN, activity.RegisterOptions{Name: "upload-cdn"})
	env.RegisterActivityWithOptions(a.WriteMetadata, activity.RegisterOptions{Name: "write-metadata"})
	env.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})

	env.ExecuteWorkflow(BatchProcessingWorkflow, BatchInput{
		BatchID:     "batch-happy",
		Total:       8,
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
	env.RegisterWorkflow(ProcessImageWorkflow)
	env.RegisterActivityWithOptions(a.ResizeImage, activity.RegisterOptions{Name: "resize-image"})
	env.RegisterActivityWithOptions(a.CreateThumbnail, activity.RegisterOptions{Name: "create-thumbnail"})
	env.RegisterActivityWithOptions(a.UploadToCDN, activity.RegisterOptions{Name: "upload-cdn"})
	env.RegisterActivityWithOptions(a.WriteMetadata, activity.RegisterOptions{Name: "write-metadata"})
	env.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})

	env.ExecuteWorkflow(BatchProcessingWorkflow, BatchInput{
		BatchID:     "batch-retry",
		Total:       4,
		FailureRate: 0,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result BatchResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 4, result.Total)
	require.Equal(t, 4, result.Processed, "bounded retries must carry first-attempt stage failures to success")
	require.Equal(t, 0, result.Failed)
}
