# Build stage: compile a static Go binary
FROM golang:1.25  AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -ldflags="-s -w" -o fbref_exporter

# Final stage: use a minimal runtime image to avoid vulnerabilities
FROM scratch

# Copy the static binary from the builder stage
COPY --from=builder /app/fbref_exporter /fbref_exporter

EXPOSE 2112

ENTRYPOINT ["/fbref_exporter"]