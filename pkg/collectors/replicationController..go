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
	// logrus.Debug("Starting ReplicationControllerCollector")
	// if token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
	// 	clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport = &http.Transport{
	// 		TLSClientConfig: &tls.Config{
	// 			InsecureSkipVerify: cfg.VegaInsecure,
	// 		},
	// 	}
	// 	clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport = transport.NewBearerAuthRoundTripper(
	// 		string(token),
	// 		clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport,
	// 	)
	// }
	logrus.Debug("ReplicationControllerCollector created successfully")
	return &ReplicationControllerCollector{
		clientset: clientset,
		config:    cfg,
	}
}

// CollectMetrics collects metrics from Kubernetes replication controllers.
func (rcc *ReplicationControllerCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := rcc.CollectReplicationControllerMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected replication controller metrics")
	return metrics, nil
}

// CollectReplicationControllerMetrics collects metrics from Kubernetes replication controllers.
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
	if rc.Annotations == nil {
		rc.Annotations = make(map[string]string)
	}

	conditions := make([]models.RCCondition, 0, len(rc.Status.Conditions))
	for _, condition := range rc.Status.Conditions {
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

	logrus.Debugf("Parsed replication controller metrics for %s/%s", rc.Namespace, rc.Name)
	return metrics
}
