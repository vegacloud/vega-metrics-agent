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
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
)

// CronJobCollector collects metrics from Kubernetes cron jobs.
type CronJobCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
	parser    cron.Parser
}

// NewCronJobCollector creates a new CronJobCollector.
func NewCronJobCollector(clientset *kubernetes.Clientset, cfg *config.Config) *CronJobCollector {
	logrus.Debug("CronJobCollector initialized successfully")
	return &CronJobCollector{
		clientset: clientset,
		config:    cfg,
		parser:    cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
	}
}

// CollectMetrics collects metrics from Kubernetes cron jobs.
func (cjc *CronJobCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	cronJobs, err := cjc.clientset.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cron jobs: %w", err)
	}
	logrus.Debugf("Successfully listed %d cron jobs", len(cronJobs.Items))

	metrics := make([]models.CronJobMetrics, 0, len(cronJobs.Items))
	for _, cj := range cronJobs.Items {
		metric, err := cjc.parseCronJobMetrics(ctx, cj)
		if err != nil {
			logrus.Warnf("Failed to parse metrics for cronjob %s/%s: %v", cj.Namespace, cj.Name, err)
			continue
		}
		metrics = append(metrics, metric)
	}

	logrus.Debugf("Collected metrics for %d cron jobs", len(metrics))
	return metrics, nil
}

// parseCronJobMetrics parses metrics from a Kubernetes cron job.
func (cjc *CronJobCollector) parseCronJobMetrics(ctx context.Context, cj batchv1.CronJob) (models.CronJobMetrics, error) {
	metrics := models.CronJobMetrics{
		Name:      cj.Name,
		Namespace: cj.Namespace,
		Schedule:  cj.Spec.Schedule,
		Labels:    cj.Labels,
		Status: models.CronJobStatus{
			Active:             len(cj.Status.Active),
			LastScheduleTime:   cjc.convertTime(cj.Status.LastScheduleTime),
			LastSuccessfulTime: cjc.convertTime(cj.Status.LastSuccessfulTime),
		},
		Spec: models.CronJobSpec{
			Suspend:                    cjc.getBoolValue(cj.Spec.Suspend),
			Concurrency:                string(cj.Spec.ConcurrencyPolicy),
			StartingDeadlineSeconds:    cjc.getInt64Value(cj.Spec.StartingDeadlineSeconds),
			SuccessfulJobsHistoryLimit: cjc.getInt32Value(cj.Spec.SuccessfulJobsHistoryLimit),
			FailedJobsHistoryLimit:     cjc.getInt32Value(cj.Spec.FailedJobsHistoryLimit),
		},
	}

	// Calculate next scheduled time
	nextSchedule, err := cjc.calculateNextSchedule(cj)
	if err != nil {
		logrus.Warnf("Failed to calculate next schedule for cronjob %s/%s: %v", cj.Namespace, cj.Name, err)
	} else {
		metrics.Status.NextScheduledTime = nextSchedule
	}

	// Get associated jobs metrics
	jobs, err := cjc.getAssociatedJobsMetrics(ctx, cj)
	if err != nil {
		logrus.Warnf("Failed to get associated jobs for cronjob %s/%s: %v", cj.Namespace, cj.Name, err)
	} else {
		metrics.JobMetrics = jobs
		metrics.Status.SuccessRate = cjc.calculateSuccessRate(jobs)
	}

	return metrics, nil
}

// getAssociatedJobsMetrics collects metrics from jobs associated with the CronJob
func (cjc *CronJobCollector) getAssociatedJobsMetrics(ctx context.Context, cj batchv1.CronJob) ([]models.JobMetrics, error) {
	selector := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", cj.Name),
	}

	jobs, err := cjc.clientset.BatchV1().Jobs(cj.Namespace).List(ctx, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	metrics := make([]models.JobMetrics, 0, len(jobs.Items))
	for _, job := range jobs.Items {
		metrics = append(metrics, cjc.parseJobMetrics(job))
	}

	return metrics, nil
}

