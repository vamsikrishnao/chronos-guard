# ==============================================================================
# STAGE 1: Secure Build Environment
# ==============================================================================
FROM golang:1.26.5-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /build

# Cache dependency layers cleanly
COPY go.mod go.sum ./
RUN go mod download

# Copy application source tree
COPY . .

# Compile optimized production binary; strip debugging symbols (-s -w)
RUN go build -ldflags="-s -w" -o chronos-guard ./cmd/server/main.go

# ==============================================================================
# STAGE 2: Minimalist Runtime Environment
# ==============================================================================
FROM alpine:3.20

# Run as non-privileged system user to defend runtime layer
RUN adduser -D -u 10001 chronosuser
USER chronosuser

WORKDIR /app

# Pull only the compiled artifact from stage 1
COPY --from=builder /build/chronos-guard .

# Expose production gRPC infrastructure ports
EXPOSE 50051 9090

ENTRYPOINT ["./chronos-guard"]