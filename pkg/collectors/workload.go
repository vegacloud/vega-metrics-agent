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
package collectors

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

type WorkloadCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewWorkloadCollector(clientset *kubernetes.Clientset, cfg *config.Config) *WorkloadCollector {
	return &WorkloadCollector{
		clientset: clientset,
		config:    cfg,
	}
}

func (wc *WorkloadCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	return wc.CollectWorkloadMetrics(ctx)
}

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
	return models.DeploymentMetrics{
		Name:              d.Name,
		Namespace:         d.Namespace,
		Replicas:          *d.Spec.Replicas,
		ReadyReplicas:     d.Status.ReadyReplicas,
		UpdatedReplicas:   d.Status.UpdatedReplicas,
		AvailableReplicas: d.Status.AvailableReplicas,
		Labels:            d.Labels,
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
	return models.StatefulSetMetrics{
		Name:            s.Name,
		Namespace:       s.Namespace,
		Replicas:        *s.Spec.Replicas,
		ReadyReplicas:   s.Status.ReadyReplicas,
		CurrentReplicas: s.Status.CurrentReplicas,
		UpdatedReplicas: s.Status.UpdatedReplicas,
		Labels:          s.Labels,
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
	return models.DaemonSetMetrics{
		Name:                   d.Name,
		Namespace:              d.Namespace,
		DesiredNumberScheduled: d.Status.DesiredNumberScheduled,
		CurrentNumberScheduled: d.Status.CurrentNumberScheduled,
		NumberReady:            d.Status.NumberReady,
		UpdatedNumberScheduled: d.Status.UpdatedNumberScheduled,
		NumberAvailable:        d.Status.NumberAvailable,
		Labels:                 d.Labels,
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
	metrics := models.JobMetrics{
		Name:      j.Name,
		Namespace: j.Namespace,
		Labels:    j.Labels,
		Completions: func() *int32 {
			if j.Spec.Completions != nil {
				return j.Spec.Completions
			}
			return nil
		}(),
		Parallelism: func() *int32 {
			if j.Spec.Parallelism != nil {
				return j.Spec.Parallelism
			}
			return nil
		}(),
		Active:    j.Status.Active,
		Succeeded: j.Status.Succeeded,
		Failed:    j.Status.Failed,
	}

	if j.Status.StartTime != nil {
		metrics.StartTime = j.Status.StartTime.Time
	}

	if j.Status.CompletionTime != nil {
		metrics.CompletionTime = j.Status.CompletionTime.Time
	}

	return metrics
}
