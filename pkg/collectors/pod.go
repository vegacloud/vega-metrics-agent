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

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// PodCollector collects metrics for all pods in the cluster
type PodCollector struct {
	clientset  *kubernetes.Clientset
	config     *config.Config
	httpClient *http.Client
}

// NewPodCollector creates a new PodCollector
func NewPodCollector(clientset *kubernetes.Clientset, cfg *config.Config) *PodCollector {
	return &PodCollector{
		clientset: clientset,
		config:    cfg,
	}
}

// CollectMetrics collects metrics for all pods in the cluster
func (pc *PodCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	logrus.WithField("collector", "PodCollector")

	// Verify client identity before collecting metrics
	if err := VerifyCollectorClient(ctx, pc.clientset, pc.config.VegaNamespace, "PodCollector"); err != nil {
		return nil, err
	}
	metrics, err := pc.CollectEnhancedPodMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected pod metrics")
	return metrics, nil
}

// CollectEnhancedPodMetrics collects metrics from Kubernetes pods.
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
		StartTime: func() *time.Time {
			if pod.Status.StartTime != nil {
				t := pod.Status.StartTime.Time
				return &t
			}
			return nil
		}(),
		Priority:          pod.Spec.Priority,
		PriorityClassName: pod.Spec.PriorityClassName,
		NodeName:          pod.Spec.NodeName,
		HostIP:            pod.Status.HostIP,
		NominatedNodeName: pod.Status.NominatedNodeName,
	}

	// Collect Pod IPs
	podIPs := make([]string, 0, len(pod.Status.PodIPs))
	for _, ip := range pod.Status.PodIPs {
		podIPs = append(podIPs, ip.IP)
	}
	metrics.PodIPs = podIPs

	// Collect Readiness Gates
	readinessGates := make([]models.PodReadinessGate, 0, len(pod.Spec.ReadinessGates))
	for _, gate := range pod.Spec.ReadinessGates {
		status := false
		for _, condition := range pod.Status.Conditions {
			if string(condition.Type) == string(gate.ConditionType) {
				status = condition.Status == v1.ConditionTrue
				break
			}
		}
		readinessGates = append(readinessGates, models.PodReadinessGate{
			ConditionType: string(gate.ConditionType),
			Status:        status,
		})
	}
	metrics.ReadinessGates = readinessGates

	// Collect Pod Conditions
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

	// Collect Resource Metrics
	for _, container := range pod.Spec.Containers {
		metrics.Requests.CPU += container.Resources.Requests.Cpu().MilliValue()
		metrics.Requests.Memory += container.Resources.Requests.Memory().Value()
		metrics.Limits.CPU += container.Resources.Limits.Cpu().MilliValue()
		metrics.Limits.Memory += container.Resources.Limits.Memory().Value()
	}

	// Get Pod Metrics from Kubelet
	podMetrics, err := pc.getPodMetrics(ctx, &pod)
	if err != nil {
		return metrics, fmt.Errorf("failed to get pod metrics: %w", err)
	}

	metrics.Containers = pc.extractContainerMetrics(pod, podMetrics)
	metrics.TotalRestarts = pc.getTotalRestarts(pod)

	// Set completion time for completed pods
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Terminated != nil {
				metrics.CompletionTime = &containerStatus.State.Terminated.FinishedAt.Time
				break
			}
		}
	}

	// Add annotations
	metrics.Annotations = pod.Annotations

	// Add volume mounts
	metrics.VolumeMounts = make([]models.VolumeMountMetrics, 0)
	for _, container := range pod.Spec.Containers {
		for _, volumeMount := range container.VolumeMounts {
			metrics.VolumeMounts = append(metrics.VolumeMounts, models.VolumeMountMetrics{
				Name:        volumeMount.Name,
				MountPath:   volumeMount.MountPath,
				ReadOnly:    volumeMount.ReadOnly,
				SubPath:     volumeMount.SubPath,
				SubPathExpr: volumeMount.SubPathExpr,
				MountPropagation: func() string {
					if volumeMount.MountPropagation != nil {
						return string(*volumeMount.MountPropagation)
					}
					return ""
				}(),
			})
		}
	}

	// Add image pull policy
	imagePullPolicies := make([]string, 0)
	for _, container := range pod.Spec.Containers {
		imagePullPolicies = append(imagePullPolicies, string(container.ImagePullPolicy))
	}
	// Use the most common pull policy, or "Mixed" if there are different policies
	if len(imagePullPolicies) > 0 {
		allSame := true
		for i := 1; i < len(imagePullPolicies); i++ {
			if imagePullPolicies[i] != imagePullPolicies[0] {
				allSame = false
				break
			}
		}
		if allSame {
			metrics.ImagePullPolicy = imagePullPolicies[0]
		} else {
			metrics.ImagePullPolicy = "Mixed"
		}
	}

	// Add service account information
	metrics.ServiceAccountName = pod.Spec.ServiceAccountName

	// Collect PDB metrics
	pdbMetrics, err := pc.collectPodDisruptionBudget(ctx, pod)
	if err != nil {
		logrus.Warnf("Failed to collect PDB metrics for pod %s/%s: %v", pod.Namespace, pod.Name, err)
	} else {
		metrics.DisruptionBudget = pdbMetrics
	}

	// Collect topology spread constraints
	metrics.TopologySpread = pc.collectTopologySpread(pod)

	// Collect pod overhead
	if pod.Spec.Overhead != nil {
		metrics.Overhead = &models.PodOverheadMetrics{
			CPU:    pod.Spec.Overhead.Cpu().String(),
			Memory: pod.Spec.Overhead.Memory().String(),
		}
	}

	// Collect scheduling gates
	for _, gate := range pod.Spec.SchedulingGates {
		metrics.SchedulingGates = append(metrics.SchedulingGates, models.PodSchedulingGate{
			Name:   gate.Name,
			Active: true,
		})
	}

	// Collect security context
	metrics.SecurityContext = pc.collectSecurityContext(pod)

	// Collect QoS details
	metrics.QoSDetails = pc.collectQoSDetails(pod)

	logrus.Debugf("Successfully collected metrics for pod %s/%s", pod.Namespace, pod.Name)
	return metrics, nil
}

