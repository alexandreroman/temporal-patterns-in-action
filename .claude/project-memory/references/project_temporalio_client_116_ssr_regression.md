---
name: "@temporalio/client 1.16+ breaks Nuxt SSR dev bundle"
description: "@temporalio/client 1.16 and 1.17 generate broken extension-less ESM imports in .nuxt/dev/index.mjs; pinned to ~1.15.0 as workaround on 2026-05-09."
type: project
---

# @temporalio/client 1.16+ breaks Nuxt SSR dev bundle

`@temporalio/client` versions ≥1.16 cause
`.nuxt/dev/index.mjs` to import 14 sub-modules
of the package (e.g.
`/.../client/lib/async-completion-client`)
as bare absolute paths without `.js`
extensions, which Node's ESM loader rejects:

> Cannot find module
> '/.../client/lib/async-completion-client'
> imported from
> '/.../.nuxt/dev/index.mjs'. Did you mean to
> import '...async-completion-client.js'?

The dev server then loops on the
"`.nuxt/dist directory has been removed.
Restarting Nuxt...`" placeholder and serves
HTTP 503 on every route. Working versions
(≤1.15) emit a single
`import ... from 'file:///.../client/lib/index.js'`
because Vite's optimizeDeps wraps the package
into one ESM module.

**Bisect (2026-05-09):**

| client version | result |
| -------------- | ------ |
| 1.13.x         | OK     |
| 1.14.0         | OK     |
| 1.15.0         | OK     |
| 1.16.0         | BROKEN |
| 1.17.0 / 1.17.1 | BROKEN |

Tested with Nuxt 4.1 (Nitro 2.12) and Nuxt
4.4 (Nitro 2.13.4) — same regression on both,
so the trigger is the package, not the Nuxt
bump in commit `84bfd65`. The `devtools`
config is also unrelated (broken with both
`enabled: true` and `enabled: false`).

The SDK bump on `84bfd65` ("Bump Temporal
SDKs and frontend dependencies") set the
range to `~1.17.0`. The reason it appeared to
work in the Priority feature worktree but not
on main: the long-running dev server in that
worktree was started before the bump and kept
serving its old in-memory bundle until cmux
pre-destroy killed it; the next fresh boot
on main hit the broken bundler path.

**Why:** keeping the dev server unblocked
matters more than chasing the upstream bug —
the affected import is `frontend/server/utils/
temporal.ts`, which Nuxt auto-loads for every
SSR render, so a single broken bundle takes
down every route, not just the pages that
talk to Temporal.

**How to apply:**

- Keep `@temporalio/client` at `~1.15.0` in
  `frontend/package.json` until upstream
  Nitro/Vite handle the post-1.16 package
  shape — or the Temporal SDK fixes the
  publishing artifact that trips the
  bundler. Lift the pin only after a
  successful smoke run of `pnpm dev` and a
  `curl -fsS http://localhost:3000/`.
- The following knobs were tried and did
  NOT help: `nitro.externals.external`,
  `nitro.externals.inline`,
  `vite.ssr.external`, `vite.ssr.noExternal`,
  `vite.optimizeDeps.include`,
  `vite.ssr.optimizeDeps.include`, fresh
  `pnpm install`, wiping `.nuxt` and
  `node_modules/.cache`. Don't burn cycles
  retrying these unless the upstream
  changelog says otherwise.
- Symptom recognition: if the Nuxt dev page
  is stuck on the loader (`__NUXT_LOADING__`
  in the HTML, HTTP 503), grep
  `frontend/.nuxt/dev/index.mjs` for
  `@temporalio/client` — if the imports lack
  `.js` and the `file://` prefix, it's this
  regression.
