# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o microcron-ce ./cmd/microcron-ce

# Runtime stage
FROM alpine:latest

# Install bash and curl for health checks
RUN apk add --no-cache bash curl ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/microcron-ce .

# Create log directory
RUN mkdir -p /var/log/microcron-ce && chmod 777 /var/log/microcron-ce

# Run as non-root user
RUN addgroup -g 1000 microcron && \
    adduser -D -u 1000 -G microcron microcron && \
    chown -R microcron:microcron /app /var/log/microcron-ce

USER microcron

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD ps aux | grep microcron-ce | grep -v grep || exit 1

# Run the application
ENTRYPOINT ["./microcron-ce"]
CMD ["--namespace=default", "--configmap=microcron-scripts", "--log-dir=/var/log/microcron-ce", "--retention-days=7"]
