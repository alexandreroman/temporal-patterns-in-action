<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Sidecar panel that tracks the four headline counters of the agent loop
 * (phase, LLM calls, tool calls, tokens) plus the catalogue of MCP tools —
 * highlighting the one currently in flight. Everything is derived from the
 * live event stream; no state is mirrored from Temporal queries.
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
  activeTool: string | null;
}

const derived = computed<Derived>(() => {
  let phase: PhaseLabel = props.events.length === 0 ? "Idle" : "LLM";
  let llmCalls = 0;
  let toolCalls = 0;
  let tokens = 0;
  let activeTool: string | null = null;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    switch (env.type) {
      case "progress.step.started":
        if (data.step === "call-llm") phase = "LLM";
        else if (data.step === "execute-mcp-tool") phase = "Tool";
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

  return { phase, llmCalls, toolCalls, tokens, activeTool };
});
</script>

<template>
  <div class="flex w-full flex-col gap-1.5 lg:w-[220px] lg:shrink-0">
    <div class="grid grid-cols-2 gap-2">
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

      <div
        class="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60 flex flex-col"
      >
        <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
          Tokens used
        </div>
        <div class="mt-auto text-sm font-medium text-slate-800 dark:text-slate-100">
          {{ derived.tokens.toLocaleString() }}
        </div>
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
