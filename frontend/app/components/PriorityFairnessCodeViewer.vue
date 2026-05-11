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
// HelpdeskRunWorkflow dispatches each ticket via a local activity
// (StartResolveTicket) that uses the Temporal client to start a brand-new
// top-level ResolveTicketWorkflow — explicitly NOT a ChildWorkflow. The
// per-ticket Priority is built inside that activity and pinned on
// StartWorkflowOptions; the ResolveTicket activity inside the new workflow
// inherits Priority via SDK semantics. fairnessOn=true sets PriorityKey +
// FairnessKey + FairnessWeight; fairnessOn=false sets only PriorityKey
// (matching workers/priority-fairness/workflow.go's `if input.FairnessOn`
// branch). Keep the four languages structurally aligned and recompute
// `ranges` after any edit — indices are 0-based offsets into `lines`.
const SOURCES_ON: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// HelpdeskRunWorkflow: dispatch each ticket via StartResolveTicket —",
      "// a local activity that uses the Temporal client to create a brand-new",
      "// top-level ResolveTicketWorkflow (NOT a ChildWorkflow). The new",
      "// workflow's StartWorkflowOptions.Priority is what the matching",
      "// service sees on every ResolveTicket schedule.",
      "var a *Activities",
      "for _, ticket := range seedTickets {",
      "    workflow.ExecuteLocalActivity(ctx, a.StartResolveTicket, StartResolveTicketInput{",
      '        WorkflowID:  fmt.Sprintf("%s-ticket-%s", parentID, ticket.ID),',
      "        Ticket:      ticket,",
      "        PriorityKey: ticket.Priority,",
      "        FairnessOn:  true,",
      "    })",
      "}",
      "",
      "// StartResolveTicket activity: build per-ticket Priority + Fairness",
      "// and hand off to the Temporal client. The ResolveTicket activity",
      "// inside the new workflow inherits Priority via SDK semantics.",
      "func (a *Activities) StartResolveTicket(ctx context.Context, in StartResolveTicketInput) error {",
      "    priority := temporal.Priority{",
      "        PriorityKey:    int(in.PriorityKey),               // P0..P3 → 1..4",
      "        FairnessKey:    string(in.Ticket.Tenant),          // tenant identifier",
      "        FairnessWeight: TenantWeight[in.Ticket.Tenant],",
      "    }",
      "    _, err := a.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{",
      "        ID: in.WorkflowID, TaskQueue: TaskQueue, Priority: priority,",
      "    }, ResolveTicketWorkflow, in.Ticket)",
      "    return err",
      "}",
    ],
    ranges: {
      "priority-build": [19, 23],
      "execute-activity": [7, 12],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// HelpdeskRunWorkflow: dispatch each ticket via startResolveTicket —",
      "// a local activity that uses the WorkflowClient to create a brand-new",
      "// top-level ResolveTicketWorkflow (NOT a ChildWorkflow). The new",
      "// workflow's WorkflowOptions.priority is what the matching service",
      "// sees on every resolveTicket schedule.",
      "HelpdeskActivities activities =",
      "    Workflow.newLocalActivityStub(HelpdeskActivities.class, opts);",
      "for (Ticket ticket : seedTickets) {",
      "    activities.startResolveTicket(new StartResolveTicketInput(",
      '        parentId + "-ticket-" + ticket.id(),',
      "        ticket,",
      "        ticket.priority(),",
      "        true",
      "    ));",
      "}",
      "",
      "// startResolveTicket activity: build per-ticket Priority + Fairness",
      "// and hand off to the WorkflowClient. The resolveTicket activity",
      "// inside the new workflow inherits Priority via SDK semantics.",
      "public void startResolveTicket(StartResolveTicketInput in) {",
      "    Priority priority = Priority.newBuilder()",
      "        .setPriorityKey(in.priorityKey())                  // P0..P3 → 1..4",
      "        .setFairnessKey(in.ticket().tenant())              // tenant identifier",
      "        .setFairnessWeight(TENANT_WEIGHT.get(in.ticket().tenant()))",
      "        .build();",
      "    WorkflowOptions opts = WorkflowOptions.newBuilder()",
      "        .setWorkflowId(in.workflowId())",
      "        .setTaskQueue(TASK_QUEUE)",
      "        .setPriority(priority)",
      "        .build();",
      "    ResolveTicketWorkflow stub =",
      "        workflowClient.newWorkflowStub(ResolveTicketWorkflow.class, opts);",
      "    WorkflowClient.start(stub::run, in.ticket());",
      "}",
    ],
    ranges: {
      "priority-build": [20, 24],
      "execute-activity": [8, 13],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# HelpdeskRunWorkflow: dispatch each ticket via start_resolve_ticket —",
      "# a local activity that uses the Temporal client to create a brand-new",
      "# top-level ResolveTicketWorkflow (NOT a child workflow). The new",
      "# workflow's start_workflow priority is what the matching service",
      "# sees on every resolve_ticket schedule.",
      "for ticket in seed_tickets:",
      "    await workflow.execute_local_activity(",
      "        start_resolve_ticket,",
      "        StartResolveTicketInput(",
      '            workflow_id=f"{parent_id}-ticket-{ticket.id}",',
      "            ticket=ticket,",
      "            priority_key=ticket.priority,",
      "            fairness_on=True,",
      "        ),",
      "        start_to_close_timeout=timedelta(seconds=5),",
      "    )",
      "",
      "# start_resolve_ticket: build per-ticket Priority + Fairness and",
      "# hand off to the Temporal client. The resolve_ticket activity inside",
      "# the new workflow inherits priority via SDK semantics.",
      "@activity.defn",
      "async def start_resolve_ticket(in_: StartResolveTicketInput) -> None:",
      "    priority = Priority(",
      "        priority_key=in_.priority_key,                # P0..P3 → 1..4",
      "        fairness_key=in_.ticket.tenant,               # tenant identifier",
      "        fairness_weight=TENANT_WEIGHT[in_.ticket.tenant],",
      "    )",
      "    await client.start_workflow(",
      "        ResolveTicketWorkflow.run, in_.ticket,",
      "        id=in_.workflow_id, task_queue=TASK_QUEUE,",
      "        priority=priority,",
      "    )",
    ],
    ranges: {
      "priority-build": [22, 26],
      // Python spans the full multi-line execute_local_activity call as the launch site.
      "execute-activity": [6, 15],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "// HelpdeskRunWorkflow: dispatch each ticket via startResolveTicket —",
      "// a local activity that uses the Temporal client to create a brand-new",
      "// top-level ResolveTicketWorkflow (NOT a ChildWorkflow). The new",
      "// workflow's workflowOptions.priority is what the matching service",
      "// sees on every resolveTicket schedule.",
      'import { proxyLocalActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const { startResolveTicket } = proxyLocalActivities<typeof activities>({",
      '    startToCloseTimeout: "5 seconds",',
      "});",
      "",
      "for (const ticket of seedTickets) {",
      "    await startResolveTicket({",
      "        workflowId: `${parentId}-ticket-${ticket.id}`,",
      "        ticket,",
      "        priorityKey: ticket.priority,",
      "        fairnessOn: true,",
      "    });",
      "}",
      "",
      "// startResolveTicket activity: build per-ticket Priority + Fairness and",
      "// hand off to the Temporal client. The resolveTicket activity inside the",
      "// new workflow inherits priority via SDK semantics.",
      "export async function startResolveTicket(in_: StartResolveTicketInput): Promise<void> {",
      "    const priority = {",
      "        priorityKey: in_.priorityKey,                  // P0..P3 → 1..4",
      "        fairnessKey: in_.ticket.tenant,                // tenant identifier",
      "        fairnessWeight: TENANT_WEIGHT[in_.ticket.tenant],",
      "    };",
      "    await client.workflow.start(resolveTicketWorkflow, {",
      "        args: [in_.ticket],",
      "        workflowId: in_.workflowId,",
      "        taskQueue: TASK_QUEUE,",
      "        priority,",
      "    });",
      "}",
    ],
    ranges: {
      "priority-build": [25, 29],
      "execute-activity": [13, 18],
    },
  },
};

