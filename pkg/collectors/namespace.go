// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/namespace.go
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

type NamespaceCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewNamespaceCollector(clientset *kubernetes.Clientset, cfg *config.Config) *NamespaceCollector {
	logrus.Debug("Creating new NamespaceCollector")
	return &NamespaceCollector{
		clientset: clientset,
		config:    cfg,
	}
}

func (nc *NamespaceCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	logrus.Debug("Collecting namespace metrics")
	return nc.CollectNamespaceMetrics(ctx)
}

func (nc *NamespaceCollector) CollectNamespaceMetrics(ctx context.Context) ([]models.NamespaceMetrics, error) {
	namespaces, err := nc.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	logrus.Debugf("Listed %d namespaces", len(namespaces.Items))

	namespaceMetrics := make([]models.NamespaceMetrics, 0, len(namespaces.Items))

	for _, ns := range namespaces.Items {
		metrics, err := nc.collectSingleNamespaceMetrics(ctx, ns)
		if err != nil {
			logrus.Warnf("Failed to collect metrics for namespace %s: %v", ns.Name, err)
			continue
		}
		namespaceMetrics = append(namespaceMetrics, metrics)
		logrus.Debugf("Collected metrics for namespace %s", ns.Name)
	}

	return namespaceMetrics, nil
}

func (nc *NamespaceCollector) collectSingleNamespaceMetrics(
	ctx context.Context,
	ns v1.Namespace,
) (models.NamespaceMetrics, error) {
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	metrics := models.NamespaceMetrics{
		Name:   ns.Name,
		Status: string(ns.Status.Phase),
		Labels: ns.Labels,
	}

	// Collect ResourceQuotas
	quotas, err := nc.clientset.CoreV1().ResourceQuotas(ns.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return metrics, fmt.Errorf("failed to list resource quotas: %w", err)
	}
	logrus.Debugf("Listed %d resource quotas for namespace %s", len(quotas.Items), ns.Name)

	for _, quota := range quotas.Items {
		metrics.ResourceQuotas = append(metrics.ResourceQuotas, nc.parseResourceQuota(quota))
	}

	// Collect LimitRanges
	limitRanges, err := nc.clientset.CoreV1().LimitRanges(ns.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return metrics, fmt.Errorf("failed to list limit ranges: %w", err)
	}
	logrus.Debugf("Listed %d limit ranges for namespace %s", len(limitRanges.Items), ns.Name)

	for _, lr := range limitRanges.Items {
		metrics.LimitRanges = append(metrics.LimitRanges, nc.parseLimitRange(lr))
	}

	// Collect usage metrics
	metrics.Usage, err = nc.collectNamespaceUsage(ctx, ns.Name)
	if err != nil {
		logrus.Warnf("Failed to collect usage metrics for namespace %s: %v", ns.Name, err)
	}

	return metrics, nil
}

func (nc *NamespaceCollector) parseResourceQuota(quota v1.ResourceQuota) models.ResourceQuotaMetrics {
	metrics := models.ResourceQuotaMetrics{
		Name: quota.Name,
	}

	for resourceName, hard := range quota.Status.Hard {
		used := quota.Status.Used[resourceName]
		metrics.Resources = append(metrics.Resources, models.ResourceMetric{
			ResourceName: string(resourceName),
			Hard:         hard.String(),
			Used:         used.String(),
		})
	}

	logrus.Debugf("Parsed resource quota %s", quota.Name)
	return metrics
}

func (nc *NamespaceCollector) parseLimitRange(lr v1.LimitRange) models.LimitRangeMetrics {
	metrics := models.LimitRangeMetrics{
		Name: lr.Name,
	}

	for _, item := range lr.Spec.Limits {
		limit := models.LimitRangeItem{
			Type: string(item.Type),
		}

		if item.Max != nil {
			limit.Max = make(map[string]string)
			for k, v := range item.Max {
				limit.Max[string(k)] = v.String()
			}
		}

		if item.Min != nil {
			limit.Min = make(map[string]string)
			for k, v := range item.Min {
				limit.Min[string(k)] = v.String()
			}
		}

		if item.Default != nil {
			limit.Default = make(map[string]string)
			for k, v := range item.Default {
				limit.Default[string(k)] = v.String()
			}
		}

		if item.DefaultRequest != nil {
			limit.DefaultRequest = make(map[string]string)
			for k, v := range item.DefaultRequest {
				limit.DefaultRequest[string(k)] = v.String()
			}
		}

		metrics.Limits = append(metrics.Limits, limit)
	}

	logrus.Debugf("Parsed limit range %s", lr.Name)
	return metrics
}

func (nc *NamespaceCollector) collectNamespaceUsage(
	ctx context.Context,
	namespace string,
) (models.ResourceUsage, error) {
	usage := models.ResourceUsage{}

	pods, err := nc.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return usage, fmt.Errorf("failed to list pods: %w", err)
	}
	logrus.Debugf("Listed %d pods for namespace %s", len(pods.Items), namespace)

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			usage.CPU += container.Resources.Requests.Cpu().MilliValue()
			usage.Memory += container.Resources.Requests.Memory().Value()
		}
	}

	pvcs, err := nc.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return usage, fmt.Errorf("failed to list PVCs: %w", err)
	}
	logrus.Debugf("Listed %d PVCs for namespace %s", len(pvcs.Items), namespace)

	for _, pvc := range pvcs.Items {
		if pvc.Spec.Resources.Requests != nil {
			if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
				usage.Storage += storage.Value()
			}
		}
	}

	usage.Pods = int64(len(pods.Items))

	logrus.Debugf("Collected usage metrics for namespace %s", namespace)
	return usage, nil
}
