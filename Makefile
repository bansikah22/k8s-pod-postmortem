# k8s-pod-postmortem Makefile
# Build, test, and release automation

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary names
BINARY_NAME=postmortem
BINARY_UNIX=$(BINARY_NAME)_unix

# Directories
CMD_DIR=./cmd/postmortem
INTERNAL_DIR=./internal
TEST_DIR=./test

# Build parameters
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Docker parameters
DOCKER_CMD=docker
DOCKER_BUILD=$(DOCKER_CMD) build
DOCKER_TAG=$(DOCKER_CMD) tag
DOCKER_PUSH=$(DOCKER_CMD) push
IMAGE_NAME?=k8s-pod-postmortem
IMAGE_TAG?=$(VERSION)
REGISTRY?=ghcr.io
REPO?=$(USER)/$(IMAGE_NAME)

# Security tools
TRIVY=trivy
GRYPE=grype
SYFT=syft
COSIGN=cosign

.PHONY: all build clean test coverage lint fmt vet security-scan help

all: clean deps lint test build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) $(CMD_DIR)

## build-linux: Build for Linux (used in Docker)
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_UNIX) $(CMD_DIR)

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	rm -f $(BINARY_NAME)

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## test: Run unit tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## coverage: Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run --timeout 5m ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w $(INTERNAL_DIR) $(CMD_DIR)

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## security-scan: Run security scanners
security-scan:
	@echo "Running security scan..."
	$(TRIVY) fs --severity HIGH,CRITICAL .
	$(GRYPE) dir:. --only-fixed

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	$(DOCKER_BUILD) -t $(REGISTRY)/$(REPO):$(IMAGE_TAG) .
	$(DOCKER_BUILD) -t $(REGISTRY)/$(REPO):latest .

## docker-push: Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	$(DOCKER_CMD) login $(REGISTRY)
	$(DOCKER_PUSH) $(REGISTRY)/$(REPO):$(IMAGE_TAG)
	$(DOCKER_PUSH) $(REGISTRY)/$(REPO):latest

## docker-scan: Scan Docker image for vulnerabilities
docker-scan:
	@echo "Scanning Docker image..."
	$(TRIVY) image $(REGISTRY)/$(REPO):$(IMAGE_TAG)

## sign-image: Sign Docker image with cosign
sign-image:
	@echo "Signing Docker image..."
	$(COSIGN) sign $(REGISTRY)/$(REPO):$(IMAGE_TAG)
	$(COSIGN) sign $(REGISTRY)/$(REPO):latest

## sbom: Generate Software Bill of Materials
sbom:
	@echo "Generating SBOM..."
	$(SYFT) packages dir:. -o spdx-json > sbom.spdx.json
	$(SYFT) packages docker:$(REGISTRY)/$(REPO):$(IMAGE_TAG) -o spdx-json > sbom-image.spdx.json

## helm: Package Helm chart
helm:
	@echo "Packaging Helm chart..."
	helm package charts/k8s-pod-postmortem -d dist/

## helm-install: Install Helm chart locally
helm-install:
	@echo "Installing Helm chart..."
	helm install k8s-pod-postmortem charts/k8s-pod-postmortem

## helm-uninstall: Uninstall Helm chart
helm-uninstall:
	@echo "Uninstalling Helm chart..."
	helm uninstall k8s-pod-postmortem

## install: Install binary to GOPATH/bin
install:
	@echo "Installing..."
	$(GOCMD) install $(LDFLAGS) $(CMD_DIR)

## check: Run all checks
check: deps fmt vet lint test security-scan

## release: Create a release
release: clean deps test build docker-build sbom sign-image docker-push

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'

.DEFAULT_GOAL := help