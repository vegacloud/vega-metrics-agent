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
	"runtime/debug"
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
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NewHPACollector")
		}
	}()

	logrus.Debug("Starting HPACollector")
	collector := &HPACollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("HPACollector created successfully")
	return collector
}

// CollectMetrics collects metrics from Kubernetes horizontal pod autoscalers.
func (hc *HPACollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in HPACollector.CollectMetrics")
		}
	}()

	metrics, err := hc.CollectHPAMetrics(ctx)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to collect HPA metrics")
		return []models.HPAMetrics{}, nil
	}

	logrus.WithField("count", len(metrics)).Debug("Successfully collected HPA metrics")
	return metrics, nil
}

// CollectHPAMetrics collects metrics from Kubernetes horizontal pod autoscalers.
func (hc *HPACollector) CollectHPAMetrics(ctx context.Context) ([]models.HPAMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in HPACollector.CollectHPAMetrics")
		}
	}()

	hpas, err := hc.clientset.AutoscalingV1().HorizontalPodAutoscalers("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to list HPAs")
		return []models.HPAMetrics{}, nil
	}

	metrics := make([]models.HPAMetrics, 0, len(hpas.Items))
	for _, hpa := range hpas.Items {
		logrus.WithFields(logrus.Fields{
			"hpa":       hpa.Name,
			"namespace": hpa.Namespace,
		}).Debug("Processing HPA")

		metric := hc.parseHPAMetrics(hpa)
		metrics = append(metrics, metric)
	}

	logrus.WithField("count", len(metrics)).Debug("Collected metrics for HPAs")
	return metrics, nil
}

func (hc *HPACollector) parseHPAMetrics(hpa autoscalingv1.HorizontalPodAutoscaler) models.HPAMetrics {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"hpa":        hpa.Name,
				"namespace":  hpa.Namespace,
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic while parsing HPA metrics")
		}
	}()

	// Initialize empty maps if nil
	if hpa.Labels == nil {
		hpa.Labels = make(map[string]string)
	}
	if hpa.Annotations == nil {
		hpa.Annotations = make(map[string]string)
	}

	// Safely handle ScaleTargetRef
	scaleTargetRef := models.ScaleTargetRef{
		Kind:       hpa.Spec.ScaleTargetRef.Kind,
		Name:       hpa.Spec.ScaleTargetRef.Name,
		APIVersion: hpa.Spec.ScaleTargetRef.APIVersion,
	}

	// Safely handle LastScaleTime
	var lastScaleTime *time.Time
	if hpa.Status.LastScaleTime != nil {
		t := hpa.Status.LastScaleTime.Time
		lastScaleTime = &t
	}

	metrics := models.HPAMetrics{
		Name:                            hpa.Name,
		Namespace:                       hpa.Namespace,
		ScaleTargetRef:                  scaleTargetRef,
		MinReplicas:                     hpa.Spec.MinReplicas,
		MaxReplicas:                     hpa.Spec.MaxReplicas,
		CurrentReplicas:                 hpa.Status.CurrentReplicas,
		DesiredReplicas:                 hpa.Status.DesiredReplicas,
		CurrentCPUUtilizationPercentage: hpa.Status.CurrentCPUUtilizationPercentage,
		TargetCPUUtilizationPercentage:  hpa.Spec.TargetCPUUtilizationPercentage,
		LastScaleTime:                   lastScaleTime,
		ObservedGeneration:              hpa.Status.ObservedGeneration,
		Labels:                          hpa.Labels,
		Annotations:                     hpa.Annotations,
		Status: models.HPAStatus{
			CurrentReplicas:                 hpa.Status.CurrentReplicas,
			DesiredReplicas:                 hpa.Status.DesiredReplicas,
			CurrentCPUUtilizationPercentage: hpa.Status.CurrentCPUUtilizationPercentage,
			LastScaleTime:                   lastScaleTime,
		},
	}

	logrus.WithFields(logrus.Fields{
		"hpa":             hpa.Name,
		"namespace":       hpa.Namespace,
		"currentReplicas": hpa.Status.CurrentReplicas,
		"desiredReplicas": hpa.Status.DesiredReplicas,
		"targetCPU":       hpa.Spec.TargetCPUUtilizationPercentage,
		"currentCPU":      hpa.Status.CurrentCPUUtilizationPercentage,
	}).Debug("Parsed HPA metrics")

	return metrics
}
