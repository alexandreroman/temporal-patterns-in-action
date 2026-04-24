<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import type { BatchStartRequest, BatchStartResponse } from "~~/shared/types";

useSeoMeta({ title: "Long-Running Batch" });

// Demo-side constants: kept in sync with the server so the UI renders the
// right `total` / `parallelism` without re-inferring from events.
const TOTAL = 48;
const PARALLELISM = 4;

type Scenario = BatchStartRequest["scenario"];

const form = reactive({
  scenario: "clean" as Scenario,
});

const workflowId = ref<string | null>(null);
const starting = ref(false);
const finalError = ref<string | null>(null);

const { events, waitForOpen } = usePatternStream("batch", workflowId);

const TERMINAL_EVENTS = new Set(["progress.workflow.completed", "progress.workflow.failed"]);

const running = computed(() => {
  if (starting.value) return true;
  if (!workflowId.value) return false;
  return !events.value.some((e) => TERMINAL_EVENTS.has(e.type));
});

async function start() {
  finalError.value = null;
  starting.value = true;
  const batchId = randomSuffix();
  // Subscribe BEFORE starting the workflow: core NATS has no replay, and
  // the first batch.item.started fires almost immediately after start —
  // we would miss it if the SSE stream opened only after the start()
  // response came back.
  workflowId.value = `batch-${batchId}`;
  try {
    await waitForOpen();
    await $fetch<BatchStartResponse>("/api/batch/start", {
      method: "POST",
      body: {
        batchId,
        scenario: form.scenario,
      },
    });
  } catch (error) {
    finalError.value = error instanceof Error ? error.message : String(error);
    workflowId.value = null;
  } finally {
    starting.value = false;
  }
}
</script>

<template>
  <section>
    <NuxtLink to="/" class="text-sm text-slate-400 hover:text-slate-100"> &larr; back </NuxtLink>

    <!-- Control bar -->
    <div class="mt-2 flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-800 bg-slate-950 text-slate-300"
        >
          <IconBatch class="h-5 w-5" />
        </span>
        <h1 class="text-2xl font-semibold tracking-tight text-slate-100">
          Long-Running Batch &mdash; Worker-Throttled Fan-Out
        </h1>
      </div>
      <div class="flex items-center gap-2">
        <select
          v-model="form.scenario"
          :disabled="running"
          class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
        >
          <option value="clean">All succeed</option>
          <option value="failures">With failures + retries</option>
        </select>
        <button
          type="button"
          :disabled="running"
          class="cursor-pointer rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
          @click="start"
        >
          {{ starting ? "Starting…" : running ? "Running…" : "Run batch" }}
        </button>
      </div>
    </div>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <BatchArchitecture :events="events" />
    </div>

    <!-- Metrics + slots: label column on the left, data on the right -->
    <div class="mt-2 grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-4">
      <h2 class="text-xs font-medium uppercase tracking-wide text-slate-400">Metrics:</h2>
      <BatchMetrics :events="events" :total="TOTAL" />

      <h2 class="text-xs font-medium uppercase tracking-wide text-slate-400">Slots:</h2>
      <BatchSlots :events="events" :parallelism="PARALLELISM" />
    </div>

    <!-- Grid -->
    <div class="mt-4">
      <BatchGrid :events="events" :total="TOTAL" />
    </div>

    <!-- Status bar -->
    <BatchStatusBar :events="events" :total="TOTAL" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <BatchCodeViewer :events="events" />
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
