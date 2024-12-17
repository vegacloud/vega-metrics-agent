// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"sync"

	// "crypto/tls"
	"encoding/json"
	// "errors"
	"fmt"
	"math"

	// "os"
	"strconv"
	"strings"

	// "sync"
	// "time"

	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// NodeCollector collects metrics from Kubernetes nodes.
type NodeCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
	// bearerToken string
	//httpClient *http.Client
}

// NewNodeCollector initializes a new NodeCollector.
func NewNodeCollector(
	clientset *kubernetes.Clientset,
	cfg *config.Config,
) (*NodeCollector, error) {
	nc := &NodeCollector{
		clientset: clientset,
		config:    cfg,
	}

	// // Get bearer token
	// token, err := nc.getBearerToken()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get bearer token: %w", err)
	// }
	// nc.bearerToken = token
	// logrus.Debug("Successfully retrieved bearer token")

	// // Create HTTP client
	// transport := &http.Transport{
	// 	TLSClientConfig: &tls.Config{
	// 		InsecureSkipVerify: cfg.VegaInsecure, //#nosec this is only off for local testing and will be true in prod.
	// 	},
	// }

	// nc.httpClient = &http.Client{
	// 	Timeout:   10 * time.Second,
	// 	Transport: transport,
	// }
	logrus.Debug("HTTP client created successfully")

	return nc, nil
}

// CollectMetrics collects metrics for all nodes
func (nc *NodeCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	nodes, err := nc.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var metrics []models.EnhancedNodeMetrics
	for _, node := range nodes.Items {
		nodeMetrics, err := nc.CollectNodeMetrics(ctx, &node)
		if err != nil {
			continue
		}
		metrics = append(metrics, *nodeMetrics)
	}
	return metrics, nil
}

// Update collectCPUMetrics to use the new method
func (nc *NodeCollector) collectCPUMetrics(ctx context.Context, node *v1.Node) (models.CPUMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics/cadvisor")
	if err != nil {
		return models.CPUMetrics{}, err
	}

	cpuMetrics := models.CPUMetrics{
		PerCoreUsage: make(map[string]uint64),
		Throttling:   models.ThrottlingMetrics{},
	}

	// Total CPU usage (equivalent to metrics-server CPU usage)
	if cpuUsage, ok := metricFamilies["container_cpu_usage_seconds_total"]; ok {
		for _, metric := range cpuUsage.Metric {
			if metric.Counter != nil {
				value := metric.Counter.GetValue() * 1e9
				if value >= 0 {
					cpuMetrics.UsageTotal = uint64(value)
				}
			}
		}
	}

	// Additional detailed CPU metrics
	if cpuUser, ok := metricFamilies["container_cpu_user_seconds_total"]; ok {
		for _, metric := range cpuUser.Metric {
			if metric.Counter != nil {
				value := metric.Counter.GetValue() * 1e9
				if value >= 0 {
					cpuMetrics.UserTime = uint64(value)
				}
			}
		}
	}

	if cpuSystem, ok := metricFamilies["container_cpu_system_seconds_total"]; ok {
		for _, metric := range cpuSystem.Metric {
			if metric.Counter != nil {
				value := metric.Counter.GetValue() * 1e9
				if value >= 0 {
					cpuMetrics.SystemTime = uint64(value)
				}
			}
		}
	}

	return cpuMetrics, nil
}

// collectRuntimeMetrics collects runtime metrics
func (nc *NodeCollector) collectRuntimeMetrics(ctx context.Context, node *v1.Node) (models.RuntimeMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics/probes")
	if err != nil {
		return models.RuntimeMetrics{}, err
	}

	runtimeMetrics := models.RuntimeMetrics{
		Operations: make(map[string]models.RuntimeOperation),
	}

	// Parse runtime operations
	if operations, ok := metricFamilies["runtime_operations_total"]; ok {
		for _, metric := range operations.Metric {
			if metric.Counter != nil {
				opType := getLabel(metric, "operation_type")
				if opType != "" {
					runtimeMetrics.Operations[opType] = models.RuntimeOperation{
						Count: uint64(metric.Counter.GetValue()),
						Type:  opType,
					}
				}
			}
		}
	}

	// Parse runtime errors
	if errors, ok := metricFamilies["runtime_operations_errors_total"]; ok {
		for _, metric := range errors.Metric {
			if metric.Counter != nil {
				opType := getLabel(metric, "operation_type")
				if op, exists := runtimeMetrics.Operations[opType]; exists {
					op.Errors = uint64(metric.Counter.GetValue())
					runtimeMetrics.Operations[opType] = op
				}
			}
		}
	}

	return runtimeMetrics, nil
}

