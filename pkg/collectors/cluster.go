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
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
)

// ClusterCollector collects cluster-wide metrics
type ClusterCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewClusterCollector creates a new cluster collector
func NewClusterCollector(clientset *kubernetes.Clientset, cfg *config.Config) *ClusterCollector {
	// Ensure we're using the in-cluster config token
	if token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.VegaInsecure,
			},
		}
		clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport = transport.NewBearerAuthRoundTripper(
			string(token),
			clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport,
		)
	}

	return &ClusterCollector{
		clientset: clientset,
		config:    cfg,
	}
}

// CollectMetrics collects cluster-wide metrics
func (cc *ClusterCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	logger := logrus.WithField("collector", "ClusterCollector")

	// Add debug logging for the client configuration
	if token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		logger.Debugf("Service account token length: %d", len(string(token)))
	}

	// Verify client identity before collecting metrics
	if err := VerifyCollectorClient(ctx, cc.clientset, cc.config.VegaNamespace, "ClusterCollector"); err != nil {
		return nil, err
	}
	version, err := cc.clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}
	clusterVersion := version.String()
	nodes, err := cc.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	logger.Debugf("Successfully listed %d nodes", len(nodes.Items))

	pods, err := cc.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	logger.Debugf("Successfully listed %d pods", len(pods.Items))

	metrics := &models.ClusterMetrics{
		KubernetesVersion: clusterVersion,
		NodeCount:         len(nodes.Items),
		PodCount:          len(pods.Items),
		ContainerCount:    cc.countContainers(pods.Items),
		NodeLabels:        make(map[string]map[string]string),
		PodLabels:         make(map[string]map[string]string),
	}

	metrics.TotalCapacity = cc.calculateTotalCapacity(nodes.Items)
	metrics.TotalAllocatable = cc.calculateTotalAllocatable(nodes.Items)
	metrics.TotalRequests = cc.calculateTotalRequests(pods.Items)
	metrics.TotalLimits = cc.calculateTotalLimits(pods.Items)

	// Collect labels for nodes
	for _, node := range nodes.Items {
		metrics.NodeLabels[node.Name] = node.Labels
	}

	// Collect labels for pods
	for _, pod := range pods.Items {
		podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		metrics.PodLabels[podKey] = pod.Labels
	}

	return metrics, nil
}

func (cc *ClusterCollector) calculateTotalCapacity(nodes []v1.Node) models.ResourceMetrics {
	total := models.ResourceMetrics{}
	for _, node := range nodes {
		total.CPU += node.Status.Capacity.Cpu().MilliValue()
		total.Memory += node.Status.Capacity.Memory().Value()
		total.Storage += node.Status.Allocatable.Storage().Value()
		if ephemeral, ok := node.Status.Allocatable[v1.ResourceEphemeralStorage]; ok {
			total.EphemeralStorage += ephemeral.Value()
		}
		total.Pods += node.Status.Allocatable.Pods().Value()

		// Collect GPU metrics if available
		if gpuQuantity, ok := node.Status.Capacity["nvidia.com/gpu"]; ok {
			gpuMetric := models.GPUMetrics{
				DeviceID:    fmt.Sprintf("node-%s-gpu", node.Name),
				MemoryTotal: utils.SafeGPUMemory(gpuQuantity.Value()), // Convert to bytes
			}
			total.GPUDevices = append(total.GPUDevices, gpuMetric)
		}
	}
	logrus.Debugf("Calculated total capacity: CPU=%d, Memory=%d, Storage=%d, EphemeralStorage=%d, Pods=%d, GPUs=%d",
		total.CPU, total.Memory, total.Storage, total.EphemeralStorage, total.Pods, len(total.GPUDevices))
	return total
}

