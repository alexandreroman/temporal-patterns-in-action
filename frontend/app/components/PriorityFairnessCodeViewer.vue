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

// One snippet per language and fairness mode: SOURCES_ON sets PriorityKey +
// FairnessKey + FairnessWeight, SOURCES_OFF sets PriorityKey only. The real
// branch lives in StartResolveTicket (workers/priority-fairness/activities.go).
// Recompute the 0-based `ranges` after any edit to `lines`.
const SOURCES_ON: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// HelpdeskRunWorkflow: start one top-level ResolveTicketWorkflow per",
      "// ticket from a local activity (NOT a ChildWorkflow).",
      "lctx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{",
      "    StartToCloseTimeout: 5 * time.Second,",
      "})",
      "var a *Activities",
      "for _, ticket := range seedTickets {",
      "    workflow.ExecuteLocalActivity(lctx, a.StartResolveTicket, StartResolveTicketInput{",
      '        WorkflowID:  fmt.Sprintf("%s-ticket-%s", parentID, ticket.ID),',
      "        Ticket:      ticket,",
      "        PriorityKey: ticket.Priority,",
      "    }).Get(ctx, nil)",
      "}",
      "",
      "// StartResolveTicket activity: build per-ticket Priority + Fairness",
      "// and hand off to the Temporal client. The ResolveTicket activity",
      "// inside the new workflow inherits Priority via SDK semantics.",
      "func (a *Activities) StartResolveTicket(ctx context.Context, in StartResolveTicketInput) error {",
      "    priority := temporal.Priority{",
      "        PriorityKey:    int(in.PriorityKey),      // P0..P3 → 1..4",
      "        FairnessKey:    string(in.Ticket.Tenant), // tenant identifier",
      "        FairnessWeight: TenantWeight[in.Ticket.Tenant],",
      "    }",
      "    _, err := a.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{",
      "        ID: in.WorkflowID, TaskQueue: TaskQueue, Priority: priority,",
      "    }, ResolveTicketWorkflow, in.Ticket)",
      "    return err",
      "}",
    ],
    ranges: {
      "priority-build": [18, 25],
      "execute-activity": [7, 11],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// HelpdeskRunWorkflow: start one top-level ResolveTicketWorkflow per",
      "// ticket from a local activity (NOT a ChildWorkflow).",
      "HelpdeskActivities activities = Workflow.newLocalActivityStub(",
      "    HelpdeskActivities.class,",
      "    LocalActivityOptions.newBuilder()",
      "        .setStartToCloseTimeout(Duration.ofSeconds(5))",
      "        .build());",
      "for (Ticket ticket : seedTickets) {",
      "    activities.startResolveTicket(new StartResolveTicketInput(",
      '        parentId + "-ticket-" + ticket.id(),',
      "        ticket,",
      "        ticket.priority()));",
      "}",
      "",
      "// startResolveTicket activity: build per-ticket Priority + Fairness",
      "// and hand off to the WorkflowClient. The resolveTicket activity",
      "// inside the new workflow inherits Priority via SDK semantics.",
      "@Override",
      "public void startResolveTicket(StartResolveTicketInput in) {",
      "    Priority priority = Priority.newBuilder()",
      "        .setPriorityKey(in.priorityKey())      // P0..P3 → 1..4",
      "        .setFairnessKey(in.ticket().tenant())  // tenant identifier",
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
      "priority-build": [19, 31],
      "execute-activity": [8, 11],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      'import { proxyLocalActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const a = proxyLocalActivities<typeof activities>({",
      '    startToCloseTimeout: "5 seconds",',
      "});",
      "",
      "// HelpdeskRunWorkflow: start one top-level ResolveTicketWorkflow per",
      "// ticket from a local activity (NOT a ChildWorkflow).",
      "for (const ticket of seedTickets) {",
      "    await a.startResolveTicket({",
      "        workflowId: `${parentId}-ticket-${ticket.id}`,",
      "        ticket,",
      "        priorityKey: ticket.priority,",
      "    });",
      "}",
      "",
      "// activities.ts — startResolveTicket: build per-ticket Priority +",
      "// Fairness and hand off to the Temporal client. The resolveTicket",
      "// activity inside the new workflow inherits priority via SDK semantics.",
      "export async function startResolveTicket(input: StartResolveTicketInput): Promise<void> {",
      "    const priority = {",
      "        priorityKey: input.priorityKey, // P0..P3 → 1..4",
      "        fairnessKey: input.ticket.tenant, // tenant identifier",
      "        fairnessWeight: TENANT_WEIGHT[input.ticket.tenant],",
      "    };",
      "    await client.workflow.start(resolveTicketWorkflow, {",
      "        args: [input.ticket],",
      "        workflowId: input.workflowId,",
      "        taskQueue: TASK_QUEUE,",
      "        priority,",
      "    });",
      "}",
    ],
    ranges: {
      "priority-build": [21, 31],
      "execute-activity": [10, 14],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# HelpdeskRunWorkflow: start one top-level ResolveTicketWorkflow per",
      "# ticket from a local activity (NOT a child workflow).",
      "for ticket in seed_tickets:",
      "    await workflow.execute_local_activity(",
      "        start_resolve_ticket,",
      "        StartResolveTicketInput(",
      '            workflow_id=f"{parent_id}-ticket-{ticket.id}",',
      "            ticket=ticket,",
      "            priority_key=ticket.priority,",
      "        ),",
      "        start_to_close_timeout=timedelta(seconds=5),",
      "    )",
      "",
      "# start_resolve_ticket activity: build per-ticket Priority + Fairness",
      "# and hand off to the Temporal client. The resolve_ticket activity",
      "# inside the new workflow inherits priority via SDK semantics.",
      "@activity.defn",
      "async def start_resolve_ticket(in_: StartResolveTicketInput) -> None:",
      "    priority = Priority(",
      "        priority_key=in_.priority_key,  # P0..P3 → 1..4",
      "        fairness_key=in_.ticket.tenant,  # tenant identifier",
      "        fairness_weight=TENANT_WEIGHT[in_.ticket.tenant],",
      "    )",
      "    await client.start_workflow(",
      "        ResolveTicketWorkflow.run, in_.ticket,",
      "        id=in_.workflow_id, task_queue=TASK_QUEUE,",
      "        priority=priority,",
      "    )",
    ],
    ranges: {
      "priority-build": [18, 27],
      "execute-activity": [3, 11],
    },
  },
};

