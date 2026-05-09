<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import type {
  PriorityFairnessSignalRequest,
  PriorityFairnessSignalResponse,
  PriorityFairnessStartRequest,
  PriorityFairnessStartResponse,
} from "~~/shared/types";
import { PRIORITIES, TENANTS } from "~/utils/priority-fairness";

useSeoMeta({ title: "Priority and Fairness" });

type Scenario = "fairness-on" | "fairness-off";

const TERMINAL_TYPES = new Set(["progress.workflow.completed", "progress.workflow.failed"]);

const form = reactive({
  scenario: "fairness-off" as Scenario,
});

const workflowId = ref<string | null>(null);
const starting = ref(false);
const finalError = ref<string | null>(null);

const { events, waitForOpen } = usePatternStream("priority-fairness", workflowId);
const state = usePriorityFairnessState(events);

const running = computed(() => {
  if (starting.value) return true;
  if (!workflowId.value) return false;
  return !events.value.some((e) => TERMINAL_TYPES.has(e.type));
});

async function start(): Promise<void> {
  finalError.value = null;
  starting.value = true;
  const fairnessOn = form.scenario === "fairness-on";
  const id = `priority-fairness-${randomSuffix()}`;
  workflowId.value = id;
  try {
    await waitForOpen();
    await $fetch<PriorityFairnessStartResponse>("/api/priority-fairness/start", {
      method: "POST",
      body: { workflowId: id, fairnessOn } satisfies PriorityFairnessStartRequest,
    });
  } catch (error) {
    finalError.value = error instanceof Error ? error.message : String(error);
    workflowId.value = null;
  } finally {
    starting.value = false;
  }
}

async function dumpAcme(): Promise<void> {
  const id = workflowId.value;
  if (!id || !running.value) return;
  try {
    await $fetch<PriorityFairnessSignalResponse>("/api/priority-fairness/dump", {
      method: "POST",
      body: { workflowId: id } satisfies PriorityFairnessSignalRequest,
    });
  } catch (error) {
    finalError.value = error instanceof Error ? error.message : String(error);
  }
}

async function injectIncident(): Promise<void> {
  const id = workflowId.value;
  if (!id || !running.value) return;
  try {
    await $fetch<PriorityFairnessSignalResponse>("/api/priority-fairness/incident", {
      method: "POST",
      body: { workflowId: id } satisfies PriorityFairnessSignalRequest,
    });
  } catch (error) {
    finalError.value = error instanceof Error ? error.message : String(error);
  }
}
</script>

<template>
  <section>
    <NuxtLink to="/" class="text-sm text-slate-400 hover:text-slate-100"> &larr; back </NuxtLink>

    <!-- Header row -->
    <div class="mt-2 flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-800 bg-slate-950 text-slate-300"
        >
          <IconPriorityFairness class="h-5 w-5" />
        </span>
        <div class="flex flex-col">
          <h1 class="text-2xl font-semibold tracking-tight text-slate-100">
            Priority and Fairness &mdash; Multi-Tenant Helpdesk
          </h1>
          <p class="text-xs text-slate-400">
            4 agents &middot; 3 tenants &middot; 4 priority levels
          </p>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <select
          v-model="form.scenario"
          :disabled="running"
          class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
        >
          <option value="fairness-off">Fairness off (Acme starves)</option>
          <option value="fairness-on">Fairness on (proportional sharing)</option>
        </select>
        <button
          type="button"
          :disabled="!running"
          class="cursor-pointer rounded-md bg-slate-700 px-3 py-1.5 text-xs font-medium text-slate-100 transition-colors hover:bg-slate-600 disabled:cursor-not-allowed disabled:opacity-50"
          @click="dumpAcme"
        >
          Acme dumps 80
        </button>
        <button
          type="button"
          :disabled="!running"
          class="cursor-pointer rounded-md bg-slate-700 px-3 py-1.5 text-xs font-medium text-slate-100 transition-colors hover:bg-slate-600 disabled:cursor-not-allowed disabled:opacity-50"
          @click="injectIncident"
        >
          + P0 incident
        </button>
        <button
          type="button"
          :disabled="running"
          class="cursor-pointer rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
          @click="start"
        >
          {{ starting ? "Starting…" : running ? "Running…" : "Run scenario" }}
        </button>
      </div>
    </div>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <PriorityFairnessArchitecture :events="events" />
    </div>

    <!-- Resolution timeline -->
    <div class="mt-3">
      <PriorityFairnessChart :spans="state.ticketHistory" :tenants="TENANTS" :running="running" />
    </div>

    <!-- Two-column body -->
    <div class="mt-3 grid gap-3 lg:grid-cols-[2fr_1fr]">
      <div class="flex flex-col gap-3">
        <PriorityFairnessTenants :tenants="TENANTS" :state="state" />
        <PriorityFairnessWorkers :agents="state.agents" />
      </div>
      <div class="lg:relative">
        <PriorityFairnessLog
          class="max-lg:max-h-[380px] lg:absolute lg:inset-0"
          :log="state.log"
          :start-time="state.startTime"
        />
      </div>
    </div>

    <!-- Legend bar -->
    <div
      class="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-3 py-2 dark:border-slate-700 dark:bg-slate-900"
    >
      <div class="flex flex-wrap items-center gap-3">
        <span
          v-for="p in PRIORITIES"
          :key="p.key"
          class="inline-flex items-center gap-1.5 text-[11px]"
        >
          <span
            class="rounded-md px-1.5 py-0.5 font-mono text-[10px] tabular-nums"
            :style="{ backgroundColor: p.bg, color: p.fg }"
          >
            {{ p.label }}
          </span>
          <span class="text-slate-500 dark:text-slate-400">{{ p.meaning }}</span>
        </span>
      </div>
      <div class="font-mono text-[11px] text-slate-500 dark:text-slate-400">
        fairnessKey = tenant &middot; weight = contract tier
      </div>
    </div>

    <!-- Status bar -->
    <PriorityFairnessStatusBar :events="events" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <PriorityFairnessCodeViewer :events="events" />
      </div>
      <div class="min-w-0 flex-1">
        <PriorityFairnessEventStream :events="events" />
      </div>
    </div>

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>
