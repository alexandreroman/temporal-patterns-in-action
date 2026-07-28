---
name: "pnpm settings live in pnpm-workspace.yaml"
description: "Frontend pnpm config belongs in frontend/pnpm-workspace.yaml; allowBuilds governs dependency build scripts"
type: project
---

# pnpm settings live in pnpm-workspace.yaml

All frontend pnpm settings live in
`frontend/pnpm-workspace.yaml`. Dependency build scripts are
governed by the `allowBuilds` map, which sets `esbuild`,
`protobufjs` and `unrs-resolver` to `false`. The same file
carries `savePrefix: "~"` (every range in `package.json` uses a
tilde) and `engineStrict: true` (enforces the `engines.node`
floor).

**Why:** pnpm 11 reads settings only from
`pnpm-workspace.yaml` (or the global
`~/.config/pnpm/config.yaml`). A `pnpm` field in
`package.json` is silently ignored, and `.npmrc` is honoured
only for auth and registry keys. `allowBuilds`
supersedes `onlyBuiltDependencies`,
`ignoredBuiltDependencies`, `neverBuiltDependencies` and
`ignoreDepScripts`. Any package with a build script that is
neither allowed nor explicitly denied fails the install with
`ERR_PNPM_IGNORED_BUILDS`, which breaks both `make bootstrap`
and the frontend image build. The three denied packages are
verified to work with their build scripts skipped.

**How to apply:** put every pnpm setting in
`frontend/pnpm-workspace.yaml`, never in `package.json` and
never in a `.npmrc`. When a dependency bump introduces a new
package with a build script, add an explicit `true`/`false`
entry under `allowBuilds`. Keep `pnpm-workspace.yaml` in the
`COPY` list of the `deps` stage in `frontend/Dockerfile`
alongside `package.json` and `pnpm-lock.yaml`, or the image
build loses the setting and fails. Verify with
`cd frontend && CI=true pnpm install --frozen-lockfile` and
`podman build --target deps .`.
