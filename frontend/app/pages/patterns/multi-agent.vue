<script setup lang="ts">
import { reactive } from "vue";
import type { MultiAgentStartRequest, MultiAgentStartResponse } from "~~/shared/types";

useSeoMeta({ title: "Multi-Agent — Deep Research" });

type Scenario = MultiAgentStartRequest["scenario"];

const form = reactive({
  scenario: "happy" as Scenario,
});

const { events, starting, running, error: finalError, run } = usePatternRun("multi-agent");

async function start() {
  const runId = randomSuffix();
  await run(`multi-agent-${runId}`, () =>
    $fetch<MultiAgentStartResponse>("/api/multi-agent/start", {
      method: "POST",
      body: { runId, scenario: form.scenario },
    }),
  );
}
</script>

<template>
  <section>
    <PatternHeader
      title="Multi-Agent — Deep Research"
      label="Run research"
      :busy-label="starting ? 'Starting…' : 'Running…'"
      :disabled="running"
      @run="start"
    >
      <template #icon><IconMultiAgent class="h-5 w-5" /></template>
      <select
        v-model="form.scenario"
        :disabled="running"
        class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
      >
        <option value="happy">All succeed</option>
        <option value="partial">Partial failure</option>
      </select>
    </PatternHeader>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <MultiAgentArchitecture :events="events" />
    </div>

    <!--
      Phase bar + agents + metrics share one grid at lg+ so Synthesis
      (row 1, col 7) and Stats (row 2, col 7) land on the same column
      track — aligning the Stats panel's left edge with Synthesis.
      At mobile the outer grid collapses to a single column and each
      block stacks vertically.
    -->
    <div
      class="mt-2 grid gap-y-3 lg:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_auto_minmax(0,1fr)_auto_minmax(0,1fr)] lg:items-stretch lg:gap-x-1"
    >
      <MultiAgentPhases :events="events" />
      <!--
        Span 5 cols (not 6) so the Research→Synthesis arrow track (col 6)
        stays empty in row 2, giving the agents panel a right-edge gap
        equal to the space between adjacent phase pills. Metrics is pinned
        to col 7 explicitly; without col-start it would auto-place into
        col 6 (the empty arrow track) instead of under Synthesis.
      -->
      <MultiAgentAgents :events="events" class="lg:col-span-5" />
      <MultiAgentMetrics :events="events" class="lg:col-start-7" />
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