// Update collectMemoryMetrics
func (nc *NodeCollector) collectMemoryMetrics(ctx context.Context, node *v1.Node) (models.MemoryMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics/cadvisor")
	if err != nil {
		return models.MemoryMetrics{}, err
	}

	memoryMetrics := models.MemoryMetrics{}

	// Total memory usage (equivalent to metrics-server memory usage)
	if memUsage, ok := metricFamilies["container_memory_usage_bytes"]; ok {
		for _, metric := range memUsage.Metric {
			if metric.Gauge != nil {
				memoryMetrics.Used = uint64(metric.Gauge.GetValue())
			}
		}
	}

	// Additional detailed memory metrics
	if memWorkingSet, ok := metricFamilies["container_memory_working_set_bytes"]; ok {
		for _, metric := range memWorkingSet.Metric {
			if metric.Gauge != nil {
				memoryMetrics.WorkingSet = uint64(metric.Gauge.GetValue())
			}
		}
	}

	if memRSS, ok := metricFamilies["container_memory_rss"]; ok {
		for _, metric := range memRSS.Metric {
			if metric.Gauge != nil {
				memoryMetrics.RSS = uint64(metric.Gauge.GetValue())
			}
		}
	}

	return memoryMetrics, nil
}

// // getBearerToken retrieves the token from a file or environment variable.
// func (nc *NodeCollector) getBearerToken() (string, error) {
// 	// First, try to read from file
// 	if nc.config.VegaBearerTokenPath != "" {
// 		tokenBytes, err := os.ReadFile(nc.config.VegaBearerTokenPath)
// 		if err == nil {
// 			token := strings.TrimSpace(string(tokenBytes))
// 			logrus.Debug("Successfully read Service Account bearer token from file")
// 			return token, nil
// 		}
// 		logrus.Printf("Failed to read bearer token from file, defaulting to BEARER_TOKEN environment variable: %v", err)
// 	}

// 	// If file read failed or no file path was provided, try environment variable
// 	token := strings.TrimSpace(os.Getenv("BEARER_TOKEN"))
// 	if token != "" {
// 		logrus.Debug("Successfully read bearer token from environment variable")
// 		return token, nil
// 	}

// 	// If both file and environment variable failed, return an error
// 	return "", errors.New("bearer token not found in file or environment")
// }

func (nc *NodeCollector) collectNetworkMetrics(ctx context.Context, node *v1.Node) (models.NetworkMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics/cadvisor")
	if err != nil {
		return models.NetworkMetrics{}, fmt.Errorf("failed to fetch network metrics: %w", err)
	}

	networkMetrics := models.NetworkMetrics{
		Interfaces: make([]models.InterfaceStats, 0),
		Summary:    models.NetworkSummary{},
	}

	interfaceMap := make(map[string]*models.InterfaceStats)

	// Process node network metrics for detailed interface stats
	for name, family := range metricFamilies {
		if !strings.HasPrefix(name, "node_network_") {
			continue
		}

		for _, metric := range family.Metric {
			interfaceName := getLabel(metric, "device")
			if interfaceName == "" {
				continue
			}

			if _, exists := interfaceMap[interfaceName]; !exists {
				interfaceMap[interfaceName] = &models.InterfaceStats{
					InterfaceName: interfaceName,
				}
			}

			if metric.Counter != nil {
				value := uint64(metric.Counter.GetValue())
				switch name {
				case "node_network_receive_bytes_total":
					interfaceMap[interfaceName].RxBytes = value
					networkMetrics.Summary.RxBytesTotal += value
				case "node_network_transmit_bytes_total":
					interfaceMap[interfaceName].TxBytes = value
					networkMetrics.Summary.TxBytesTotal += value
				case "node_network_receive_packets_total":
					interfaceMap[interfaceName].RxPackets = value
				case "node_network_transmit_packets_total":
					interfaceMap[interfaceName].TxPackets = value
				case "node_network_receive_errs_total":
					interfaceMap[interfaceName].RxErrors = value
				case "node_network_transmit_errs_total":
					interfaceMap[interfaceName].TxErrors = value
				}
			}
		}
	}

	// Convert interface map to slice
	for _, v := range interfaceMap {
		networkMetrics.Interfaces = append(networkMetrics.Interfaces, *v)
	}

	// Also collect container network metrics if they provide additional information
	if rxBytes, ok := metricFamilies["container_network_receive_bytes_total"]; ok {
		for _, metric := range rxBytes.Metric {
			if metric.Counter != nil {
				networkMetrics.Summary.ContainerRxBytesTotal += uint64(metric.Counter.GetValue())
			}
		}
	}

	if txBytes, ok := metricFamilies["container_network_transmit_bytes_total"]; ok {
		for _, metric := range txBytes.Metric {
			if metric.Counter != nil {
				networkMetrics.Summary.ContainerTxBytesTotal += uint64(metric.Counter.GetValue())
			}
		}
	}

	return networkMetrics, nil
}

