---
name: "@temporalio/client is loaded through createRequire"
description: "@temporalio/client 1.16+ breaks the Nuxt dev SSR bundle; the frontend loads it as CommonJS and force-traces it."
type: project
---

# @temporalio/client is loaded through createRequire

`frontend/server/utils/temporal.ts` loads
`@temporalio/client` with `createRequire`, not with a plain
`import`, and re-exports `WorkflowNotFoundError` so the whole
server shares that single load. `nuxt.config.ts` lists the
package under `nitro.externals.traceInclude`.

**Why:** from 1.16 onward the package makes Nitro emit
extension-less ESM imports of its sub-modules
(`/…/client/lib/async-completion-client`, no `file://` prefix
and no `.js`), which Node's ESM loader rejects. Every SSR route
then answers HTTP 500 — `server/utils/temporal.ts` is
auto-loaded for every render, so one broken bundle takes down
the whole site, not just the Temporal pages.

The trigger is the package, not the framework or the bundler:
it reproduces on Nuxt 4.1 (Nitro 2.12), Nuxt 4.4 (Nitro 2.13.4)
and Nuxt 4.5, the last of which builds with Vite 8 / rolldown
rather than esbuild. Versions ≤1.15 emit a single
`file:///…/client/lib/index.js` import and are unaffected.

**Only the dev bundle exhibits it.** `pnpm build` and the
container image succeed on the affected versions, so the E2E
net — which exercises only the container — cannot catch it.

Three constraints make the CommonJS load work:

- Resolution is anchored on `process.argv[1]`, the running
  server entry. See [[project_nitro_import_meta_url_placeholder]]
  for why `import.meta.url` is unusable here.
- `traceInclude` is required because a `require` is invisible
  to Nitro's dependency tracer; without it the package is
  absent from `.output/server/node_modules` and the container
  fails at boot.
- Both call sites share one load, so `instanceof
  WorkflowNotFoundError` compares against a single class
  identity.

Configuration-only alternatives do not work:
`nitro.externals.external`, `vite.ssr.external`,
`vite.ssr.noExternal`, `vite.optimizeDeps.include` and
`vite.ssr.optimizeDeps.include` all leave the malformed imports
in place. `nitro.externals.inline` on the client alone raises
`Cannot read properties of undefined (reading 'api')`; inlining
the whole `@temporalio` family raises `Cannot set properties of
undefined (setting 'Long')` from the protobufjs-generated
`@temporalio/proto/protos/root.js`.

**How to apply:** keep the `createRequire` load and the
`traceInclude` entry together — removing either one alone
breaks a different environment, and each carries a comment
pointing at the other. When bumping the package, validate on
three surfaces, because each catches what the others miss:
`pnpm dev` plus `curl` on `/`, `pnpm build`, and booting the
built server the way the container does
(`node .output/server/index.mjs`) plus `curl`. Should upstream
fix the published artifact, a plain `import` becomes possible
again and both pieces can go.
