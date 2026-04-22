<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import type { MultiAgentStartRequest, MultiAgentStartResponse } from "~~/shared/types";

useSeoMeta({ title: "Multi-Agent — Deep Research" });

type Scenario = MultiAgentStartRequest["scenario"];

const form = reactive({
  scenario: "happy" as Scenario,
});

const workflowId = ref<string | null>(null);
const starting = ref(false);
const finalError = ref<string | null>(null);

const { events, waitForOpen } = usePatternStream("multi-agent", workflowId);

const TERMINAL_EVENTS = new Set(["progress.workflow.completed", "progress.workflow.failed"]);

const running = computed(() => {
  if (starting.value) return true;
  if (!workflowId.value) return false;
  return !events.value.some((e) => TERMINAL_EVENTS.has(e.type));
});

function randomSuffix(): string {
  return Math.random().toString(36).slice(2, 8);
}

async function start() {
  finalError.value = null;
  starting.value = true;
  const runId = randomSuffix();
  // Subscribe BEFORE starting the workflow: core NATS has no replay, and
  // the user.prompt event fires almost immediately after start — we would
  // miss it if the SSE stream opened only after start() returned.
  workflowId.value = `multi-agent-${runId}`;
  try {
    await waitForOpen();
    await $fetch<MultiAgentStartResponse>("/api/multi-agent/start", {
      method: "POST",
      body: { runId, scenario: form.scenario },
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
          <IconMultiAgent class="h-5 w-5" />
        </span>
        <h1 class="text-2xl font-semibold tracking-tight text-slate-100">
          Multi-Agent &mdash; Deep Research
        </h1>
      </div>
      <div class="flex items-center gap-2">
        <select
          v-model="form.scenario"
          :disabled="running"
          class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
        >
          <option value="happy">All succeed</option>
          <option value="partial">Partial failure</option>
        </select>
        <button
          type="button"
          :disabled="running"
          class="cursor-pointer rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
          @click="start"
        >
          {{ starting ? "Starting…" : running ? "Running…" : "Run research" }}
        </button>
      </div>
    </div>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <MultiAgentArchitecture :events="events" />
    </div>

    <!-- Phase bar -->
    <div class="mt-2">
      <MultiAgentPhases :events="events" />
    </div>

    <!-- Agents + metrics -->
    <div class="mt-3 flex flex-col gap-3 lg:flex-row lg:items-start">
      <MultiAgentAgents :events="events" />
      <MultiAgentMetrics :events="events" />
    </div>

    <!-- Status bar -->
    <MultiAgentStatusBar :events="events" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <MultiAgentCodeViewer :events="events" />
      </div>
      <div class="min-w-0 flex-1">
        <MultiAgentEventStream :events="events" />
      </div>
    </div>

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>
