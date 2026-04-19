<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import type { SagaStartRequest, SagaStartResponse } from "~~/shared/types";

useSeoMeta({ title: "Saga" });

type FailAt = NonNullable<SagaStartRequest["failAt"]>;

const form = reactive({
  failAt: "shipping" as FailAt,
});

const workflowId = ref<string | null>(null);
const starting = ref(false);
const finalError = ref<string | null>(null);

const { events, waitForOpen } = usePatternStream("saga", workflowId);

const TERMINAL_EVENTS = new Set(["progress.workflow.completed", "progress.workflow.failed"]);

const running = computed(() => {
  if (starting.value) return true;
  if (!workflowId.value) return false;
  return !events.value.some((e) => TERMINAL_EVENTS.has(e.type));
});

function randomSuffix(): string {
  // 6-char base36 is plenty for a per-run order ID in a demo.
  return Math.random().toString(36).slice(2, 8);
}

async function start() {
  finalError.value = null;
  starting.value = true;
  const orderId = `order-${randomSuffix()}`;
  // Subscribe BEFORE starting the workflow: core NATS has no replay, and
  // the first progress.step.started (reserve-inventory) fires almost
  // immediately after start — we would miss it if the SSE stream opened
  // only after the start() response came back.
  workflowId.value = `saga-${orderId}`;
  try {
    await waitForOpen();
    await $fetch<SagaStartResponse>("/api/saga/start", {
      method: "POST",
      body: {
        customerId: "alice",
        orderId,
        amount: 1200,
        failAt: form.failAt,
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
      <h1 class="text-2xl font-semibold tracking-tight text-slate-100">
        Saga Pattern &mdash; Order Processing
      </h1>
      <div class="flex items-center gap-2">
        <select
          v-model="form.failAt"
          :disabled="running"
          class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
        >
          <option value="">No failure</option>
          <option value="inventory">Fail at reserve inventory</option>
          <option value="payment">Fail at charge payment</option>
          <option value="shipping">Fail at ship order</option>
          <option value="notification">Fail at send confirmation</option>
        </select>
        <button
          type="button"
          :disabled="running"
          class="rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:opacity-50"
          @click="start"
        >
          {{ starting ? "Starting…" : running ? "Running…" : "Run saga" }}
        </button>
      </div>
    </div>

    <!-- Architecture diagram -->
    <div class="mt-8">
      <SagaArchitecture :events="events" />
    </div>

    <!-- Pipeline -->
    <div class="mt-8 flex items-center justify-center gap-4">
      <h2 class="shrink-0 text-xs font-medium uppercase tracking-wide text-slate-400">Workflow:</h2>
      <SagaPipeline :events="events" />
    </div>

    <!-- Code + event stream -->
    <div class="mt-12 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <SagaCodeViewer :events="events" />
      </div>
      <div class="min-w-0 flex-1">
        <SagaEventStream :events="events" />
      </div>
    </div>

    <!-- Status bar -->
    <SagaStatusBar :events="events" class="mt-4" />

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>
