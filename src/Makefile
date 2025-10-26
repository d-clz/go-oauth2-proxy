.PHONY: help build run test clean docker-build docker-push deploy

# Variables
APP_NAME := token-gateway
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
REGISTRY := your-registry
IMAGE := $(REGISTRY)/$(APP_NAME):$(VERSION)
IMAGE_LATEST := $(REGISTRY)/$(APP_NAME):latest

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Binary name
BINARY_NAME := token-gateway
BINARY_PATH := ./$(BINARY_NAME)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

build: deps ## Build the binary
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/gateway

run: ## Run locally (requires config.yaml and GOOGLE_APPLICATION_CREDENTIALS)
	$(GOCMD) run ./cmd/gateway/main.go -config config.yaml -log-level debug

run-prod: build ## Run built binary
	$(BINARY_PATH) -config config.yaml

test: ## Run tests
	$(GOTEST) -v ./...

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

docker-build: ## Build Docker image
	docker build -t $(IMAGE) -t $(IMAGE_LATEST) .

docker-push: docker-build ## Push Docker image to registry
	docker push $(IMAGE)
	docker push $(IMAGE_LATEST)

docker-run: docker-build ## Run Docker container locally
	docker run -p 8080:8080 \
		-v $(PWD)/config.yaml:/app/config.yaml \
		-v $(PWD)/key.json:/app/key.json \
		-e GOOGLE_APPLICATION_CREDENTIALS=/app/key.json \
		$(IMAGE)

lint: ## Run linters
	golangci-lint run ./...

fmt: ## Format code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

check: fmt vet lint ## Run all checks

# Development helpers
dev-setup: ## Setup development environment
	@echo "Installing dependencies..."
	$(GOGET) -u golang.org/x/tools/cmd/goimports
	@echo "Development environment ready!"

watch: ## Watch for changes and rebuild (requires entr)
	find . -name '*.go' | entr -r make run

# Quick test endpoints
test-health: ## Test health endpoint
	curl http://localhost:8080/healthz

test-ready: ## Test ready endpoint
	curl http://localhost:8080/readyz

test-metrics: ## Test metrics endpoint
	curl http://localhost:8080/metrics | jq

test-token-info: ## Test token-info endpoint
	curl http://localhost:8080/token-info | jq

test-proxy: ## Test proxy request
	curl -v http://localhost:8080/api/test

# Kubernetes deployment (requires kubectl)
k8s-deploy: ## Deploy to Kubernetes
	kubectl apply -f deployments/k8s/

k8s-logs: ## Follow logs in Kubernetes
	kubectl logs -f -l app=token-gateway -n auth-system

k8s-describe: ## Describe pod in Kubernetes
	kubectl describe pod -l app=token-gateway -n auth-system

k8s-delete: ## Delete from Kubernetes
	kubectl delete -f deployments/k8s/
