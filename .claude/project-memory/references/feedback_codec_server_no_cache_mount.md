---
name: "Go Dockerfiles: cache mounts need an explicit id"
description: "podman breaks go build for id-less type=cache mounts; always set id=gomod/id=gobuild"
type: feedback
---

# Go Dockerfiles: cache mounts need an explicit id

Every Go build Dockerfile (`workers/*/Dockerfile` and
`codec-server/Dockerfile`) uses `--mount=type=cache` with an
explicit `id`: `id=gomod` for `/go/pkg/mod` and `id=gobuild`
for `/root/.cache/go-build`. Never use an `id`-less cache mount
on the Go module cache.

**Why:** on this machine (podman 5.8.3, overlay driver) an
`id`-less `type=cache` mount on `/go/pkg/mod` breaks `go build`
package→module resolution — the build fails with `no required
module provides package github.com/...` even though `go mod
download` populated the cache, the module is physically
present, and `go list -m all` resolves it. Adding a stable
`id` fixes it completely; it reproduces in a single isolated
build and affects both the self-contained worker module and
the codec-server module that has a local `replace`
(=> ../workers), so it is not `replace`-specific and not a
concurrency race. This matches the skillbox `go-rules`
container template, which already mandates explicit ids
(`skills/go-rules/references/containers.md`).

**How to apply:** keep
`RUN --mount=type=cache,id=gomod,target=/go/pkg/mod go mod
download` and the build step with both
`id=gomod` + `id=gobuild` mounts. Use the same id across every
`RUN` so the caches are shared and persist across builds. If a
new Go container build is added, follow the same rule. Verify
the whole stack with `make app-up`. Relates to
[[project_codec_server_ui_endpoint]].