func (nc *NodeCollector) collectDiskMetrics(ctx context.Context, node *v1.Node) (models.DiskMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics/cadvisor")
	if err != nil {
		return models.DiskMetrics{}, fmt.Errorf("failed to fetch disk metrics: %w", err)
	}

	diskMetrics := models.DiskMetrics{
		Devices: make([]models.NodeDiskStats, 0),
	}

	deviceMap := make(map[string]*models.NodeDiskStats)

	// Process per-device metrics
	for name, family := range metricFamilies {
		if !strings.HasPrefix(name, "node_disk_") {
			continue
		}

		for _, metric := range family.Metric {
			device := getLabel(metric, "device")
			if device == "" {
				continue
			}

			if _, exists := deviceMap[device]; !exists {
				deviceMap[device] = &models.NodeDiskStats{
					Device: device,
				}
			}

			if metric.Counter != nil {
				value := uint64(metric.Counter.GetValue())
				switch name {
				case "node_disk_read_bytes_total":
					deviceMap[device].ReadBytes = value
					diskMetrics.ReadBytes += safeInt64(value)
				case "node_disk_written_bytes_total":
					deviceMap[device].WriteBytes = value
					diskMetrics.WriteBytes += safeInt64(value)
				case "node_disk_reads_completed_total":
					deviceMap[device].ReadOps = value
				case "node_disk_writes_completed_total":
					deviceMap[device].WriteOps = value
				}
			}
		}
	}

	// Convert device map to slice
	for _, device := range deviceMap {
		diskMetrics.Devices = append(diskMetrics.Devices, *device)
	}

	return diskMetrics, nil
}

func (nc *NodeCollector) collectProcessMetrics(ctx context.Context, node *v1.Node) (models.ProcessMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics")
	if err != nil {
		return models.ProcessMetrics{}, fmt.Errorf("failed to fetch process metrics: %w", err)
	}

	var processMetrics models.ProcessMetrics

	if procCount, ok := metricFamilies["process_cpu_seconds_total"]; ok {
		processMetrics.ProcessCount = len(procCount.Metric)
	}

	return processMetrics, nil
}

// parseResourceList converts Kubernetes resource list to ResourceMetrics
func (nc *NodeCollector) parseResourceList(rl v1.ResourceList) models.ResourceMetrics {
	metrics := models.ResourceMetrics{}

	// CPU is reported in cores, convert to millicores
	if cpu, ok := rl[v1.ResourceCPU]; ok {
		metrics.CPU = cpu.MilliValue()
	}

	// Memory is reported in bytes
	if memory, ok := rl[v1.ResourceMemory]; ok {
		metrics.Memory = memory.Value()
	}

	// Persistent Storage
	if storage, ok := rl[v1.ResourceStorage]; ok {
		metrics.Storage = storage.Value()
	}

	// Ephemeral storage should be tracked separately from persistent storage
	if ephemeralStorage, ok := rl[v1.ResourceEphemeralStorage]; ok {
		metrics.EphemeralStorage = ephemeralStorage.Value()
	}

	// Add pods resource if available
	if pods, ok := rl[v1.ResourcePods]; ok {
		metrics.Pods = pods.Value()
	}

	return metrics
}

