#!/bin/bash

set -euo pipefail

# === Configuration ===
KIND_VERSION="v0.22.0"
K8S_IMAGE="kindest/node:v1.30.0"
CONFIG_FILE="kind-config.yaml"
RESOURCES_FILE="./test-resources.yaml"
CLUSTER_PREFIX="testcluster"
SLEEP_TIME=20

# === Generate Random Cluster Name ===
RAND_SUFFIX=$(head /dev/urandom | tr -dc a-z0-9 | head -c 8)
CLUSTER_NAME="${CLUSTER_PREFIX}-${RAND_SUFFIX}"
echo "$(date +%T) - Generated cluster name: $CLUSTER_NAME"

# === Download kind if not already present ===
if [[ ! -x "./kind" ]]; then
  echo "$(date +%T) - Downloading kind $KIND_VERSION..."
  curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
  chmod +x ./kind
fi

# === Create the Kind Cluster ===
echo "$(date +%T) - Creating Kind cluster: $CLUSTER_NAME"
./kind create cluster --name "$CLUSTER_NAME" --config "$CONFIG_FILE" --image="$K8S_IMAGE"

# === Wait for Cluster to Settle ===
echo "$(date +%T) - Sleeping $SLEEP_TIME seconds to allow the cluster to settle"
sleep "$SLEEP_TIME"

# === Set kubectl context (optional if only one kind cluster is running) ===
export KUBECONFIG="$(kind get kubeconfig-path --name="$CLUSTER_NAME" 2>/dev/null || echo "$HOME/.kube/config")"

# === Apply Test Resources ===
echo "$(date +%T) - Applying test resources"
kubectl apply -f "$RESOURCES_FILE"

# === Optional: Debug into a pod ===
#DEBUG_NAMESPACE="test-metrics"
#DEBUG_POD="test-pod"
#DEBUG_TARGET="main-contai"
#DEBUG_IMAGE="busybox"

#echo "$(date +%T) - Starting kubectl debug session (namespace: $DEBUG_NAMESPACE)"
#kubectl debug -n "$DEBUG_NAMESPACE" "$DEBUG_POD" --image="$DEBUG_IMAGE" -it --target="$DEBUG_TARGET"

# === Optional: Cleanup on exit ===
# trap 'echo "Cleaning up..."; ./kind delete cluster --name "$CLUSTER_NAME"' EXIT
