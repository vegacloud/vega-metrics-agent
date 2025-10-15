# Copyright 2024 Vega Cloud, Inc.
#
# Use of this software is governed by the Business Source License
# included in the file licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with
# the Business Source License, use of this software will be governed
# by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.
# Variables
APPLICATION = vega-metrics-agent
DOCKER_IMAGE = public.ecr.aws/c0f8b9o4/vegacloud/${APPLICATION}
VERSION = $(shell cat pkg/config/VERSION)
DOCKER_IMAGE_DEV = public.ecr.aws/c0f8b9o4/vegacloud/${APPLICATION}-test
GOLANG_VERSION ?= 1.23

# Detect if we're using podman or docker
# Check if docker command exists and if it's actually podman emulating docker
CONTAINER_RUNTIME := $(shell if command -v docker >/dev/null 2>&1; then \
	if docker --version 2>&1 | grep -qi podman; then \
		echo "podman"; \
	else \
		echo "docker"; \
	fi; \
elif command -v podman >/dev/null 2>&1; then \
	echo "podman"; \
else \
	echo "docker"; \
fi)

# Go commands
GO_BUILD = CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/amd64/${APPLICATION}
GO_BUILD_ARM = CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/arm64/${APPLICATION}

GO_FMT = go fmt ./...
GO_LINT = golangci-lint run
GO_SEC = ${HOME}/go/bin/gosec ./...
GO_TEST = go test ./...
GO_VET = go vet ./...

# Docker/Podman commands (defined conditionally based on runtime)
ifeq ($(CONTAINER_RUNTIME),podman)
    # Use buildah for multi-architecture builds with Podman
    # Buildah works better with QEMU for cross-platform builds
    DOCKER_BUILD = buildah manifest rm ${DOCKER_IMAGE}:${VERSION} 2>/dev/null || true && \
	buildah manifest rm ${DOCKER_IMAGE}:latest 2>/dev/null || true && \
	buildah manifest create ${DOCKER_IMAGE}:${VERSION} && \
	buildah manifest create ${DOCKER_IMAGE}:latest && \
	buildah bud --arch amd64 \
		--build-arg golang_version=${GOLANG_VERSION} \
		--manifest ${DOCKER_IMAGE}:${VERSION} \
		-f Dockerfile . && \
	buildah bud --arch arm64 \
		--build-arg golang_version=${GOLANG_VERSION} \
		--manifest ${DOCKER_IMAGE}:${VERSION} \
		-f Dockerfile . && \
	buildah bud --arch amd64 \
		--build-arg golang_version=${GOLANG_VERSION} \
		--manifest ${DOCKER_IMAGE}:latest \
		-f Dockerfile . && \
	buildah bud --arch arm64 \
		--build-arg golang_version=${GOLANG_VERSION} \
		--manifest ${DOCKER_IMAGE}:latest \
		-f Dockerfile . && \
	buildah manifest push --all ${DOCKER_IMAGE}:${VERSION} docker://${DOCKER_IMAGE}:${VERSION} && \
	buildah manifest push --all ${DOCKER_IMAGE}:latest docker://${DOCKER_IMAGE}:latest

    DOCKER_BUILD_DEV = buildah manifest rm ${DOCKER_IMAGE_DEV}:${VERSION} 2>/dev/null || true && \
	buildah manifest create ${DOCKER_IMAGE_DEV}:${VERSION} && \
	buildah bud --arch amd64 \
		--build-arg golang_version=${GOLANG_VERSION} \
		--manifest ${DOCKER_IMAGE_DEV}:${VERSION} \
		-f Dockerfile . && \
	buildah bud --arch arm64 \
		--build-arg golang_version=${GOLANG_VERSION} \
		--manifest ${DOCKER_IMAGE_DEV}:${VERSION} \
		-f Dockerfile . && \
	buildah manifest push --all ${DOCKER_IMAGE_DEV}:${VERSION} docker://${DOCKER_IMAGE_DEV}:${VERSION}
