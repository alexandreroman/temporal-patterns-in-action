# syntax=docker/dockerfile:1
# Build context: repo root — we need both ./workers (for the shared
# encryption package) and ./codec-server (this binary).
FROM golang:1.25-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux
COPY workers ./workers
COPY codec-server ./codec-server
WORKDIR /src/codec-server
# Explicit id= on the cache mounts is required: under podman/buildah an
# id-less type=cache mount on the Go module cache breaks `go build` module
# resolution ("no required module provides package" even though the modules
# are present and `go list -m all` resolves them). With a stable id it works
# and both caches persist across builds. Same rule as the worker Dockerfiles.
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build -ldflags='-s -w' -o /out/codec-server .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/codec-server /usr/local/bin/codec-server
EXPOSE 8888
ENTRYPOINT ["/usr/local/bin/codec-server"]
