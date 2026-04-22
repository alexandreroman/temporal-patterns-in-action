<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Sidecar counters for the deep-research run: phase, LLM calls, web
 * searches, sources found, tokens. Token counts come from each LLM/search
 * event the worker emits so the counter ticks through realistic (non-round)
 * numbers — mirroring the agent pattern's AgentStatePanel.
 */

type Phase =
  | "Idle"
  | "Planning"
  | "Query gen"
  | "Research"
  | "Research (partial)"
  | "Synthesis"
  | "Done";

const LLM_STEPS = new Set(["plan-research", "generate-queries", "synthesize-report"]);

interface Derived {
  phase: Phase;
  llmCalls: number;
  webSearches: number;
  sourcesFound: number;
  tokens: number;
}

const props = defineProps<{
  events: EventEnvelope[];
}>();

const derived = computed<Derived>(() => {
  let phase: Phase = props.events.length === 0 ? "Idle" : "Planning";
  let llmCalls = 0;
  let webSearches = 0;
  let sourcesFound = 0;
  let tokens = 0;
  let anySearchFailed = false;
  let terminal = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    const step = typeof data.step === "string" ? data.step : "";

    switch (env.type) {
      case "progress.step.completed":
        if (LLM_STEPS.has(step)) llmCalls++;
        break;
      case "progress.step.started":
        if (step === "plan-research") phase = "Planning";
        else if (step === "generate-queries") phase = "Query gen";
        else if (step === "synthesize-report") phase = "Synthesis";
        break;
      case "multi-agent.plan.ready":
      case "multi-agent.queries.ready":
      case "multi-agent.report.ready":
        if (typeof data.tokens === "number") tokens += data.tokens;
        break;
      case "multi-agent.fanout.started":
        phase = "Research";
        break;
      case "multi-agent.search.started":
        webSearches++;
        break;
      case "multi-agent.search.completed": {
        if (typeof data.sourcesFound === "number") sourcesFound += data.sourcesFound;
        if (typeof data.tokens === "number") tokens += data.tokens;
        break;
      }
      case "multi-agent.search.failed":
        anySearchFailed = true;
        break;
      case "progress.workflow.completed":
      case "progress.workflow.failed":
        terminal = true;
        break;
    }
  }

  if (phase === "Research" && anySearchFailed) phase = "Research (partial)";
  if (terminal) phase = "Done";

  return { phase, llmCalls, webSearches, sourcesFound, tokens };
});

const displayedTokens = useCountTween(() => derived.value.tokens);

interface Card {
  label: string;
  value: () => string;
  mono?: boolean;
}

const CARDS: readonly Card[] = [
  { label: "Phase", value: () => derived.value.phase, mono: true },
  { label: "LLM calls", value: () => String(derived.value.llmCalls) },
  { label: "Web searches", value: () => String(derived.value.webSearches) },
  { label: "Sources found", value: () => String(derived.value.sourcesFound) },
  { label: "Tokens", value: () => displayedTokens.value.toLocaleString() },
];
</script>

<template>
  <!--
    Small screens: 2x2 grid of counters with Phase spanning the top row
    (5 cards total). At lg+: revert to the vertical flex stack so the
    panel lines up with the agents column.
  -->
  <div class="grid grid-cols-2 gap-1.5 lg:flex lg:h-full lg:w-full lg:flex-col">
    <div
      v-for="(card, idx) in CARDS"
      :key="card.label"
      class="flex flex-col rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60 lg:flex-1"
      :class="{ 'col-span-2 lg:col-span-1': idx === 0 }"
    >
      <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        {{ card.label }}
      </div>
      <div
        class="mt-auto text-slate-800 dark:text-slate-100 tabular-nums"
        :class="card.mono ? 'font-mono text-[12px] whitespace-nowrap' : 'text-sm font-medium'"
      >
        {{ card.value() }}
      </div>
    </div>
  </div>
</template>
