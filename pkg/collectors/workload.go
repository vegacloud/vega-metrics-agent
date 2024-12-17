// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/workload.go

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
	v1 "k8s.io/api/core/v1"
)

// WorkloadCollector collects metrics from Kubernetes workloads.
type WorkloadCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewWorkloadCollector creates a new WorkloadCollector.
func NewWorkloadCollector(clientset *kubernetes.Clientset, cfg *config.Config) *WorkloadCollector {
	return &WorkloadCollector{
		clientset: clientset,
		config:    cfg,
	}
}

// CollectMetrics collects metrics from Kubernetes workloads.
func (wc *WorkloadCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	return wc.CollectWorkloadMetrics(ctx)
}

// CollectWorkloadMetrics collects metrics from Kubernetes workloads.
func (wc *WorkloadCollector) CollectWorkloadMetrics(ctx context.Context) (*models.WorkloadMetrics, error) {
	metrics := &models.WorkloadMetrics{}

	var err error
	metrics.Deployments, err = wc.collectDeploymentMetrics(ctx)
	if err != nil {
		logrus.Warnf("Failed to collect deployment metrics: %v", err)
	} else {
		logrus.Debug("Successfully collected deployment metrics")
	}

	metrics.StatefulSets, err = wc.collectStatefulSetMetrics(ctx)
	if err != nil {
		logrus.Warnf("Failed to collect statefulset metrics: %v", err)
	} else {
		logrus.Debug("Successfully collected statefulset metrics")
	}

	metrics.DaemonSets, err = wc.collectDaemonSetMetrics(ctx)
	if err != nil {
		logrus.Warnf("Failed to collect daemonset metrics: %v", err)
	} else {
		logrus.Debug("Successfully collected daemonset metrics")
	}

	metrics.Jobs, err = wc.collectJobMetrics(ctx)
	if err != nil {
		logrus.Warnf("Failed to collect job metrics: %v", err)
	} else {
		logrus.Debug("Successfully collected job metrics")
	}

	return metrics, nil
}

// collectDeploymentMetrics collects metrics from Kubernetes deployments.
func (wc *WorkloadCollector) collectDeploymentMetrics(ctx context.Context) ([]models.DeploymentMetrics, error) {
	deployments, err := wc.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	metrics := make([]models.DeploymentMetrics, 0, len(deployments.Items))

	for _, d := range deployments.Items {
		metrics = append(metrics, wc.parseDeploymentMetrics(d))
	}

	logrus.Debugf("Collected metrics for %d deployments", len(metrics))
	return metrics, nil
}

func (wc *WorkloadCollector) parseDeploymentMetrics(d appsv1.Deployment) models.DeploymentMetrics {
	if d.Labels == nil {
		d.Labels = make(map[string]string)
	}

	conditions := make([]string, 0)
	for _, condition := range d.Status.Conditions {
		conditions = append(conditions, string(condition.Type))
	}

	return models.DeploymentMetrics{
		Name:               d.Name,
		Namespace:          d.Namespace,
		Replicas:           *d.Spec.Replicas,
		ReadyReplicas:      d.Status.ReadyReplicas,
		UpdatedReplicas:    d.Status.UpdatedReplicas,
		AvailableReplicas:  d.Status.AvailableReplicas,
		Labels:             d.Labels,
		CollisionCount:     d.Status.CollisionCount,
		Conditions:         conditions,
		Generation:         d.Generation,
		ObservedGeneration: d.Status.ObservedGeneration,
	}
}

func (wc *WorkloadCollector) collectStatefulSetMetrics(ctx context.Context) ([]models.StatefulSetMetrics, error) {
	statefulSets, err := wc.clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}
	metrics := make([]models.StatefulSetMetrics, 0, len(statefulSets.Items))

	for _, s := range statefulSets.Items {
		metrics = append(metrics, wc.parseStatefulSetMetrics(s))
	}

	logrus.Debugf("Collected metrics for %d statefulsets", len(metrics))
	return metrics, nil
}

func (wc *WorkloadCollector) parseStatefulSetMetrics(s appsv1.StatefulSet) models.StatefulSetMetrics {
	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}

	conditions := make([]string, 0)
	for _, condition := range s.Status.Conditions {
		conditions = append(conditions, string(condition.Type))
	}

	return models.StatefulSetMetrics{
		Name:               s.Name,
		Namespace:          s.Namespace,
		Replicas:           *s.Spec.Replicas,
		ReadyReplicas:      s.Status.ReadyReplicas,
		CurrentReplicas:    s.Status.CurrentReplicas,
		UpdatedReplicas:    s.Status.UpdatedReplicas,
		AvailableReplicas:  s.Status.AvailableReplicas,
		Labels:             s.Labels,
		CollisionCount:     s.Status.CollisionCount,
		Conditions:         conditions,
		Generation:         s.Generation,
		ObservedGeneration: s.Status.ObservedGeneration,
	}
}

