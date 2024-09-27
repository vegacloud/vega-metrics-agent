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
package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

type JobCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewJobCollector(clientset *kubernetes.Clientset, cfg *config.Config) *JobCollector {
	collector := &JobCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("JobCollector created successfully")
	return collector
}

func (jc *JobCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := jc.CollectJobMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected job metrics")
	return metrics, nil
}

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
		metrics := models.JobMetrics{
			Name:      job.Name,
			Namespace: job.Namespace,
			Labels:    job.Labels,
			Completions: func() *int32 {
				if job.Spec.Completions != nil {
					return job.Spec.Completions
				}
				return nil
			}(),
			Parallelism: func() *int32 {
				if job.Spec.Parallelism != nil {
					return job.Spec.Parallelism
				}
				return nil
			}(),
			Active:    job.Status.Active,
			Succeeded: job.Status.Succeeded,
			Failed:    job.Status.Failed,
			StartTime: func() time.Time {
				if job.Status.StartTime != nil {
					return job.Status.StartTime.Time
				}
				return time.Time{}
			}(),
			CompletionTime: func() time.Time {
				if job.Status.CompletionTime != nil {
					return job.Status.CompletionTime.Time
				}
				return time.Time{}
			}(),
		}
		jobMetrics = append(jobMetrics, metrics)
		logrus.Debugf("Collected metrics for job %s/%s", job.Namespace, job.Name)
	}

	return jobMetrics, nil
}
