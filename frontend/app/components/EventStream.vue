<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

type DotColor = "blue" | "green" | "red" | "amber";

const props = withDefaults(
  defineProps<{
    events: EventEnvelope[];
    labelFor: (env: EventEnvelope) => string;
    dotColor?: (env: EventEnvelope) => DotColor;
  }>(),
  { dotColor: () => (): DotColor => "blue" },
);

const reversed = computed(() => [...props.events].reverse());

const startTime = computed(() => {
  const first = props.events[0];
  if (!first) return null;
  const t = new Date(first.time).getTime();
  return Number.isNaN(t) ? null : t;
});

const DOT_CLS: Record<DotColor, string> = {
  blue: "bg-blue-500",
  green: "bg-emerald-500",
  red: "bg-rose-500",
  amber: "bg-amber-500",
};

function formatTime(env: EventEnvelope): string {
  if (startTime.value === null) return "";
  const t = new Date(env.time).getTime();
  if (Number.isNaN(t)) return env.time;
  const elapsed = Math.max(0, (t - startTime.value) / 1000);
  if (elapsed === 0) return "0";
  return `+${elapsed.toFixed(1)}s`;
}
</script>

<template>
  <div
    class="flex flex-col overflow-hidden rounded-xl border border-slate-200 dark:border-slate-700"
  >
    <div class="border-b border-slate-200 dark:border-slate-700">
      <div
        class="border-b-2 border-transparent px-4 py-2 text-xs font-medium text-slate-700 dark:text-slate-300"
      >
        Event stream
      </div>
    </div>
    <div class="h-80 overflow-y-auto px-3 py-2">
      <p v-if="events.length === 0" class="py-4 text-center text-xs text-slate-400">
        No events yet.
      </p>
      <TransitionGroup name="event">
        <div
          v-for="env in reversed"
          :key="env.id"
          class="event-row flex items-baseline gap-2 border-b border-slate-100 py-1.5 text-[11px] last:border-0 dark:border-slate-800"
        >
          <span class="mt-1 size-1.5 shrink-0 rounded-full" :class="DOT_CLS[props.dotColor(env)]" />
          <span class="w-12 shrink-0 text-right font-mono text-[10px] text-slate-400">
            {{ formatTime(env) }}
          </span>
          <span class="text-slate-600 dark:text-slate-400">
            {{ labelFor(env) }}
          </span>
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>

<style scoped>
.event-enter-active {
  transition:
    opacity 0.3s ease-out,
    transform 0.3s ease-out;
}
.event-enter-from {
  opacity: 0;
  transform: translateY(-6px);
}
.event-move {
  transition: transform 0.25s ease-out;
}
.event-row {
  animation: event-flash 1s ease-out;
}
@keyframes event-flash {
  0% {
    background-color: rgb(59 130 246 / 0.18);
  }
  100% {
    background-color: transparent;
  }
}
</style>
