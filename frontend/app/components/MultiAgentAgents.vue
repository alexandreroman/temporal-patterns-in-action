<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Three cards, one per subtopic / child workflow. Topic name and queries
 * come from multi-agent.plan.ready / queries.ready; per-query state flips
 * running/ok/failed from search.started/completed/failed; card state
 * rolls up on child.completed/failed.
 */

type CardState = "idle" | "running" | "done" | "partial" | "failed";
type ChipState = "idle" | "running" | "ok" | "failed";

interface Topic {
  index: number;
  name: string;
  queries: { text: string; state: ChipState }[];
  state: CardState;
}

const props = defineProps<{
  events: EventEnvelope[];
}>();

const topics = computed<Topic[]>(() => {
  // Seed three placeholder cards so the layout is present before the
  // plan.ready event arrives. The plan.ready event overrides name/index.
  const byIndex = new Map<number, Topic>();
  for (let i = 0; i < 3; i++) {
    byIndex.set(i, { index: i, name: `Subtopic ${i + 1}`, queries: [], state: "idle" });
  }

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;

    switch (env.type) {
      case "multi-agent.plan.ready": {
        const subtopics = Array.isArray(data.subtopics) ? data.subtopics : [];
        for (const s of subtopics) {
          const sub = s as Record<string, unknown>;
          const idx = typeof sub.index === "number" ? sub.index : -1;
          if (idx < 0) continue;
          const existing = byIndex.get(idx);
          const name = typeof sub.name === "string" ? sub.name : `Subtopic ${idx + 1}`;
          byIndex.set(idx, {
            index: idx,
            name,
            queries: existing?.queries ?? [],
            state: existing?.state ?? "idle",
          });
        }
        break;
      }
      case "multi-agent.queries.ready": {
        const perTopic = Array.isArray(data.queries) ? data.queries : [];
        for (const t of perTopic) {
          const topic = t as Record<string, unknown>;
          const idx = typeof topic.topicIndex === "number" ? topic.topicIndex : -1;
          if (idx < 0) continue;
          const card = byIndex.get(idx);
          if (!card) continue;
          const qs = Array.isArray(topic.queries) ? (topic.queries as unknown[]) : [];
          card.queries = qs.map((q) => ({
            text: String(q ?? ""),
            state: "idle" as ChipState,
          }));
          if (typeof topic.topicName === "string" && topic.topicName) card.name = topic.topicName;
        }
        break;
      }
      case "multi-agent.fanout.started": {
        for (const card of byIndex.values()) {
          if (card.state === "idle") card.state = "running";
        }
        break;
      }
      case "multi-agent.search.started": {
        const ti = typeof data.topicIndex === "number" ? data.topicIndex : -1;
        const qi = typeof data.queryIndex === "number" ? data.queryIndex : -1;
        const card = byIndex.get(ti);
        if (!card) break;
        card.state = "running";
        const chip = card.queries[qi];
        if (chip) chip.state = "running";
        break;
      }
      case "multi-agent.search.completed": {
        const ti = typeof data.topicIndex === "number" ? data.topicIndex : -1;
        const qi = typeof data.queryIndex === "number" ? data.queryIndex : -1;
        const card = byIndex.get(ti);
        if (!card) break;
        const chip = card.queries[qi];
        if (chip) chip.state = "ok";
        break;
      }
      case "multi-agent.search.failed": {
        const ti = typeof data.topicIndex === "number" ? data.topicIndex : -1;
        const qi = typeof data.queryIndex === "number" ? data.queryIndex : -1;
        const card = byIndex.get(ti);
        if (!card) break;
        const chip = card.queries[qi];
        if (chip) chip.state = "failed";
        break;
      }
      case "multi-agent.child.completed": {
        const ti = typeof data.topicIndex === "number" ? data.topicIndex : -1;
        const card = byIndex.get(ti);
        if (!card) break;
        card.state = data.partial ? "partial" : "done";
        break;
      }
      case "multi-agent.child.failed": {
        const ti = typeof data.topicIndex === "number" ? data.topicIndex : -1;
        const card = byIndex.get(ti);
        if (!card) break;
        card.state = "failed";
        break;
      }
    }
  }

  return [...byIndex.values()].sort((a, b) => a.index - b.index);
});

// Cards reuse the neutral slate border of the Stats panel at every state;
// the tag label and per-query chips carry the running/done/partial/failed
// signal on their own.
const CARD_CLS = "border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-800/60";

const TAG_LABEL: Record<CardState, string> = {
  idle: "idle",
  running: "running",
  done: "done",
  partial: "partial",
  failed: "failed",
};

const CHIP_CLS: Record<ChipState, string> = {
  idle: "border-slate-200 bg-white text-slate-600 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300",
  running:
    "border-blue-300 bg-blue-50 text-blue-700 dark:border-blue-500 dark:bg-blue-950 dark:text-blue-200",
  ok: "border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-500 dark:bg-emerald-950 dark:text-emerald-200",
  failed:
    "border-rose-300 bg-rose-50 text-rose-700 dark:border-rose-500 dark:bg-rose-950 dark:text-rose-200",
};
</script>

<template>
  <div class="flex flex-1 flex-col gap-1.5">
    <div
      v-for="topic in topics"
      :key="topic.index"
      class="rounded-md border px-3 py-2 lg:flex-1"
      :class="CARD_CLS"
    >
      <div class="flex items-center justify-between gap-2">
        <span class="text-xs font-medium text-slate-800 dark:text-slate-100">
          Research: {{ topic.name }}
        </span>
        <span
          class="rounded border border-slate-200 bg-white px-1.5 py-0.5 font-mono text-[10px] text-slate-600 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300"
        >
          {{ TAG_LABEL[topic.state] }}
        </span>
      </div>
      <TransitionGroup
        v-if="topic.queries.length > 0"
        tag="div"
        name="pill"
        appear
        class="mt-1.5 flex flex-wrap items-start gap-2"
      >
        <span
          v-for="(q, qi) in topic.queries"
          :key="qi"
          class="rounded border px-2 py-1 font-mono text-[10px] transition-all duration-300"
          :class="CHIP_CLS[q.state]"
          :style="{ animationDelay: `${qi * 70}ms` }"
          :title="q.text"
        >
          {{ q.text }}
        </span>
      </TransitionGroup>
    </div>
  </div>
</template>

<style scoped>
.pill-enter-active,
.pill-appear-active {
  animation: pill-in 350ms ease-out both;
}
@keyframes pill-in {
  from {
    opacity: 0;
    transform: translateY(4px) scale(0.92);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
</style>
