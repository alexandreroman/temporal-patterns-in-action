<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { AGENT_PROMPT } from "~~/shared/constants";
import type {
  AgentApprovalRequest,
  AgentApprovalResponse,
  AgentStartRequest,
  AgentStartResponse,
} from "~~/shared/types";

useSeoMeta({ title: "Durable AI Agent" });

type Scenario = AgentStartRequest["scenario"];

const form = reactive({
  scenario: "happy" as Scenario,
});

const { workflowId, events, starting, running, error: finalError, run } = usePatternRun("agent");
const approving = ref(false);

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

async function start() {
  const runId = randomSuffix();
  await run(`agent-${runId}`, () =>
    $fetch<AgentStartResponse>("/api/agent/start", {
      method: "POST",
      body: { runId, scenario: form.scenario },
    }),
  );
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
</script>

<template>
  <section>
    <PatternHeader
      title="Durable AI Agent — Travel Planner"
      label="Run agent"
      :busy-label="starting ? 'Starting…' : 'Running…'"
      :disabled="running"
      @run="start"
    >
      <template #icon><IconAgent class="h-5 w-5" /></template>
      <select
        v-model="form.scenario"
        :disabled="running"
        class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
      >
        <option value="happy">Happy path</option>
        <option value="retry">LLM timeout + retry</option>
        <option value="approval">Human-in-the-loop</option>
      </select>
    </PatternHeader>

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
    <!--
      At lg+ the conversation is taken out of the flow and stretched over its
      column, so the state panel alone sets the row height and the
      conversation scrolls inside it. On mobile it keeps its own h-72 scroller.
    -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row lg:items-stretch">
      <div class="min-w-0 flex-1 lg:relative">
        <AgentConversation
          class="lg:absolute lg:inset-0"
          :events="events"
          :pending-prompt="running ? AGENT_PROMPT : null"
        />
      </div>
      <AgentStatePanel :events="events" />
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