else
    # Use docker buildx for multi-platform builds
    DOCKER_BUILD = docker buildx build -f Dockerfile \
		--build-arg golang_version=${GOLANG_VERSION} \
		--platform linux/amd64,linux/arm64 \
		-t ${DOCKER_IMAGE}:${VERSION} \
		-t ${DOCKER_IMAGE}:latest  \
		--push .

    DOCKER_BUILD_DEV = docker buildx build -f Dockerfile \
		--build-arg golang_version=${GOLANG_VERSION} \
		--platform linux/amd64,linux/arm64 \
		-t ${DOCKER_IMAGE_DEV}:${VERSION} \
		--push .
endif


# Default target
.PHONY: all
all: fmt vet lint sec test build docker-build

# Dev target
.PHONY: alldev
alldev: fmt vet lint sec build docker-build-dev

# Dev target without security checks
.PHONY: alldevnosec
alldevnosec: fmt vet lint build docker-build-dev


# Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	${GO_FMT}

# Run linters
.PHONY: lint
lint:
	@echo "Running Go linters..."
	${GO_LINT}

# Run security checks
.PHONY: sec
sec:
	@echo "Running security checks..."
	${GO_SEC}

# Run tests
.PHONY: test
test:
	@echo "Running Go tests..."
	${GO_TEST}

# Run Go vet
.PHONY: vet
vet:
	@echo "Running Go vet..."
	${GO_VET}

# Build Go binary locally
.PHONY: build
build:
	@echo "Building Go application locally..."
	mkdir -p bin
	${GO_BUILD}
	${GO_BUILD_ARM}

# Check which container runtime is detected
.PHONY: check-runtime
check-runtime:
	@echo "Detected container runtime: ${CONTAINER_RUNTIME}"
ifeq ($(CONTAINER_RUNTIME),podman)
	@echo "Build tool: buildah (for multi-architecture support)"
	@echo "Platforms: linux/amd64, linux/arm64"
else
	@echo "Build tool: docker buildx"
	@echo "Platforms: linux/amd64, linux/arm64"
endif

# Build Docker image
.PHONY: docker-build
docker-build: check-runtime
	@echo "Building Docker image using ${CONTAINER_RUNTIME}..."
	${DOCKER_BUILD}


# Build Docker image
.PHONY: docker-build-dev
docker-build-dev: check-runtime
	@echo "Building Docker dev image using ${CONTAINER_RUNTIME}..."
	${DOCKER_BUILD_DEV}


# Push Docker image
#.PHONY: docker-push
#docker-push:
#	docker push ${DOCKER_IMAGE}:${VERSION}
#	docker push ${DOCKER_IMAGE}:latest

#.PHONY: docker-push-dev
#docker-push-dev:
#	docker push ${DOCKER_IMAGE_DEV}:${VERSION}

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf bin/

# Help target
.PHONY: help
help:
	@echo "Usage:"
	@echo "  make all              - Format, vet, lint, sec, test, build locally, and build the Docker image"
	@echo "  make alldev           - Format, vet, lint, sec, build locally, and build the Docker dev image"
	@echo "  make alldevnosec      - Format, vet, lint, build locally, and build the Docker dev image (skip security checks)"
	@echo "  make fmt              - Format the Go code"
	@echo "  make lint             - Run Go linters"
	@echo "  make sec              - Run security checks"
	@echo "  make test             - Run Go tests"
	@echo "  make vet              - Run Go vet"
	@echo "  make build            - Build the Go application locally"
	@echo "  make docker-build     - Build multi-arch Docker image (Docker: buildx, Podman: buildah)"
	@echo "  make docker-build-dev - Build multi-arch Docker dev image (Docker: buildx, Podman: buildah)"
	@echo "  make check-runtime    - Display container runtime and build tool information"
	@echo "  make clean            - Clean build artifacts"
	@echo ""
	@echo "Note: Automatically detects Docker or Podman and uses appropriate build tool"
	@echo "      Docker uses 'buildx', Podman uses 'buildah' for multi-architecture builds"
