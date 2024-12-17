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
	"runtime/debug"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// ReplicationControllerCollector collects metrics from Kubernetes replication controllers.
type ReplicationControllerCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewReplicationControllerCollector creates a new ReplicationControllerCollector.
func NewReplicationControllerCollector(clientset *kubernetes.Clientset,
	cfg *config.Config) *ReplicationControllerCollector {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NewReplicationControllerCollector")
		}
	}()

	logrus.Debug("Creating new ReplicationControllerCollector")
	collector := &ReplicationControllerCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("ReplicationControllerCollector created successfully")
	return collector
}

// CollectMetrics collects metrics from Kubernetes replication controllers.
func (rcc *ReplicationControllerCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in ReplicationControllerCollector.CollectMetrics")
		}
	}()

	metrics, err := rcc.CollectReplicationControllerMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to collect replication controller metrics")
		return []models.ReplicationControllerMetrics{}, nil
	}

	logrus.WithField("count", len(metrics)).Debug("Successfully collected replication controller metrics")
	return metrics, nil
}

// CollectReplicationControllerMetrics collects metrics from Kubernetes replication controllers.
func (rcc *ReplicationControllerCollector) CollectReplicationControllerMetrics(
	ctx context.Context) ([]models.ReplicationControllerMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in CollectReplicationControllerMetrics")
		}
	}()

	rcs, err := rcc.clientset.CoreV1().ReplicationControllers("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list replication controllers")
		return []models.ReplicationControllerMetrics{}, nil
	}

	logrus.WithField("count", len(rcs.Items)).Debug("Successfully listed replication controllers")
	metrics := make([]models.ReplicationControllerMetrics, 0, len(rcs.Items))

	for _, rc := range rcs.Items {
		logrus.WithFields(logrus.Fields{
			"rc":        rc.Name,
			"namespace": rc.Namespace,
		}).Debug("Processing replication controller")

		metrics = append(metrics, rcc.parseReplicationControllerMetrics(rc))
	}

	return metrics, nil
}

func (rcc *ReplicationControllerCollector) parseReplicationControllerMetrics(
	rc v1.ReplicationController) models.ReplicationControllerMetrics {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"rc":         rc.Name,
				"namespace":  rc.Namespace,
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in parseReplicationControllerMetrics")
		}
	}()

	if rc.Labels == nil {
		rc.Labels = make(map[string]string)
	}
	if rc.Annotations == nil {
		rc.Annotations = make(map[string]string)
	}

	conditions := make([]models.RCCondition, 0, len(rc.Status.Conditions))
	for _, condition := range rc.Status.Conditions {
		if condition.LastTransitionTime.IsZero() {
			logrus.WithFields(logrus.Fields{
				"rc":        rc.Name,
				"namespace": rc.Namespace,
				"condition": condition.Type,
			}).Debug("Skipping condition with zero transition time")
			continue
		}

		conditions = append(conditions, models.RCCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: &condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	metrics := models.ReplicationControllerMetrics{
		Name:                 rc.Name,
		Namespace:            rc.Namespace,
		Replicas:             rc.Status.Replicas,
		ReadyReplicas:        rc.Status.ReadyReplicas,
		AvailableReplicas:    rc.Status.AvailableReplicas,
		Labels:               rc.Labels,
		ObservedGeneration:   rc.Status.ObservedGeneration,
		FullyLabeledReplicas: rc.Status.FullyLabeledReplicas,
		Conditions:           conditions,
		Annotations:          rc.Annotations,
		CreationTimestamp:    &rc.CreationTimestamp.Time,
	}

	logrus.WithFields(logrus.Fields{
		"rc":         rc.Name,
		"namespace":  rc.Namespace,
		"replicas":   metrics.Replicas,
		"ready":      metrics.ReadyReplicas,
		"available":  metrics.AvailableReplicas,
		"conditions": len(metrics.Conditions),
	}).Debug("Collected metrics for replication controller")

	return metrics
}
