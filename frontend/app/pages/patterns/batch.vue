<script setup lang="ts">
import { reactive } from "vue";
import { BATCH_TOTAL } from "~~/shared/constants";
import type { BatchStartRequest, BatchStartResponse } from "~~/shared/types";

useSeoMeta({ title: "Long-Running Batch" });

// Mirrors `windowSize` in `workers/batch/workflow.go`; display-only (the
// server does not receive it).
const PARALLELISM = 4;

type Scenario = BatchStartRequest["scenario"];

const form = reactive({
  scenario: "clean" as Scenario,
});

const { events, starting, running, error: finalError, run } = usePatternRun("batch");

async function start() {
  const batchId = randomSuffix();
  await run(`batch-${batchId}`, () =>
    $fetch<BatchStartResponse>("/api/batch/start", {
      method: "POST",
      body: {
        batchId,
        scenario: form.scenario,
      },
    }),
  );
}
</script>

<template>
  <section>
    <PatternHeader
      title="Long-Running Batch — Sliding-Window Fan-Out"
      label="Run batch"
      :busy-label="starting ? 'Starting…' : 'Running…'"
      :disabled="running"
      @run="start"
    >
      <template #icon><IconBatch class="h-5 w-5" /></template>
      <select
        v-model="form.scenario"
        :disabled="running"
        class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
      >
        <option value="clean">All succeed</option>
        <option value="failures">With failures + retries</option>
      </select>
    </PatternHeader>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <BatchArchitecture :events="events" />
    </div>

    <!-- Metrics + slots: label column on the left, data on the right -->
    <div class="mt-2 grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-4">
      <h2 class="text-xs font-medium uppercase tracking-wide text-slate-400">Metrics:</h2>
      <BatchMetrics :events="events" :total="BATCH_TOTAL" />

      <h2 class="text-xs font-medium uppercase tracking-wide text-slate-400">Slots:</h2>
      <BatchSlots :events="events" :parallelism="PARALLELISM" />
    </div>

    <!-- Grid -->
    <div class="mt-4">
      <BatchGrid :events="events" :total="BATCH_TOTAL" />
    </div>

    <!-- Status bar -->
    <BatchStatusBar :events="events" :total="BATCH_TOTAL" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <BatchCodeViewer :events="events" :total="BATCH_TOTAL" />
      </div>
      <div class="min-w-0 flex-1">
        <BatchEventStream :events="events" />
      </div>
    </div>

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>
