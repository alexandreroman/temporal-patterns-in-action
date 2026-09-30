<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
  total: number;
}>();

const lang = useCodeLang();

type StepKey = "dispatch" | "drain" | "summary";

interface BatchSource extends CodeSource {
  stepLines: Record<StepKey, [number, number]>;
}

const SOURCES: Record<CodeLang, BatchSource> = {
  go: {
    label: "Go",
    lines: [
      "const windowSize = 4 // max children in flight",
      "",
      "func BatchProcessingWorkflow(ctx workflow.Context, in BatchInput) (BatchResult, error) {",
      "    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{",
      "        StartToCloseTimeout: 10 * time.Second, // summary only; children set their own",
      "    })",
      "    rootID := workflow.GetInfo(ctx).WorkflowExecution.ID",
      "    result := BatchResult{BatchID: in.BatchID, Total: in.Total}",
      "    selector := workflow.NewSelector(ctx)",
      "    inFlight := 0",
      "",
      "    // Sliding window: start a child only when a slot is free.",
      "    for i := range in.Total {",
      "        if inFlight == windowSize {",
      "            selector.Select(ctx) // wait for a child to finish, freeing a slot",
      "            inFlight--",
      "        }",
      "        childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{",
      '            WorkflowID: fmt.Sprintf("%s-item-%03d", rootID, i),',
      "        })",
      "        f := workflow.ExecuteChildWorkflow(childCtx, ProcessImageWorkflow, ImageInput{",
      "            BatchID: in.BatchID, RootWorkflowID: rootID, Index: i, FailureRate: in.FailureRate,",
      "        })",
      "        selector.AddFuture(f, func(f workflow.Future) {",
      "            if err := f.Get(ctx, nil); err != nil {",
      "                result.Failed++",
      "            } else {",
      "                result.Processed++",
      "            }",
      "        })",
      "        inFlight++",
      "    }",
      "",
      "    // Drain the last children still in the window.",
      "    for ; inFlight > 0; inFlight-- {",
      "        selector.Select(ctx)",
      "    }",
      "",
      "    var a *Activities",
      "    return result, workflow.ExecuteActivity(ctx, a.ReportBatchSummary, result).",
      "        Get(ctx, nil)",
      "}",
      "",
      "func ProcessImageWorkflow(ctx workflow.Context, in ImageInput) error {",
      "    ctx = workflow.WithActivityOptions(ctx, stageActivityOptions())",
      "    var a *Activities",
      "    stageIn := StageInput{BatchID: in.BatchID, Index: in.Index, FailureRate: in.FailureRate}",
      "    // Stages run sequentially. Each activity is retried per the retry policy;",
      "    // only after retries are exhausted does the error surface and the parent",
      "    // counts this image as failed.",
      "    for _, stage := range []any{a.ResizeImage, a.CreateThumbnail, a.UploadToCDN, a.WriteMetadata} {",
      "        if err := workflow.ExecuteActivity(ctx, stage, stageIn).Get(ctx, nil); err != nil {",
      "            return err",
      "        }",
      "    }",
      "    return nil",
      "}",
    ],
    stepLines: {
      dispatch: [11, 31],
      drain: [33, 36],
      summary: [38, 40],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// BatchProcessingWorkflowImpl",
      "private static final int WINDOW_SIZE = 4; // max children in flight",
      "private final Activities activities = Workflow.newActivityStub(Activities.class,",
      "    ActivityOptions.newBuilder().setStartToCloseTimeout(Duration.ofSeconds(10)).build());",
      "private int inFlight = 0;",
      "",
      "@Override",
      "public BatchResult processBatch(BatchInput in) {",
      "    String rootId = Workflow.getInfo().getWorkflowId();",
      "    var result = new BatchResult(in.batchId(), in.total());",
      "",
      "    // Sliding window: start a child only when a slot is free.",
      "    for (int i = 0; i < in.total(); i++) {",
      "        Workflow.await(() -> inFlight < WINDOW_SIZE);",
      "        var opts = ChildWorkflowOptions.newBuilder()",
      '            .setWorkflowId(String.format("%s-item-%03d", rootId, i))',
      "            .build();",
      "        var child = Workflow.newChildWorkflowStub(ProcessImageWorkflow.class, opts);",
      "        var input = new ImageInput(in.batchId(), rootId, i, in.failureRate());",
      "        inFlight++;",
      "        Async.procedure(child::processImage, input).handle((ignored, failure) -> {",
      "            if (failure == null) {",
      "                result.incProcessed();",
      "            } else {",
      "                result.incFailed();",
      "            }",
      "            inFlight--;",
      "            return null;",
      "        });",
      "    }",
      "",
      "    // Drain the last children still in the window.",
      "    Workflow.await(() -> inFlight == 0);",
      "",
      "    activities.reportBatchSummary(result);",
      "    return result;",
      "}",
      "",
      "// ProcessImageWorkflowImpl (a separate workflow implementation)",
      "private final Activities activities = Workflow.newActivityStub(Activities.class,",
      "    ActivityOptions.newBuilder()",
      "        .setStartToCloseTimeout(Duration.ofSeconds(10))",
      "        .setRetryOptions(RetryOptions.newBuilder()",
      "            .setInitialInterval(Duration.ofMillis(500))",
      "            .setBackoffCoefficient(1.5)",
      "            .setMaximumAttempts(3)",
      "            .build())",
      "        .build());",
      "",
      "@Override",
      "public void processImage(ImageInput in) {",
      "    // Stages run sequentially. Each activity is retried per the retry policy;",
      "    // only after retries are exhausted does the error surface and the parent",
      "    // counts this image as failed.",
      "    activities.resizeImage(in);",
      "    activities.createThumbnail(in);",
      "    activities.uploadToCdn(in);",
      "    activities.writeMetadata(in);",
      "}",
    ],
    stepLines: {
      dispatch: [11, 29],
      drain: [31, 32],
      summary: [34, 34],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      'import { condition, executeChild, proxyActivities, workflowInfo } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const WINDOW_SIZE = 4; // max children in flight",
      "",
      "const { reportBatchSummary } = proxyActivities<typeof activities>({",
      '    startToCloseTimeout: "10 seconds",',
      "});",
      "",
      "export async function batchProcessingWorkflow(input: BatchInput): Promise<BatchResult> {",
      "    const rootId = workflowInfo().workflowId;",
      "    const result: BatchResult = { batchId: input.batchId, total: input.total, processed: 0, failed: 0 };",
      "    let inFlight = 0;",
      "",
      "    // Sliding window: start a child only when a slot is free.",
      "    for (let i = 0; i < input.total; i++) {",
      "        await condition(() => inFlight < WINDOW_SIZE);",
      "        inFlight++;",
      "        void executeChild(processImageWorkflow, {",
      "            args: [{ batchId: input.batchId, rootId, index: i, failureRate: input.failureRate }],",
      '            workflowId: `${rootId}-item-${String(i).padStart(3, "0")}`,',
      "        })",
      "            .then(",
      "                () => { result.processed++; },",
      "                () => { result.failed++; },",
      "            )",
      "            .finally(() => { inFlight--; });",
      "    }",
      "",
      "    // Drain the last children still in the window.",
      "    await condition(() => inFlight === 0);",
      "",
      "    await reportBatchSummary(result);",
      "    return result;",
      "}",
      "",
      "export async function processImageWorkflow(input: ImageInput): Promise<void> {",
      "    const a = proxyActivities<typeof activities>({",
      '        startToCloseTimeout: "10 seconds",',
      '        retry: { initialInterval: "500ms", backoffCoefficient: 1.5, maximumAttempts: 3 },',
      "    });",
      "    // Stages run sequentially. Each activity is retried per the retry policy;",
      "    // only after retries are exhausted does the error surface and the parent",
      "    // counts this image as failed.",
      "    await a.resizeImage(input);",
      "    await a.createThumbnail(input);",
      "    await a.uploadToCdn(input);",
      "    await a.writeMetadata(input);",
      "}",
    ],
    stepLines: {
      dispatch: [14, 27],
      drain: [29, 30],
      summary: [32, 32],
    },
  },
  python: {
    label: "Python",
    lines: [
      "WINDOW_SIZE = 4  # max children in flight",
      "",
      "@workflow.defn",
      "class BatchProcessingWorkflow:",
      "    def __init__(self) -> None:",
      "        self._in_flight = 0",
      "",
      "    @workflow.run",
      "    async def run(self, in_: BatchInput) -> BatchResult:",
      "        root_id = workflow.info().workflow_id",
      "        result = BatchResult(in_.batch_id, total=in_.total)",
      "",
      "        async def process(i: int) -> None:",
      "            try:",
      "                await workflow.execute_child_workflow(",
      "                    ProcessImageWorkflow.run,",
      "                    ImageInput(in_.batch_id, root_id, i, in_.failure_rate),",
      '                    id=f"{root_id}-item-{i:03d}")',
      "                result.processed += 1",
      "            except ChildWorkflowError:",
      "                result.failed += 1",
      "            finally:",
      "                self._in_flight -= 1",
      "",
      "        # Sliding window: start a child only when a slot is free.",
      "        for i in range(in_.total):",
      "            await workflow.wait_condition(lambda: self._in_flight < WINDOW_SIZE)",
      "            self._in_flight += 1",
      "            asyncio.create_task(process(i))",
      "",
      "        # Drain the last children still in the window.",
      "        await workflow.wait_condition(lambda: self._in_flight == 0)",
      "",
      "        await workflow.execute_activity(",
      "            report_batch_summary, result,",
      "            start_to_close_timeout=timedelta(seconds=10))",
      "        return result",
      "",
      "@workflow.defn",
      "class ProcessImageWorkflow:",
      "    @workflow.run",
      "    async def run(self, in_: ImageInput) -> None:",
      "        # Stages run sequentially. Each activity is retried per the retry policy;",
      "        # only after retries are exhausted does the ActivityError surface and the",
      "        # parent counts this image as failed.",
      "        opts = {",
      '            "start_to_close_timeout": timedelta(seconds=10),',
      '            "retry_policy": RetryPolicy(',
      "                initial_interval=timedelta(milliseconds=500),",
      "                backoff_coefficient=1.5,",
      "                maximum_attempts=3,",
      "            ),",
      "        }",
      "        await workflow.execute_activity(resize_image, in_, **opts)",
      "        await workflow.execute_activity(create_thumbnail, in_, **opts)",
      "        await workflow.execute_activity(upload_to_cdn, in_, **opts)",
      "        await workflow.execute_activity(write_metadata, in_, **opts)",
    ],
    stepLines: {
      dispatch: [12, 28],
      drain: [30, 31],
      summary: [33, 35],
    },
  },
};

