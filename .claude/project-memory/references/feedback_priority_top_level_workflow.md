---
name: "Priority pattern: top-level workflow per ticket, not ChildWorkflow"
description: "In priority-fairness, each ticket runs as its own top-level workflow started from a local activity via the Temporal client — never as workflow.ExecuteChildWorkflow."
type: feedback
---

# Priority pattern: top-level workflow per ticket, not ChildWorkflow

In the priority-fairness pattern, every ticket
must be resolved by a **separate top-level
workflow** (`ResolveTicketWorkflow`) started
from inside `HelpdeskRunWorkflow` via a local
activity (`StartResolveTicket`) that calls
`client.ExecuteWorkflow`. Do NOT switch this
to `workflow.ExecuteChildWorkflow`, and do NOT
collapse it back to a single workflow that
schedules `ResolveTicket` activities directly.

The per-ticket `temporal.Priority` (Key + the
optional Fairness Key/Weight) lives on the
new workflow's `StartWorkflowOptions.Priority`;
the `ResolveTicket` activity inside the new
workflow inherits Priority via SDK semantics,
so the matching service still sees per-task
Priority on every schedule.

`ResolveTicketWorkflow` itself stays
signal-free — its history is exactly
Started → ResolveTicket → Completed. For each
dispatched ticket, `HelpdeskRunWorkflow` spawns
a waiter coroutine (`workflow.Go`) that runs a
`WaitTicketDone` local activity; that activity
long-polls `client.GetWorkflow(...).Get(...)`
on the per-ticket workflow id and returns when
it closes, at which point the waiter pushes the
ticket id onto an in-workflow buffered channel
the drain loop reads (plus listens for
`inject-p0-incident`) instead of holding
ChildWorkflow futures or relying on a signal-
back from the per-ticket workflow.

**Why:** the user asked twice in the same
conversation. First: drop the ChildWorkflow
wrapper; then, after a wrong reading where I
removed the wrapper entirely and called
`ResolveTicket` directly with
`ActivityOptions.Priority`, the user
clarified: "Non je veux effectivement lancer
un nouveau workflow par ticket, juste ne pas
utiliser un ChildWorkflow". The demo is about
showing per-task Priority on independent
top-level workflows, not on activities of one
parent and not on children.

**How to apply:** when editing
`workers/priority-fairness/workflow.go` or
its CodeViewer snippets, keep the
parent → local-activity → `client.ExecuteWorkflow`
→ top-level `ResolveTicketWorkflow` chain
intact, with completion observed via the
`WaitTicketDone` long-poll local activity (no
signal-back). If a refactor seems to simplify
by removing the per-ticket workflow or
turning it into a child, stop and confirm
with the user first.
