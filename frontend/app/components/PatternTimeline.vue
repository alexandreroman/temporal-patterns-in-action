<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

type StepState = "pending" | "running" | "done" | "failed" | "compensated";
type WorkflowState =
  | "idle"
  | "started"
  | "running"
  | "completed"
  | "failed"
  | "compensating"
  | "compensated";

interface StepInfo {
  id: string;
  state: StepState;
  attempt: number;
  durationMs?: number;
  error?: string;
}

const props = defineProps<{
  steps: string[];
  events: EventEnvelope[];
}>();

const derived = computed(() => {
  const map = new Map<string, StepInfo>();
  for (const id of props.steps) {
    map.set(id, { id, state: "pending", attempt: 0 });
  }
  let workflow: WorkflowState = "idle";
  let compensationActive = false;
  let workflowError: string | undefined;
  let workflowDurationMs: number | undefined;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    switch (env.type) {
      case "progress.workflow.started":
        workflow = "started";
        break;
      case "progress.workflow.completed":
        workflow = "completed";
        workflowDurationMs = numberOf(data.durationMs);
        break;
      case "progress.workflow.failed":
        workflow = compensationActive ? "compensated" : "failed";
        workflowError = stringOf(data.error);
        break;
      case "progress.step.started": {
        const step = stringOf(data.step);
        const info = step ? map.get(step) : undefined;
        if (info) {
          info.state = "running";
          info.attempt = numberOf(data.attempt) ?? info.attempt;
          workflow = "running";
        }
        break;
      }
      case "progress.step.completed": {
        const step = stringOf(data.step);
        const info = step ? map.get(step) : undefined;
        if (info) {
          info.state = "done";
          info.durationMs = numberOf(data.durationMs);
          info.attempt = numberOf(data.attempt) ?? info.attempt;
        }
        break;
      }
      case "progress.step.failed": {
        const step = stringOf(data.step);
        const info = step ? map.get(step) : undefined;
        if (info) {
          info.state = "failed";
          info.error = stringOf(data.error);
          info.attempt = numberOf(data.attempt) ?? info.attempt;
        }
        break;
      }
      case "progress.compensation.started":
        compensationActive = true;
        workflow = "compensating";
        for (const info of map.values()) {
          if (info.state === "done") info.state = "compensated";
        }
        break;
      case "progress.compensation.completed":
        compensationActive = false;
        workflow = "compensated";
        break;
    }
  }

  return {
    workflow,
    workflowError,
    workflowDurationMs,
    steps: props.steps.map((id) => map.get(id)!),
  };
});

function numberOf(value: unknown): number | undefined {
  return typeof value === "number" ? value : undefined;
}

function stringOf(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function formatDuration(ms?: number): string {
  if (ms === undefined) return "";
  if (ms < 1000) return `${ms} ms`;
  return `${(ms / 1000).toFixed(2)} s`;
}

const stepBadge: Record<StepState, string> = {
  pending: "bg-slate-100 text-slate-400 ring-slate-200",
  running: "bg-blue-100 text-blue-700 ring-blue-200",
  done: "bg-emerald-100 text-emerald-700 ring-emerald-200",
  failed: "bg-rose-100 text-rose-700 ring-rose-200",
  compensated: "bg-amber-100 text-amber-700 ring-amber-200",
};

const stepGlyph: Record<StepState, string> = {
  pending: "·",
  running: "…",
  done: "✓",
  failed: "×",
  compensated: "↺",
};

const workflowBadge: Record<WorkflowState, string> = {
  idle: "bg-slate-100 text-slate-500",
  started: "bg-blue-100 text-blue-700",
  running: "bg-blue-100 text-blue-700",
  completed: "bg-emerald-100 text-emerald-700",
  failed: "bg-rose-100 text-rose-700",
  compensating: "bg-amber-100 text-amber-700",
  compensated: "bg-amber-100 text-amber-800",
};
</script>

<template>
  <div>
    <header class="flex flex-wrap items-center gap-3">
      <span
        class="inline-flex items-center rounded-full px-3 py-1 text-xs font-medium"
        :class="workflowBadge[derived.workflow]"
      >
        {{ derived.workflow }}
      </span>
      <span v-if="derived.workflowDurationMs !== undefined" class="text-xs text-slate-500">
        {{ formatDuration(derived.workflowDurationMs) }}
      </span>
      <span v-if="derived.workflowError" class="text-xs text-rose-600">
        {{ derived.workflowError }}
      </span>
    </header>

    <ol class="mt-4 space-y-2">
      <li
        v-for="step in derived.steps"
        :key="step.id"
        class="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-4 py-3"
      >
        <span
          class="inline-flex size-6 items-center justify-center rounded-full text-xs font-medium ring-1"
          :class="stepBadge[step.state]"
        >
          {{ stepGlyph[step.state] }}
        </span>
        <span class="text-sm">{{ step.id }}</span>
        <span v-if="step.state === 'running'" class="ml-auto text-xs text-blue-600">
          running… (attempt {{ step.attempt }})
        </span>
        <span
          v-else-if="step.state === 'done' && step.durationMs !== undefined"
          class="ml-auto text-xs text-slate-500"
        >
          {{ formatDuration(step.durationMs) }}
        </span>
        <span v-else-if="step.state === 'failed'" class="ml-auto text-xs text-rose-600">
          {{ step.error ?? "failed" }}
        </span>
        <span v-else-if="step.state === 'compensated'" class="ml-auto text-xs text-amber-700">
          compensated
        </span>
      </li>
    </ol>
  </div>
</template>
