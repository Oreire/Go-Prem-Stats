# # ---------------------
# # Build stage: compile a static Go binary
# # ---------------------
# FROM golang:1.25 AS builder

# WORKDIR /app

# # Install dependencies
# COPY go.mod go.sum ./
# RUN go mod download

# # Copy source and build a static binary
# COPY . .
# RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
#     go build -ldflags="-s -w" -o fbref_exporter

# # ---------------------
# # Final stage: minimal runtime
# # ---------------------
# FROM alpine:latest

# # Install CA certificates for HTTPS requests
# RUN apk --no-cache add ca-certificates curl

# # Create a non-root user for better security
# RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# USER appuser

# WORKDIR /app

# # Copy the static binary from the builder stage
# COPY --from=builder /app/fbref_exporter /app/fbref_exporter

# # Expose metrics port
# EXPOSE 2113

# # Run the exporter
# ENTRYPOINT ["/app/fbref_exporter"]

# ------------------------ Builder Stage ------------------------
FROM golang:1.25-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go modules files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -o fbref_exporter .

# ------------------------ Final Stage ------------------------
FROM alpine:latest

# Install certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/fbref_exporter .

# Expose metrics port
EXPOSE 2113

# Start exporter
ENTRYPOINT ["./fbref_exporter"]
