<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
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
  let terminalCompleted = false;
  let terminalFailed = false;

  const addTokens = (data: Record<string, unknown>) => {
    if (typeof data.tokens === "number") tokens += data.tokens;
  };

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
        addTokens(data);
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
        addTokens(data);
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

  return { phase, llmCalls, webSearches, sourcesFound, tokens };
});

// Tween the token counter so viewers see it tick up (e.g. 612 → 1459)
// instead of snapping. Honors prefers-reduced-motion and snaps on reset
// (new run). Mirrors AgentStatePanel.useCountTween.
function useCountTween(source: () => number) {
  const displayed = ref(0);
  let frame: number | null = null;
  const cancel = () => {
    if (frame !== null) {
      cancelAnimationFrame(frame);
      frame = null;
    }
  };

  watch(
    source,
    (target, previous) => {
      cancel();
      const from = displayed.value;
      if (target === from) return;

      const reduceMotion =
        typeof window !== "undefined" &&
        window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
      if (reduceMotion || target < (previous ?? 0)) {
        displayed.value = target;
        return;
      }

      const delta = target - from;
      const duration = Math.min(800, 300 + Math.min(delta, 500));
      const start = performance.now();

      const step = (now: number) => {
        const t = Math.min(1, (now - start) / duration);
        const eased = 1 - Math.pow(1 - t, 3);
        displayed.value = Math.round(from + delta * eased);
        if (t < 1) {
          frame = requestAnimationFrame(step);
        } else {
          frame = null;
        }
      };
      frame = requestAnimationFrame(step);
    },
    { immediate: true },
  );

  onBeforeUnmount(cancel);
  return displayed;
}

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
