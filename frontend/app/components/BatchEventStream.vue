<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";

defineProps<{
  events: EventEnvelope[];
}>();

type DotColor = "blue" | "green" | "red" | "amber";

function shortType(type: string): string {
  return type.replace(/^(progress|batch)\./, "");
}

function serviceLabel(service: unknown): string {
  if (typeof service !== "string") return "";
  const map: Record<string, string> = {
    resize: "Resize",
    thumbnail: "Thumbnail",
    cdn: "CDN",
    metadata: "Metadata",
  };
  return map[service] ?? service;
}

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const step = data.step ? String(data.step) : "";
  const error = data.error ? String(data.error) : "";
  const index = typeof data.index === "number" ? data.index : null;
  const attempt = typeof data.attempt === "number" ? data.attempt : null;
  const service = serviceLabel(data.service);

  switch (env.type) {
    case "progress.workflow.completed":
      return "Workflow completed";
    case "progress.workflow.failed":
      return `Workflow failed: ${error}`;
    case "progress.step.started":
      return `${step} started (attempt ${attempt ?? 1})`;
    case "progress.step.completed":
      return `${step} completed`;
    case "progress.step.failed":
      return `${step} failed: ${error}`;
    case "batch.item.started":
      return `Item #${index} started — ${service} (attempt ${attempt ?? 1})`;
    case "batch.item.completed":
      return `Item #${index} done`;
    case "batch.item.attempt_failed":
      return `Item #${index} failed — ${service}: ${error}`;
    case "batch.summary.reported": {
      const total = data.total ?? "?";
      const processed = data.processed ?? 0;
      const failed = data.failed ?? 0;
      return `Batch summary: ${processed}/${total} ok, ${failed} failed`;
    }
    default:
      if (env.type.startsWith("batch.") || env.type.startsWith("progress.")) {
        return shortType(env.type);
      }
      return env.type;
  }
}

function dotColor(env: EventEnvelope): DotColor {
  const t = env.type;
  if (t.includes("attempt_failed")) return "amber";
  if (t.includes("started")) return "blue";
  if (t.includes("completed") || t.includes("reported")) return "green";
  if (t.includes("failed")) return "red";
  return "blue";
}
</script>

<template>
  <EventStream :events="events" :label-for="eventLabel" :dot-color="dotColor" />
</template>
