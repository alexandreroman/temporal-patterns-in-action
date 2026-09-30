<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { StatusState, StatusTone } from "~/types/status-bar";

const props = defineProps<{
  events: EventEnvelope[];
}>();

const derived = computed<StatusState>(() => {
  if (props.events.length === 0) {
    return { tone: "idle", message: "Ready — choose a scenario and run" };
  }

  let tone: StatusTone = "running";
  let message = "Starting agent…";
  // Set by agent.approval.received: whether the human rejected the plan, or
  // approved it and the agent is resuming.
  let rejected = false;
  let resumed = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    switch (env.type) {
      case "progress.step.started":
        if (data.step === "call-llm") {
          tone = "running";
          message = resumed ? "Resuming — LLM reasoning" : "LLM reasoning…";
        } else if (data.step === "execute-mcp-tool") {
          tone = "running";
          message = "MCP tool running…";
        }
        break;
      case "progress.step.failed":
        // Interceptor fires on every retry attempt — keep it transient.
        tone = "running";
        message = "LLM call failed — retrying";
        break;
      case "agent.tool.started":
        tone = "running";
        message = `MCP: ${String(data.name ?? "")}`;
        break;
      case "agent.approval.requested":
        tone = "running";
        message = "Workflow durably waiting for approval…";
        break;
      case "agent.approval.received":
        rejected = !data.approved;
        resumed = !!data.approved;
        tone = "running";
        message = data.approved ? "Approved — resuming agent" : "Rejected — stopping";
        break;
      case "progress.workflow.completed":
        tone = "success";
        message = "Agent completed — plan delivered";
        break;
      case "progress.workflow.failed":
        tone = "error";
        message = rejected
          ? "Agent stopped — plan rejected"
          : `Agent failed: ${String(data.error ?? "")}`;
        break;
    }
  }

  return { tone, message };
});
</script>

<template>
  <StatusBar :tone="derived.tone" :message="derived.message" />
</template>
