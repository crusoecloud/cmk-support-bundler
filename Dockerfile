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
ENV HELM_VERSION=v3.17.3
RUN apk add --no-cache ca-certificates curl && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/ && \
    curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
    rm -rf /tmp/linux-amd64

# Copy binary from builder
COPY --from=builder /slurm-bundler /slurm-bundler

# Run as non-root user
RUN adduser -D -u 1000 debuguser
USER debuguser

ENTRYPOINT ["/slurm-bundler"]
