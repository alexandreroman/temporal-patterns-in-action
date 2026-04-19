<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
}>();

const lang = useCodeLang();

type StepKey = "dispatch" | "slot" | "drain" | "summary" | "retry";

interface BatchSource extends CodeSource {
  stepLines: Partial<Record<StepKey, [number, number]>>;
}

const SOURCES: Record<CodeLang, BatchSource> = {
  go: {
    label: "Go",
    lines: [
      "func BatchProcessingWorkflow(ctx workflow.Context, in BatchInput) (BatchResult, error) {",
      "    var a *Activities",
      "    result := BatchResult{BatchID: in.BatchID, Total: in.Total}",
      "",
      "    sem := workflow.NewSemaphore(ctx, int64(in.Parallelism))",
      "    futures := make([]workflow.Future, 0, in.Total)",
      "",
      "    // Dispatch: acquire a slot per item so at most Parallelism are in flight.",
      "    for i := 0; i < in.Total; i++ {",
      "        if err := sem.Acquire(ctx, 1); err != nil {",
      "            return result, err",
      "        }",
      "        item := ImageItem{Index: i, Service: services[i%len(services)]}",
      "        f := workflow.ExecuteActivity(ctx, a.ProcessImage, item)",
      "        futures = append(futures, f)",
      "",
      "        // Release the slot as soon as the activity settles.",
      "        workflow.Go(ctx, func(gctx workflow.Context) {",
      "            defer sem.Release(1)",
      "            _ = f.Get(gctx, nil)",
      "        })",
      "    }",
      "",
      "    // Drain: wait for every activity to report pass/fail.",
      "    for _, f := range futures {",
      "        if err := f.Get(ctx, nil); err != nil {",
      "            result.Failed++",
      "        } else {",
      "            result.Processed++",
      "        }",
      "    }",
      "",
      "    return result, workflow.ExecuteActivity(ctx, a.ReportBatchSummary, result).",
      "        Get(ctx, nil)",
      "}",
    ],
    stepLines: {
      dispatch: [7, 14],
      slot: [16, 20],
      retry: [12, 14],
      drain: [22, 29],
      summary: [31, 32],
    },
  },
  java: {
    label: "Java",
    lines: [
      "@WorkflowMethod",
      "public BatchResult processBatch(BatchInput in) {",
      "    var result = new BatchResult(in.batchId(), in.total());",
      "    var inflight = new ArrayList<Promise<Void>>();",
      "",
      "    // Dispatch: keep at most `parallelism` promises in flight.",
      "    for (int i = 0; i < in.total(); i++) {",
      "        var item = new ImageItem(i, services.get(i % services.size()));",
      "        Promise<Void> p = Async.procedure(activities::processImage, item);",
      "        inflight.add(p);",
      "",
      "        // Slot release: wait for any one to settle before the next dispatch.",
      "        if (inflight.size() >= in.parallelism()) {",
      "            Promise.anyOf(inflight).get();",
      "            inflight.removeIf(Promise::isCompleted);",
      "        }",
      "    }",
      "",
      "    // Drain: account each remaining promise in the result counters.",
      "    for (var p : inflight) {",
      "        try { p.get(); result.incProcessed(); }",
      "        catch (ActivityFailure e) { result.incFailed(); }",
      "    }",
      "",
      "    activities.reportBatchSummary(result);",
      "    return result;",
      "}",
    ],
    stepLines: {
      dispatch: [5, 9],
      slot: [11, 15],
      retry: [7, 9],
      drain: [18, 22],
      summary: [24, 25],
    },
  },
  python: {
    label: "Python",
    lines: [
      "@workflow.defn",
      "class BatchProcessingWorkflow:",
      "    @workflow.run",
      "    async def run(self, in_: BatchInput) -> BatchResult:",
      "        result = BatchResult(in_.batch_id, total=in_.total)",
      "        sem = asyncio.Semaphore(in_.parallelism)",
      "        tasks: list[asyncio.Task] = []",
      "",
      "        # Dispatch: await a free slot before every activity start.",
      "        for i in range(in_.total):",
      "            await sem.acquire()",
      "            item = ImageItem(index=i, service=SERVICES[i % len(SERVICES)])",
      "            task = asyncio.create_task(",
      "                workflow.execute_activity(",
      "                    process_image, item,",
      "                    start_to_close_timeout=timedelta(seconds=10),",
      "                    heartbeat_timeout=timedelta(seconds=5)))",
      "            task.add_done_callback(lambda _t: sem.release())",
      "            tasks.append(task)",
      "",
      "        # Drain: account each settled task on the workflow goroutine.",
      "        for t in asyncio.as_completed(tasks):",
      "            try: await t; result.processed += 1",
      "            except ActivityError: result.failed += 1",
      "",
      "        await workflow.execute_activity(",
      "            report_batch_summary, result,",
      "            start_to_close_timeout=timedelta(seconds=10))",
      "        return result",
    ],
    stepLines: {
      dispatch: [8, 18],
      slot: [10, 10],
      retry: [11, 16],
      drain: [20, 23],
      summary: [25, 28],
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
  if (latest.type === "batch.item.attempt_failed") {
    return src.stepLines.retry ?? src.stepLines.dispatch ?? null;
  }
  if (latest.type === "batch.item.started" && latest.attempt > 1) {
    return src.stepLines.retry ?? src.stepLines.dispatch ?? null;
  }
  // batch.item.started (attempt 1) or batch.item.completed → dispatch phase.
  return src.stepLines.dispatch ?? null;
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="currentHighlight" />
</template>
