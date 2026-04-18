FROM golang:1.25-alpine AS build
ARG PATTERN=saga
WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags='-s -w' -o /out/worker ./${PATTERN}/cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]
