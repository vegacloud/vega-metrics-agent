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
func (dsc *DaemonSetCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := dsc.CollectDaemonSetMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected daemon set metrics")
	return metrics, nil
}

// CollectDaemonSetMetrics collects metrics for all daemon sets in the cluster
func (dsc *DaemonSetCollector) CollectDaemonSetMetrics(ctx context.Context) ([]models.DaemonSetMetrics, error) {
	daemonSets, err := dsc.clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemon sets: %w", err)
	}
	logrus.Debugf("Successfully listed %d daemon sets", len(daemonSets.Items))

	metrics := make([]models.DaemonSetMetrics, 0, len(daemonSets.Items))

	for _, ds := range daemonSets.Items {
		metrics = append(metrics, dsc.parseDaemonSetMetrics(ds))
	}

	logrus.Debugf("Collected metrics for %d daemon sets", len(metrics))
	return metrics, nil
}

// parseDaemonSetMetrics parses metrics for a single daemon set
func (dsc *DaemonSetCollector) parseDaemonSetMetrics(ds appsv1.DaemonSet) models.DaemonSetMetrics {
	if ds.Labels == nil {
		ds.Labels = make(map[string]string)
	}
	return models.DaemonSetMetrics{
		Name:                   ds.Name,
		Namespace:              ds.Namespace,
		DesiredNumberScheduled: ds.Status.DesiredNumberScheduled,
		CurrentNumberScheduled: ds.Status.CurrentNumberScheduled,
		NumberReady:            ds.Status.NumberReady,
		UpdatedNumberScheduled: ds.Status.UpdatedNumberScheduled,
		NumberAvailable:        ds.Status.NumberAvailable,
		Labels:                 ds.Labels,
	}
}
