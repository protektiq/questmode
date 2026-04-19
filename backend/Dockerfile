# syntax=docker/dockerfile:1
FROM golang:1.22-bookworm AS builder
WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY src ./src
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./src

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /
COPY --from=builder /server /server
USER nonroot:nonroot
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/server"]
