---
name: "priority-fairness dials its own client locally"
description: "Keep shared events.RunWorker untouched: priority-fairness's main dials its own Temporal client.Client and passes it to Activities.Client, instead of widening the register-callback signature."
type: feedback
---

# priority-fairness dials its own client locally

The priority-fairness pattern needs a
`client.Client` inside its activities (so
`StartResolveTicket` can call
`client.ExecuteWorkflow`). Dial it **locally**
in `workers/priority-fairness/cmd/worker/main.go`
and assign it to `Activities.Client`. The
shared `events.RunWorker` still constructs its
own internal client for the worker; the extra
dial here is intentional and isolated to this
pattern's `main`.

Do NOT widen the `register` callback in
`workers/events/bootstrap.go` to pass the
client through to every pattern, and do NOT
touch the other patterns' `cmd/worker/main.go`
to absorb a new parameter.

**Why:** a bootstrap-signature change would touch
6 unrelated patterns just to serve this demo; the
localized dial keeps the blast radius inside
`workers/priority-fairness/`. The cross-pattern
refactor is explicitly out of scope.

**How to apply:** whenever something in
priority-fairness needs a Temporal API the
worker normally hides (start a workflow, send
a signal, run a query, list executions), wire
it from the locally dialled client in this
module's main. Avoid the temptation to "clean
this up" by moving the dial into
`events.RunWorker` — that change is out of scope.
