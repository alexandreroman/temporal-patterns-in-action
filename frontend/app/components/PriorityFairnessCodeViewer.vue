<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
  fairnessOn: boolean;
}>();

const lang = useCodeLang();

type RangeKey = "priority-build" | "execute-activity";

interface PrioritySource extends CodeSource {
  ranges: Record<RangeKey, [number, number]>;
}

// Two snippet variants per language mirror the workflow's actual code path:
// fairnessOn=true sets PriorityKey + FairnessKey + FairnessWeight; fairnessOn=false
// sets only PriorityKey (matching workers/priority-fairness/workflow.go's
// `if input.FairnessOn` branch). Keep the four languages structurally aligned
// and recompute `ranges` after any edit — indices are 0-based offsets into `lines`.
const SOURCES_ON: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// Workflow: pick a ticket, attach Priority + Fairness to the activity.",
      "func ResolveTicketWorkflow(ctx workflow.Context, ticket Ticket) error {",
      "    tenantWeight := map[string]float32{",
      '        "acme":  10, // Mission Critical',
      '        "brick":  3, // Enterprise',
      '        "solo":   1, // Business',
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
      '        "acme",  10f, // Mission Critical',
      '        "brick",  3f, // Enterprise',
      '        "solo",   1f  // Business',
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
      "execute-activity": [23, 23],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# Workflow: pick a ticket, attach Priority + Fairness to the activity.",
      "TENANT_WEIGHT: dict[str, float] = {",
      '    "acme": 10.0,   # Mission Critical',
      '    "brick": 3.0,   # Enterprise',
      '    "solo": 1.0,    # Business',
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
      // Python passes priority/timeout as kwargs to execute_activity, so the
      // whole call is the launch site — span the full multi-line call.
      "execute-activity": [19, 23],
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
      "    acme: 10,   // Mission Critical",
      "    brick: 3,   // Enterprise",
      "    solo: 1,    // Business",
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
      "execute-activity": [24, 24],
    },
  },
};

const SOURCES_OFF: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// Workflow: pick a ticket, attach Priority to the activity.",
      "// Fairness disabled: only PriorityKey is set, so the matching service",
      "// falls back to FIFO within each priority bucket — Acme can starve.",
      "func ResolveTicketWorkflow(ctx workflow.Context, ticket Ticket) error {",
      "    priority := temporal.Priority{",
      "        PriorityKey: int(ticket.Priority), // P0..P3 → 1..4",
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
      "priority-build": [4, 6],
      "execute-activity": [14, 14],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// Workflow: pick a ticket, attach Priority to the activity.",
      "// Fairness disabled: only priorityKey is set, so the matching service",
      "// falls back to FIFO within each priority bucket — Acme can starve.",
      "@WorkflowMethod",
      "public void resolveTicket(Ticket ticket) {",
      "    Priority priority = Priority.newBuilder()",
      "        .setPriorityKey(ticket.priority())  // P0..P3 → 1..4",
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
      "priority-build": [5, 7],
      "execute-activity": [15, 15],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# Workflow: pick a ticket, attach Priority to the activity.",
      "# Fairness disabled: only priority_key is set, so the matching service",
      "# falls back to FIFO within each priority bucket — Acme can starve.",
      "@workflow.defn",
      "class ResolveTicketWorkflow:",
      "    @workflow.run",
      "    async def run(self, ticket: Ticket) -> None:",
      "        priority = Priority(",
      "            priority_key=ticket.priority,  # P0..P3 → 1..4",
      "        )",
      "",
      "        await workflow.execute_activity(",
      "            resolve_ticket, ticket,",
      "            priority=priority,",
      "            start_to_close_timeout=timedelta(seconds=5),",
      "        )",
    ],
    ranges: {
      "priority-build": [7, 9],
      "execute-activity": [11, 15],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "// Workflow: pick a ticket, attach Priority to the activity.",
      "// Fairness disabled: only priorityKey is set, so the matching service",
      "// falls back to FIFO within each priority bucket — Acme can starve.",
      'import { proxyActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "export async function resolveTicketWorkflow(ticket: Ticket): Promise<void> {",
      "    const priority = {",
      "        priorityKey: ticket.priority, // P0..P3 → 1..4",
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
      "priority-build": [7, 9],
      "execute-activity": [16, 16],
    },
  },
};

const sources = computed<Record<CodeLang, PrioritySource>>(() =>
  props.fairnessOn ? SOURCES_ON : SOURCES_OFF,
);

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
  const src = sources.value[lang.value];
  const latest = latestRelevantType(props.events);
  if (!latest) return null;

  switch (latest) {
    // Announcement activities (run.seeded, dump.executed, incident.injected)
    // fire BEFORE the priority-attached activities are dispatched — they map
    // to the per-ticket Priority construction the workflow is about to do.
    case "helpdesk.run.seeded":
    case "helpdesk.dump.executed":
    case "helpdesk.incident.injected":
      return src.ranges["priority-build"];
    // ticket.assigned / ticket.resolved are emitted from inside ResolveTicket
    // itself — the priority-attached activity is running and then returns, so
    // both anchor the highlight on workflow.ExecuteActivity(...).
    case "helpdesk.ticket.assigned":
    case "helpdesk.ticket.resolved":
      return src.ranges["execute-activity"];
    default:
      return null;
  }
});
</script>

<template>
  <CodeViewer :sources="sources" :highlight="currentHighlight" />
</template>
