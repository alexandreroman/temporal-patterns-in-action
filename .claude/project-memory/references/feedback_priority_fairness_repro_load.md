---
name: "Don't abort priority-fairness runs mid-flight when reproducing bugs"
description: "Each priority-fairness scenario spawns ~120 top-level workflows; aborting reproductions before they drain piles workflows up on the dev Temporal server and may force a restart."
type: feedback
---

# Don't abort priority-fairness runs mid-flight

When reproducing intermittent bugs on
`/patterns/priority-fairness`, let each run finish (the
`HelpdeskRunWorkflow` plus its ~120 spawned per-ticket
`ResolveTicketWorkflow` executions) before triggering the
next one. Do **not** kill the SSE consumer mid-run or
hammer "Run scenario" in rapid succession just to
amplify the race.

**Why:** the dev Temporal server runs in a 512 MB
container ([[event-architecture]] / `compose.yaml`).
Each scenario adds 1 helpdesk workflow + 120 ticket
workflows + dozens of long-poll waiters to the
namespace. Aborting the SSE only stops the UI; the
workflows keep running. Piling unfinished runs up
overloads the Temporal server, and it may need a restart
once the namespace is congested — at which point
reproductions become unreliable for unrelated reasons.

**How to apply:** for repro loops over this pattern,
serialise the trials: trigger a run, wait until
`progress.workflow.completed` (or the status bar shows
"Drained — N total resolutions"), then start the next.
5–10 sequential runs are enough to expose intermittent
races. If iteration speed matters, prefer adding
diagnostic logs once and watching a single failing run
rather than burning more workflows. If you ever see the
Temporal UI lag, list-workflows hang, or healthchecks
flap, restart the `temporal` compose service before
continuing. The same caution applies to any future
pattern that fans out to dozens of top-level workflows
per scenario.
