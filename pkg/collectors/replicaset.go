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
	collector := &ReplicaSetCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("ReplicaSetCollector created successfully")
	return collector
}

// CollectMetrics collects metrics for all ReplicaSets in the cluster
func (rsc *ReplicaSetCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := rsc.CollectReplicaSetMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected ReplicaSet metrics")
	return metrics, nil
}

// CollectReplicaSetMetrics collects metrics for all ReplicaSets in the cluster
func (rsc *ReplicaSetCollector) CollectReplicaSetMetrics(ctx context.Context) ([]models.ReplicaSetMetrics, error) {
	replicaSets, err := rsc.clientset.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ReplicaSets: %w", err)
	}
	logrus.Debugf("Successfully listed %d ReplicaSets", len(replicaSets.Items))

	metrics := make([]models.ReplicaSetMetrics, 0, len(replicaSets.Items))

	for _, rs := range replicaSets.Items {
		metrics = append(metrics, rsc.parseReplicaSetMetrics(rs))
	}

	logrus.Debugf("Collected metrics for %d ReplicaSets", len(metrics))
	return metrics, nil
}

// parseReplicaSetMetrics parses metrics for a single ReplicaSet
func (rsc *ReplicaSetCollector) parseReplicaSetMetrics(rs appsv1.ReplicaSet) models.ReplicaSetMetrics {
	if rs.Labels == nil {
		rs.Labels = make(map[string]string)
	}
	return models.ReplicaSetMetrics{
		Name:              rs.Name,
		Namespace:         rs.Namespace,
		Replicas:          *rs.Spec.Replicas,
		ReadyReplicas:     rs.Status.ReadyReplicas,
		AvailableReplicas: rs.Status.AvailableReplicas,
		Labels:            rs.Labels,
	}
}