function latestRelevant(events: EventEnvelope[]): string | null {
  for (let i = events.length - 1; i >= 0; i--) {
    const env = events[i];
    if (!env) continue;
    if (
      env.type === "progress.workflow.completed" ||
      env.type === "progress.workflow.failed" ||
      env.type === "batch.summary.reported" ||
      env.type === "batch.item.started" ||
      env.type === "batch.item.completed" ||
      env.type === "batch.item.attempt_failed"
    ) {
      return env.type;
    }
  }
  return null;
}

const lastChildStarted = computed(() =>
  props.events.some((env) => {
    const data = env.data as Record<string, unknown>;
    return env.type === "batch.item.started" && data.index === props.total - 1;
  }),
);

const currentHighlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];
  const latest = latestRelevant(props.events);
  if (!latest) return null;

  // The summary is the workflow's last step, so it stays lit once the
  // workflow completes or fails.
  if (
    latest === "batch.summary.reported" ||
    latest === "progress.workflow.completed" ||
    latest === "progress.workflow.failed"
  ) {
    return src.stepLines.summary;
  }
  // batch.item.* events: the window loop runs until the last child has been
  // started; after that the workflow is draining the window.
  if (lastChildStarted.value) {
    return src.stepLines.drain;
  }
  return src.stepLines.dispatch;
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="currentHighlight" />
</template>
