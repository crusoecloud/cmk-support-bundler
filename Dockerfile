# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /slurm-bundler ./cmd/slurm-bundler

# Final stage
FROM alpine:3.20

# Install ca-certificates, kubectl, and helm
RUN apk add --no-cache ca-certificates curl && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/ && \
    curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | sh

# Copy binary from builder
COPY --from=builder /slurm-bundler /slurm-bundler

# Run as non-root user
RUN adduser -D -u 1000 debuguser
USER debuguser

ENTRYPOINT ["/slurm-bundler"]
