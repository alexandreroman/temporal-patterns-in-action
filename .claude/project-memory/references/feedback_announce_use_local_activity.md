---
name: "Announce activities must be local activities in priority-aware patterns"
description: "When a parent workflow has no explicit Priority, regular announce activities inherit the default priority key 3 and block prioritised dispatch behind the backlog — use ExecuteLocalActivity instead."
type: feedback
---

# Announce activities must be local activities in priority-aware patterns

In any pattern where the parent workflow dispatches
work at varying priorities (e.g. `priority-fairness`),
the parent's announce / state-emission activities must
run as **local activities** (`workflow.ExecuteLocalActivity`),
not regular `workflow.ExecuteActivity`.

**Why:** The parent workflow is started without an explicit
`Priority` on `StartWorkflowOptions`, so the server treats
it as priority key 3 (default). Every regular activity it
schedules inherits that priority. With `matching.useNewMatcher=true`
and a deep prioritised backlog (e.g. 120 resolve-ticket
activities on the same task queue), the parent's announce
sits in the matching service queue behind every P1 task,
which can mean 10–15 s of wait. Because the parent calls
`.Get(ctx, nil)` on that announce future before dispatching
the follow-up high-priority work (e.g. the P0 child workflow
for an injected incident), the dispatch is delayed by the
same amount — the symptom is "the urgent ticket waits for
all the other tickets to finish".

Local activities run inline in the workflow task on the
same worker, never enter the matching service, and never
consume one of the worker's bounded activity slots, so
the announce→dispatch ordering the UI relies on is preserved
while latency stays in the millisecond range.

**How to apply:** When introducing a new pattern that uses
`matching.useNewMatcher` priority/fairness, default the
parent's announce-style activities to `ExecuteLocalActivity`.
Only escalate to a regular activity if the announce needs
durable retries beyond the workflow task. Discovered in
`workers/priority-fairness/workflow.go` (HelpdeskRunWorkflow)
on 2026-05-11 while debugging "injected P0 waits for backlog".
