<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
}>();

const lang = useCodeLang();

type RangeKey = "priority-build" | "activity-options" | "execute-activity";

interface PrioritySource extends CodeSource {
  ranges: Record<RangeKey, [number, number]>;
}

// Each snippet illustrates how a Temporal worker attaches Priority + Fairness
// to an activity invocation: priorityKey from the ticket's P0..P3, fairnessKey
// from the tenant id, fairnessWeight from the contract tier. Keep the four
// languages structurally aligned and recompute `ranges` after any edit — the
// indices are 0-based offsets into `lines`.
const SOURCES: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// Workflow: pick a ticket, attach Priority + Fairness to the activity.",
      "func ResolveTicketWorkflow(ctx workflow.Context, ticket Ticket) error {",
      "    tenantWeight := map[string]float32{",
      '        "acme":  10, // Enterprise',
      '        "brick":  3, // Pro',
      '        "solo":   1, // Free',
      "    }",
      "",
      "    // Build the Priority for this task: lower priorityKey runs sooner;",
      "    // fairnessKey + fairnessWeight balance throughput across tenants.",
      "    priority := temporal.Priority{",
      "        PriorityKey:    int(ticket.Priority),       // P0..P3 → 1..4",
      '        FairnessKey:    string(ticket.Tenant),      // "acme" | "brick" | "solo"',
      "        FairnessWeight: tenantWeight[ticket.Tenant],",
      "    }",
      "",
      "    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{",
      "        Priority:            priority,",
      "        StartToCloseTimeout: 5 * time.Second,",
      "    })",
      "",
      "    var a *Activities",
      "    return workflow.ExecuteActivity(ctx, a.ResolveTicket, ticket).Get(ctx, nil)",
      "}",
    ],
    ranges: {
      "priority-build": [8, 14],
      "activity-options": [16, 19],
      "execute-activity": [22, 22],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// Workflow: pick a ticket, attach Priority + Fairness to the activity.",
      "@WorkflowMethod",
      "public void resolveTicket(Ticket ticket) {",
      "    Map<String, Float> tenantWeight = Map.of(",
      '        "acme",  10f, // Enterprise',
      '        "brick",  3f, // Pro',
      '        "solo",   1f  // Free',
      "    );",
      "",
      "    // Build the Priority for this task: lower priorityKey runs sooner;",
      "    // fairnessKey + fairnessWeight balance throughput across tenants.",
      "    Priority priority = Priority.newBuilder()",
      "        .setPriorityKey(ticket.priority())          // P0..P3 → 1..4",
      '        .setFairnessKey(ticket.tenant())            // "acme" | "brick" | "solo"',
      "        .setFairnessWeight(tenantWeight.get(ticket.tenant()))",
      "        .build();",
      "",
      "    ActivityOptions options = ActivityOptions.newBuilder()",
      "        .setPriority(priority)",
      "        .setStartToCloseTimeout(Duration.ofSeconds(5))",
      "        .build();",
      "",
      "    HelpdeskActivities activities = Workflow.newActivityStub(HelpdeskActivities.class, options);",
      "    activities.resolveTicket(ticket);",
      "}",
    ],
    ranges: {
      "priority-build": [9, 15],
      "activity-options": [17, 20],
      "execute-activity": [23, 23],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# Workflow: pick a ticket, attach Priority + Fairness to the activity.",
      "TENANT_WEIGHT: dict[str, float] = {",
      '    "acme": 10.0,   # Enterprise',
      '    "brick": 3.0,   # Pro',
      '    "solo": 1.0,    # Free',
      "}",
      "",
      "@workflow.defn",
      "class ResolveTicketWorkflow:",
      "    @workflow.run",
      "    async def run(self, ticket: Ticket) -> None:",
      "        # Build the Priority for this task: lower priority_key runs sooner;",
      "        # fairness_key + fairness_weight balance throughput across tenants.",
      "        priority = Priority(",
      "            priority_key=ticket.priority,            # P0..P3 → 1..4",
      '            fairness_key=ticket.tenant,              # "acme" | "brick" | "solo"',
      "            fairness_weight=TENANT_WEIGHT[ticket.tenant],",
      "        )",
      "",
      "        await workflow.execute_activity(",
      "            resolve_ticket, ticket,",
      "            priority=priority,",
      "            start_to_close_timeout=timedelta(seconds=5),",
      "        )",
    ],
    ranges: {
      "priority-build": [11, 17],
      "activity-options": [19, 23],
      "execute-activity": [19, 19],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "// Workflow: pick a ticket, attach Priority + Fairness to the activity.",
      'import { proxyActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const TENANT_WEIGHT: Record<string, number> = {",
      "    acme: 10,   // Enterprise",
      "    brick: 3,   // Pro",
      "    solo: 1,    // Free",
      "};",
      "",
      "export async function resolveTicketWorkflow(ticket: Ticket): Promise<void> {",
      "    // Build the Priority for this task: lower priorityKey runs sooner;",
      "    // fairnessKey + fairnessWeight balance throughput across tenants.",
      "    const priority = {",
      "        priorityKey: ticket.priority,                // P0..P3 → 1..4",
      '        fairnessKey: ticket.tenant,                  // "acme" | "brick" | "solo"',
      "        fairnessWeight: TENANT_WEIGHT[ticket.tenant],",
      "    };",
      "",
      "    const { resolveTicket } = proxyActivities<typeof activities>({",
      "        priority,",
      '        startToCloseTimeout: "5 seconds",',
      "    });",
      "",
      "    await resolveTicket(ticket);",
      "}",
    ],
    ranges: {
      "priority-build": [11, 17],
      "activity-options": [19, 22],
      "execute-activity": [24, 24],
    },
  },
};

function latestRelevantType(events: EventEnvelope[]): string | null {
  for (let i = events.length - 1; i >= 0; i--) {
    const env = events[i];
    if (!env) continue;
    switch (env.type) {
      case "helpdesk.ticket.assigned":
      case "helpdesk.ticket.resolved":
      case "helpdesk.dump.executed":
      case "helpdesk.incident.injected":
      case "helpdesk.run.seeded":
      case "progress.workflow.completed":
      case "progress.workflow.failed":
        return env.type;
      default:
        continue;
    }
  }
  return null;
}

const currentHighlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];
  const latest = latestRelevantType(props.events);
  if (!latest) return null;

  switch (latest) {
    case "helpdesk.ticket.assigned":
    case "helpdesk.dump.executed":
    case "helpdesk.incident.injected":
      return src.ranges["priority-build"];
    case "helpdesk.ticket.resolved":
      return src.ranges["execute-activity"];
    case "helpdesk.run.seeded":
      return src.ranges["activity-options"];
    default:
      return null;
  }
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="currentHighlight" />
</template>
