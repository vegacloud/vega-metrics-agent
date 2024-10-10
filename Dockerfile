# Copyright 2024 Vega Cloud, Inc.
#
# Use of this software is governed by the Business Source License
# included in the file licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with
# the Business Source License, use of this software will be governed
# by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.
# Build Stage
ARG golang_version=1.22
FROM golang:${golang_version}-alpine AS builder

# Set the working directory
WORKDIR /app

# Install git and ca-certificates for dependency management
RUN apk update && \
    apk upgrade && \
    apk add --no-cache git ca-certificates

# Accept build arguments
ARG app_version

# Copy go.mod and go.sum to leverage caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with the version injected
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${app_version}" -o vega-metrics-agent

# Runtime Stage
#FROM gcr.io/distroless/base:nonroot
FROM golang:${golang_version}-alpine

RUN apk update && \
    apk upgrade && \
    apk add --no-cache git ca-certificates curl bash

# Copy the binary from the builder stage
RUN mkdir -p /app
COPY --from=builder /app/vega-metrics-agent /app/vega-metrics-agent
WORKDIR /app
RUN chmod 777 /app
# Set the entrypoint
ENTRYPOINT ["/app/vega-metrics-agent"]

