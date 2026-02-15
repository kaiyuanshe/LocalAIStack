# Makefile for LocalAIStack

# Variables
BINARY_NAME=las
SERVER_BINARY=$(BINARY_NAME)-server
CLI_BINARY=$(BINARY_NAME)
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
LDFLAGS=-ldflags "-X github.com/zhuangbiaowei/LocalAIStack/internal/config.Version=$(VERSION) -s -w"
COVERAGE_THRESHOLD ?= 40

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Platform-specific build
GOOS_LINUX=linux
GOARCH_AMD64=amd64
GOARCH_ARM64=arm64

.PHONY: all
all: clean test build

.PHONY: help
help:
	@echo "Make targets:"
	@echo "  make all           - Run clean, test, and build"
	@echo "  make build         - Build binaries for current platform"
	@echo "  make build-server  - Build server binary"
	@echo "  make build-cli     - Build CLI binary"
	@echo "  make build-all     - Build binaries for all platforms"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make fmt           - Format Go code"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make deps          - Download dependencies"
	@echo "  make tidy          - Tidy go.mod"
	@echo "  make run-server    - Run server (development)"
	@echo "  make run-cli       - Run CLI (development)"

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(SERVER_BINARY) $(CLI_BINARY)
	@echo "Done"

.PHONY: deps
deps:
	$(GOCMD) mod download

.PHONY: tidy
tidy:
	$(GOMOD) tidy

.PHONY: build
build: build-server build-cli

.PHONY: build-server
build-server:
	@echo "Building server binary..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY) ./cmd/server
	@echo "Server binary: $(BUILD_DIR)/$(SERVER_BINARY)"

.PHONY: build-cli
build-cli:
	@echo "Building CLI binary..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY) ./cmd/cli
	@echo "CLI binary: $(BUILD_DIR)/$(CLI_BINARY)"

.PHONY: build-all
build-all:
	@echo "Building binaries for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building for linux/amd64..."
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_AMD64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-linux-amd64 ./cmd/server
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_AMD64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-linux-amd64 ./cmd/cli
	@echo "Building for linux/arm64..."
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_ARM64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-linux-arm64 ./cmd/server
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_ARM64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-linux-arm64 ./cmd/cli
	@echo "Done"

.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test
	@echo "Coverage report:"
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@coverage=$$($(GOCMD) tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}'); \
	echo "Total coverage: $$coverage%"; \
	awk -v coverage="$$coverage" -v threshold="$(COVERAGE_THRESHOLD)" 'BEGIN { if (coverage + 0 < threshold) { printf "Coverage %s%% is below threshold %s%%\\n", coverage, threshold; exit 1 } }'

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/"; \
	fi

.PHONY: install
install: build
	@echo "Installing binaries..."
	@install -d /usr/local/bin
	@install -m 755 $(BUILD_DIR)/$(SERVER_BINARY) /usr/local/bin/$(SERVER_BINARY)
	@install -m 755 $(BUILD_DIR)/$(CLI_BINARY) /usr/local/bin/$(CLI_BINARY)
	@echo "Installed to /usr/local/bin"

.PHONY: uninstall
uninstall:
	@echo "Uninstalling binaries..."
	@rm -f /usr/local/bin/$(SERVER_BINARY) /usr/local/bin/$(CLI_BINARY)
	@echo "Uninstalled"

.PHONY: run-server
run-server: build-server
	$(BUILD_DIR)/$(SERVER_BINARY)

.PHONY: run-cli
run-cli: build-cli
	$(BUILD_DIR)/$(CLI_BINARY) --help

# Development targets
.PHONY: dev
dev:
	@echo "Development environment setup..."
	$(GOCMD) mod tidy
	$(GOCMD) mod download
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY) ./cmd/server
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY) ./cmd/cli

# Docker variables
REGISTRY ?= ghcr.io/zhuangbiaowei
IMAGE_NAME ?= localaistack
DOCKER_TARGET ?= runtime-alpine

.PHONY: docker-build
docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--target $(DOCKER_TARGET) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		.

.PHONY: docker-build-cuda
docker-build-cuda:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--target runtime-cuda \
		-t $(IMAGE_NAME):$(VERSION)-cuda \
		-t $(IMAGE_NAME):cuda \
		.

.PHONY: docker-build-all
docker-build-all: docker-build docker-build-cuda

.PHONY: docker-buildx
docker-buildx:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--target $(DOCKER_TARGET) \
		-t $(REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		-t $(REGISTRY)/$(IMAGE_NAME):latest \
		--push .

.PHONY: docker-buildx-cuda
docker-buildx-cuda:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--target runtime-cuda \
		-t $(REGISTRY)/$(IMAGE_NAME):$(VERSION)-cuda \
		-t $(REGISTRY)/$(IMAGE_NAME):cuda \
		--push .

.PHONY: docker-run
docker-run:
	docker run --rm -it \
		-v ~/.localaistack:/home/appuser/.localaistack \
		-v ~/.ollama:/home/appuser/.ollama \
		$(IMAGE_NAME):$(VERSION) --help

.PHONY: docker-run-gpu
docker-run-gpu:
	docker run --rm -it --gpus all \
		-v ~/.localaistack:/home/appuser/.localaistack \
		-v ~/.ollama:/home/appuser/.ollama \
		$(IMAGE_NAME):$(VERSION)-cuda --help

.PHONY: docker-push
docker-push: docker-build
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest
