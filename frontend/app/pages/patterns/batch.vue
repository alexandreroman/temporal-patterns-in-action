<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import type { BatchStartRequest, BatchStartResponse } from "~~/shared/types";

useSeoMeta({ title: "Long-Running Batch" });

// Demo-side constants: the server defaults match these so we do not have to
// echo them in the start payload. Declared here only so the UI components can
// render the right `total` / `parallelism` without re-inferring from events.
const TOTAL = 48;
const PARALLELISM = 4;

type Scenario = BatchStartRequest["scenario"];

const form = reactive({
  scenario: "failures" as Scenario,
});

const workflowId = ref<string | null>(null);
const starting = ref(false);
const finalError = ref<string | null>(null);

const { events, status } = usePatternStream("batch", workflowId);

const TERMINAL_EVENTS = new Set(["progress.workflow.completed", "progress.workflow.failed"]);

const running = computed(() => {
  if (starting.value) return true;
  if (!workflowId.value) return false;
  return !events.value.some((e) => TERMINAL_EVENTS.has(e.type));
});

function randomSuffix(): string {
  // 6-char base36 is plenty for a per-run batch ID in a demo.
  return Math.random().toString(36).slice(2, 8);
}

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
    await waitForStreamOpen();
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

function waitForStreamOpen(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (status.value === "open") return resolve();
    const stop = watch(status, (s) => {
      if (s === "open") {
        stop();
        resolve();
      } else if (s === "error" || s === "closed") {
        stop();
        reject(new Error(`event stream ${s}`));
      }
    });
  });
}
</script>

<template>
  <section>
    <NuxtLink to="/" class="text-sm text-slate-400 hover:text-slate-100"> &larr; back </NuxtLink>

    <!-- Control bar -->
    <div class="mt-2 flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-2xl font-semibold tracking-tight text-slate-100">
        Long-Running Batch &mdash; Parallel Sliding Window
      </h1>
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
          class="rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:opacity-50"
          @click="start"
        >
          {{ starting ? "Starting…" : running ? "Running…" : "Run batch" }}
        </button>
      </div>
    </div>

    <!-- Architecture diagram -->
    <div class="mt-8">
      <BatchArchitecture :events="events" />
    </div>

    <!-- Metrics -->
    <div class="mt-8">
      <BatchMetrics :events="events" :total="TOTAL" />
    </div>

    <!-- Slots -->
    <div class="mt-4">
      <BatchSlots :events="events" :parallelism="PARALLELISM" />
    </div>

    <!-- Grid -->
    <div class="mt-6">
      <BatchGrid :events="events" :total="TOTAL" />
    </div>

    <!-- Code + event stream -->
    <div class="mt-12 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <BatchCodeViewer :events="events" />
      </div>
      <div class="min-w-0 flex-1">
        <BatchEventStream :events="events" />
      </div>
    </div>

    <!-- Status bar -->
    <BatchStatusBar :events="events" :total="TOTAL" class="mt-4" />

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>
