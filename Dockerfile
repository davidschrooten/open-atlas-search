FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o open-atlas-search .

# Final stage
FROM alpine:latest

# Install runtime dependencies including wget for health checks
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user for security
RUN addgroup -g 1001 appuser && \
    adduser -D -u 1001 -G appuser appuser

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/open-atlas-search .

# Create directories for indexes and config
RUN mkdir -p /var/lib/indexes /etc/config && \
    chown -R appuser:appuser /app /var/lib/indexes /etc/config

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check for container runtime
HEALTHCHECK --interval=30s --timeout=3s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./open-atlas-search", "server"]
