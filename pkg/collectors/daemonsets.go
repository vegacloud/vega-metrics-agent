// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/daemonsets.go

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// DaemonSetCollector collects metrics for daemon sets
type DaemonSetCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewDaemonSetCollector creates a new DaemonSetCollector instance
func NewDaemonSetCollector(clientset *kubernetes.Clientset, cfg *config.Config) *DaemonSetCollector {
	collector := &DaemonSetCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("DaemonSetCollector created successfully")
	return collector
}

// CollectMetrics collects metrics for daemon sets
func (dc *DaemonSetCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	daemonsets, err := dc.clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}

	metrics := make([]models.DaemonSetMetrics, 0, len(daemonsets.Items))
	for _, ds := range daemonsets.Items {
		metric := dc.convertDaemonSetToMetrics(&ds)
		metrics = append(metrics, metric)
	}

	logrus.Debugf("Collected metrics for %d daemonsets", len(metrics))
	return metrics, nil
}

// convertDaemonSetToMetrics converts a Kubernetes DaemonSet to metrics
func (dc *DaemonSetCollector) convertDaemonSetToMetrics(ds *appsv1.DaemonSet) models.DaemonSetMetrics {
	conditions := make([]models.DaemonSetCondition, 0, len(ds.Status.Conditions))
	for _, condition := range ds.Status.Conditions {
		conditions = append(conditions, models.DaemonSetCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: &condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	return models.DaemonSetMetrics{
		Name:                   ds.Name,
		Namespace:              ds.Namespace,
		DesiredNumberScheduled: ds.Status.DesiredNumberScheduled,
		CurrentNumberScheduled: ds.Status.CurrentNumberScheduled,
		NumberReady:            ds.Status.NumberReady,
		UpdatedNumberScheduled: ds.Status.UpdatedNumberScheduled,
		NumberAvailable:        ds.Status.NumberAvailable,
		NumberUnavailable:      ds.Status.NumberUnavailable,
		NumberMisscheduled:     ds.Status.NumberMisscheduled,
		Labels:                 ds.Labels,
		Annotations:            ds.Annotations,
		CreationTimestamp:      &ds.CreationTimestamp.Time,
		CollisionCount:         ds.Status.CollisionCount,
		Status: models.DaemonSetStatus{
			ObservedGeneration: ds.Status.ObservedGeneration,
		},
		Conditions: conditions,
	}
}