func (nc *NodeCollector) collectFilesystemMetrics(ctx context.Context, node *v1.Node) (models.FSMetrics, error) {
	rawStats, err := FetchRawStatsViaKubelet(ctx, nc.clientset, node.Name, "stats/summary")
	if err != nil {
		return models.FSMetrics{}, fmt.Errorf("failed to fetch filesystem stats: %w", err)
	}

	var stats struct {
		Node struct {
			Fs struct {
				CapacityBytes  uint64 `json:"capacityBytes"`
				UsedBytes      uint64 `json:"usedBytes"`
				AvailableBytes uint64 `json:"availableBytes"`
				InodesFree     uint64 `json:"inodesFree"`
				InodesUsed     uint64 `json:"inodesUsed"`
			} `json:"fs"`
		} `json:"node"`
	}

	if err := json.Unmarshal(rawStats, &stats); err != nil {
		return models.FSMetrics{}, fmt.Errorf("failed to unmarshal filesystem stats: %w", err)
	}

	return models.FSMetrics{
		TotalBytes:     stats.Node.Fs.CapacityBytes,
		UsedBytes:      stats.Node.Fs.UsedBytes,
		AvailableBytes: stats.Node.Fs.AvailableBytes,
		Inodes: models.InodeStats{
			Free:  stats.Node.Fs.InodesFree,
			Used:  stats.Node.Fs.InodesUsed,
			Total: stats.Node.Fs.InodesFree + stats.Node.Fs.InodesUsed,
		},
	}, nil
}

func (nc *NodeCollector) collectContainerStats(ctx context.Context, node *v1.Node) (models.ContainerStats, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics/cadvisor")
	if err != nil {
		return models.ContainerStats{}, fmt.Errorf("failed to fetch container metrics: %w", err)
	}

	stats := models.ContainerStats{
		PerContainer: make(map[string]models.ContainerMetrics),
	}

	// Basic container count from running containers
	if count, ok := metricFamilies["container_last_seen"]; ok {
		stats.RunningCount = len(count.Metric)
	}

	return stats, nil
}

func (nc *NodeCollector) collectSystemMetrics(ctx context.Context, node *v1.Node) (models.SystemMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics")
	if err != nil {
		return models.SystemMetrics{}, fmt.Errorf("failed to fetch system metrics: %w", err)
	}

	var systemMetrics models.SystemMetrics

	// Parse system metrics
	if nodeUptimeMetric, ok := metricFamilies["node_boot_time_seconds"]; ok {
		for _, metric := range nodeUptimeMetric.Metric {
			if metric.Gauge != nil {
				systemMetrics.BootTimeSeconds = float64(metric.Gauge.GetValue())
			}
		}
	}

	// Add additional system information
	cpuValue := node.Status.Capacity.Cpu().Value()
	if cpuValue >= 0 {
		systemMetrics.NumCPUs = cpuValue
	}
	systemMetrics.KernelVersion = node.Status.NodeInfo.KernelVersion
	systemMetrics.OSVersion = node.Status.NodeInfo.OSImage
	systemMetrics.MachineID = node.Status.NodeInfo.MachineID
	systemMetrics.SystemUUID = node.Status.NodeInfo.SystemUUID

	return systemMetrics, nil
}

// safeInt64 converts uint64 to int64 safely
func safeInt64(val uint64) int64 {
	if val > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(val)
}

// // safeInt32 converts uint64 to int32 safely
// func safeInt32(val uint64) int32 {
// 	if val > math.MaxInt32 {
// 		return math.MaxInt32
// 	}
// 	return int32(val)
// }

// // safeUint64 converts int64 to uint64 safely
// func safeUint64(val int64) uint64 {
// 	if val < 0 {
// 		return 0
// 	}
// 	return uint64(val)
// }

func (nc *NodeCollector) parseLatencyMetric(family *dto.MetricFamily) models.LatencyMetric {
	latencyMetric := models.LatencyMetric{}

	for _, metric := range family.Metric {
		if metric.Summary != nil {
			for _, quantile := range metric.Summary.Quantile {
				switch *quantile.Quantile {
				case 0.50:
					latencyMetric.P50 = *quantile.Value
				case 0.90:
					latencyMetric.P90 = *quantile.Value
				case 0.99:
					latencyMetric.P99 = *quantile.Value
				}
			}
			if metric.Summary.SampleCount != nil {
				count := *metric.Summary.SampleCount
				latencyMetric.Count = safeInt64(count)
			}
		}
	}

	return latencyMetric
}

