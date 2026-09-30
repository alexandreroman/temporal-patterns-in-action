# Project Memory

> When a new decision **contradicts** an existing
> memory note, do NOT silently override it.
> Instead: surface the conflict, quote the
> existing memory, explain how the new decision
> differs, and ask for explicit confirmation
> before updating. **Do NOT take any action** —
> no tool calls, no file writes — until confirmed.

> **Note wording** — state permanent facts in the
> present tense. A note read out of context must
> not reveal what it replaces or what just
> happened. Ban narration markers: "now", "no
> longer", "previously / used to", "reverses /
> replaces", "kept", "changed to", "reintroduce",
> "the user asked to". Phrase prohibitions
> positively ("the API is versioned under /v2"),
> not as the negation of a former state. Test:
> remove the note from its context — if a sentence
> only makes sense knowing the prior state,
> rewrite it.

## Working rules

- [Coding conventions](references/feedback_coding_conventions.md) — line lengths, markdown style, LTS rule.
- [Agent delegation](references/feedback_agent_delegation.md) — code-writer for code, code-reviewer for reviews, temporal skill.
- [Demo-first priorities](references/feedback_demo_priorities.md) — favour visibility and short forms; skip production robustness.
- [Runbook: new pattern](references/project_adding_new_pattern.md) — six steps: workers/, main, Dockerfile+compose, PATTERNS, frontend, README.

## Architecture and Temporal

- [Event architecture (NATS)](references/event-architecture.md) — subjects, envelope, progress/business split, kebab-case steps.
- [Temporal conventions](references/feedback_temporal_conventions.md) — determinism, `workflowcheck`, task-queue/workflow-name contract.
- [Saga activities: txID first](references/feedback_saga_idempotency_key_first.md) — idempotency key is the first arg after `ctx`.
- [Batch throttling: in-workflow sliding window](references/project_batch_throttling.md) — Selector window caps in-flight children.
- [Announce activities → ExecuteLocalActivity](references/feedback_announce_use_local_activity.md) — regular announces inherit key 3 and queue.
- [Priority pattern: top-level workflow per ticket](references/feedback_priority_top_level_workflow.md) — local activity + ExecuteWorkflow.
- [Don't abort priority-fairness runs mid-flight](references/feedback_priority_fairness_repro_load.md) — 120+ workflows per run; serialise.
- [Codec Server is opt-in by design](references/project_codec_server_ui_endpoint.md) — no --ui-codec-endpoint; glasses icon toggle.
- [Rogue host workers](references/feedback_rogue_host_workers.md) — a stale host `go run` worker steals tasks; check pollers first.
- [Verify dynamic config keys](references/project_verify_dynamic_config_keys.md) — start-dev ignores unknown keys; wrong-type probe.

## Frontend

- [Frontend component conventions](references/feedback_frontend_component_conventions.md) — generic shells + `<Pattern>*.vue` wrappers.
- [Keep CodeViewer snippets in sync](references/feedback_codeviewer_snippet_sync.md) — mirror all four languages; recompute ranges.
- [CodeViewer snippet idioms](references/feedback_codeviewer_snippet_idioms.md) — per-SDK conventions: options, failures, Java/Python/TS.
- [SSE start-up handshake](references/feedback_sse_startup_handshake.md) — subscribe → nc.flush() → first push; flush:"sync" watch.
- [Default scenario to happy path](references/feedback_default_scenario_happy_path.md) — scenario selectors default to success.
- [Realistic animated token counters](references/feedback_token_counters.md) — non-round tokens from the worker + `useCountTween`.
- [Stable Vue keys for placeholder items](references/feedback_stable_keys_for_placeholder_messages.md) — placeholder and row share a key.
- [Status bar: no out-in transition](references/feedback_statusbar_no_outin_transition.md) — wedges on hidden tabs; use CSS keyframes.
- [Dynamic NuxtLink via <component :is>](references/feedback_nuxtlink_dynamic_component.md) — `resolveComponent("NuxtLink")`.
- [Nuxt SSR browser globals](references/feedback_nuxt_ssr_browser_globals.md) — guard in onMounted; smoke routes with `curl`.
- [Nuxt server env vars: process.env](references/feedback_nuxt_runtime_env.md) — runtimeConfig defaults bake at build time.
- [@temporalio/client via createRequire](references/project_temporalio_client_116_ssr_regression.md) — ESM imports break; CJS + traceInclude.
- [Nitro import.meta.url placeholder](references/project_nitro_import_meta_url_placeholder.md) — anchor on `process.argv[1]`; boot output.
- [pnpm settings in pnpm-workspace.yaml](references/project_pnpm_config_in_workspace_yaml.md) — `allowBuilds`; Dockerfile copies it.

## Infra and workspaces

- [Casper compose/dev port isolation](references/project_casper_compose_port_isolation.md) — override from CASPER_PORT; `!override`.
- [cmux compose port isolation](references/project_cmux_compose_port_isolation.md) — Makefile defaults CASPER_PORT to CMUX_PORT.
- [Casper info panel mirrors make endpoints](references/feedback_casper_info_panel.md) — app-up/dev publish, down targets clear.
- [Browser tests: casper load is a hidden page](references/feedback_casper_browser_hidden_page.md) — use `casper browser open`.
- [Node healthcheck: use 127.0.0.1](references/feedback_node_healthcheck_ipv6.md) — busybox wget tries `::1`; Nuxt is IPv4.
