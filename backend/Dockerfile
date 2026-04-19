# syntax=docker/dockerfile:1
FROM cgr.dev/chainguard/go:latest AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=0 go build -o /app/backend ./cmd

FROM cgr.dev/chainguard/glibc-dynamic:latest
COPY --from=builder /app/backend /usr/local/bin/backend
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/usr/local/bin/backend"]
