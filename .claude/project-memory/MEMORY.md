# Project Memory

> When a new decision **contradicts** an existing
> memory note, do NOT silently override it.
> Instead: surface the conflict, quote the
> existing memory, explain how the new decision
> differs, and ask for explicit confirmation
> before updating. **Do NOT take any action** —
> no tool calls, no file writes — until confirmed.

- [Event architecture (NATS)](references/event-architecture.md) — subject hierarchy, envelope, progress/business split, determinism rule, kebab-case activity naming gotcha.
- [Dynamic NuxtLink via <component :is>](references/feedback_nuxtlink_dynamic_component.md) — use `resolveComponent("NuxtLink")`, not the string `'NuxtLink'`, or the element renders inert.
- [Coding conventions](references/feedback_coding_conventions.md) — line lengths, markdown style, LTS rule, no compound shell commands, hyphenated `docker-compose` in committed files.
- [Agent delegation](references/feedback_agent_delegation.md) — code-writer for code; code-reviewer for reviews; temporal skill for workflow work.
- [Temporal conventions](references/feedback_temporal_conventions.md) — determinism, `workflowcheck`, and the task-queue/workflow-name contract.
- [Runbook: new pattern](references/project_adding_new_pattern.md) — 5-step checklist covering workers/, workers/Makefile, and frontend additions.
- [Demo-first priorities](references/feedback_demo_priorities.md) — bias toward visibility and short inline forms; skip production robustness unless the demo itself showcases it.
- [Rogue host workers](references/feedback_rogue_host_workers.md) — stale `go run` worker on the host can steal tasks from the container; check `task-queue describe` before blaming Docker caching.
- [SSE endpoints need an immediate initial push](references/feedback_sse_initial_flush.md) — push one chunk right after `subscribe()` or Node/h3 holds response headers until the 15s heartbeat, blocking `EventSource.onopen`.
- [Frontend component conventions](references/feedback_frontend_component_conventions.md) — generic shells in `components/`; pattern logic lives in `<Pattern><Component>.vue` wrappers.
- [Saga activities: txID first](references/feedback_saga_idempotency_key_first.md) — saga activities take `txID` as the first business arg after `ctx`; keeps the idempotency key visible in logs and UI.
- [Batch throttling on worker, not workflow](references/project_batch_throttling.md) — Batch pattern throttles via worker options in all four SDKs; no semaphore variant in the demo.
- [Keep CodeViewer snippets in sync](references/feedback_codeviewer_snippet_sync.md) — any edit to one language snippet must land alongside matching edits in the other three.
