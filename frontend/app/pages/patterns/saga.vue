<script setup lang="ts">
import { reactive } from "vue";
import type { SagaStartRequest, SagaStartResponse } from "~~/shared/types";

useSeoMeta({ title: "Saga" });

type FailAt = NonNullable<SagaStartRequest["failAt"]>;

const form = reactive({
  failAt: "" as FailAt,
});

const { events, starting, running, error: finalError, run } = usePatternRun("saga");

async function start() {
  const orderId = `order-${randomSuffix()}`;
  await run(`saga-${orderId}`, () =>
    $fetch<SagaStartResponse>("/api/saga/start", {
      method: "POST",
      body: {
        customerId: "alice",
        orderId,
        amount: 1200,
        failAt: form.failAt,
      },
    }),
  );
}
</script>

<template>
  <section>
    <PatternHeader
      title="Saga Pattern — Order Processing"
      label="Run saga"
      :busy-label="starting ? 'Starting…' : 'Running…'"
      :disabled="running"
      @run="start"
    >
      <template #icon><IconSaga class="h-5 w-5" /></template>
      <select
        v-model="form.failAt"
        :disabled="running"
        class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
      >
        <option value="">No failure</option>
        <option value="fraud">Fail at check fraud</option>
        <option value="shipment">Fail at prepare shipment</option>
        <option value="charge">Fail at charge customer</option>
        <option value="notification">Fail at send confirmation</option>
      </select>
    </PatternHeader>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <SagaArchitecture :events="events" />
    </div>

    <!-- Pipeline -->
    <div class="mt-2 flex items-center justify-center gap-4">
      <h2 class="shrink-0 text-xs font-medium uppercase tracking-wide text-slate-400">Workflow:</h2>
      <SagaPipeline :events="events" />
    </div>

    <!-- Status bar -->
    <SagaStatusBar :events="events" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <SagaCodeViewer :events="events" />
      </div>
      <div class="min-w-0 flex-1">
        <SagaEventStream :events="events" />
      </div>
    </div>

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>
