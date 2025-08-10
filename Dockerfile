# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o actionlint-mcp .

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    shellcheck \
    py3-pyflakes

# Create non-root user
RUN addgroup -g 1001 -S mcp && \
    adduser -u 1001 -S mcp -G mcp

# Copy binary from builder
COPY --from=builder /app/actionlint-mcp /usr/local/bin/actionlint-mcp

# Set ownership
RUN chown mcp:mcp /usr/local/bin/actionlint-mcp && \
    chmod +x /usr/local/bin/actionlint-mcp

# Switch to non-root user
USER mcp

# Set environment variables for optional tools
ENV SHELLCHECK_COMMAND=shellcheck \
    PYFLAKES_COMMAND=pyflakes

# Expose MCP default port (if needed)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["actionlint-mcp", "-version"] || exit 1

# Run the MCP server
ENTRYPOINT ["actionlint-mcp"]