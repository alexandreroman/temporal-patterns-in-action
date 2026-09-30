<script setup lang="ts">
import { computed, reactive } from "vue";
import type {
  PriorityFairnessSignalRequest,
  PriorityFairnessSignalResponse,
  PriorityFairnessStartRequest,
  PriorityFairnessStartResponse,
} from "~~/shared/types";
import { PRIORITIES, TENANTS } from "~/utils/priority-fairness";

useSeoMeta({ title: "Priority and Fairness" });

type Scenario = "fairness-on" | "fairness-off";

const form = reactive({
  scenario: "fairness-off" as Scenario,
});

const {
  workflowId,
  events,
  starting,
  running,
  error: finalError,
  run,
} = usePatternRun("priority-fairness");
const state = usePriorityFairnessState(events);

// Show the mode the displayed run was started with, read from its seed event:
// once a run is over, the select may already point at the next scenario.
const fairnessOn = computed(() => {
  const seeded = events.value.find((e) => e.type === "helpdesk.run.seeded");
  const runFairnessOn = (seeded?.data as { fairnessOn?: unknown } | undefined)?.fairnessOn;
  return typeof runFairnessOn === "boolean" ? runFairnessOn : form.scenario === "fairness-on";
});

async function start(): Promise<void> {
  const id = `priority-fairness-${randomSuffix()}`;
  const body: PriorityFairnessStartRequest = {
    workflowId: id,
    fairnessOn: form.scenario === "fairness-on",
  };
  await run(id, () =>
    $fetch<PriorityFairnessStartResponse>("/api/priority-fairness/start", {
      method: "POST",
      body,
    }),
  );
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
    <PatternHeader
      title="Priority and Fairness — Multi-Tenant Helpdesk"
      label="Run scenario"
      :busy-label="starting ? 'Starting…' : 'Running…'"
      :disabled="running"
      @run="start"
    >
      <template #icon><IconPriorityFairness class="h-5 w-5" /></template>
      <select
        v-model="form.scenario"
        :disabled="running"
        class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
      >
        <option value="fairness-off">Fairness OFF</option>
        <option value="fairness-on">Fairness ON</option>
      </select>
    </PatternHeader>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <PriorityFairnessArchitecture :events="events" />
    </div>

    <!-- Resolution timeline -->
    <div class="mt-3">
      <PriorityFairnessChart
        :spans="state.ticketHistory"
        :tenants="TENANTS"
        :running="running"
        @inject-incident="injectIncident"
      />
    </div>

    <!-- Two-column body -->
    <div class="mt-3 grid gap-3 lg:grid-cols-[2fr_1fr]">
      <div class="flex flex-col gap-3">
        <PriorityFairnessTenants :tenants="TENANTS" :state="state" :fairness-on="fairnessOn" />
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
      class="mt-3 flex flex-wrap items-center justify-center gap-3 rounded-xl border border-slate-200 bg-white px-3 py-2 dark:border-slate-700 dark:bg-slate-900"
    >
      <div class="flex flex-wrap items-center justify-center gap-3">
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
    </div>

    <!-- Status bar -->
    <PriorityFairnessStatusBar :events="events" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <PriorityFairnessCodeViewer :events="events" :fairness-on="fairnessOn" />
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
