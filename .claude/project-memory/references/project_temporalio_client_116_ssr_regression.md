---
name: "@temporalio/client 1.16+ breaks Nuxt SSR dev bundle"
description: "@temporalio/client 1.16 and 1.17 generate broken extension-less ESM imports in .nuxt/dev/index.mjs; pinned to ~1.15.0 as a workaround."
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

**Bisect:**

| client version | result |
| -------------- | ------ |
| 1.13.x         | OK     |
| 1.14.0         | OK     |
| 1.15.0         | OK     |
| 1.16.0         | BROKEN |
| 1.17.0 / 1.17.1 | BROKEN |

The regression reproduces on both Nuxt 4.1
(Nitro 2.12) and Nuxt 4.4 (Nitro 2.13.4), so
the trigger is the package, not Nuxt. The
`devtools` config is unrelated (broken with
both `enabled: true` and `enabled: false`).

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
