<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { EntityCartProgress } from "~~/shared/types";

/**
 * Sidecar panel summarising the entity workflow's signal/query counters.
 * Counts come primarily from the live event stream; when the page has just
 * received a getCart query response we trust those numbers in place of the
 * estimated ones.
 */

type Status =
  | "Idle"
  | "Waiting for signals"
  | "Signal: addItem"
  | "Signal: updateQty"
  | "Signal: removeItem"
  | "Signal: checkout"
  | "Query: getCart"
  | "Completed";

const props = defineProps<{
  events: EventEnvelope[];
  progress?: EntityCartProgress | null;
}>();

interface Derived {
  status: Status;
  signalsReceived: number;
  queriesAnswered: number;
}

const derived = computed<Derived>(() => {
  let status: Status = props.events.length === 0 ? "Idle" : "Waiting for signals";
  let signalsReceived = 0;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;

    switch (env.type) {
      case "entity.item.added":
        status = "Signal: addItem";
        signalsReceived++;
        break;
      case "entity.qty.updated":
        status = "Signal: updateQty";
        signalsReceived++;
        break;
      case "entity.item.removed":
        status = "Signal: removeItem";
        signalsReceived++;
        break;
      case "progress.step.started":
        if (data.step === "process-payment") {
          status = "Signal: checkout";
          signalsReceived++;
        }
        break;
      case "progress.step.completed":
        // Transient "working" status settles back to idle once the activity
        // acknowledges — mirrors the way the workflow reopens its selector.
        if (status !== "Completed") {
          status = "Waiting for signals";
        }
        break;
      case "entity.query.getCart":
        status = "Query: getCart";
        break;
      case "progress.workflow.completed":
        status = "Completed";
        break;
      case "progress.workflow.failed":
        status = "Completed";
        break;
    }
  }

  // When a fresh getCart query response is available, prefer its authoritative
  // counts over our local estimates.
  const prog = props.progress;
  return {
    status,
    signalsReceived: prog ? prog.signalsReceived : signalsReceived,
    queriesAnswered: prog ? prog.queriesAnswered : 0,
  };
});

const displayedSignals = useCountTween(() => derived.value.signalsReceived);
const displayedQueries = useCountTween(() => derived.value.queriesAnswered);
</script>

<template>
  <div class="flex w-full flex-col gap-1.5 lg:w-[220px] lg:shrink-0">
    <div
      class="flex flex-col rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60"
    >
      <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Workflow status
      </div>
      <div class="mt-auto font-mono text-[12px] text-slate-800 dark:text-slate-100">
        {{ derived.status }}
      </div>
    </div>

    <div class="grid grid-cols-2 gap-2">
      <div
        class="flex flex-col rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60"
      >
        <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
          Signals
        </div>
        <div class="mt-auto text-sm font-medium tabular-nums text-slate-800 dark:text-slate-100">
          {{ displayedSignals }}
        </div>
      </div>
      <div
        class="flex flex-col rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60"
      >
        <div class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
          Queries
        </div>
        <div class="mt-auto text-sm font-medium tabular-nums text-slate-800 dark:text-slate-100">
          {{ displayedQueries }}
        </div>
      </div>
    </div>
  </div>
</template>
