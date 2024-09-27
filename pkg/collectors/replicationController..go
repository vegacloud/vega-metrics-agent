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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

type ReplicationControllerCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewReplicationControllerCollector(clientset *kubernetes.Clientset,
	cfg *config.Config) *ReplicationControllerCollector {
	collector := &ReplicationControllerCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("ReplicationControllerCollector created successfully")
	return collector
}

func (rcc *ReplicationControllerCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := rcc.CollectReplicationControllerMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected replication controller metrics")
	return metrics, nil
}

func (rcc *ReplicationControllerCollector) CollectReplicationControllerMetrics(
	ctx context.Context) ([]models.ReplicationControllerMetrics, error) {
	rcs, err := rcc.clientset.CoreV1().ReplicationControllers("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list replication controllers: %w", err)
	}
	logrus.Debugf("Successfully listed %d replication controllers", len(rcs.Items))

	metrics := make([]models.ReplicationControllerMetrics, 0, len(rcs.Items))
	for _, rc := range rcs.Items {
		metrics = append(metrics, rcc.parseReplicationControllerMetrics(rc))
	}

	logrus.Debugf("Collected metrics for %d replication controllers", len(metrics))
	return metrics, nil
}

func (rcc *ReplicationControllerCollector) parseReplicationControllerMetrics(
	rc v1.ReplicationController) models.ReplicationControllerMetrics {
	if rc.Labels == nil {
		rc.Labels = make(map[string]string)
	}
	metrics := models.ReplicationControllerMetrics{
		Name:              rc.Name,
		Namespace:         rc.Namespace,
		Replicas:          rc.Status.Replicas,
		ReadyReplicas:     rc.Status.ReadyReplicas,
		AvailableReplicas: rc.Status.AvailableReplicas,
		Labels:            rc.Labels,
	}

	logrus.Debugf("Parsed replication controller metrics for %s/%s", rc.Namespace, rc.Name)
	return metrics
}
