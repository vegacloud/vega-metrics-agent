# Vega Kubernetes Metrics Agent

## Overview

The **Vega Kubernetes Metrics Agent** is a robust tool designed to gather, process, and upload various Kubernetes cluster metrics. It tracks a wide range of metrics, including those related to nodes, pods, clusters, persistent volumes, namespaces, workloads, networking, and more. The agent is flexible, supporting both in-cluster and out-of-cluster deployment scenarios.

## Key Features

- **Node Metrics**: Detailed metrics for each node, including capacity, allocatable resources, and usage.
- **Pod Metrics**: Metrics for all pods, covering resource requests, limits, and usage.
- **Cluster Metrics**: Aggregated cluster-wide metrics like total capacity, allocatable resources, and usage.
- **Persistent Volume Metrics**: Metrics for persistent volumes and persistent volume claims.
- **Namespace Metrics**: Tracks namespace resource quotas, limit ranges, and usage.
- **Workload Metrics**: Metrics for deployments, stateful sets, daemon sets, and jobs.
- **Networking Metrics**: Gathers metrics related to services and ingresses.
- **Horizontal Pod Autoscaler (HPA)**: Monitors metrics for HPAs.
- **Replication Controller & ReplicaSet Metrics**: Tracks metrics for replication controllers and replica sets.
- **Health Check**: Provides a simple health check endpoint.

## Installation

### Prerequisites

Before installing, ensure the following:

- A Kubernetes cluster
- Go 1.16+ (if building from source)
- AWS credentials (for S3 uploads)

### Building from Source

1. Clone the repository:
    ```sh
    git clone https://github.com/vegacloud/vega-metrics-agent.git
    cd vega-metrics-agent
    ```

2. Build the project:
    ```sh
    make build
    ```

3. Run the agent:
    ```sh
    ./vega-metrics-agent --help
    ```

### Configuration

The Vega Metrics Agent can be configured using either environment variables or a configuration file. Below are the key configuration parameters:

| Environment Variable         | Description                                 | Required  |
|------------------------------|---------------------------------------------|-----------|
| VEGA_CLIENT_ID              | Client ID for authentication generated from Client                | Yes       |
| VEGA_CLIENT_SECRET          | Client secret for authentication.           | Yes       |
| VEGA_CLUSTER_NAME           | Name of the Kubernetes cluster.             | Yes       |
| VEGA_ORG_SLUG               | Your Vega Cloud Organization slug.                          | Yes       |
| VEGA_POLL_INTERVAL          | Interval for polling metrics.               | No        |
| VEGA_UPLOAD_REGION          | AWS region for S3 uploads.                  | No        |
| LOG_LEVEL                   | Log level (e.g., INFO, DEBUG).              | No        |
| VEGA_INSECURE               | Use insecure connections.                   | No        |
| VEGA_WORK_DIR               | Working directory for temporary files.      | No        |
| VEGA_COLLECTION_RETRY_LIMIT | Retry limit for metric collection.          | No        |
| VEGA_BEARER_TOKEN_PATH      | Path to the bearer token file.              | No        |
| VEGA_NAMESPACE              | Kubernetes namespace for agent deployment.  | No        |

Additional parameters for local testing and debugging include:

- AGENT_ID: Unique identifier for the agent.
- SHOULD_AGENT_CHECK_IN: Determines if the agent should check in with the metrics server.
- START_COLLECTION_NOW: Start metric collection immediately.
- SAVE_LOCAL: Save metrics locally.
- MAX_CONCURRENCY: Set the maximum concurrency for metric collection.

### Deploying in Kubernetes

The default namespace for the agent is \`vegacloud\`. When deployed, the agent gathers metrics from the Kubernetes API and directly from the cluster nodes. Metrics are uploaded every 10 minutes to an Amazon S3 bucket. 

Ensure that the agent has outbound access to:

- **api.vegacloud.io** (port 443) — for pre-signed URL retrieval.
- **vegametricsocean.s3.us-west-2.amazonaws.com** (port 443) — for uploading data.

If your cluster is behind a firewall, add these addresses to your outbound allowlist.

### Supported Kubernetes Versions

The Vega Kubernetes Metrics Agent supports Kubernetes versions up to 1.30 across cloud platforms such as AWS (EKS), Google Cloud (GKE), Azure (AKS), and Oracle Cloud (OKE).

### Deploying in Kubernetes using Helm Charts ###
- Install the kubectl command line tool
- Install the helm command line tool
- Modify the values.yaml file in ./deploy/charts/vega-metrics-agent/values.yaml
- In the charts/vega-metrics-agent directory, run `helm install vega-metrics ./delpoy/charts/vega-metrics-agent` (this sets up the service account, role, and rolebinding for the agent in your cluster)

You should now see a 'vegacloud' namespace with the agent running as a deployment. 

## Usage

### Running the Agent

To run the agent, execute:

```sh
vega-metrics-agent --help
```

This displays help for available flags, such as:

- \`--client_id\`: Client ID for authentication.
- \`--client_secret\`: Client secret for authentication.
- \`--cluster_name\`: The name of the Kubernetes cluster.
- \`--log_level\`: Set log verbosity (DEBUG, INFO, WARN, ERROR).

### Health Check

You can check the agent’s health by accessing the ```/health``` endpoint:

### Networking and System Requirements

#### Networking

Ensure the container running the agent allows outbound HTTPS requests to the following:

- api.vegacloud.io (port 443)
- vegametricsocean.s3.us-west-2.amazonaws.com (port 443)

#### Resource Recommendations

Based on the number of nodes in your cluster, here are the suggested CPU and memory requirements for the agent:

| Nodes  | CPU Request | CPU Limit | Mem Request | Mem Limit |
|--------|-------------|-----------|-------------|-----------|
| < 100  | 500m        | 1000m     | 2Gi         | 4Gi       |
| 100-200| 1000m       | 1500m     | 4Gi         | 8Gi       |
| 200-500| 1500m       | 2000m     | 8Gi         | 16Gi      |
| 500-1000| 2000m      | 3000m     | 16Gi        | 24Gi      |
| 1000+  | 3000m       | -         | 24Gi        | -         |

These values can be adjusted according to your actual workload.

## Contributing

We welcome contributions! If you have any suggestions, improvements, or bug reports, feel free to open an issue or submit a pull request.

## License

This project is licensed under the Business Source License (BSL) 1.1. After the specified change date, it will be governed by the Apache License 2.0.

## Contact

For any questions or support, reach out to us at [support@vegacloud.io](mailto:support@vegacloud.io).
