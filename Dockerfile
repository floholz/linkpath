FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o linkpath .

# ===========================
# Runtime stage
# ===========================
FROM alpine:3.19

LABEL org.opencontainers.image.title="LinkPath" \
      org.opencontainers.image.description="URL-path-aware link and note aggregator" \
      org.opencontainers.image.url="https://github.com/floholz/linkpath" \
      org.opencontainers.image.source="https://github.com/floholz/linkpath" \
      org.opencontainers.image.licenses="GPL-3.0-only"

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/linkpath .

# Create data directory
RUN mkdir -p /app/pb_data

EXPOSE 8090
EXPOSE 8080

VOLUME ["/app/pb_data"]

ENTRYPOINT ["/app/linkpath"]
CMD ["serve", "--http=0.0.0.0:8090", "--app-http=0.0.0.0:8080"]