const SOURCES_OFF: Record<CodeLang, PrioritySource> = {
  go: {
    label: "Go",
    lines: [
      "// HelpdeskRunWorkflow: start one top-level ResolveTicketWorkflow per",
      "// ticket from a local activity (NOT a ChildWorkflow).",
      "lctx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{",
      "    StartToCloseTimeout: 5 * time.Second,",
      "})",
      "var a *Activities",
      "for _, ticket := range seedTickets {",
      "    workflow.ExecuteLocalActivity(lctx, a.StartResolveTicket, StartResolveTicketInput{",
      '        WorkflowID:  fmt.Sprintf("%s-ticket-%s", parentID, ticket.ID),',
      "        Ticket:      ticket,",
      "        PriorityKey: ticket.Priority,",
      "    }).Get(ctx, nil)",
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
      "priority-build": [17, 22],
      "execute-activity": [7, 11],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// HelpdeskRunWorkflow: start one top-level ResolveTicketWorkflow per",
      "// ticket from a local activity (NOT a ChildWorkflow).",
      "HelpdeskActivities activities = Workflow.newLocalActivityStub(",
      "    HelpdeskActivities.class,",
      "    LocalActivityOptions.newBuilder()",
      "        .setStartToCloseTimeout(Duration.ofSeconds(5))",
      "        .build());",
      "for (Ticket ticket : seedTickets) {",
      "    activities.startResolveTicket(new StartResolveTicketInput(",
      '        parentId + "-ticket-" + ticket.id(),',
      "        ticket,",
      "        ticket.priority()));",
      "}",
      "",
      "// startResolveTicket activity: build Priority and hand off to the",
      "// WorkflowClient.",
      "@Override",
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
      "priority-build": [18, 28],
      "execute-activity": [8, 11],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      'import { proxyLocalActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const a = proxyLocalActivities<typeof activities>({",
      '    startToCloseTimeout: "5 seconds",',
      "});",
      "",
      "// HelpdeskRunWorkflow: start one top-level ResolveTicketWorkflow per",
      "// ticket from a local activity (NOT a ChildWorkflow).",
      "for (const ticket of seedTickets) {",
      "    await a.startResolveTicket({",
      "        workflowId: `${parentId}-ticket-${ticket.id}`,",
      "        ticket,",
      "        priorityKey: ticket.priority,",
      "    });",
      "}",
      "",
      "// activities.ts — startResolveTicket: build Priority and hand off to",
      "// the Temporal client.",
      "export async function startResolveTicket(input: StartResolveTicketInput): Promise<void> {",
      "    const priority = {",
      "        priorityKey: input.priorityKey, // P0..P3 → 1..4",
      "    };",
      "    await client.workflow.start(resolveTicketWorkflow, {",
      "        args: [input.ticket],",
      "        workflowId: input.workflowId,",
      "        taskQueue: TASK_QUEUE,",
      "        priority,",
      "    });",
      "}",
    ],
    ranges: {
      "priority-build": [20, 28],
      "execute-activity": [10, 14],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# HelpdeskRunWorkflow: start one top-level ResolveTicketWorkflow per",
      "# ticket from a local activity (NOT a child workflow).",
      "for ticket in seed_tickets:",
      "    await workflow.execute_local_activity(",
      "        start_resolve_ticket,",
      "        StartResolveTicketInput(",
      '            workflow_id=f"{parent_id}-ticket-{ticket.id}",',
      "            ticket=ticket,",
      "            priority_key=ticket.priority,",
      "        ),",
      "        start_to_close_timeout=timedelta(seconds=5),",
      "    )",
      "",
      "# start_resolve_ticket activity: build Priority and hand off to the",
      "# Temporal client.",
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
      "priority-build": [17, 24],
      "execute-activity": [3, 11],
    },
  },
};

// Seed and incident announcements fire right before the dispatch loop, so
// they light the local-activity call site. Assignment and resolution order is
// decided by the Priority pinned on each started workflow, so ticket events
// light the Priority construction.
const EVENT_TO_RANGE: Record<string, RangeKey> = {
  "helpdesk.run.seeded": "execute-activity",
  "helpdesk.incident.injected": "execute-activity",
  "helpdesk.ticket.assigned": "priority-build",
  "helpdesk.ticket.resolved": "priority-build",
};

const sources = computed<Record<CodeLang, PrioritySource>>(() =>
  props.fairnessOn ? SOURCES_ON : SOURCES_OFF,
);

// Workflow completion/failure events are not mapped, so the last ticket
// event keeps its range highlighted once the run is over.
const currentHighlight = computed<[number, number] | null>(() => {
  const src = sources.value[lang.value];
  for (let i = props.events.length - 1; i >= 0; i--) {
    const env = props.events[i];
    if (!env) continue;
    const key = EVENT_TO_RANGE[env.type];
    if (key) return src.ranges[key];
  }
  return null;
});
</script>

<template>
  <CodeViewer :sources="sources" :highlight="currentHighlight" />
</template>
