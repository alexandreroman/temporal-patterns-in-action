<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Sidecar panel that tracks the four headline counters of the agent loop
 * (phase, LLM calls, tool calls, tokens) plus the catalogue of MCP tools —
 * highlighting the one currently in flight. When a retry is detected, a
 * full "Without Temporal" twin group (LLM calls, Tool calls, Tokens used)
 * fades in below the originals in amber, showing the work a naive
 * (non-durable) agent would have had to redo by restarting the loop from
 * scratch. Everything is derived from the live event stream; no state is
 * mirrored from Temporal queries.
 */

const TOOLS = [
  "search_flights",
  "search_hotels",
  "get_calendar",
  "book_flight",
  "book_hotel",
  "send_itinerary",
] as const;

type PhaseLabel = "Idle" | "LLM" | "Tool" | "Approval" | "Done";

const props = defineProps<{
  events: EventEnvelope[];
}>();

interface Derived {
  phase: PhaseLabel;
  llmCalls: number;
  toolCalls: number;
  tokens: number;
  naiveLLMCalls: number;
  naiveToolCalls: number;
  naiveTokens: number;
  hasRetry: boolean;
  activeTool: string | null;
}

const derived = computed<Derived>(() => {
  let phase: PhaseLabel = props.events.length === 0 ? "Idle" : "LLM";
  let llmCalls = 0;
  let toolCalls = 0;
  let tokens = 0;
  // Work a naive agent would redo by restarting the loop on each retry.
  let replayLLM = 0;
  let replayTool = 0;
  let replayTokens = 0;
  let activeTool: string | null = null;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    switch (env.type) {
      case "progress.step.started":
        if (data.step === "call-llm") phase = "LLM";
        else if (data.step === "execute-mcp-tool") phase = "Tool";
        break;
      case "progress.step.failed":
        // A naive (non-durable) agent would have to restart the whole loop,
        // repaying every call and token spent before the failure.
        if (data.step === "call-llm") {
          replayLLM += llmCalls;
          replayTool += toolCalls;
          replayTokens += tokens;
        }
        break;
      case "agent.llm.responded": {
        llmCalls++;
        const t = typeof data.tokens === "number" ? data.tokens : 0;
        tokens += t;
        break;
      }
      case "agent.tool.started":
        activeTool = typeof data.name === "string" ? data.name : null;
        phase = "Tool";
        break;
      case "agent.tool.completed":
        toolCalls++;
        activeTool = null;
        break;
      case "agent.approval.requested":
        phase = "Approval";
        break;
      case "agent.approval.received":
        phase = "LLM";
        break;
      case "progress.workflow.completed":
        phase = "Done";
        activeTool = null;
        break;
      case "progress.workflow.failed":
        phase = "Done";
        activeTool = null;
        break;
    }
  }

  return {
    phase,
    llmCalls,
    toolCalls,
    tokens,
    naiveLLMCalls: llmCalls + replayLLM,
    naiveToolCalls: toolCalls + replayTool,
    naiveTokens: tokens + replayTokens,
    hasRetry: replayTokens > 0,
    activeTool,
  };
});

// Tween a counter so viewers see it tick up (e.g. 50 → 100) instead of
// snapping. Honors prefers-reduced-motion and snaps on reset (new run).
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
      // Scale duration with jump size so big leaps feel proportional, capped at 800ms.
      const duration = Math.min(800, 300 + Math.min(delta, 500));
      const start = performance.now();

      const step = (now: number) => {
        const t = Math.min(1, (now - start) / duration);
        const eased = 1 - Math.pow(1 - t, 3); // easeOutCubic
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
const displayedNaiveTokens = useCountTween(() => derived.value.naiveTokens);
</script>

<template>
  <div class="flex w-full flex-col gap-1.5 lg:w-[220px] lg:shrink-0">
    <div
      class="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60 flex flex-col"
    >
      <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Agent phase
      </div>
      <div
        class="mt-auto font-mono text-[12px] text-slate-800 dark:text-slate-100 whitespace-nowrap"
      >
        {{ derived.phase }}
      </div>
    </div>

    <div class="grid grid-cols-2 gap-2">
      <div
        class="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60 flex flex-col"
      >
        <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
          LLM calls
        </div>
        <div class="mt-auto text-sm font-medium text-slate-800 dark:text-slate-100">
          {{ derived.llmCalls }}
        </div>
      </div>
      <div
        class="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60 flex flex-col"
      >
        <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
          Tool calls
        </div>
        <div class="mt-auto text-sm font-medium text-slate-800 dark:text-slate-100">
          {{ derived.toolCalls }}
        </div>
      </div>
    </div>

    <div
      class="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60 flex flex-col"
    >
      <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Tokens used
      </div>
      <div class="mt-auto text-sm font-medium text-slate-800 dark:text-slate-100 tabular-nums">
        {{ displayedTokens.toLocaleString() }}
      </div>
    </div>

    <!--
      "Without Temporal" twin group: when a retry happens, a full amber
      copy of the three counters fades in below the originals so viewers
      can compare the durable run against what a naive agent would have
      had to redo.
    -->
    <Transition
      enter-active-class="transition-all duration-500 ease-out"
      enter-from-class="opacity-0 -translate-y-2 scale-95"
      enter-to-class="opacity-100 translate-y-0 scale-100"
      leave-active-class="transition-all duration-300 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div v-if="derived.hasRetry" class="flex flex-col gap-1.5">
        <div
          class="text-[10px] uppercase tracking-wide text-amber-700 dark:text-amber-300 text-center"
        >
          Without Temporal
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div
            class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 dark:border-amber-700 dark:bg-amber-950/40 flex flex-col"
          >
            <div
              class="text-[10px] uppercase tracking-wide text-amber-700 dark:text-amber-300 whitespace-nowrap"
            >
              LLM calls
            </div>
            <div class="mt-auto text-sm font-medium text-amber-800 dark:text-amber-200">
              {{ derived.naiveLLMCalls }}
            </div>
          </div>
          <div
            class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 dark:border-amber-700 dark:bg-amber-950/40 flex flex-col"
          >
            <div
              class="text-[10px] uppercase tracking-wide text-amber-700 dark:text-amber-300 whitespace-nowrap"
            >
              Tool calls
            </div>
            <div class="mt-auto text-sm font-medium text-amber-800 dark:text-amber-200">
              {{ derived.naiveToolCalls }}
            </div>
          </div>
        </div>
        <div
          class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 dark:border-amber-700 dark:bg-amber-950/40 flex flex-col"
        >
          <div
            class="text-[10px] uppercase tracking-wide text-amber-700 dark:text-amber-300 whitespace-nowrap"
          >
            Tokens used
          </div>
          <div class="mt-auto text-sm font-medium text-amber-800 dark:text-amber-200 tabular-nums">
            {{ displayedNaiveTokens.toLocaleString() }}
          </div>
        </div>
      </div>
    </Transition>

    <div
      class="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60 lg:flex-1 lg:min-h-0"
    >
      <div class="mb-1 text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        MCP tools
      </div>
      <div class="flex flex-wrap gap-1">
        <span
          v-for="t in TOOLS"
          :key="t"
          class="rounded-md border px-1.5 py-0.5 font-mono text-[10px] transition-all duration-200"
          :class="
            derived.activeTool === t
              ? 'border-blue-300 bg-blue-50 text-blue-700 dark:border-blue-500 dark:bg-blue-950 dark:text-blue-200'
              : 'border-slate-200 bg-white text-slate-600 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300'
          "
        >
          {{ t }}
        </span>
      </div>
    </div>
  </div>
</template>
