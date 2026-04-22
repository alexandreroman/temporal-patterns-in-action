<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Live shopping-cart panel. Derives its contents directly from the event
 * stream so the viewer sees items appear, quantities update, and rows
 * disappear without round-tripping through queries.
 */

interface Row {
  itemId: string;
  name: string;
  priceCents: number;
  qty: number;
}

const props = defineProps<{
  events: EventEnvelope[];
  workflowId: string | null;
}>();

const rows = computed<Row[]>(() => {
  const map = new Map<string, Row>();

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    switch (env.type) {
      case "entity.item.added": {
        const itemId = String(data.itemId ?? "");
        if (!itemId) break;
        const name = String(data.name ?? itemId);
        const priceCents = Number(data.priceCents ?? 0);
        const qty = Number(data.qty ?? 1);
        const existing = map.get(itemId);
        if (existing) {
          existing.qty += qty;
          existing.name = name;
          existing.priceCents = priceCents;
        } else {
          map.set(itemId, { itemId, name, priceCents, qty });
        }
        break;
      }
      case "entity.qty.updated": {
        const itemId = String(data.itemId ?? "");
        const qty = Number(data.qty ?? 0);
        const existing = map.get(itemId);
        if (!existing) break;
        if (qty <= 0) map.delete(itemId);
        else existing.qty = qty;
        break;
      }
      case "entity.item.removed": {
        const itemId = String(data.itemId ?? "");
        if (itemId) map.delete(itemId);
        break;
      }
    }
  }

  return [...map.values()];
});

const totalCents = computed(() => rows.value.reduce((sum, r) => sum + r.priceCents * r.qty, 0));

// Tween the displayed total so viewers see the number tick up between
// signals. Mirrors the inline implementation in AgentStatePanel /
// MultiAgentMetrics.
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
        if (t < 1) frame = requestAnimationFrame(step);
        else frame = null;
      };
      frame = requestAnimationFrame(step);
    },
    { immediate: true },
  );

  onBeforeUnmount(cancel);
  return displayed;
}

const displayedTotalCents = useCountTween(() => totalCents.value);

function formatDollars(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}
</script>

<template>
  <div
    class="flex h-full min-h-[28rem] flex-col rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900"
  >
    <div
      class="flex items-center justify-between gap-3 border-b border-slate-200 px-4 py-2 dark:border-slate-700"
    >
      <div class="text-xs font-medium text-slate-700 dark:text-slate-300">Shopping cart</div>
      <div
        v-if="workflowId"
        class="truncate font-mono text-[10px] text-slate-400 dark:text-slate-500"
        :title="workflowId"
      >
        {{ workflowId }}
      </div>
    </div>

    <div class="flex-1 px-3 py-2">
      <p v-if="rows.length === 0" class="py-6 text-center text-xs text-slate-400">
        Cart is empty — run the scenario
      </p>
      <TransitionGroup v-else name="row" tag="ul" class="flex flex-col">
        <li
          v-for="r in rows"
          :key="r.itemId"
          class="flex items-baseline gap-3 border-b border-slate-100 py-2 text-[12px] last:border-0 dark:border-slate-800"
        >
          <span class="flex-1 truncate text-slate-700 dark:text-slate-200">
            {{ r.name }}
            <span class="ml-1 text-slate-400 dark:text-slate-500">× {{ r.qty }}</span>
          </span>
          <span class="font-mono tabular-nums text-slate-600 dark:text-slate-300">
            {{ formatDollars(r.priceCents * r.qty) }}
          </span>
        </li>
      </TransitionGroup>
    </div>

    <div
      class="flex items-center justify-between border-t border-slate-200 px-4 py-2 text-sm dark:border-slate-700"
    >
      <span class="text-slate-500 dark:text-slate-400">Total</span>
      <span
        class="font-mono text-base font-semibold tabular-nums text-slate-900 dark:text-slate-100"
      >
        {{ formatDollars(displayedTotalCents) }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.row-enter-active {
  transition:
    opacity 0.3s ease-out,
    transform 0.3s ease-out;
}
.row-enter-from {
  opacity: 0;
  transform: translateY(-4px);
}
.row-leave-active {
  transition:
    opacity 0.2s ease-in,
    transform 0.2s ease-in;
  position: absolute;
}
.row-leave-to {
  opacity: 0;
  transform: translateX(8px);
}
.row-move {
  transition: transform 0.25s ease-out;
}
</style>
