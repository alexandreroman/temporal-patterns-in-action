<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Four-pill phase bar: Plan -> Query gen -> Research -> Synthesis.
 * Each phase's state is derived from the live event stream. Research
 * drops to `warn` if any search.failed was observed but the report still
 * produced (partial result path).
 */

type PhaseState = "idle" | "active" | "done" | "warn";
type PhaseId = "plan" | "queries" | "research" | "synthesis";

interface Phase {
  id: PhaseId;
  label: string;
}

const PHASES: readonly Phase[] = [
  { id: "plan", label: "Plan" },
  { id: "queries", label: "Query gen" },
  { id: "research", label: "Research" },
  { id: "synthesis", label: "Synthesis" },
];

const props = defineProps<{
  events: EventEnvelope[];
}>();

const states = computed<Record<PhaseId, PhaseState>>(() => {
  const map: Record<PhaseId, PhaseState> = {
    plan: "idle",
    queries: "idle",
    research: "idle",
    synthesis: "idle",
  };

  let anySearchFailed = false;
  let reportReady = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    const step = typeof data.step === "string" ? data.step : "";

    switch (env.type) {
      case "progress.step.started":
        if (step === "plan-research") map.plan = "active";
        else if (step === "generate-queries") map.queries = "active";
        else if (step === "synthesize-report") map.synthesis = "active";
        break;
      case "multi-agent.plan.ready":
        map.plan = "done";
        break;
      case "multi-agent.queries.ready":
        map.queries = "done";
        break;
      case "multi-agent.fanout.started":
        map.research = "active";
        break;
      case "multi-agent.search.failed":
        anySearchFailed = true;
        break;
      case "multi-agent.report.ready":
        reportReady = true;
        map.research = anySearchFailed ? "warn" : "done";
        map.synthesis = "done";
        break;
      case "progress.workflow.completed":
        if (map.plan === "idle") map.plan = "done";
        if (map.queries === "idle") map.queries = "done";
        if (map.research === "active") {
          map.research = anySearchFailed ? "warn" : "done";
        }
        if (map.synthesis === "active" || (reportReady && map.synthesis === "idle")) {
          map.synthesis = "done";
        }
        break;
      case "progress.workflow.failed":
        for (const id of ["plan", "queries", "research", "synthesis"] as PhaseId[]) {
          if (map[id] === "active") map[id] = "warn";
        }
        break;
    }
  }

  return map;
});

const CLS: Record<PhaseState, string> = {
  idle: "border-slate-200 bg-slate-50 text-slate-500 dark:border-slate-700 dark:bg-slate-800/60 dark:text-slate-400",
  active:
    "border-blue-300 bg-blue-50 text-blue-700 dark:border-blue-500 dark:bg-blue-950 dark:text-blue-200",
  done: "border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-500 dark:bg-emerald-950 dark:text-emerald-200",
  warn: "border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-500 dark:bg-amber-950 dark:text-amber-200",
};
</script>

<template>
  <!--
    At mobile: self-contained flex row.
    At lg+: `contents` so the 4 pills + 3 arrows drop directly into the
    parent grid in patterns/multi-agent.vue, sharing its column tracks
    with the agents + metrics row below so Synthesis lines up with Stats.
  -->
  <div class="flex items-center gap-1 lg:contents">
    <template v-for="(phase, idx) in PHASES" :key="phase.id">
      <span
        v-if="idx > 0"
        class="shrink-0 self-center px-1 text-xs text-slate-400 dark:text-slate-500"
        aria-hidden="true"
        >&rarr;</span
      >
      <div
        class="flex-1 rounded-md border px-2 py-1.5 text-center text-xs font-medium transition-all duration-300 lg:flex-none"
        :class="CLS[states[phase.id]]"
      >
        {{ phase.label }}
      </div>
    </template>
  </div>
</template>
