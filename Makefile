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

# Go commands
GO_BUILD = CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/amd64/${APPLICATION}
GO_BUILD_ARM = CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/arm64/${APPLICATION}

GO_FMT = go fmt ./...
GO_LINT = golangci-lint run
GO_SEC = ${HOME}/go/bin/gosec ./...
GO_TEST = go test ./...
GO_VET = go vet ./...

# Docker commands
DOCKER_BUILD = docker buildx build -f Dockerfile \
	--build-arg golang_version=${GOLANG_VERSION} \
	--build-arg app_version=${VERSION} \
	--platform linux/amd64,linux/arm64 \
	-t ${DOCKER_IMAGE}:${VERSION} \
	-t ${DOCKER_IMAGE}:latest  \
        --push .

DOCKER_BUILD_DEV = docker buildx build -f Dockerfile \
	--build-arg golang_version=${GOLANG_VERSION} \
	--build-arg app_version=${VERSION} \
	--platform linux/amd64,linux/arm64 \
	-t ${DOCKER_IMAGE_DEV}:${VERSION} \
        --push .


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

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	${DOCKER_BUILD}


# Build Docker image
.PHONY: docker-build-dev
docker-build-dev:
	@echo "Building Docker dev image..."
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
	@echo "  make fmt              - Format the Go code"
	@echo "  make lint             - Run Go linters"
	@echo "  make sec              - Run security checks"
	@echo "  make test             - Run Go tests"
	@echo "  make vet              - Run Go vet"
	@echo "  make build            - Build the Go application locally"
	@echo "  make docker-build     - Build Docker image using Dockerfile"
	@echo "  make docker-push      - Push Docker image to registry"
	@echo "  make clean            - Clean build artifacts"
