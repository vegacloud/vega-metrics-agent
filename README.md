# Vega Kubernetes Metrics Agent

## Overview

The **Vega Kubernetes Metrics Agent** is a robust Go-based tool designed to gather, process, and upload comprehensive metrics from Kubernetes clusters. It collects a wide range of metrics across nodes, pods, persistent volumes, namespaces, workloads, networking components, and more. The agent supports both in-cluster and out-of-cluster deployment scenarios, providing flexibility for various operational environments.

## Architecture

The agent uses a collector-based architecture where:
- Each collector specializes in gathering specific metrics (nodes, pods, etc.)
- Metrics are collected in parallel with configurable concurrency
- Data is securely uploaded to S3 storage using pre-signed URLs
- Support for both metrics API and direct Kubelet collection with automatic fallback

## Key Features

- **Comprehensive Metrics Collection**:
  - **Node Metrics**: Detailed metrics for each node including capacity, allocatable resources, usage, and hardware details
  - **Pod Metrics**: Metrics for all pods covering resource requests, limits, usage, and container status
  - **Cluster Metrics**: Aggregated cluster-wide metrics with cloud provider detection
  - **Persistent Volume Metrics**: Metrics for persistent volumes, claims, and storage classes
  - **Namespace Metrics**: Resource quotas, limit ranges, and detailed usage
  - **Workload Metrics**: Deployments, stateful sets, daemon sets, jobs, and cron jobs
  - **Networking Metrics**: Services, ingresses, and network policies
  - **Orchestration Metrics**: HPAs, replication controllers, and replica sets

- **Multiple Collection Methods**:
  - **Metrics API Integration**: Primary collection through the `metrics.k8s.io` API
  - **Kubelet Direct Collection**: Fallback mechanism for detailed node metrics
  - **Automatic Failover**: Graceful degradation if primary collection method fails

- **Advanced Configuration**:
  - **API Rate Limiting**: Configurable QPS, Burst, and Timeout settings
  - **Concurrency Control**: Parallel collection with throttling
  - **Customizable Parameters**: Extensive configuration options through environment variables and flags

- **Operational Features**:
  - **Health Check**: Simple HTTP endpoint for liveness monitoring
  - **Secure Authentication**: Bearer token authentication for API access
  - **Cloud Provider Detection**: Automatic identification of AWS (EKS), Azure (AKS), GCP (GKE)
  - **Agent Check-in**: Optional capability to report agent status

## Installation

### Prerequisites

Before installing, ensure the following:

- A Kubernetes cluster
- Go 1.23+ (if building from source)
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

### Deploying with Helm Chart

The easiest way to deploy the Vega Metrics Agent is using the provided Helm chart.

#### Basic Installation

```sh
helm install vega-metrics-agent ./charts/vega-metrics-agent \
  --set vega.clientId=YOUR_CLIENT_ID \
  --set vega.clientSecret=YOUR_CLIENT_SECRET \
  --set vega.orgSlug=YOUR_ORG_SLUG \
  --set vega.clusterName=YOUR_CLUSTER_NAME
```

#### Customizing API Rate Limiting

To customize the API rate limiting settings, use the following parameters:

```sh
helm install vega-metrics-agent ./charts/vega-metrics-agent \
  --set vega.clientId=YOUR_CLIENT_ID \
  --set vega.clientSecret=YOUR_CLIENT_SECRET \
  --set vega.orgSlug=YOUR_ORG_SLUG \
  --set vega.clusterName=YOUR_CLUSTER_NAME \
  --set apiRateLimiting.qps=200 \
  --set apiRateLimiting.burst=300 \
  --set apiRateLimiting.timeout=15
```

#### Adjusting Concurrency

To control the maximum number of concurrent metric collection operations:

```sh
helm install vega-metrics-agent ./charts/vega-metrics-agent \
  --set maxConcurrency=12
```

#### Using a Custom Values File

Create a file named `custom-values.yaml` with your desired configuration:

```yaml
vega:
  clientId: "YOUR_CLIENT_ID"
  clientSecret: "YOUR_CLIENT_SECRET"
  orgSlug: "YOUR_ORG_SLUG"
  clusterName: "YOUR_CLUSTER_NAME"

apiRateLimiting:
  qps: 200
  burst: 300
  timeout: 15

maxConcurrency: 12

resources:
  requests:
    memory: "4Gi"
    cpu: "750m"
  limits:
    memory: "8Gi"
    cpu: "1500m"
```

Then install using:

```sh
helm install vega-metrics-agent ./charts/vega-metrics-agent -f custom-values.yaml
```

