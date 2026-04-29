# projects/Dockerfile
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# RUN wget -qO /bin/grpc_health_probe \
#     https://github.com/grpc-ecosystem/grpc-health-probe/releases/latest/download/grpc_health_probe-linux-amd64 \
#     && chmod +x /bin/grpc_health_probe

COPY . .
RUN go build -o service ./cmd/sources/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/service .
COPY --from=builder /app/configs ./configs
# COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe


CMD ["./service"]