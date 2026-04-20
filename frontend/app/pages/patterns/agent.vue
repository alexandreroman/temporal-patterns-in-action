<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import type {
  AgentApprovalRequest,
  AgentApprovalResponse,
  AgentStartRequest,
  AgentStartResponse,
} from "~~/shared/types";

useSeoMeta({ title: "Durable AI Agent" });

type Scenario = AgentStartRequest["scenario"];

const PROMPT =
  "Plan a 5-day trip to Tokyo in October. Budget: $3000. I like food, temples, and nightlife.";

const form = reactive({
  scenario: "happy" as Scenario,
});

const workflowId = ref<string | null>(null);
const starting = ref(false);
const finalError = ref<string | null>(null);
const approving = ref(false);

const { events, waitForOpen } = usePatternStream("agent", workflowId);

const TERMINAL_EVENTS = new Set(["progress.workflow.completed", "progress.workflow.failed"]);

const running = computed(() => {
  if (starting.value) return true;
  if (!workflowId.value) return false;
  return !events.value.some((e) => TERMINAL_EVENTS.has(e.type));
});

const awaitingApproval = computed(() => {
  let pending = false;
  for (const e of events.value) {
    if (e.type === "agent.approval.requested") pending = true;
    else if (e.type === "agent.approval.received") pending = false;
  }
  return pending && running.value;
});

// While approving is true, don't flip it back until the workflow
// acknowledges the signal via agent.approval.received.
watch(awaitingApproval, (now) => {
  if (!now) approving.value = false;
});

function randomSuffix(): string {
  // 6-char base36 is plenty for a per-run agent ID in a demo.
  return Math.random().toString(36).slice(2, 8);
}

async function start() {
  finalError.value = null;
  starting.value = true;
  const runId = randomSuffix();
  // Subscribe BEFORE starting the workflow: core NATS has no replay, and the
  // user-prompt event fires almost immediately — we would miss it if the SSE
  // stream opened only after the start() response came back.
  workflowId.value = `agent-${runId}`;
  try {
    await waitForOpen();
    await $fetch<AgentStartResponse>("/api/agent/start", {
      method: "POST",
      body: { runId, scenario: form.scenario },
    });
  } catch (error) {
    finalError.value = error instanceof Error ? error.message : String(error);
    workflowId.value = null;
  } finally {
    starting.value = false;
  }
}

async function respond(approved: boolean) {
  if (!workflowId.value) return;
  approving.value = true;
  try {
    await $fetch<AgentApprovalResponse>("/api/agent/approval", {
      method: "POST",
      body: {
        workflowId: workflowId.value,
        approved,
      } satisfies AgentApprovalRequest,
    });
  } catch (error) {
    finalError.value = error instanceof Error ? error.message : String(error);
    approving.value = false;
  }
}

const statePanelRef = ref<{ $el?: HTMLElement } | null>(null);
const convoWrapperRef = ref<HTMLElement | null>(null);
const panelHeight = ref<number | null>(null);

const convoHeightStyle = computed(() =>
  panelHeight.value !== null ? { height: `${panelHeight.value}px` } : {},
);

let resizeObserver: ResizeObserver | null = null;
let mediaQuery: MediaQueryList | null = null;

function updatePanelHeight() {
  const el = statePanelRef.value?.$el as HTMLElement | undefined;
  if (!el) {
    panelHeight.value = null;
    return;
  }
  // Only pin height on desktop layout; on mobile the row stacks vertically
  // and the conversation should keep its own h-72 scroller.
  if (mediaQuery && !mediaQuery.matches) {
    panelHeight.value = null;
    return;
  }
  panelHeight.value = el.getBoundingClientRect().height;
}

onMounted(() => {
  mediaQuery = window.matchMedia("(min-width: 1024px)");
  mediaQuery.addEventListener("change", updatePanelHeight);

  const el = statePanelRef.value?.$el as HTMLElement | undefined;
  if (el) {
    resizeObserver = new ResizeObserver(() => updatePanelHeight());
    resizeObserver.observe(el);
  }
  updatePanelHeight();
});

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  mediaQuery?.removeEventListener("change", updatePanelHeight);
});
</script>

<template>
  <section>
    <NuxtLink to="/" class="text-sm text-slate-400 hover:text-slate-100"> &larr; back </NuxtLink>

    <!-- Control bar -->
    <div class="mt-2 flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-800 bg-slate-950 text-slate-300"
        >
          <IconAgent class="h-5 w-5" />
        </span>
        <h1 class="text-2xl font-semibold tracking-tight text-slate-100">
          Durable AI Agent &mdash; Travel Planner
        </h1>
      </div>
      <div class="flex items-center gap-2">
        <select
          v-model="form.scenario"
          :disabled="running"
          class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
        >
          <option value="happy">Happy path</option>
          <option value="retry">LLM timeout + retry</option>
          <option value="approval">Human-in-the-loop</option>
        </select>
        <button
          type="button"
          :disabled="running"
          class="cursor-pointer rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
          @click="start"
        >
          {{ starting ? "Starting…" : running ? "Running…" : "Run agent" }}
        </button>
      </div>
    </div>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <AgentArchitecture :events="events" />
    </div>

    <!-- Approval banner — visible only while the workflow is durably waiting -->
    <div
      class="approval-row grid transition-[grid-template-rows,opacity] duration-300 ease-out"
      :class="awaitingApproval ? 'mt-2 grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'"
      aria-live="polite"
    >
      <div class="min-h-0 overflow-hidden">
        <div
          class="approval-pulse flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-600 dark:bg-amber-950 dark:text-amber-200"
        >
          <span>Workflow suspended — the agent is waiting for a human decision.</span>
          <div class="flex gap-2">
            <button
              type="button"
              :disabled="approving"
              class="cursor-pointer rounded-md bg-emerald-600 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:opacity-60"
              @click="respond(true)"
            >
              Approve
            </button>
            <button
              type="button"
              :disabled="approving"
              class="cursor-pointer rounded-md bg-rose-600 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-rose-500 disabled:opacity-60"
              @click="respond(false)"
            >
              Reject
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Conversation + state panel -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row lg:items-start">
      <div ref="convoWrapperRef" class="min-w-0 flex-1" :style="convoHeightStyle">
        <AgentConversation :events="events" :pending-prompt="running ? PROMPT : null" />
      </div>
      <AgentStatePanel ref="statePanelRef" :events="events" />
    </div>

    <!-- Status bar -->
    <AgentStatusBar :events="events" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <AgentCodeViewer :events="events" />
      </div>
      <div class="min-w-0 flex-1">
        <AgentEventStream :events="events" />
      </div>
    </div>

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>

<style scoped>
@keyframes approval-pulse {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.55);
  }
  50% {
    box-shadow: 0 0 0 6px rgba(245, 158, 11, 0);
  }
}
.approval-pulse {
  animation: approval-pulse 1.8s ease-in-out infinite;
}
@media (prefers-reduced-motion: reduce) {
  .approval-pulse {
    animation: none;
  }
  .approval-row {
    transition: none;
  }
}
</style>