func (nc *NodeCollector) collectVolumeMetrics(ctx context.Context, node *v1.Node) (models.VolumeMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics")
	if err != nil {
		return models.VolumeMetrics{}, fmt.Errorf("failed to fetch volume metrics: %w", err)
	}

	volumeMetrics := models.VolumeMetrics{
		OperationLatency: make(map[string]models.LatencyMetric),
	}

	// Parse volume metrics
	if attachCount, ok := metricFamilies["volume_manager_total_volumes"]; ok {
		for _, metric := range attachCount.Metric {
			if metric.Gauge != nil {
				volumeMetrics.InUseCount = uint64(metric.Gauge.GetValue())
			}
		}
	}

	// Parse operation latency metrics if available
	if latencyMetrics, ok := metricFamilies["storage_operation_duration_seconds"]; ok {
		for _, metric := range latencyMetrics.Metric {
			if metric.Summary != nil {
				opType := getLabel(metric, "operation_name")
				if opType != "" {
					volumeMetrics.OperationLatency[opType] = nc.parseLatencyMetric(latencyMetrics)
				}
			}
		}
	}

	return volumeMetrics, nil
}

func (nc *NodeCollector) collectNodeInfo(node *v1.Node) models.NodeInfo {
	return models.NodeInfo{
		Architecture:            node.Status.NodeInfo.Architecture,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		OSImage:                 node.Status.NodeInfo.OSImage,
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
	}
}

func (nc *NodeCollector) collectHardwareTopology(ctx context.Context, node *v1.Node) (*models.HardwareTopology, error) {
	cpuValue := node.Status.Capacity.Cpu().Value()

	// Since cpuValue is already an int64, this check is unnecessary
	// You can remove the check entirely or if you want to be extra careful:
	if cpuValue < 0 {
		return nil, fmt.Errorf("invalid negative CPU value: %d", cpuValue)
	}

	topology := &models.HardwareTopology{
		Sockets: cpuValue,
		Cores:   cpuValue,
	}

	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics/cadvisor")
	if err != nil {
		return topology, nil // Return basic topology even if detailed info fails
	}

	// Parse NUMA information if available
	if numaInfo, ok := metricFamilies["node_numa_memory_bytes"]; ok {
		for _, metric := range numaInfo.Metric {
			if nodeIDStr := getLabel(metric, "numa_node"); nodeIDStr != "" {
				nodeID, err := strconv.ParseInt(nodeIDStr, 10, 32)
				if err != nil {
					continue
				}
				topology.NUMANodes = append(topology.NUMANodes, models.NUMANode{
					ID:     int32(nodeID),
					Memory: uint64(metric.Gauge.GetValue()),
				})
			}
		}
	}

	return topology, nil
}

func (nc *NodeCollector) collectPowerMetrics(ctx context.Context, node *v1.Node) (*models.PowerMetrics, error) {
	metricFamilies, err := FetchMetricsViaKubelet(ctx, nc.clientset, node.Name, "metrics/cadvisor")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch power metrics: %w", err)
	}

	powerMetrics := &models.PowerMetrics{}

	if power, ok := metricFamilies["node_power_watts"]; ok {
		for _, metric := range power.Metric {
			if metric.Gauge != nil {
				powerMetrics.CurrentWatts = metric.Gauge.GetValue()
				break
			}
		}
	}

	return powerMetrics, nil
}

func (nc *NodeCollector) collectNodeTaintsAndTolerations(node v1.Node) []models.NodeTaint {
	taints := make([]models.NodeTaint, 0, len(node.Spec.Taints))
	for _, taint := range node.Spec.Taints {
		nodeTaint := models.NodeTaint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		}
		if taint.TimeAdded != nil {
			timeAdded := taint.TimeAdded.Time
			nodeTaint.TimeAdded = &timeAdded
		}
		taints = append(taints, nodeTaint)
	}
	return taints
}

func (nc *NodeCollector) collectNodeLease(ctx context.Context, nodeName string) (*models.NodeLease, error) {
	lease, err := nc.clientset.CoordinationV1().Leases("kube-node-lease").Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &models.NodeLease{
		HolderIdentity:       *lease.Spec.HolderIdentity,
		LeaseDurationSeconds: *lease.Spec.LeaseDurationSeconds,
		AcquireTime:          &lease.Spec.AcquireTime.Time,
		RenewTime:            &lease.Spec.RenewTime.Time,
	}, nil
}