#### Helm Chart Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `apiRateLimiting.qps` | Kubernetes API requests per second | 100 |
| `apiRateLimiting.burst` | Maximum burst of API requests allowed | 100 |
| `apiRateLimiting.timeout` | API request timeout in seconds | 10 |
| `maxConcurrency` | Maximum concurrent collector operations | 8 |
| `resources.requests.memory` | Memory request for the agent | "2Gi" |
| `resources.requests.cpu` | CPU request for the agent | "500m" |
| `resources.limits.memory` | Memory limit for the agent | "4Gi" |
| `resources.limits.cpu` | CPU limit for the agent | "1000m" |
| `replicaCount` | Number of agent replicas to run | 1 |
| `image.repository` | Agent container image repository | public.ecr.aws/c0f8b9o4/vegacloud/vega-metrics-agent |
| `image.tag` | Agent container image tag | 1.1.3 |
| `image.pullPolicy` | Container image pull policy | Always |

**Note**: If you do not specify the API rate limiting or concurrency parameters, the agent will use its built-in defaults, which are optimized for most use cases.

### Configuration

The Vega Metrics Agent can be configured using either environment variables or command-line flags. Below are the key configuration parameters:

| Environment Variable         | Description                                 | Required  | Default Value |
|------------------------------|---------------------------------------------|-----------|---------------|
| VEGA_CLIENT_ID              | Client ID for authentication                | Yes       | |
| VEGA_CLIENT_SECRET          | Client secret for authentication            | Yes       | |
| VEGA_CLUSTER_NAME           | Name of the Kubernetes cluster              | Yes       | |
| VEGA_ORG_SLUG               | Your Vega Cloud Organization slug           | Yes       | |
| VEGA_POLL_INTERVAL          | Interval for polling metrics                | No        | 60m |
| VEGA_UPLOAD_REGION          | AWS region for S3 uploads                   | No        | us-west-2 |
| LOG_LEVEL                   | Log level (DEBUG, INFO, WARN, ERROR)        | No        | INFO |
| VEGA_INSECURE               | Use insecure connections                    | No        | false |
| VEGA_WORK_DIR               | Working directory for temporary files       | No        | /tmp |
| VEGA_COLLECTION_RETRY_LIMIT | Retry limit for metric collection           | No        | 3 |
| VEGA_BEARER_TOKEN_PATH      | Path to the bearer token file               | No        | /var/run/secrets/kubernetes.io/serviceaccount/token |
| VEGA_NAMESPACE              | Kubernetes namespace for agent deployment   | No        | vegacloud |
| VEGA_QPS                    | API rate limiter for requests per second    | No        | 100 |
| VEGA_BURST                  | API rate limiter burst allowance            | No        | 100 |
| VEGA_TIMEOUT                | Timeout for API requests                    | No        | 10s |
| VEGA_MAX_CONCURRENCY        | Maximum number of concurrent collectors     | No        | 8 |

Additional parameters for local testing and debugging include:

- AGENT_ID: Unique identifier for the agent
- SHOULD_AGENT_CHECK_IN: Determines if the agent should check in with the metrics server
- START_COLLECTION_NOW: Start metric collection immediately
- SAVE_LOCAL: Save metrics locally

#### API Rate Limiting Configuration

The agent provides the following parameters to control the rate of API requests to the Kubernetes API server:

- **QPS (Queries Per Second)**: Controls the sustainable rate of requests to the Kubernetes API. Default is 100 QPS.
- **Burst**: Sets the maximum burst of requests allowed beyond the QPS rate. Default is 100 requests.
- **Timeout**: Sets the timeout for individual API requests. Default is 10 seconds.

These settings can be adjusted based on your cluster size and API server capacity. For larger clusters or environments with high API server load, you may need to tune these values to prevent overwhelming the Kubernetes API server.

Example environment variable configuration:
```
VEGA_QPS=200
VEGA_BURST=300
VEGA_TIMEOUT=15s
```

Example command line configuration:
```
--qps=200 --burst=300 --timeout=15s
```

### Deploying in Kubernetes

The default namespace for the agent is `vegacloud`. When deployed, the agent gathers metrics from the Kubernetes API and directly from the cluster nodes. Metrics are uploaded to an Amazon S3 bucket at the configured interval.

Ensure that the agent has outbound access to:

- **api.vegacloud.io** (port 443) — for pre-signed URL retrieval and check-ins
- **vegametricsocean.s3.us-west-2.amazonaws.com** (port 443) — for uploading data

If your cluster is behind a firewall, add these addresses to your outbound allowlist.

### Supported Kubernetes Versions

The Vega Kubernetes Metrics Agent supports Kubernetes versions up to 1.30 across cloud platforms such as AWS (EKS), Google Cloud (GKE), Azure (AKS), and Oracle Cloud (OKE). The agent automatically detects the cloud provider and adapts its collection methods accordingly.

## Usage

### Running the Agent

To run the agent, execute:

```sh
vega-metrics-agent --help
```

This displays help for available flags, such as:

- `--client_id`: Client ID for authentication
- `--client_secret`: Client secret for authentication
- `--cluster_name`: The name of the Kubernetes cluster
- `--log_level`: Set log verbosity (DEBUG, INFO, WARN, ERROR)
- `--qps`: Set API request rate limit in queries per second
- `--burst`: Set API request burst allowance
- `--timeout`: Set API request timeout duration (e.g., 10s, 15s, 1m)
- `--max_concurrency`: Set maximum concurrent collector operations

