package batch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"

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
		events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.RootWorkflowID, in.RootRunID, TypeItemAttemptFailed, map[string]any{
			"index":   in.Index,
			"service": in.Service,
			"attempt": 1,
			"error":   "forced first-attempt failure",
		})
		return errors.New("forced first-attempt failure")
	}
	return a.Activities.CreateThumbnail(ctx, in)
}

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

func TestBatchProcessingWorkflow_WindowBoundsInFlightChildren(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Track how many children are in flight. The started listener runs on each
	// child's goroutine, hence the mutex. The completed listener runs before the
	// parent resumes, so a finished child is uncounted before the next starts.
	var (
		mu          sync.Mutex
		inFlight    int
		maxInFlight int
	)
	env.SetOnChildWorkflowStartedListener(func(*workflow.Info, workflow.Context, converter.EncodedValues) {
		mu.Lock()
		defer mu.Unlock()
		inFlight++
		maxInFlight = max(maxInFlight, inFlight)
	})
	env.SetOnChildWorkflowCompletedListener(func(*workflow.Info, converter.EncodedValue, error) {
		mu.Lock()
		defer mu.Unlock()
		inFlight--
	})

	a := &Activities{Publisher: events.NopPublisher{}}
	env.RegisterWorkflow(ProcessImageWorkflow)
	env.RegisterActivityWithOptions(a.ResizeImage, activity.RegisterOptions{Name: "resize-image"})
	env.RegisterActivityWithOptions(a.CreateThumbnail, activity.RegisterOptions{Name: "create-thumbnail"})
	env.RegisterActivityWithOptions(a.UploadToCDN, activity.RegisterOptions{Name: "upload-cdn"})
	env.RegisterActivityWithOptions(a.WriteMetadata, activity.RegisterOptions{Name: "write-metadata"})
	env.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})

	env.ExecuteWorkflow(BatchProcessingWorkflow, BatchInput{
		BatchID:     "batch-window",
		Total:       10,
		FailureRate: 0,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result BatchResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 10, result.Processed)
	require.Equal(t, windowSize, maxInFlight, "the window must fill up but never overflow")
}

func TestBatchProcessingWorkflow_ItemFailuresDoNotFailBatch(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// The CDN stage always fails, so every item exhausts its retries.
	uploadAlwaysFails := func(ctx context.Context, in StageInput) error {
		return errors.New("cdn unavailable")
	}

	a := &Activities{Publisher: events.NopPublisher{}}
	env.RegisterWorkflow(ProcessImageWorkflow)
	env.RegisterActivityWithOptions(a.ResizeImage, activity.RegisterOptions{Name: "resize-image"})
	env.RegisterActivityWithOptions(a.CreateThumbnail, activity.RegisterOptions{Name: "create-thumbnail"})
	env.RegisterActivityWithOptions(uploadAlwaysFails, activity.RegisterOptions{Name: "upload-cdn"})
	env.RegisterActivityWithOptions(a.WriteMetadata, activity.RegisterOptions{Name: "write-metadata"})
	env.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})

	env.ExecuteWorkflow(BatchProcessingWorkflow, BatchInput{
		BatchID:     "batch-failures",
		Total:       6,
		FailureRate: 0,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError(), "item failures must not fail the batch")

	var result BatchResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 6, result.Total)
	require.Equal(t, 0, result.Processed)
	require.Equal(t, 6, result.Failed)
}
