<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
}>();

const lang = useCodeLang();

type StepKey = "dispatch" | "drain" | "summary";

interface BatchSource extends CodeSource {
  stepLines: Partial<Record<StepKey, [number, number]>>;
}

const SOURCES: Record<CodeLang, BatchSource> = {
  go: {
    label: "Go",
    lines: [
      "func BatchProcessingWorkflow(ctx workflow.Context, in BatchInput) (BatchResult, error) {",
      "    rootID := workflow.GetInfo(ctx).WorkflowExecution.ID",
      "    result := BatchResult{BatchID: in.BatchID, Total: in.Total}",
      "    futures := make([]workflow.Future, 0, in.Total)",
      "",
      "    // Dispatch: start every child immediately. The worker caps how many",
      "    // stage activities run in parallel via MaxConcurrentActivityExecutionSize,",
      "    // which throttles the effective sliding window.",
      "    for i := 0; i < in.Total; i++ {",
      "        childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{",
      '            WorkflowID: fmt.Sprintf("%s-item-%03d", rootID, i),',
      "        })",
      "        f := workflow.ExecuteChildWorkflow(childCtx, ProcessImageWorkflow,",
      "            ImageInput{Index: i, RootWorkflowID: rootID})",
      "        futures = append(futures, f)",
      "    }",
      "",
      "    // Drain: wait for every child to report pass/fail.",
      "    for _, f := range futures {",
      "        if err := f.Get(ctx, nil); err != nil {",
      "            result.Failed++",
      "        } else {",
      "            result.Processed++",
      "        }",
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
      "    // Stages run sequentially. Each activity is retried per the retry policy;",
      "    // only after retries are exhausted does the error surface and the parent",
      "    // counts this image as failed.",
      "    for _, stage := range []any{a.ResizeImage, a.CreateThumbnail, a.UploadToCDN, a.WriteMetadata} {",
      "        if err := workflow.ExecuteActivity(ctx, stage, stageInputFor(in)).Get(ctx, nil); err != nil {",
      "            return err",
      "        }",
      "    }",
      "    return nil",
      "}",
    ],
    stepLines: {
      dispatch: [6, 16],
      drain: [18, 25],
      summary: [28, 29],
    },
  },
  java: {
    label: "Java",
    lines: [
      "@WorkflowMethod",
      "public BatchResult processBatch(BatchInput in) {",
      "    String rootId = Workflow.getInfo().getWorkflowId();",
      "    var result = new BatchResult(in.batchId(), in.total());",
      "    var futures = new ArrayList<Promise<Void>>();",
      "",
      "    // Dispatch: start every child immediately. The worker caps how many",
      "    // stage activities run in parallel via",
      "    // WorkerOptions.setMaxConcurrentActivityExecutionSize, which throttles",
      "    // the effective sliding window.",
      "    for (int i = 0; i < in.total(); i++) {",
      "        var opts = ChildWorkflowOptions.newBuilder()",
      '            .setWorkflowId(String.format("%s-item-%03d", rootId, i))',
      "            .build();",
      "        var child = Workflow.newChildWorkflowStub(ProcessImageWorkflow.class, opts);",
      "        var input = new ImageInput(in.batchId(), rootId, i, in.failureRate());",
      "        futures.add(Async.procedure(child::processImage, input));",
      "    }",
      "",
      "    // Drain: account each child in the result counters.",
      "    for (var p : futures) {",
      "        try { p.get(); result.incProcessed(); }",
      "        catch (ChildWorkflowFailure e) { result.incFailed(); }",
      "    }",
      "",
      "    activities.reportBatchSummary(result);",
      "    return result;",
      "}",
      "",
      "@WorkflowMethod",
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
      dispatch: [7, 18],
      drain: [20, 24],
      summary: [26, 27],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      'import { executeChild, proxyActivities, workflowInfo } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const { reportBatchSummary } = proxyActivities<typeof activities>({",
      '    startToCloseTimeout: "10 seconds",',
      "});",
      "",
      "export async function batchProcessingWorkflow(in_: BatchInput): Promise<BatchResult> {",
      "    const rootId = workflowInfo().workflowId;",
      "    const result: BatchResult = { batchId: in_.batchId, total: in_.total, processed: 0, failed: 0 };",
      "    const futures: Promise<void>[] = [];",
      "",
      "    // Dispatch: start every child immediately. The worker caps how many",
      "    // stage activities run in parallel via maxConcurrentActivityTaskExecutions,",
      "    // which throttles the effective sliding window.",
      "    for (let i = 0; i < in_.total; i++) {",
      "        futures.push(",
      "            executeChild(processImageWorkflow, {",
      "                args: [{ batchId: in_.batchId, rootId, index: i, failureRate: in_.failureRate }],",
      '                workflowId: `${rootId}-item-${String(i).padStart(3, "0")}`,',
      "            })",
      "                .then(() => { result.processed++; })",
      "                .catch(() => { result.failed++; }),",
      "        );",
      "    }",
      "",
      "    // Drain: wait for every child to settle.",
      "    await Promise.all(futures);",
      "",
      "    await reportBatchSummary(result);",
      "    return result;",
      "}",
      "",
      "export async function processImageWorkflow(in_: ImageInput): Promise<void> {",
      "    const a = proxyActivities<typeof activities>({",
      '        startToCloseTimeout: "10 seconds",',
      '        retry: { initialInterval: "500ms", backoffCoefficient: 1.5, maximumAttempts: 3 },',
      "    });",
      "    // Stages run sequentially. Each activity is retried per the retry policy;",
      "    // only after retries are exhausted does the error surface and the parent",
      "    // counts this image as failed.",
      "    await a.resizeImage(in_);",
      "    await a.createThumbnail(in_);",
      "    await a.uploadToCdn(in_);",
      "    await a.writeMetadata(in_);",
      "}",
    ],
    stepLines: {
      dispatch: [12, 25],
      drain: [27, 28],
      summary: [30, 31],
    },
  },
  python: {
    label: "Python",
    lines: [
      "@workflow.defn",
      "class BatchProcessingWorkflow:",
      "    @workflow.run",
      "    async def run(self, in_: BatchInput) -> BatchResult:",
      "        root_id = workflow.info().workflow_id",
      "        result = BatchResult(in_.batch_id, total=in_.total)",
      "        tasks: list[asyncio.Task] = []",
      "",
      "        # Dispatch: start every child immediately. The worker caps how many",
      "        # stage activities run in parallel via max_concurrent_activities,",
      "        # which throttles the effective sliding window.",
      "        for i in range(in_.total):",
      "            tasks.append(asyncio.create_task(workflow.execute_child_workflow(",
      "                ProcessImageWorkflow.run,",
      "                ImageInput(in_.batch_id, root_id, i, in_.failure_rate),",
      '                id=f"{root_id}-item-{i:03d}")))',
      "",
      "        # Drain: account each settled child on the workflow task.",
      "        for t in asyncio.as_completed(tasks):",
      "            try: await t; result.processed += 1",
      "            except ChildWorkflowError: result.failed += 1",
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
      '        opts = {"start_to_close_timeout": timedelta(seconds=10)}',
      "        await workflow.execute_activity(resize_image, in_, **opts)",
      "        await workflow.execute_activity(create_thumbnail, in_, **opts)",
      "        await workflow.execute_activity(upload_to_cdn, in_, **opts)",
      "        await workflow.execute_activity(write_metadata, in_, **opts)",
    ],
    stepLines: {
      dispatch: [9, 16],
      drain: [18, 21],
      summary: [23, 25],
    },
  },
};

function latestRelevant(events: EventEnvelope[]): { type: string; attempt: number } | null {
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
      const data = env.data as Record<string, unknown>;
      const attempt = typeof data.attempt === "number" ? data.attempt : 1;
      return { type: env.type, attempt };
    }
  }
  return null;
}

const currentHighlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];
  const latest = latestRelevant(props.events);
  if (!latest) return null;

  // Terminal: clear the highlight like saga does.
  if (latest.type === "progress.workflow.completed") return null;
  if (latest.type === "progress.workflow.failed") return null;

  if (latest.type === "batch.summary.reported") {
    return src.stepLines.summary ?? null;
  }
  // batch.item.started / completed / attempt_failed → dispatch phase.
  return src.stepLines.dispatch ?? null;
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="currentHighlight" />
</template>
