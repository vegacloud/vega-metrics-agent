// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/job.go

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// JobCollector collects metrics from Kubernetes jobs.
type JobCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewJobCollector creates a new JobCollector.
func NewJobCollector(clientset *kubernetes.Clientset, cfg *config.Config) *JobCollector {
	collector := &JobCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("JobCollector created successfully")
	return collector
}

// CollectMetrics collects metrics from Kubernetes jobs.
func (jc *JobCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := jc.CollectJobMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected job metrics")
	return metrics, nil
}

// CollectJobMetrics collects metrics from Kubernetes jobs.
func (jc *JobCollector) CollectJobMetrics(ctx context.Context) ([]models.JobMetrics, error) {
	jobs, err := jc.clientset.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	logrus.Debugf("Successfully listed %d jobs", len(jobs.Items))

	jobMetrics := make([]models.JobMetrics, 0, len(jobs.Items))

	for _, job := range jobs.Items {
		if job.Labels == nil {
			job.Labels = make(map[string]string)
		}

		var duration *time.Duration
		if job.Status.CompletionTime != nil && job.Status.StartTime != nil {
			d := job.Status.CompletionTime.Sub(job.Status.StartTime.Time)
			duration = &d
		}

		status := calculateJobStatus(&job)

		metrics := models.JobMetrics{
			Name:            job.Name,
			Namespace:       job.Namespace,
			Labels:          job.Labels,
			Active:          job.Status.Active,
			Succeeded:       job.Status.Succeeded,
			Failed:          job.Status.Failed,
			Status:          status,
			StartTime:       timePtr(job.Status.StartTime),
			CompletionTime:  timePtr(job.Status.CompletionTime),
			Duration:        duration,
			Parallelism:     job.Spec.Parallelism,
			Completions:     job.Spec.Completions,
			BackoffLimit:    job.Spec.BackoffLimit,
			Suspended:       job.Spec.Suspend != nil && *job.Spec.Suspend,
			CreationTime:    &job.CreationTimestamp.Time,
			Conditions:      convertJobConditions(job.Status.Conditions),
			ResourceMetrics: jc.collectJobResourceMetrics(ctx, &job),
		}

		jobMetrics = append(jobMetrics, metrics)
		logrus.Debugf("Collected metrics for job %s/%s", job.Namespace, job.Name)
	}

	return jobMetrics, nil
}

// Helper functions

func timePtr(t *metav1.Time) *time.Time {
	if t == nil {
		return nil
	}
	tt := t.Time
	return &tt
}

func calculateJobStatus(job *batchv1.Job) string {
	if job.Status.Succeeded > 0 {
		return "Completed"
	}
	if job.Status.Failed > 0 {
		return "Failed"
	}
	if job.Status.Active > 0 {
		return "Active"
	}
	if job.Spec.Suspend != nil && *job.Spec.Suspend {
		return "Suspended"
	}
	return "Pending"
}

func convertJobConditions(conditions []batchv1.JobCondition) []models.JobCondition {
	result := make([]models.JobCondition, 0, len(conditions))
	for _, c := range conditions {
		condition := models.JobCondition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			LastProbeTime:      timePtr(&c.LastProbeTime),
			LastTransitionTime: timePtr(&c.LastTransitionTime),
			Reason:             c.Reason,
			Message:            c.Message,
		}
		result = append(result, condition)
	}
	return result
}

func (jc *JobCollector) collectJobResourceMetrics(ctx context.Context, job *batchv1.Job) models.ResourceMetrics {
	selector := metav1.LabelSelector{MatchLabels: job.Spec.Selector.MatchLabels}
	labelSelector, err := metav1.LabelSelectorAsSelector(&selector)
	if err != nil {
		logrus.Errorf("Failed to create selector for job %s/%s: %v", job.Namespace, job.Name, err)
		return models.ResourceMetrics{}
	}

	pods, err := jc.clientset.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		logrus.Errorf("Failed to list pods for job %s/%s: %v", job.Namespace, job.Name, err)
		return models.ResourceMetrics{}
	}

	metrics := models.ResourceMetrics{}
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			metrics.CPU += container.Resources.Requests.Cpu().MilliValue()
			metrics.Memory += container.Resources.Requests.Memory().Value()
			if container.Resources.Requests.Storage() != nil {
				metrics.Storage += container.Resources.Requests.Storage().Value()
			}
		}
	}

	return metrics
}
