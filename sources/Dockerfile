# projects/Dockerfile
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# ADD GRPC HEALTH PROBE FOR CHECK SERVICE STATUS
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; else ARCH="amd64"; fi && \
    wget -qO /bin/grpc_health_probe \
    https://github.com/grpc-ecosystem/grpc-health-probe/releases/latest/download/grpc_health_probe-linux-${ARCH} \
    && chmod +x /bin/grpc_health_probe

COPY . .
# RUN MIGRATIONS
RUN go run ./cmd/migrate/main.go
# BUILD APP
RUN go build -o service ./cmd/sources/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/service .
COPY --from=builder /app/configs ./configs
# COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe


CMD ["./service"]