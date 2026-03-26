FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/agent ./cmd/agent/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/agent /app/agent

# Default entrypoint runs the agent
ENTRYPOINT ["/app/agent"]