func (nc *NodeCollector) collectExtendedResources(node v1.Node) map[string]models.ExtendedResource {
	resources := make(map[string]models.ExtendedResource)

	for resourceName, quantity := range node.Status.Capacity {
		if !isStandardResource(resourceName) {
			capUnstructured := quantity.ToUnstructured()
			allocUnstructured := node.Status.Allocatable[resourceName].ToUnstructured()

			resources[string(resourceName)] = models.ExtendedResource{
				Name:        string(resourceName),
				Capacity:    fmt.Sprintf("%v", capUnstructured),
				Allocatable: fmt.Sprintf("%v", allocUnstructured),
			}
		}
	}

	return resources
}

// Helper function to identify standard resources
func isStandardResource(resourceName v1.ResourceName) bool {
	standardResources := []v1.ResourceName{
		v1.ResourceCPU,
		v1.ResourceMemory,
		v1.ResourceStorage,
		v1.ResourceEphemeralStorage,
		v1.ResourcePods,
	}

	for _, std := range standardResources {
		if resourceName == std {
			return true
		}
	}
	return false
}

func (nc *NodeCollector) collectVolumeHealthMetrics(ctx context.Context, node *v1.Node) ([]models.VolumeHealthMetrics, error) {
	pods, err := nc.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
	})
	if err != nil {
		return nil, err
	}

	var volumeHealthMetrics []models.VolumeHealthMetrics

	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				pvc, err := nc.clientset.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(ctx, volume.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
				if err != nil {
					continue
				}

				metric := models.VolumeHealthMetrics{
					VolumeName: volume.Name,
					PodName:    pod.Name,
					Namespace:  pod.Namespace,
					State:      string(pvc.Status.Phase),
				}

				volumeHealthMetrics = append(volumeHealthMetrics, metric)
			}
		}
	}

	return volumeHealthMetrics, nil
}

