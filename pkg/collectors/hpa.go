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

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// HPACollector collects metrics from Kubernetes horizontal pod autoscalers.
type HPACollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewHPACollector creates a new HPACollector.
func NewHPACollector(clientset *kubernetes.Clientset, cfg *config.Config) *HPACollector {
	collector := &HPACollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("HPACollector created successfully")
	return collector
}

// CollectMetrics collects metrics from Kubernetes horizontal pod autoscalers.
func (hc *HPACollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := hc.CollectHPAMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected HPA metrics")
	return metrics, nil
}

// CollectHPAMetrics collects metrics from Kubernetes horizontal pod autoscalers.
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
	metrics := models.HPAMetrics{
		Name:      hpa.Name,
		Namespace: hpa.Namespace,
		ScaleTargetRef: models.ScaleTargetRef{
			Kind:       hpa.Spec.ScaleTargetRef.Kind,
			Name:       hpa.Spec.ScaleTargetRef.Name,
			APIVersion: hpa.Spec.ScaleTargetRef.APIVersion,
		},
		MinReplicas:                     hpa.Spec.MinReplicas,
		MaxReplicas:                     hpa.Spec.MaxReplicas,
		CurrentReplicas:                 hpa.Status.CurrentReplicas,
		DesiredReplicas:                 hpa.Status.DesiredReplicas,
		CurrentCPUUtilizationPercentage: hpa.Status.CurrentCPUUtilizationPercentage,
		TargetCPUUtilizationPercentage:  hpa.Spec.TargetCPUUtilizationPercentage,
		LastScaleTime: func() *time.Time {
			if hpa.Status.LastScaleTime != nil {
				t := hpa.Status.LastScaleTime.Time
				return &t
			}
			return nil
		}(),
		ObservedGeneration: hpa.Status.ObservedGeneration,
		Labels:             hpa.Labels,
		Annotations:        hpa.Annotations,
		Status: models.HPAStatus{
			CurrentReplicas:                 hpa.Status.CurrentReplicas,
			DesiredReplicas:                 hpa.Status.DesiredReplicas,
			CurrentCPUUtilizationPercentage: hpa.Status.CurrentCPUUtilizationPercentage,
			LastScaleTime: func() *time.Time {
				if hpa.Status.LastScaleTime != nil {
					t := hpa.Status.LastScaleTime.Time
					return &t
				}
				return nil
			}(),
		},
	}

	logrus.Debugf("Parsed HPA metrics for HPA %s/%s", hpa.Namespace, hpa.Name)
	return metrics
}