func (cc *ClusterCollector) calculateTotalAllocatable(nodes []v1.Node) models.ResourceMetrics {
	total := models.ResourceMetrics{}
	for _, node := range nodes {
		total.CPU += node.Status.Allocatable.Cpu().MilliValue()
		total.Memory += node.Status.Allocatable.Memory().Value()
		total.Storage += node.Status.Allocatable.Storage().Value()
		if ephemeral, ok := node.Status.Allocatable[v1.ResourceEphemeralStorage]; ok {
			total.EphemeralStorage += ephemeral.Value()
		}
		total.Pods += node.Status.Allocatable.Pods().Value()

		// Collect GPU metrics if available
		if gpuQuantity, ok := node.Status.Allocatable["nvidia.com/gpu"]; ok {
			gpuMetric := models.GPUMetrics{
				DeviceID:    fmt.Sprintf("node-%s-gpu", node.Name),
				MemoryTotal: utils.SafeGPUMemory(gpuQuantity.Value()), // Convert to bytes
			}
			total.GPUDevices = append(total.GPUDevices, gpuMetric)
		}
	}
	logrus.Debugf("Calculated total allocatable: CPU=%d, Memory=%d, Storage=%d, EphemeralStorage=%d, Pods=%d, GPUs=%d",
		total.CPU, total.Memory, total.Storage, total.EphemeralStorage, total.Pods, len(total.GPUDevices))
	return total
}

func (cc *ClusterCollector) calculateTotalRequests(pods []v1.Pod) models.ResourceMetrics {
	total := models.ResourceMetrics{}
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			total.CPU += container.Resources.Requests.Cpu().MilliValue()
			total.Memory += container.Resources.Requests.Memory().Value()
			if ephemeral, ok := container.Resources.Requests[v1.ResourceEphemeralStorage]; ok {
				total.EphemeralStorage += ephemeral.Value()
			}

			// Sum GPU requests if available
			if gpuQuantity, ok := container.Resources.Requests["nvidia.com/gpu"]; ok {
				gpuMetric := models.GPUMetrics{
					DeviceID:    fmt.Sprintf("pod-%s-%s-gpu", pod.Namespace, pod.Name),
					MemoryTotal: utils.SafeGPUMemory(gpuQuantity.Value()), // Convert to bytes
				}
				total.GPUDevices = append(total.GPUDevices, gpuMetric)
			}
		}
	}
	total.Pods = int64(len(pods))
	logrus.Debugf("Calculated total requests: CPU=%d, Memory=%d, EphemeralStorage=%d, Pods=%d, GPUs=%d",
		total.CPU, total.Memory, total.EphemeralStorage, total.Pods, len(total.GPUDevices))
	return total
}

func (cc *ClusterCollector) calculateTotalLimits(pods []v1.Pod) models.ResourceMetrics {
	total := models.ResourceMetrics{}
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			total.CPU += container.Resources.Limits.Cpu().MilliValue()
			total.Memory += container.Resources.Limits.Memory().Value()
			if ephemeral, ok := container.Resources.Limits[v1.ResourceEphemeralStorage]; ok {
				total.EphemeralStorage += ephemeral.Value()
			}

			// Sum GPU limits if available
			if gpuQuantity, ok := container.Resources.Limits["nvidia.com/gpu"]; ok {
				gpuMetric := models.GPUMetrics{
					DeviceID:    fmt.Sprintf("pod-%s-%s-gpu", pod.Namespace, pod.Name),
					MemoryTotal: utils.SafeGPUMemory(gpuQuantity.Value()),
				}
				total.GPUDevices = append(total.GPUDevices, gpuMetric)
			}
		}
	}
	total.Pods = int64(len(pods))
	logrus.Debugf("Calculated total limits: CPU=%d, Memory=%d, EphemeralStorage=%d, Pods=%d, GPUs=%d",
		total.CPU, total.Memory, total.EphemeralStorage, total.Pods, len(total.GPUDevices))
	return total
}

func (cc *ClusterCollector) countContainers(pods []v1.Pod) int {
	count := 0
	for _, pod := range pods {
		count += len(pod.Spec.Containers)
		count += len(pod.Spec.InitContainers)
		count += len(pod.Spec.EphemeralContainers)
	}
	return count
}
