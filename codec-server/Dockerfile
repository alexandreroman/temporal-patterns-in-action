# syntax=docker/dockerfile:1
# Build context: repo root — we need both ./workers (for the shared
# encryption package) and ./codec-server (this binary).
FROM golang:1.27-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux
# Both manifests are needed to resolve the module graph: codec-server's
# go.mod replaces the workers module with ../workers. Copying only the
# manifests first keeps the download layer cached across source edits.
COPY workers/go.mod workers/go.sum ./workers/
COPY codec-server/go.mod codec-server/go.sum ./codec-server/
WORKDIR /src/codec-server
# Explicit id= on the cache mounts is required: under podman/buildah an
# id-less type=cache mount on the Go module cache breaks `go build` module
# resolution ("no required module provides package" even though the modules
# are present and `go list -m all` resolves them). With a stable id it works
# and both caches persist across builds. Same rule as the worker Dockerfiles.
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download
COPY workers /src/workers
COPY codec-server /src/codec-server
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build -trimpath -ldflags='-s -w' -o /out/codec-server .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/codec-server /usr/local/bin/codec-server
EXPOSE 8888
ENTRYPOINT ["/usr/local/bin/codec-server"]
