# Build stage
FROM golang:1.22-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o arangodb-bk-restore .

# Final stage
FROM scratch

# Copy ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/arangodb-bk-restore /arangodb-bk-restore

# Copy example config
COPY --from=builder /app/config.yml.example /config.yml.example

# Set entrypoint
ENTRYPOINT ["/arangodb-bk-restore"]

# Default command
CMD ["--help"]

# Metadata
LABEL org.opencontainers.image.title="ArangoDB Backup & Restore Tool"
LABEL org.opencontainers.image.description="CLI tool for backing up and restoring ArangoDB databases with S3-compatible storage"
LABEL org.opencontainers.image.vendor="Apito"
LABEL org.opencontainers.image.source="https://github.com/apito-io/arangodb-bk-restore"
LABEL org.opencontainers.image.documentation="https://github.com/apito-io/arangodb-bk-restore/blob/main/README.md"
