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
// the parent builds per-ticket Priority and attaches it to ChildWorkflowOptions
// when spawning a ResolveTicketWorkflow child; the activity inside the child
// inherits that Priority via SDK semantics. fairnessOn=true sets PriorityKey +
// FairnessKey + FairnessWeight; fairnessOn=false sets only PriorityKey
// (matching workers/priority-fairness/workflow.go's `if input.FairnessOn`
// branch). Keep the four languages structurally aligned and recompute `ranges`
// after any edit — indices are 0-based offsets into `lines`.
const SOURCES_ON: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// Parent: build per-ticket Priority + Fairness, attach to the child.",
      "// The activity inside the child inherits Priority from its workflow.",
      "tenantWeight := map[string]float32{",
      '    "mission-critical": 10,',
      '    "enterprise":        3,',
      '    "business":          1,',
      "}",
      "",
      "for _, ticket := range seedTickets {",
      "    // Lower priorityKey runs sooner; fairnessKey + fairnessWeight",
      "    // balance throughput across tenants inside each priority bucket.",
      "    priority := temporal.Priority{",
      "        PriorityKey:    int(ticket.Priority),       // P0..P3 → 1..4",
      '        FairnessKey:    string(ticket.Tenant),      // tenant identifier',
      "        FairnessWeight: tenantWeight[ticket.Tenant],",
      "    }",
      "    cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{",
      "        Priority: priority,",
      "    })",
      "    workflow.ExecuteChildWorkflow(cctx, ResolveTicketWorkflow, ticket)",
      "}",
      "",
      "// Child: minimal — runs the activity. Priority is inherited from the",
      "// child workflow's ChildWorkflowOptions, so it is NOT set here.",
      "func ResolveTicketWorkflow(ctx workflow.Context, ticket Ticket) error {",
      "    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{",
      "        StartToCloseTimeout: 5 * time.Second,",
      "    })",
      "    var a *Activities",
      "    return workflow.ExecuteActivity(ctx, a.ResolveTicket, ticket).Get(ctx, nil)",
      "}",
    ],
    ranges: {
      "priority-build": [11, 15],
      "execute-activity": [29, 29],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// Parent: build per-ticket Priority + Fairness, attach to the child.",
      "// The activity inside the child inherits Priority from its workflow.",
      "Map<String, Float> tenantWeight = Map.of(",
      '    "mission-critical", 10f,',
      '    "enterprise",        3f,',
      '    "business",          1f',
      ");",
      "",
      "for (Ticket ticket : seedTickets) {",
      "    // Lower priorityKey runs sooner; fairnessKey + fairnessWeight",
      "    // balance throughput across tenants inside each priority bucket.",
      "    Priority priority = Priority.newBuilder()",
      "        .setPriorityKey(ticket.priority())          // P0..P3 → 1..4",
      '        .setFairnessKey(ticket.tenant())            // tenant identifier',
      "        .setFairnessWeight(tenantWeight.get(ticket.tenant()))",
      "        .build();",
      "    ChildWorkflowOptions childOptions = ChildWorkflowOptions.newBuilder()",
      "        .setPriority(priority)",
      "        .build();",
      "    ResolveTicketWorkflow child =",
      "        Workflow.newChildWorkflowStub(ResolveTicketWorkflow.class, childOptions);",
      "    child.run(ticket);",
      "}",
      "",
      "// Child: minimal — runs the activity. Priority is inherited from the",
      "// child workflow's ChildWorkflowOptions, so it is NOT set here.",
      "@WorkflowMethod",
      "public void run(Ticket ticket) {",
      "    ActivityOptions options = ActivityOptions.newBuilder()",
      "        .setStartToCloseTimeout(Duration.ofSeconds(5))",
      "        .build();",
      "    HelpdeskActivities activities = Workflow.newActivityStub(HelpdeskActivities.class, options);",
      "    activities.resolveTicket(ticket);",
      "}",
    ],
    ranges: {
      "priority-build": [11, 15],
      "execute-activity": [32, 32],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# Parent: build per-ticket Priority + Fairness, attach to the child.",
      "# The activity inside the child inherits Priority from its workflow.",
      "TENANT_WEIGHT: dict[str, float] = {",
      '    "mission-critical": 10.0,',
      '    "enterprise":        3.0,',
      '    "business":          1.0,',
      "}",
      "",
      "for ticket in seed_tickets:",
      "    # Lower priority_key runs sooner; fairness_key + fairness_weight",
      "    # balance throughput across tenants inside each priority bucket.",
      "    priority = Priority(",
      "        priority_key=ticket.priority,            # P0..P3 → 1..4",
      '        fairness_key=ticket.tenant,              # tenant identifier',
      "        fairness_weight=TENANT_WEIGHT[ticket.tenant],",
      "    )",
      "    await workflow.execute_child_workflow(",
      "        ResolveTicketWorkflow.run, ticket,",
      "        priority=priority,",
      "    )",
      "",
      "# Child: minimal — runs the activity. Priority is inherited from the",
      "# child workflow, so it is NOT passed to execute_activity.",
      "@workflow.defn",
      "class ResolveTicketWorkflow:",
      "    @workflow.run",
      "    async def run(self, ticket: Ticket) -> None:",
      "        await workflow.execute_activity(",
      "            resolve_ticket, ticket,",
      "            start_to_close_timeout=timedelta(seconds=5),",
      "        )",
    ],
    ranges: {
      "priority-build": [11, 15],
      // Python spans the full multi-line execute_activity call as the launch site.
      "execute-activity": [27, 30],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "// Parent: build per-ticket Priority + Fairness, attach to the child.",
      "// The activity inside the child inherits Priority from its workflow.",
      'import { executeChild, proxyActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const TENANT_WEIGHT: Record<string, number> = {",
      '    "mission-critical": 10,',
      '    "enterprise":        3,',
      '    "business":          1,',
      "};",
      "",
      "for (const ticket of seedTickets) {",
      "    // Lower priorityKey runs sooner; fairnessKey + fairnessWeight",
      "    // balance throughput across tenants inside each priority bucket.",
      "    const priority = {",
      "        priorityKey: ticket.priority,                // P0..P3 → 1..4",
      '        fairnessKey: ticket.tenant,                  // tenant identifier',
      "        fairnessWeight: TENANT_WEIGHT[ticket.tenant],",
      "    };",
      "    await executeChild(resolveTicketWorkflow, {",
      "        args: [ticket],",
      "        priority,",
      "    });",
      "}",
      "",
      "// Child: minimal — runs the activity. Priority is inherited from the",
      "// child workflow, so it is NOT passed to proxyActivities.",
      "export async function resolveTicketWorkflow(ticket: Ticket): Promise<void> {",
      "    const { resolveTicket } = proxyActivities<typeof activities>({",
      '        startToCloseTimeout: "5 seconds",',
      "    });",
      "    await resolveTicket(ticket);",
      "}",
    ],
    ranges: {
      "priority-build": [14, 18],
      "execute-activity": [31, 31],
    },
  },
};