### Health Check

You can check the agent's health by accessing the `/health` endpoint on port 80. This endpoint returns a simple "OK" response if the agent is running properly.

### Networking and System Requirements

#### Networking

Ensure the container running the agent allows outbound HTTPS requests to the following:

- api.vegacloud.io (port 443)
- vegametricsocean.s3.us-west-2.amazonaws.com (port 443)

#### Resource Recommendations

Based on the number of nodes in your cluster, here are guidelines for CPU and memory resources for the agent. Please note: Your mileage may vary depending on cluster size and configuration:

| Nodes   | CPU Request | CPU Limit | Mem Request | Mem Limit |
|---------|-------------|-----------|-------------|-----------|
| < 50    | 200m        | 500m      | 512Mi       | 1Gi       |
| 50-100  | 300m        | 700m      | 1Gi         | 2Gi       |
| 100-250 | 500m        | 1000m     | 2Gi         | 4Gi       |
| 250-500 | 750m        | 1500m     | 4Gi         | 8Gi       |
| 500-1000| 1000m       | 2000m     | 8Gi         | 12Gi      |
| 1000+   | 1500m       | 3000m     | 12Gi        | 16Gi      |

**Notes on resource allocation:**

- **CPU requests** should be set to allow guaranteed minimum CPU resources. The metrics agent doesn't need high CPU most of the time but benefits from having a consistent baseline.
- **CPU limits** should be set moderately higher than requests to allow for metric collection spikes. Setting CPU limits too low can cause throttling that may interrupt metrics collection.
- **Memory requests** should be set to accommodate the baseline memory footprint plus overhead for metrics processing.
- **Memory limits** should be set higher than requests to prevent OOM (Out of Memory) kills during peak collection periods.

These values should be adjusted based on your specific monitoring needs, collection frequency, and the total number of metrics being collected. For clusters with high pod density or custom metric collection, increase these values accordingly.

## Contributing

We welcome contributions! If you have any suggestions, improvements, or bug reports, feel free to open an issue or submit a pull request.

## License

This project is licensed under the Business Source License (BSL) 1.1. After the specified change date, it will be governed by the Apache License 2.0.

## Support
- Enterprise Customers: Support for the Vega Kubernetes Metrics Agent is available through the [Vega Cloud Support Portal](https://support.vegacloud.io/).
- Community and General Public support: Support is best effort. Please file an issue, bug report, or feature request using the [GitHub Issues](
    https://github.com/vegacloud/vega-metrics-agent/issues).

## Recent Changes

### Helm Chart Enhancements

The Helm chart now includes support for configuring the API rate limiting and concurrency settings. These settings help you optimize the agent's performance in various cluster environments:

- **API Rate Limiting**: Control the rate at which the agent makes API requests to the Kubernetes API server through the `apiRateLimiting.qps`, `apiRateLimiting.burst`, and `apiRateLimiting.timeout` parameters.
- **Concurrency Control**: Adjust the maximum number of concurrent collection operations through the `maxConcurrency` parameter.

If you do not specify these parameters, the agent will use its built-in defaults, which are optimized for most use cases.

### API Rate Limiting Enhancement

The Metrics Agent now includes configurable rate limiting for Kubernetes API requests through the QPS, Burst, and Timeout settings. These parameters allow you to fine-tune how the agent interacts with your Kubernetes API server:

- **QPS (Queries Per Second)**: Controls the sustained request rate to the Kubernetes API server.
- **Burst**: Allows temporary spikes in request rates while maintaining a sustainable average.
- **Timeout**: Sets the maximum duration for API requests before they time out.

For larger clusters or environments with high API server load, adjusting these values can help prevent the agent from overwhelming your Kubernetes API server.

### Metrics Collection Enhancement

The Metrics Agent now supports collecting CPU metrics via the `metrics.k8s.io` API. This change provides the following benefits:

1. **Improved Reliability**: The agent can now retrieve metrics directly from the metrics-server instead of requiring access to the kubelet API on each node.

2. **Better Security**: The agent no longer needs to access the kubelet API directly, reducing the security permissions required.

3. **Fallback Mechanism**: If metrics-server data is unavailable, the agent will automatically fall back to the traditional kubelet-based collection method.

## Requirements

- Kubernetes cluster 
- The latest kubernetes metrics-server installed (optional)
- Service account with appropriate RBAC permissions

## Troubleshooting

If you encounter issues with CPU metrics and you have installed the metrics-server:

1. Verify that the metrics-server is installed and running in your cluster:
   ```
   kubectl get pods -n kube-system | grep metrics-server
   ```

2. Check that the metrics API is available:
   ```
   kubectl get apiservices | grep metrics
   ```

3. Test direct access to the metrics API:
   ```
   kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes
   ```

4. Check the agent logs for any connection errors
