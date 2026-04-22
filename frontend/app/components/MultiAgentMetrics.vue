<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Sidecar counters for the deep-research run: phase, LLM calls, web
 * searches, sources found, tokens. Token count is scripted (800 per LLM
 * call, 100 per search) since the worker does not report tokens.
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
const TOKENS_PER_LLM_CALL = 800;
const TOKENS_PER_SEARCH = 100;

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
  let anySearchFailed = false;
  let terminalCompleted = false;
  let terminalFailed = false;

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
      case "multi-agent.fanout.started":
        phase = "Research";
        break;
      case "multi-agent.search.started":
        webSearches++;
        break;
      case "multi-agent.search.completed": {
        const found = typeof data.sourcesFound === "number" ? data.sourcesFound : 0;
        sourcesFound += found;
        break;
      }
      case "multi-agent.search.failed":
        anySearchFailed = true;
        break;
      case "multi-agent.child.completed":
        if (data.partial) anySearchFailed = true;
        break;
      case "progress.workflow.completed":
        terminalCompleted = true;
        break;
      case "progress.workflow.failed":
        terminalFailed = true;
        break;
    }
  }

  if (phase === "Research" && anySearchFailed) phase = "Research (partial)";
  if (terminalCompleted) phase = "Done";
  else if (terminalFailed) phase = "Done";

  const tokens = llmCalls * TOKENS_PER_LLM_CALL + webSearches * TOKENS_PER_SEARCH;

  return { phase, llmCalls, webSearches, sourcesFound, tokens };
});

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
  { label: "Tokens", value: () => derived.value.tokens.toLocaleString() },
];
</script>

<template>
  <div class="flex w-full flex-col gap-1.5 lg:w-[220px] lg:shrink-0">
    <div
      v-for="card in CARDS"
      :key="card.label"
      class="flex flex-col rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60"
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