const SOURCES_OFF: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// HelpdeskRunWorkflow: dispatch each ticket via StartResolveTicket —",
      "// a local activity that uses the Temporal client to create a brand-new",
      "// top-level ResolveTicketWorkflow (NOT a ChildWorkflow). Fairness off:",
      "// only PriorityKey is set on StartWorkflowOptions, so the matching",
      "// service falls back to FIFO within each priority bucket.",
      "var a *Activities",
      "for _, ticket := range seedTickets {",
      "    workflow.ExecuteLocalActivity(ctx, a.StartResolveTicket, StartResolveTicketInput{",
      '        WorkflowID:  fmt.Sprintf("%s-ticket-%s", parentID, ticket.ID),',
      "        Ticket:      ticket,",
      "        PriorityKey: ticket.Priority,",
      "        FairnessOn:  false,",
      "    })",
      "}",
      "",
      "// StartResolveTicket activity: build Priority and hand off to the",
      "// Temporal client.",
      "func (a *Activities) StartResolveTicket(ctx context.Context, in StartResolveTicketInput) error {",
      "    priority := temporal.Priority{",
      "        PriorityKey: int(in.PriorityKey), // P0..P3 → 1..4",
      "    }",
      "    _, err := a.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{",
      "        ID: in.WorkflowID, TaskQueue: TaskQueue, Priority: priority,",
      "    }, ResolveTicketWorkflow, in.Ticket)",
      "    return err",
      "}",
    ],
    ranges: {
      "priority-build": [18, 20],
      "execute-activity": [7, 12],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// HelpdeskRunWorkflow: dispatch each ticket via startResolveTicket —",
      "// a local activity that uses the WorkflowClient to create a brand-new",
      "// top-level ResolveTicketWorkflow (NOT a ChildWorkflow). Fairness off:",
      "// only priorityKey is set on WorkflowOptions, so the matching service",
      "// falls back to FIFO within each priority bucket.",
      "HelpdeskActivities activities =",
      "    Workflow.newLocalActivityStub(HelpdeskActivities.class, opts);",
      "for (Ticket ticket : seedTickets) {",
      "    activities.startResolveTicket(new StartResolveTicketInput(",
      '        parentId + "-ticket-" + ticket.id(),',
      "        ticket,",
      "        ticket.priority(),",
      "        false",
      "    ));",
      "}",
      "",
      "// startResolveTicket activity: build Priority and hand off to the",
      "// WorkflowClient.",
      "public void startResolveTicket(StartResolveTicketInput in) {",
      "    Priority priority = Priority.newBuilder()",
      "        .setPriorityKey(in.priorityKey())  // P0..P3 → 1..4",
      "        .build();",
      "    WorkflowOptions opts = WorkflowOptions.newBuilder()",
      "        .setWorkflowId(in.workflowId())",
      "        .setTaskQueue(TASK_QUEUE)",
      "        .setPriority(priority)",
      "        .build();",
      "    ResolveTicketWorkflow stub =",
      "        workflowClient.newWorkflowStub(ResolveTicketWorkflow.class, opts);",
      "    WorkflowClient.start(stub::run, in.ticket());",
      "}",
    ],
    ranges: {
      "priority-build": [19, 21],
      "execute-activity": [8, 13],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# HelpdeskRunWorkflow: dispatch each ticket via start_resolve_ticket —",
      "# a local activity that uses the Temporal client to create a brand-new",
      "# top-level ResolveTicketWorkflow (NOT a child workflow). Fairness off:",
      "# only priority_key is set on start_workflow, so the matching service",
      "# falls back to FIFO within each priority bucket.",
      "for ticket in seed_tickets:",
      "    await workflow.execute_local_activity(",
      "        start_resolve_ticket,",
      "        StartResolveTicketInput(",
      '            workflow_id=f"{parent_id}-ticket-{ticket.id}",',
      "            ticket=ticket,",
      "            priority_key=ticket.priority,",
      "            fairness_on=False,",
      "        ),",
      "        start_to_close_timeout=timedelta(seconds=5),",
      "    )",
      "",
      "# start_resolve_ticket: build Priority and hand off to the Temporal client.",
      "@activity.defn",
      "async def start_resolve_ticket(in_: StartResolveTicketInput) -> None:",
      "    priority = Priority(",
      "        priority_key=in_.priority_key,  # P0..P3 → 1..4",
      "    )",
      "    await client.start_workflow(",
      "        ResolveTicketWorkflow.run, in_.ticket,",
      "        id=in_.workflow_id, task_queue=TASK_QUEUE,",
      "        priority=priority,",
      "    )",
    ],
    ranges: {
      "priority-build": [20, 22],
      "execute-activity": [6, 15],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "// HelpdeskRunWorkflow: dispatch each ticket via startResolveTicket —",
      "// a local activity that uses the Temporal client to create a brand-new",
      "// top-level ResolveTicketWorkflow (NOT a ChildWorkflow). Fairness off:",
      "// only priorityKey is set on workflowOptions, so the matching service",
      "// falls back to FIFO within each priority bucket.",
      'import { proxyLocalActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const { startResolveTicket } = proxyLocalActivities<typeof activities>({",
      '    startToCloseTimeout: "5 seconds",',
      "});",
      "",
      "for (const ticket of seedTickets) {",
      "    await startResolveTicket({",
      "        workflowId: `${parentId}-ticket-${ticket.id}`,",
      "        ticket,",
      "        priorityKey: ticket.priority,",
      "        fairnessOn: false,",
      "    });",
      "}",
      "",
      "// startResolveTicket activity: build Priority and hand off to the Temporal client.",
      "export async function startResolveTicket(in_: StartResolveTicketInput): Promise<void> {",
      "    const priority = {",
      "        priorityKey: in_.priorityKey, // P0..P3 → 1..4",
      "    };",
      "    await client.workflow.start(resolveTicketWorkflow, {",
      "        args: [in_.ticket],",
      "        workflowId: in_.workflowId,",
      "        taskQueue: TASK_QUEUE,",
      "        priority,",
      "    });",
      "}",
    ],
    ranges: {
      "priority-build": [23, 25],
      "execute-activity": [13, 18],
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
    // Announcement activities (run.seeded, incident.injected) fire BEFORE the
    // priority-attached activities are dispatched — they map to the per-ticket
    // Priority construction the workflow is about to do.
    case "helpdesk.run.seeded":
    case "helpdesk.incident.injected":
      return src.ranges["priority-build"];
    // ticket.assigned / ticket.resolved are emitted from inside ResolveTicket
    // itself — the priority-pinned top-level workflow is running, so both
    // anchor the highlight on the StartResolveTicket dispatch site.
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
