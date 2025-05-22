# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git curl

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for Linux container
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/vote .

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/vote .

# Copy config files
COPY config/config.yaml ./config/
COPY prometheus.yml ./

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=development

# Run the application
CMD ["./vote"]