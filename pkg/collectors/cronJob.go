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
	"time"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

type CronJobCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewCronJobCollector(clientset *kubernetes.Clientset, cfg *config.Config) *CronJobCollector {
	collector := &CronJobCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("CronJobCollector created successfully")
	return collector
}

func (cjc *CronJobCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := cjc.CollectCronJobMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected cron job metrics")
	return metrics, nil
}

func (cjc *CronJobCollector) CollectCronJobMetrics(ctx context.Context) ([]models.CronJobMetrics, error) {
	cronJobs, err := cjc.clientset.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cron jobs: %w", err)
	}
	logrus.Debugf("Successfully listed %d cron jobs", len(cronJobs.Items))

	metrics := make([]models.CronJobMetrics, 0, len(cronJobs.Items))

	for _, cj := range cronJobs.Items {
		metrics = append(metrics, cjc.parseCronJobMetrics(cj))
	}

	logrus.Debugf("Collected metrics for %d cron jobs", len(metrics))
	return metrics, nil
}

func (cjc *CronJobCollector) parseCronJobMetrics(cj batchv1.CronJob) models.CronJobMetrics {
	// Create an empty cj.Labels if cj.Labels is nil
	if cj.Labels == nil {
		cj.Labels = make(map[string]string)
	}
	metrics := models.CronJobMetrics{
		Name:       cj.Name,
		Namespace:  cj.Namespace,
		Schedule:   cj.Spec.Schedule,
		Suspend:    cj.Spec.Suspend != nil && *cj.Spec.Suspend,
		ActiveJobs: len(cj.Status.Active),
		Labels:     cj.Labels,
		LastScheduleTime: func() *time.Time {
			if cj.Status.LastScheduleTime != nil {
				t := cj.Status.LastScheduleTime.Time
				return &t
			}
			return nil
		}(),
	}

	logrus.Debugf("Parsed cron job metrics for cron job %s/%s", cj.Namespace, cj.Name)
	return metrics
}
