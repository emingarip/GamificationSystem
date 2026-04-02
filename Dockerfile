# Build Stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY internal/muscle/go.mod internal/muscle/go.sum ./
COPY internal/muscle/ ./internal/muscle/

# Download dependencies
RUN go mod download

# Build the application
WORKDIR /app/internal/muscle
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gamification-server .

# Production Stage
FROM alpine:3.19

# Install minimal dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/internal/muscle/gamification-server .

# Copy configuration files
COPY --chown=appuser:appgroup .env* ./
COPY docker-compose.yml ./

# Copy docs portal build output
COPY --chown=appuser:appgroup docs-portal/build ./docs-portal/build
COPY --chown=appuser:appgroup docs-portal/static ./docs-portal/static

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 3000 3001

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Run the application
CMD ["./gamification-server"]
