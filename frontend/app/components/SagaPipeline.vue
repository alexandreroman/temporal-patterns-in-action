<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Pipeline of the saga's four steps + compensations. Stacks vertically with
 * ↓ separators below the `sm` breakpoint; horizontal with → separators above.
 * Each chip shows the forward-step name by default; when the step is being
 * compensated or has been reverted, the label swaps to the compensation name
 * and a warning icon appears. A failed forward step keeps its name but also
 * shows the warning icon. State is derived from the live event stream.
 */

type StepState = "pending" | "active" | "done" | "failed" | "compensating" | "reverted";

const STEP_IDS = [
  "reserve-inventory",
  "charge-payment",
  "ship-order",
  "send-confirmation",
] as const;
type StepId = (typeof STEP_IDS)[number];

interface Step {
  id: StepId;
  name: string;
  comp: string;
}

const STEPS: readonly Step[] = [
  { id: "reserve-inventory", name: "Reserve inventory", comp: "Release inventory" },
  { id: "charge-payment", name: "Charge payment", comp: "Refund payment" },
  { id: "ship-order", name: "Ship order", comp: "Cancel shipment" },
  { id: "send-confirmation", name: "Send confirmation", comp: "Retract email" },
];

const COMP_TO_STEP: Record<string, StepId> = {
  "release-inventory": "reserve-inventory",
  "refund-payment": "charge-payment",
  "cancel-shipment": "ship-order",
  "retract-email": "send-confirmation",
};

const FORWARD_IDS = new Set<string>(STEP_IDS);

const props = defineProps<{
  events: EventEnvelope[];
}>();

const states = computed<Record<StepId, StepState>>(() => {
  const map: Record<StepId, StepState> = {
    "reserve-inventory": "pending",
    "charge-payment": "pending",
    "ship-order": "pending",
    "send-confirmation": "pending",
  };

  let compensating = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    const rawStep = String(data.step ?? "");

    switch (env.type) {
      case "progress.step.started": {
        const compTarget = COMP_TO_STEP[rawStep];
        if (compTarget) {
          map[compTarget] = "compensating";
        } else if (FORWARD_IDS.has(rawStep)) {
          map[rawStep as StepId] = "active";
        }
        break;
      }
      case "progress.step.completed": {
        const compTarget = COMP_TO_STEP[rawStep];
        if (compTarget) {
          map[compTarget] = "reverted";
        } else if (FORWARD_IDS.has(rawStep)) {
          map[rawStep as StepId] = "done";
        }
        break;
      }
      case "progress.step.failed": {
        if (FORWARD_IDS.has(rawStep)) {
          map[rawStep as StepId] = "failed";
        }
        break;
      }
      case "progress.compensation.started":
        compensating = true;
        break;
      case "progress.compensation.completed":
        compensating = false;
        break;
    }
  }

  // Fallback: during compensation, any previously-done step that has not yet
  // emitted its specific compensation event should still look "reverted".
  if (!compensating) return map;
  for (const id of STEP_IDS) {
    if (map[id] === "done") map[id] = "reverted";
  }
  return map;
});

const chipClass: Record<StepState, string> = {
  pending:
    "border-slate-200 bg-slate-50 text-slate-500 " +
    "dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400",
  active:
    "border-blue-300 bg-blue-50 text-blue-700 ring-2 ring-blue-200 " +
    "dark:border-blue-500 dark:bg-blue-950 dark:text-blue-200 " +
    "dark:ring-blue-800",
  done:
    "border-emerald-300 bg-emerald-50 text-emerald-700 " +
    "dark:border-emerald-600 dark:bg-emerald-950 dark:text-emerald-200",
  failed:
    "border-rose-300 bg-rose-50 text-rose-700 " +
    "dark:border-rose-600 dark:bg-rose-950 dark:text-rose-200",
  compensating:
    "border-amber-300 bg-amber-50 text-amber-700 ring-2 ring-amber-200 " +
    "dark:border-amber-500 dark:bg-amber-950 dark:text-amber-200 " +
    "dark:ring-amber-800",
  reverted:
    "border-amber-200 bg-amber-50/40 text-amber-700/60 opacity-60 " +
    "dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200/70",
};
</script>

<template>
  <div class="py-1">
    <div class="mx-auto flex w-fit flex-col items-center gap-0 sm:flex-row sm:items-stretch">
      <template v-for="(step, idx) in STEPS" :key="step.id">
        <span
          v-if="idx > 0"
          aria-hidden="true"
          class="flex h-6 w-6 shrink-0 items-center justify-center text-sm text-slate-400 sm:hidden"
          >&darr;</span
        >
        <span
          v-if="idx > 0"
          aria-hidden="true"
          class="hidden w-6 shrink-0 items-center justify-center text-sm text-slate-400 sm:flex"
          >&rarr;</span
        >
        <div
          class="min-w-[128px] rounded-lg border px-3 py-2 text-center transition-all duration-300"
          :class="chipClass[states[step.id]]"
        >
          <div class="flex items-center justify-center gap-1 text-[12px] font-medium">
            <svg
              v-if="states[step.id] === 'failed'
                || states[step.id] === 'compensating'
                || states[step.id] === 'reverted'"
              aria-hidden="true"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="h-[13px] w-[13px] shrink-0"
            >
              <path d="M12 9v4" />
              <path d="M12 17h.01" />
              <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z" />
            </svg>
            <span>{{
              states[step.id] === "compensating" || states[step.id] === "reverted"
                ? step.comp
                : step.name
            }}</span>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
