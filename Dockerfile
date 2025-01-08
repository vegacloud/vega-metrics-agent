ARG golang_version=1.23

# Use architecture-specific Alpine image
FROM --platform=linux/${TARGETARCH} golang:${golang_version}-alpine

# Define TARGETARCH within the build stage
ARG TARGETARCH

# Install necessary runtime dependencies
RUN apk update && \
    apk upgrade && \
    apk add --no-cache git ca-certificates curl bash

# Copy the pre-built binary for the target architecture
COPY bin/${TARGETARCH}/vega-metrics-agent /vega-metrics-agent

# Set the entrypoint
ENTRYPOINT ["/vega-metrics-agent"]
