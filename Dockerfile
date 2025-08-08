# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o flickr cmd/flickr/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1000 flickr && \
    adduser -D -u 1000 -G flickr flickr

# Copy binary from builder
COPY --from=builder /build/flickr /usr/local/bin/flickr

# Set ownership
RUN chown flickr:flickr /usr/local/bin/flickr

# Switch to non-root user
USER flickr

# Set entrypoint
ENTRYPOINT ["flickr"]

# Default command (show help)
CMD ["--help"]