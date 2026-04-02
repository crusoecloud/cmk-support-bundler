# CMK Support Bundler Makefile

IMAGE ?= ghcr.io/crusoecloud/cmk/cmk-support-bundler
VERSION ?= latest
CONTAINER_TOOL ?= docker

.PHONY: all
all: build

##@ Development

.PHONY: build
build: ## Build the slurm-bundler binary
	go build -o bin/slurm-bundler ./cmd/slurm-bundler

.PHONY: run
run: build ## Run locally (requires kubeconfig with access to slurm namespace)
	./bin/slurm-bundler

.PHONY: test
test: ## Run unit tests
	go test -v ./...

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

##@ Build

.PHONY: docker-build
docker-build: ## Build Docker image
	$(CONTAINER_TOOL) build -t $(IMAGE):$(VERSION) .

.PHONY: docker-push
docker-push: ## Push Docker image
	$(CONTAINER_TOOL) push $(IMAGE):$(VERSION)

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
