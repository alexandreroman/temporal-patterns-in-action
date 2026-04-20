# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags='-s -w' -o /out/worker ./batch/cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]