const SOURCES_OFF: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// Parent: build per-ticket Priority, attach to the child.",
      "// Fairness disabled: only PriorityKey is set, so the matching service",
      "// falls back to FIFO within each priority bucket — the lowest-tier backlog drains last.",
      "for _, ticket := range seedTickets {",
      "    priority := temporal.Priority{",
      "        PriorityKey: int(ticket.Priority), // P0..P3 → 1..4",
      "    }",
      "    cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{",
      "        Priority: priority,",
      "    })",
      "    workflow.ExecuteChildWorkflow(cctx, ResolveTicketWorkflow, ticket)",
      "}",
      "",
      "// Child: minimal — runs the activity. Priority is inherited from the",
      "// child workflow's ChildWorkflowOptions, so it is NOT set here.",
      "func ResolveTicketWorkflow(ctx workflow.Context, ticket Ticket) error {",
      "    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{",
      "        StartToCloseTimeout: 5 * time.Second,",
      "    })",
      "    var a *Activities",
      "    return workflow.ExecuteActivity(ctx, a.ResolveTicket, ticket).Get(ctx, nil)",
      "}",
    ],
    ranges: {
      "priority-build": [4, 6],
      "execute-activity": [20, 20],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// Parent: build per-ticket Priority, attach to the child.",
      "// Fairness disabled: only priorityKey is set, so the matching service",
      "// falls back to FIFO within each priority bucket — the lowest-tier backlog drains last.",
      "for (Ticket ticket : seedTickets) {",
      "    Priority priority = Priority.newBuilder()",
      "        .setPriorityKey(ticket.priority())  // P0..P3 → 1..4",
      "        .build();",
      "    ChildWorkflowOptions childOptions = ChildWorkflowOptions.newBuilder()",
      "        .setPriority(priority)",
      "        .build();",
      "    ResolveTicketWorkflow child =",
      "        Workflow.newChildWorkflowStub(ResolveTicketWorkflow.class, childOptions);",
      "    child.run(ticket);",
      "}",
      "",
      "// Child: minimal — runs the activity. Priority is inherited from the",
      "// child workflow's ChildWorkflowOptions, so it is NOT set here.",
      "@WorkflowMethod",
      "public void run(Ticket ticket) {",
      "    ActivityOptions options = ActivityOptions.newBuilder()",
      "        .setStartToCloseTimeout(Duration.ofSeconds(5))",
      "        .build();",
      "    HelpdeskActivities activities = Workflow.newActivityStub(HelpdeskActivities.class, options);",
      "    activities.resolveTicket(ticket);",
      "}",
    ],
    ranges: {
      "priority-build": [4, 6],
      "execute-activity": [23, 23],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# Parent: build per-ticket Priority, attach to the child.",
      "# Fairness disabled: only priority_key is set, so the matching service",
      "# falls back to FIFO within each priority bucket — the lowest-tier backlog drains last.",
      "for ticket in seed_tickets:",
      "    priority = Priority(",
      "        priority_key=ticket.priority,  # P0..P3 → 1..4",
      "    )",
      "    await workflow.execute_child_workflow(",
      "        ResolveTicketWorkflow.run, ticket,",
      "        priority=priority,",
      "    )",
      "",
      "# Child: minimal — runs the activity. Priority is inherited from the",
      "# child workflow, so it is NOT passed to execute_activity.",
      "@workflow.defn",
      "class ResolveTicketWorkflow:",
      "    @workflow.run",
      "    async def run(self, ticket: Ticket) -> None:",
      "        await workflow.execute_activity(",
      "            resolve_ticket, ticket,",
      "            start_to_close_timeout=timedelta(seconds=5),",
      "        )",
    ],
    ranges: {
      "priority-build": [4, 6],
      "execute-activity": [18, 21],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "// Parent: build per-ticket Priority, attach to the child.",
      "// Fairness disabled: only priorityKey is set, so the matching service",
      "// falls back to FIFO within each priority bucket — the lowest-tier backlog drains last.",
      'import { executeChild, proxyActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "for (const ticket of seedTickets) {",
      "    const priority = {",
      "        priorityKey: ticket.priority, // P0..P3 → 1..4",
      "    };",
      "    await executeChild(resolveTicketWorkflow, {",
      "        args: [ticket],",
      "        priority,",
      "    });",
      "}",
      "",
      "// Child: minimal — runs the activity. Priority is inherited from the",
      "// child workflow, so it is NOT passed to proxyActivities.",
      "export async function resolveTicketWorkflow(ticket: Ticket): Promise<void> {",
      "    const { resolveTicket } = proxyActivities<typeof activities>({",
      '        startToCloseTimeout: "5 seconds",',
      "    });",
      "    await resolveTicket(ticket);",
      "}",
    ],
    ranges: {
      "priority-build": [7, 9],
      "execute-activity": [22, 22],
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
      case "helpdesk.burst.executed":
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
    // Announcement activities (run.seeded, burst.executed, incident.injected)
    // fire BEFORE the priority-attached activities are dispatched — they map
    // to the per-ticket Priority construction the workflow is about to do.
    case "helpdesk.run.seeded":
    case "helpdesk.burst.executed":
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
