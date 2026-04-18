<script setup lang="ts">
import { reactive, ref } from "vue";
import type { SagaStartRequest, SagaStartResponse } from "~~/shared/types";

useSeoMeta({ title: "Saga" });

type FailAt = NonNullable<SagaStartRequest["failAt"]>;

const form = reactive({
  failAt: "shipping" as FailAt,
});

const workflowId = ref<string | null>(null);
const starting = ref(false);
const finalError = ref<string | null>(null);

const { events, status } = usePatternStream("saga", workflowId);

function randomSuffix(): string {
  // 6-char base36 is plenty for a per-run order ID in a demo.
  return Math.random().toString(36).slice(2, 8);
}

async function start() {
  finalError.value = null;
  starting.value = true;
  const orderId = `order-${randomSuffix()}`;
  try {
    const response = await $fetch<SagaStartResponse>("/api/saga/start", {
      method: "POST",
      body: {
        customerId: "alice",
        orderId,
        amount: 1200,
        failAt: form.failAt,
      },
    });
    workflowId.value = response.workflowId;
  } catch (error) {
    finalError.value = error instanceof Error ? error.message : String(error);
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
        Saga pattern &mdash; live architecture
      </h1>
      <div class="flex items-center gap-2">
        <select
          v-model="form.failAt"
          class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200"
        >
          <option value="">No failure</option>
          <option value="inventory">Fail at reserve inventory</option>
          <option value="payment">Fail at charge payment</option>
          <option value="shipping">Fail at ship order</option>
          <option value="notification">Fail at send confirmation</option>
        </select>
        <button
          type="button"
          :disabled="starting"
          class="rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:opacity-50"
          @click="start"
        >
          {{ starting ? "Starting…" : "Run saga" }}
        </button>
      </div>
    </div>

    <div
      v-if="workflowId"
      class="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-slate-400"
    >
      <span>
        Workflow ID:
        <code class="rounded bg-slate-800 px-1.5 py-0.5 text-slate-200">
          {{ workflowId }}
        </code>
      </span>
      <span class="text-slate-500">stream: {{ status }}</span>
    </div>

    <!-- Architecture diagram -->
    <div class="mt-4">
      <SagaArchitecture :events="events" />
    </div>

    <!-- Pipeline -->
    <SagaPipeline :events="events" class="mt-4" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 flex-1">
        <SagaCodeViewer :events="events" />
      </div>
      <div class="lg:w-[280px] lg:shrink-0">
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