func (pc *PodCollector) getPodMetrics(ctx context.Context, pod *v1.Pod) (*models.PodMetrics, error) {
	// Get node internal IP where the pod is running
	node, err := pc.clientset.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	var nodeAddress string
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			nodeAddress = addr.Address
			break
		}
	}

	// Construct URL for kubelet metrics
	metricsURL := fmt.Sprintf("https://%s:10250/metrics/resource", nodeAddress)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", metricsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Get bearer token from service account
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Errorf("failed to read service account token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+string(token))

	// Make request
	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics from kubelet: %w", err)
	}
	defer resp.Body.Close()

	// Parse metrics
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}

	// Create pod metrics object
	metrics := &models.PodMetrics{
		Name:       pod.Name,
		Namespace:  pod.Namespace,
		Usage:      models.ResourceMetrics{},
		Containers: make([]models.ContainerMetrics, 0),
	}

	containerMetrics := make(map[string]*models.ContainerMetrics)

	// Parse container metrics for the pod
	for _, family := range metricFamilies {
		for _, metric := range family.Metric {
			labels := make(map[string]string)
			for _, label := range metric.Label {
				labels[*label.Name] = *label.Value
			}

			// Match metrics for this specific pod
			if labels["pod"] == pod.Name && labels["namespace"] == pod.Namespace {
				containerName := labels["container"]

				// Initialize container metrics if not exists
				if _, exists := containerMetrics[containerName]; !exists {
					containerMetrics[containerName] = &models.ContainerMetrics{
						Name: containerName,
					}
				}

				// Update container metrics based on metric type
				switch family.GetName() {
				case "container_cpu_usage_seconds_total":
					value := int64(*metric.Counter.Value * 1000) // Convert to millicores
					containerMetrics[containerName].UsageNanos = value
					metrics.Usage.CPU += value
				case "container_memory_working_set_bytes":
					value := int64(*metric.Gauge.Value)
					containerMetrics[containerName].UsageBytes = value
					metrics.Usage.Memory += value
					containerMetrics[containerName].Memory.WorkingSet = uint64(*metric.Gauge.Value)
				}
			}
		}
	}

	// Convert map to slice
	for _, cm := range containerMetrics {
		metrics.Containers = append(metrics.Containers, *cm)
	}

	return metrics, nil
}

