// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/hpa.go
package collectors

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

type HPACollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewHPACollector(clientset *kubernetes.Clientset, cfg *config.Config) *HPACollector {
	collector := &HPACollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("HPACollector created successfully")
	return collector
}

func (hc *HPACollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := hc.CollectHPAMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected HPA metrics")
	return metrics, nil
}

func (hc *HPACollector) CollectHPAMetrics(ctx context.Context) ([]models.HPAMetrics, error) {
	hpas, err := hc.clientset.AutoscalingV1().HorizontalPodAutoscalers("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list HPAs: %w", err)
	}
	logrus.Debugf("Successfully listed %d HPAs", len(hpas.Items))

	metrics := make([]models.HPAMetrics, 0, len(hpas.Items))
	for _, hpa := range hpas.Items {
		metrics = append(metrics, hc.parseHPAMetrics(hpa))
	}

	logrus.Debugf("Collected metrics for %d HPAs", len(metrics))
	return metrics, nil
}

func (hc *HPACollector) parseHPAMetrics(hpa autoscalingv1.HorizontalPodAutoscaler) models.HPAMetrics {
	if hpa.Labels == nil {
		hpa.Labels = make(map[string]string)
	}
	metrics := models.HPAMetrics{
		Name:            hpa.Name,
		Namespace:       hpa.Namespace,
		CurrentReplicas: hpa.Status.CurrentReplicas,
		DesiredReplicas: hpa.Status.DesiredReplicas,
		Labels:          hpa.Labels,
		CurrentCPUUtilizationPercentage: func() *int32 {
			if hpa.Status.CurrentCPUUtilizationPercentage != nil {
				return hpa.Status.CurrentCPUUtilizationPercentage
			}
			return nil
		}(),
		TargetCPUUtilizationPercentage: func() *int32 {
			if hpa.Spec.TargetCPUUtilizationPercentage != nil {
				return hpa.Spec.TargetCPUUtilizationPercentage
			}
			return nil
		}(),
	}

	logrus.Debugf("Parsed HPA metrics for HPA %s/%s", hpa.Namespace, hpa.Name)
	return metrics
}