func (nc *NodeCollector) collectVolumeAttachmentMetrics(ctx context.Context, nodeName string) ([]models.VolumeAttachmentMetrics, error) {
	attachments, err := nc.clientset.StorageV1().VolumeAttachments().List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return nil, err
	}

	var metrics []models.VolumeAttachmentMetrics

	for _, attachment := range attachments.Items {
		metric := models.VolumeAttachmentMetrics{
			VolumeName: func() string {
				if attachment.Spec.Source.PersistentVolumeName != nil {
					return *attachment.Spec.Source.PersistentVolumeName
				}
				return ""
			}(),
			AttachmentState: string(attachment.Status.AttachmentMetadata["attached"]),
			AttachTime:      attachment.CreationTimestamp.Time,
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// CollectNodeMetrics collects all metrics for a node
func (nc *NodeCollector) CollectNodeMetrics(ctx context.Context, node *v1.Node) (*models.EnhancedNodeMetrics, error) {
	metrics := &models.EnhancedNodeMetrics{
		Name:        node.Name,
		Conditions:  nc.getNodeConditions(node),
		Labels:      node.Labels,
		Annotations: node.Annotations,
		NodeInfo:    nc.collectNodeInfo(node),
	}

	// Collect resource metrics
	metrics.Capacity = nc.parseResourceList(node.Status.Capacity)
	metrics.Allocatable = nc.parseResourceList(node.Status.Allocatable)

	// Collect all detailed metrics in parallel using goroutines
	var wg sync.WaitGroup
	var errChan = make(chan error, 15) // Buffer for potential errors
	var mu sync.Mutex

	detailedMetrics := models.NodeDetailedMetrics{}

	// Helper function to safely update metrics
	safeUpdate := func(f func()) {
		mu.Lock()
		defer mu.Unlock()
		f()
	}

	// CPU Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if cpuMetrics, err := nc.collectCPUMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.CPU = cpuMetrics })
		} else {
			errChan <- fmt.Errorf("CPU metrics: %w", err)
		}
	}()

	// Memory Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if memMetrics, err := nc.collectMemoryMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.Memory = memMetrics })
		} else {
			errChan <- fmt.Errorf("memory metrics: %w", err)
		}
	}()

	// Network Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if netMetrics, err := nc.collectNetworkMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.Network = netMetrics })
		} else {
			errChan <- fmt.Errorf("network metrics: %w", err)
		}
	}()

	// Disk Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if diskMetrics, err := nc.collectDiskMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.DiskIO = diskMetrics })
		} else {
			errChan <- fmt.Errorf("disk metrics: %w", err)
		}
	}()

	// Filesystem Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if fsMetrics, err := nc.collectFilesystemMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.Filesystem = fsMetrics })
		} else {
			errChan <- fmt.Errorf("filesystem metrics: %w", err)
		}
	}()

	// Process Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if procMetrics, err := nc.collectProcessMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.Process = procMetrics })
		} else {
			errChan <- fmt.Errorf("process metrics: %w", err)
		}
	}()

	// Runtime Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if runtimeMetrics, err := nc.collectRuntimeMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.Runtime = runtimeMetrics })
		} else {
			errChan <- fmt.Errorf("runtime metrics: %w", err)
		}
	}()

	// System Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if sysMetrics, err := nc.collectSystemMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.System = sysMetrics })
		} else {
			errChan <- fmt.Errorf("system metrics: %w", err)
		}
	}()

	// Volume Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if volMetrics, err := nc.collectVolumeMetrics(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.Volume = volMetrics })
		} else {
			errChan <- fmt.Errorf("volume metrics: %w", err)
		}
	}()

	// Container Stats
	wg.Add(1)
	go func() {
		defer wg.Done()
		if containerStats, err := nc.collectContainerStats(ctx, node); err == nil {
			safeUpdate(func() { detailedMetrics.Container = containerStats })
		} else {
			errChan <- fmt.Errorf("container stats: %w", err)
		}
	}()

	// Hardware Topology
	wg.Add(1)
	go func() {
		defer wg.Done()
		if topology, err := nc.collectHardwareTopology(ctx, node); err == nil {
			safeUpdate(func() { metrics.HardwareTopology = topology })
		} else {
			errChan <- fmt.Errorf("hardware topology: %w", err)
		}
	}()

	// Power Metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if powerMetrics, err := nc.collectPowerMetrics(ctx, node); err == nil {
			safeUpdate(func() { metrics.PowerMetrics = powerMetrics })
		} else {
			errChan <- fmt.Errorf("power metrics: %w", err)
		}
	}()

	// Volume Health
	wg.Add(1)
	go func() {
		defer wg.Done()
		if volumeHealth, err := nc.collectVolumeHealthMetrics(ctx, node); err == nil {
			safeUpdate(func() { metrics.VolumeHealth = volumeHealth })
		} else {
			errChan <- fmt.Errorf("volume health: %w", err)
		}
	}()

	// Volume Attachments
	wg.Add(1)
	go func() {
		defer wg.Done()
		if attachments, err := nc.collectVolumeAttachmentMetrics(ctx, node.Name); err == nil {
			safeUpdate(func() { metrics.VolumeAttachments = attachments })
		} else {
			errChan <- fmt.Errorf("volume attachments: %w", err)
		}
	}()

	// Node Lease
	wg.Add(1)
	go func() {
		defer wg.Done()
		if lease, err := nc.collectNodeLease(ctx, node.Name); err == nil {
			safeUpdate(func() { metrics.Lease = lease })
		} else {
			errChan <- fmt.Errorf("node lease: %w", err)
		}
	}()

	// Wait for all collectors to complete
	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	// Add non-parallel collections
	metrics.Taints = nc.collectNodeTaintsAndTolerations(*node)
	metrics.ExtendedResources = nc.collectExtendedResources(*node)
	metrics.DetailedMetrics = detailedMetrics

	// If there were any errors, log them but don't fail the entire collection
	if len(errors) > 0 {
		logrus.WithField("errors", strings.Join(errors, "; ")).
			Warn("Some metrics collections failed")
	}

	return metrics, nil
}

func (nc *NodeCollector) getNodeConditions(node *v1.Node) models.NodeConditions {
	conditions := models.NodeConditions{}
	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case v1.NodeReady:
			conditions.Ready = condition.Status == v1.ConditionTrue
		case v1.NodeMemoryPressure:
			conditions.MemoryPressure = condition.Status == v1.ConditionTrue
		case v1.NodeDiskPressure:
			conditions.DiskPressure = condition.Status == v1.ConditionTrue
		case v1.NodePIDPressure:
			conditions.PIDPressure = condition.Status == v1.ConditionTrue
		}
	}
	return conditions
}

// Helper function to get label value
func getLabel(metric *dto.Metric, name string) string {
	for _, label := range metric.Label {
		if label.GetName() == name {
			return label.GetValue()
		}
	}
	return ""
}