func (wc *WorkloadCollector) collectDaemonSetMetrics(ctx context.Context) ([]models.DaemonSetMetrics, error) {
	daemonSets, err := wc.clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}
	metrics := make([]models.DaemonSetMetrics, 0, len(daemonSets.Items))

	for _, d := range daemonSets.Items {
		metrics = append(metrics, wc.parseDaemonSetMetrics(d))
	}

	logrus.Debugf("Collected metrics for %d daemonsets", len(metrics))
	return metrics, nil
}

func (wc *WorkloadCollector) parseDaemonSetMetrics(d appsv1.DaemonSet) models.DaemonSetMetrics {
	if d.Labels == nil {
		d.Labels = make(map[string]string)
	}

	conditions := make([]models.DaemonSetCondition, 0)
	for _, condition := range d.Status.Conditions {
		conditions = append(conditions, models.DaemonSetCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: &condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	return models.DaemonSetMetrics{
		Name:                   d.Name,
		Namespace:              d.Namespace,
		DesiredNumberScheduled: d.Status.DesiredNumberScheduled,
		CurrentNumberScheduled: d.Status.CurrentNumberScheduled,
		NumberReady:            d.Status.NumberReady,
		UpdatedNumberScheduled: d.Status.UpdatedNumberScheduled,
		NumberAvailable:        d.Status.NumberAvailable,
		NumberUnavailable:      d.Status.NumberUnavailable,
		NumberMisscheduled:     d.Status.NumberMisscheduled,
		Labels:                 d.Labels,
		Generation:             d.Generation,
		ObservedGeneration:     d.Status.ObservedGeneration,
		Conditions:             conditions,
	}
}

func (wc *WorkloadCollector) collectJobMetrics(ctx context.Context) ([]models.JobMetrics, error) {
	jobs, err := wc.clientset.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	metrics := make([]models.JobMetrics, 0, len(jobs.Items))

	for _, j := range jobs.Items {
		metrics = append(metrics, wc.parseJobMetrics(j))
	}

	logrus.Debugf("Collected metrics for %d jobs", len(metrics))
	return metrics, nil
}

func (wc *WorkloadCollector) parseJobMetrics(j batchv1.Job) models.JobMetrics {
	if j.Labels == nil {
		j.Labels = make(map[string]string)
	}

	conditions := make([]models.JobCondition, 0)
	for _, condition := range j.Status.Conditions {
		conditions = append(conditions, models.JobCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastProbeTime:      &condition.LastProbeTime.Time,
			LastTransitionTime: &condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	metrics := models.JobMetrics{
		Name:             j.Name,
		Namespace:        j.Namespace,
		Labels:           j.Labels,
		Active:           j.Status.Active,
		Succeeded:        j.Status.Succeeded,
		Failed:           j.Status.Failed,
		Status:           getJobStatus(j.Status),
		CompletedIndexes: j.Status.CompletedIndexes,
		Conditions:       conditions,
		Generation:       j.Generation,
	}

	// Add existing time-related fields
	if j.Status.StartTime != nil {
		metrics.StartTime = &j.Status.StartTime.Time
	}
	if j.Status.CompletionTime != nil {
		metrics.CompletionTime = &j.Status.CompletionTime.Time
	}
	if metrics.StartTime != nil {
		endTime := time.Now()
		if metrics.CompletionTime != nil {
			endTime = *metrics.CompletionTime
		}
		duration := endTime.Sub(*metrics.StartTime)
		metrics.Duration = &duration
	}

	metrics.ResourceMetrics = wc.getJobResourceMetrics(j)
	return metrics
}

// Helper function to get job status
func getJobStatus(status batchv1.JobStatus) string {
	switch {
	case status.Succeeded > 0:
		return "Succeeded"
	case status.Failed > 0:
		return "Failed"
	case status.Active > 0:
		return "Active"
	default:
		return "Pending"
	}
}

// Helper function to get resource metrics for a job
func (wc *WorkloadCollector) getJobResourceMetrics(job batchv1.Job) models.ResourceMetrics {
	metrics := models.ResourceMetrics{}

	if job.Spec.Template.Spec.Containers == nil {
		return metrics
	}

	for _, container := range job.Spec.Template.Spec.Containers {
		if container.Resources.Requests != nil {
			metrics.CPU += container.Resources.Requests.Cpu().MilliValue()
			metrics.Memory += container.Resources.Requests.Memory().Value()
			if storage := container.Resources.Requests.Storage(); storage != nil {
				metrics.Storage += storage.Value()
			}
			if ephemeral, ok := container.Resources.Requests[v1.ResourceEphemeralStorage]; ok {
				metrics.EphemeralStorage += ephemeral.Value()
			}
		}
	}

	return metrics
}
