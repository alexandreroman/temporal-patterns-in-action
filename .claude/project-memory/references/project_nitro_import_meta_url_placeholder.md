---
name: "Nitro rewrites import.meta.url to a placeholder"
description: "In built Nitro server chunks import.meta.url is file:///_entry.js; anchor path resolution on process.argv[1] instead."
type: project
---

# Nitro rewrites import.meta.url to a placeholder

In the frontend's built server output, `import.meta.url`
evaluates to the literal `file:///_entry.js` in every chunk
except the entry itself. Code in `frontend/server/` that needs
a real on-disk anchor uses `process.argv[1]` instead.

**Why:** Nitro's `import-meta` Rollup plugin performs that
substitution at build time. The entry chunk publishes the true
URL through `globalThis._importMeta_`, but it does so *after*
other chunks have already evaluated, so deferring the lookup to
call time does not help either — a Nitro plugin running during
chunk evaluation still reads the placeholder.

Anything resolving paths from `import.meta.url` therefore
resolves from the filesystem root. For
`createRequire(import.meta.url)` the result is
`MODULE_NOT_FOUND` at boot. `process.argv[1]` is the running
server entry, which sits next to the `node_modules` tree that
holds traced dependencies: `.output/server/index.mjs` in the
container, the Nuxt CLI under `frontend/node_modules` in dev.
One expression covers both, with no build-time branching.

**The failure is invisible to the standard checks.** `nuxt
typecheck`, `eslint`, `prettier --check` and `pnpm build` all
pass on a server that cannot start, because none of them boot
the output. See
[[project_temporalio_client_116_ssr_regression]] for the case
that surfaced it.

**How to apply:** in `frontend/server/` code, resolve paths
from `process.argv[1]`, never from `import.meta.url`. After any
change touching module resolution in the server bundle, boot
the built server (`node .output/server/index.mjs`) and `curl`
a route — that step is what distinguishes a working build from
one that merely compiles.
