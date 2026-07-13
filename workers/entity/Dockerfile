# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux
COPY go.mod go.sum ./
# Explicit id= on the cache mounts is required: under podman/buildah an
# id-less type=cache mount on the Go module cache breaks `go build` module
# resolution ("no required module provides package"). With a stable id it
# works and both the module and build caches persist across builds.
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build -ldflags='-s -w' -o /out/worker ./entity/cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]