// parseJobMetrics parses metrics from a Kubernetes job
func (cjc *CronJobCollector) parseJobMetrics(job batchv1.Job) models.JobMetrics {
	metrics := models.JobMetrics{
		Name:           job.Name,
		Namespace:      job.Namespace,
		StartTime:      cjc.convertTime(job.Status.StartTime),
		CompletionTime: cjc.convertTime(job.Status.CompletionTime),
		Active:         job.Status.Active,
		Succeeded:      job.Status.Succeeded,
		Failed:         job.Status.Failed,
		Status:         cjc.getJobStatus(job.Status),
		Duration:       cjc.calculateJobDuration(job.Status),
		Labels:         job.Labels,
		Resources:      cjc.getJobResourceMetrics(job),
	}
	return metrics
}

// Helper functions

func (cjc *CronJobCollector) calculateNextSchedule(cj batchv1.CronJob) (*time.Time, error) {
	schedule, err := cjc.parser.Parse(cj.Spec.Schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schedule: %w", err)
	}

	var baseTime time.Time
	if cj.Status.LastScheduleTime != nil {
		baseTime = cj.Status.LastScheduleTime.Time
	} else {
		baseTime = time.Now()
	}

	next := schedule.Next(baseTime)
	return &next, nil
}

func (cjc *CronJobCollector) calculateSuccessRate(jobs []models.JobMetrics) float64 {
	if len(jobs) == 0 {
		return 0
	}

	completed := 0
	succeeded := 0
	for _, job := range jobs {
		if job.CompletionTime != nil {
			completed++
			if job.Succeeded > 0 {
				succeeded++
			}
		}
	}

	if completed == 0 {
		return 0
	}

	return float64(succeeded) / float64(completed) * 100
}
func (cjc *CronJobCollector) getJobResourceMetrics(job batchv1.Job) models.ResourceMetrics {
	resources := models.ResourceMetrics{}

	if job.Spec.Template.Spec.Containers == nil {
		return resources
	}

	for _, container := range job.Spec.Template.Spec.Containers {
		if container.Resources.Requests != nil {
			resources.CPU += container.Resources.Requests.Cpu().MilliValue()
			resources.Memory += container.Resources.Requests.Memory().Value()
			if storage, ok := container.Resources.Requests[v1.ResourceStorage]; ok {
				resources.Storage += storage.Value()
			}
			if ephemeral, ok := container.Resources.Requests[v1.ResourceEphemeralStorage]; ok {
				resources.EphemeralStorage += ephemeral.Value()
			}
		}

		if gpuQuantity, ok := container.Resources.Requests["nvidia.com/gpu"]; ok {
			gpuMetric := models.GPUMetrics{
				DeviceID:    fmt.Sprintf("job-%s-%s-gpu", job.Namespace, job.Name),
				MemoryTotal: utils.SafeGPUMemory(gpuQuantity.Value()),
				MemoryUsed:  0,
				OptMetrics: models.OptionalMetrics{
					DutyCycle:   0,
					Temperature: 0,
					PowerUsage:  0,
				},
			}
			resources.GPUDevices = append(resources.GPUDevices, gpuMetric)
		}
	}

	return resources
}

// Utility functions for handling nullable values
func (cjc *CronJobCollector) convertTime(t *metav1.Time) *time.Time {
	if t == nil {
		return nil
	}
	converted := t.Time
	return &converted
}

func (cjc *CronJobCollector) getBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func (cjc *CronJobCollector) getInt64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func (cjc *CronJobCollector) getInt32Value(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func (cjc *CronJobCollector) getJobStatus(status batchv1.JobStatus) string {
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

func (cjc *CronJobCollector) calculateJobDuration(status batchv1.JobStatus) *time.Duration {
	if status.StartTime == nil {
		return nil
	}

	endTime := time.Now()
	if status.CompletionTime != nil {
		endTime = status.CompletionTime.Time
	}

	duration := endTime.Sub(status.StartTime.Time)
	return &duration
}
