// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/pod.go
package collectors

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// PodCollector collects metrics for all pods in the cluster
type PodCollector struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsclientset.Clientset
	config        *config.Config
}

// NewPodCollector creates a new PodCollector instance
func NewPodCollector(
	clientset *kubernetes.Clientset,
	metricsClient *metricsclientset.Clientset,
	cfg *config.Config,
) *PodCollector {
	collector := &PodCollector{
		clientset:     clientset,
		metricsClient: metricsClient,
		config:        cfg,
	}
	logrus.Debug("PodCollector created successfully")
	return collector
}

// CollectMetrics collects metrics for all pods in the cluster
func (pc *PodCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := pc.CollectEnhancedPodMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected pod metrics")
	return metrics, nil
}

func (pc *PodCollector) CollectEnhancedPodMetrics(ctx context.Context) ([]models.EnhancedPodMetrics, error) {
	pods, err := pc.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	logrus.Debugf("Successfully listed %d pods", len(pods.Items))

	enhancedPodMetrics := make([]models.EnhancedPodMetrics, 0, len(pods.Items))

	for _, pod := range pods.Items {
		metrics, err := pc.collectSinglePodMetrics(ctx, pod)
		if err != nil {
			logrus.Warnf("Failed to collect metrics for pod %s/%s: %v", pod.Namespace, pod.Name, err)
			continue
		}
		enhancedPodMetrics = append(enhancedPodMetrics, metrics)
	}

	logrus.Debug("Successfully collected enhanced pod metrics")
	return enhancedPodMetrics, nil
}

func (pc *PodCollector) collectSinglePodMetrics(ctx context.Context, pod v1.Pod) (models.EnhancedPodMetrics, error) {
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	metrics := models.EnhancedPodMetrics{
		PodMetrics: models.PodMetrics{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
			Labels:    pod.Labels,
		},
		QoSClass: string(pod.Status.QOSClass),
	}

	for _, condition := range pod.Status.Conditions {
		switch condition.Type {
		case v1.PodScheduled:
			metrics.Conditions.PodScheduled = condition.Status == v1.ConditionTrue
		case v1.PodInitialized:
			metrics.Conditions.Initialized = condition.Status == v1.ConditionTrue
		case v1.ContainersReady:
			metrics.Conditions.ContainersReady = condition.Status == v1.ConditionTrue
		case v1.PodReady:
			metrics.Conditions.Ready = condition.Status == v1.ConditionTrue
		}
	}

	for _, container := range pod.Spec.Containers {
		metrics.Requests.CPU += container.Resources.Requests.Cpu().MilliValue()
		metrics.Requests.Memory += container.Resources.Requests.Memory().Value()
		metrics.Limits.CPU += container.Resources.Limits.Cpu().MilliValue()
		metrics.Limits.Memory += container.Resources.Limits.Memory().Value()
	}

	podMetrics, err := pc.metricsClient.
		MetricsV1beta1().
		PodMetricses(pod.Namespace).
		Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		return metrics, fmt.Errorf("failed to get pod metrics: %w", err)
	}

	metrics.Containers = pc.extractContainerMetrics(pod, podMetrics)
	metrics.TotalRestarts = pc.getTotalRestarts(pod)

	logrus.Debugf("Successfully collected metrics for pod %s/%s", pod.Namespace, pod.Name)
	return metrics, nil
}

func (pc *PodCollector) extractContainerMetrics(
	pod v1.Pod,
	podMetrics *metricsv1beta1.PodMetrics,
) []models.ContainerMetrics {

	containerMetrics := make([]models.ContainerMetrics, 0, len(podMetrics.Containers))

	for _, container := range pod.Status.ContainerStatuses {
		metrics := models.ContainerMetrics{
			Name:         container.Name,
			RestartCount: container.RestartCount,
			Ready:        container.Ready,
			State:        getContainerState(container.State),
		}

		if container.LastTerminationState.Terminated != nil {
			metrics.LastTerminated = container.LastTerminationState.Terminated.Reason
		}

		for _, c := range podMetrics.Containers {
			if c.Name == container.Name {
				metrics.CPU = c.Usage.Cpu().MilliValue()
				metrics.Memory = c.Usage.Memory().Value()
				break
			}
		}

		containerMetrics = append(containerMetrics, metrics)
	}

	logrus.Debugf("Successfully extracted container metrics for pod %s/%s", pod.Namespace, pod.Name)
	return containerMetrics
}

func (pc *PodCollector) getTotalRestarts(pod v1.Pod) int32 {
	var totalRestarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		totalRestarts += containerStatus.RestartCount
	}
	logrus.Debugf("Total restarts for pod %s/%s: %d", pod.Namespace, pod.Name, totalRestarts)
	return totalRestarts
}

func getContainerState(state v1.ContainerState) string {
	if state.Running != nil {
		return "Running"
	}
	if state.Waiting != nil {
		return "Waiting"
	}
	if state.Terminated != nil {
		return "Terminated"
	}
	return "Unknown"
}
