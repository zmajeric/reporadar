
# Simple Dockerfile for go-ingest service (to be refined later)
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o go-ingest ./...

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/go-ingest /app/go-ingest

ENV PORT=8080
CMD ["/app/go-ingest"]
