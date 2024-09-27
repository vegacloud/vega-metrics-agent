// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package collectors

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
)

type ClusterCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
	k8sConfig utils.K8sClientConfig
}

func NewClusterCollector(
	clientset *kubernetes.Clientset,
	cfg *config.Config,
	clientCFG utils.K8sClientConfig,
) *ClusterCollector {
	collector := &ClusterCollector{
		clientset: clientset,
		config:    cfg,
		k8sConfig: clientCFG,
	}
	logrus.Debug("ClusterCollector created successfully")
	return collector
}

func (cc *ClusterCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := cc.CollectClusterMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected cluster metrics")
	return metrics, nil
}

func (cc *ClusterCollector) CollectClusterMetrics(ctx context.Context) (*models.ClusterMetrics, error) {
	nodes, err := cc.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	logrus.Debugf("Successfully listed %d nodes", len(nodes.Items))

	pods, err := cc.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	logrus.Debugf("Successfully listed %d pods", len(pods.Items))

	metrics := &models.ClusterMetrics{
		KubernetesVersion: cc.k8sConfig.ClusterVersion,
		NodeCount:         len(nodes.Items),
		PodCount:          len(pods.Items),
		ContainerCount:    cc.countContainers(pods.Items),
		NodeLabels:        make(map[string]map[string]string), // New field for node labels
		PodLabels:         make(map[string]map[string]string), // New field for pod labels
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
		podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name) // Unique key for pod (namespace/name)
		metrics.PodLabels[podKey] = pod.Labels
	}

	logrus.Debug("Successfully calculated cluster metrics")
	return metrics, nil
}

func (cc *ClusterCollector) countContainers(pods []v1.Pod) int {
	count := 0
	for _, pod := range pods {
		count += len(pod.Spec.Containers)
	}
	logrus.Debugf("Counted %d containers", count)
	return count
}

func (cc *ClusterCollector) calculateTotalCapacity(nodes []v1.Node) models.ResourceMetrics {
	total := models.ResourceMetrics{}
	for _, node := range nodes {
		total.CPU += node.Status.Capacity.Cpu().MilliValue()
		total.Memory += node.Status.Capacity.Memory().Value()
		total.Pods += node.Status.Capacity.Pods().Value()
	}
	logrus.Debugf("Calculated total capacity: CPU=%d, Memory=%d, Pods=%d", total.CPU, total.Memory, total.Pods)
	return total
}

func (cc *ClusterCollector) calculateTotalAllocatable(nodes []v1.Node) models.ResourceMetrics {
	total := models.ResourceMetrics{}
	for _, node := range nodes {
		total.CPU += node.Status.Allocatable.Cpu().MilliValue()
		total.Memory += node.Status.Allocatable.Memory().Value()
		total.Pods += node.Status.Allocatable.Pods().Value()
	}
	logrus.Debugf("Calculated total allocatable: CPU=%d, Memory=%d, Pods=%d", total.CPU, total.Memory, total.Pods)
	return total
}

func (cc *ClusterCollector) calculateTotalRequests(pods []v1.Pod) models.ResourceMetrics {
	total := models.ResourceMetrics{}
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			total.CPU += container.Resources.Requests.Cpu().MilliValue()
			total.Memory += container.Resources.Requests.Memory().Value()
		}
	}
	total.Pods = int64(len(pods))
	logrus.Debugf("Calculated total requests: CPU=%d, Memory=%d, Pods=%d", total.CPU, total.Memory, total.Pods)
	return total
}

func (cc *ClusterCollector) calculateTotalLimits(pods []v1.Pod) models.ResourceMetrics {
	total := models.ResourceMetrics{}
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			total.CPU += container.Resources.Limits.Cpu().MilliValue()
			total.Memory += container.Resources.Limits.Memory().Value()
		}
	}
	total.Pods = int64(len(pods))
	logrus.Debugf("Calculated total limits: CPU=%d, Memory=%d, Pods=%d", total.CPU, total.Memory, total.Pods)
	return total
}
