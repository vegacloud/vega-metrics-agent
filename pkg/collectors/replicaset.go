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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// ReplicaSetCollector collects metrics for ReplicaSets
type ReplicaSetCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewReplicaSetCollector creates a new ReplicaSetCollector instance
func NewReplicaSetCollector(clientset *kubernetes.Clientset, cfg *config.Config) *ReplicaSetCollector {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NewReplicaSetCollector")
		}
	}()

	logrus.Debug("Creating new ReplicaSetCollector")
	collector := &ReplicaSetCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("ReplicaSetCollector created successfully")
	return collector
}

// CollectMetrics collects metrics for all ReplicaSets in the cluster
func (rsc *ReplicaSetCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in ReplicaSetCollector.CollectMetrics")
		}
	}()

	metrics, err := rsc.CollectReplicaSetMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to collect replicaset metrics")
		return []models.ReplicaSetMetrics{}, nil
	}

	logrus.WithField("count", len(metrics)).Debug("Successfully collected replicaset metrics")
	return metrics, nil
}

// CollectReplicaSetMetrics collects metrics for all ReplicaSets in the cluster
func (rsc *ReplicaSetCollector) CollectReplicaSetMetrics(ctx context.Context) ([]models.ReplicaSetMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in CollectReplicaSetMetrics")
		}
	}()

	replicaSets, err := rsc.clientset.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list ReplicaSets")
		return []models.ReplicaSetMetrics{}, nil
	}

	logrus.WithField("count", len(replicaSets.Items)).Debug("Successfully listed ReplicaSets")
	metrics := make([]models.ReplicaSetMetrics, 0, len(replicaSets.Items))

	for _, rs := range replicaSets.Items {
		logrus.WithFields(logrus.Fields{
			"replicaset": rs.Name,
			"namespace":  rs.Namespace,
		}).Debug("Processing ReplicaSet")

		metrics = append(metrics, rsc.parseReplicaSetMetrics(rs))
	}

	return metrics, nil
}

// parseReplicaSetMetrics parses metrics for a single ReplicaSet
func (rsc *ReplicaSetCollector) parseReplicaSetMetrics(rs appsv1.ReplicaSet) models.ReplicaSetMetrics {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"replicaset": rs.Name,
				"namespace":  rs.Namespace,
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in parseReplicaSetMetrics")
		}
	}()

	if rs.Labels == nil {
		rs.Labels = make(map[string]string)
	}
	if rs.Annotations == nil {
		rs.Annotations = make(map[string]string)
	}

	conditions := make([]models.RSCondition, 0, len(rs.Status.Conditions))
	for _, condition := range rs.Status.Conditions {
		if condition.LastTransitionTime.IsZero() {
			logrus.WithFields(logrus.Fields{
				"replicaset": rs.Name,
				"namespace":  rs.Namespace,
				"condition":  condition.Type,
			}).Debug("Skipping condition with zero transition time")
			continue
		}

		conditions = append(conditions, models.RSCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: &condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	metrics := models.ReplicaSetMetrics{
		Name:                 rs.Name,
		Namespace:            rs.Namespace,
		Replicas:             *rs.Spec.Replicas,
		ReadyReplicas:        rs.Status.ReadyReplicas,
		AvailableReplicas:    rs.Status.AvailableReplicas,
		CurrentReplicas:      rs.Status.Replicas,
		FullyLabeledReplicas: rs.Status.FullyLabeledReplicas,
		ObservedGeneration:   rs.Status.ObservedGeneration,
		Conditions:           conditions,
		Labels:               rs.Labels,
		Annotations:          rs.Annotations,
		CreationTimestamp:    &rs.CreationTimestamp.Time,
	}

	logrus.WithFields(logrus.Fields{
		"replicaset": rs.Name,
		"namespace":  rs.Namespace,
		"replicas":   metrics.Replicas,
		"ready":      metrics.ReadyReplicas,
		"available":  metrics.AvailableReplicas,
		"conditions": len(metrics.Conditions),
	}).Debug("Collected metrics for ReplicaSet")

	return metrics
}
