---
name: "Saga activities take txID as first business argument"
description: "Every saga activity receives the transaction/idempotency key (txID) as the first parameter after ctx, before the business payload."
type: feedback
---

# Saga activities take txID as first business argument

Every saga activity signature is
`func (a *Activities) Xxx(ctx context.Context, txID string, ...)`.
The workflow extracts `txID := input.TransactionID` once at entry,
then threads it through every `workflow.ExecuteActivity(...)` call
(forward and compensation) as the first business argument.

**Why:** The idempotency key drives correctness of retried activities
on the downstream service. Keeping it at position 1 makes it visible
in logs, in the Temporal UI's activity-input pane, and in the
pedagogical code snippets rendered by
`frontend/app/components/SagaCodeViewer.vue` (Go/Java/Python all
follow the same convention). Burying it inside an `input` struct
hides the pattern from viewers of the demo.

**How to apply:** When adding or editing a saga activity, keep `txID`
as the first parameter after `ctx`, log it as
`"transactionId", txID`, and match the call site in `workflow.go`.
Do not rename to `transactionID` or drop the log field — the frontend
Temporal timeline and NATS envelopes rely on the consistent naming.