func (pc *PodCollector) extractContainerMetrics(
	pod v1.Pod,
	podMetrics *models.PodMetrics,
) []models.ContainerMetrics {

	containerMetrics := make([]models.ContainerMetrics, 0, len(pod.Status.ContainerStatuses))

	for _, container := range pod.Status.ContainerStatuses {
		metrics := models.ContainerMetrics{
			Name:         container.Name,
			RestartCount: container.RestartCount,
			Ready:        container.Ready,
			State:        getContainerState(container.State),
		}

		if container.LastTerminationState.Terminated != nil {
			metrics.LastTerminationReason = container.LastTerminationState.Terminated.Reason
		}

		// Match container usage from podMetrics
		for _, containerName := range podMetrics.Containers {
			if containerName.Name == container.Name {
				metrics.CPU = containerName.CPU
				metrics.Memory = containerName.Memory
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

func (pc *PodCollector) collectPodDisruptionBudget(ctx context.Context, pod v1.Pod) (*models.PodDisruptionBudgetMetrics, error) {
	pdbs, err := pc.clientset.PolicyV1().PodDisruptionBudgets(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pdb := range pdbs.Items {
		selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
		if err != nil {
			continue
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			return &models.PodDisruptionBudgetMetrics{
				MinAvailable:       pdb.Spec.MinAvailable.String(),
				MaxUnavailable:     pdb.Spec.MaxUnavailable.String(),
				CurrentHealthy:     pdb.Status.CurrentHealthy,
				DesiredHealthy:     pdb.Status.DesiredHealthy,
				DisruptionsAllowed: pdb.Status.DisruptionsAllowed,
				ExpectedPods:       pdb.Status.ExpectedPods,
			}, nil
		}
	}
	return nil, nil
}

func (pc *PodCollector) collectTopologySpread(pod v1.Pod) []models.TopologySpreadConstraint {
	var constraints []models.TopologySpreadConstraint
	for _, constraint := range pod.Spec.TopologySpreadConstraints {
		constraints = append(constraints, models.TopologySpreadConstraint{
			MaxSkew:           constraint.MaxSkew,
			TopologyKey:       constraint.TopologyKey,
			WhenUnsatisfiable: string(constraint.WhenUnsatisfiable),
			LabelSelector:     constraint.LabelSelector.String(),
			MinDomains:        constraint.MinDomains,
		})
	}
	return constraints
}

func (pc *PodCollector) collectSecurityContext(pod v1.Pod) *models.SecurityContextMetrics {
	if pod.Spec.SecurityContext == nil {
		return nil
	}

	return &models.SecurityContextMetrics{
		RunAsUser:    pod.Spec.SecurityContext.RunAsUser,
		RunAsGroup:   pod.Spec.SecurityContext.RunAsGroup,
		FSGroup:      pod.Spec.SecurityContext.FSGroup,
		RunAsNonRoot: pod.Spec.SecurityContext.RunAsNonRoot,
		SELinuxOptions: func() map[string]string {
			if pod.Spec.SecurityContext.SELinuxOptions != nil {
				return map[string]string{
					"user":  pod.Spec.SecurityContext.SELinuxOptions.User,
					"role":  pod.Spec.SecurityContext.SELinuxOptions.Role,
					"type":  pod.Spec.SecurityContext.SELinuxOptions.Type,
					"level": pod.Spec.SecurityContext.SELinuxOptions.Level,
				}
			}
			return nil
		}(),
	}
}

func (pc *PodCollector) collectQoSDetails(pod v1.Pod) *models.QoSMetrics {
	qosMetrics := &models.QoSMetrics{
		Class:            string(pod.Status.QOSClass),
		CPUGuaranteed:    true,
		MemoryGuaranteed: true,
	}

	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests.Cpu().IsZero() {
			qosMetrics.CPUGuaranteed = false
		}
		if container.Resources.Requests.Memory().IsZero() {
			qosMetrics.MemoryGuaranteed = false
		}
	}

	return qosMetrics
}
