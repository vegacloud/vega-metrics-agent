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

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"runtime/debug"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// NamespaceCollector collects metrics from Kubernetes namespaces.
type NamespaceCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewNamespaceCollector creates a new NamespaceCollector.
func NewNamespaceCollector(clientset *kubernetes.Clientset, cfg *config.Config) *NamespaceCollector {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NewNamespaceCollector")
		}
	}()

	logrus.Debug("Creating new NamespaceCollector")
	return &NamespaceCollector{
		clientset: clientset,
		config:    cfg,
	}
}

// CollectMetrics collects metrics from Kubernetes namespaces.
func (nc *NamespaceCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NamespaceCollector.CollectMetrics")
		}
	}()

	logrus.Debug("Collecting namespace metrics")
	metrics, err := nc.CollectNamespaceMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to collect namespace metrics")
		return []models.NamespaceMetrics{}, nil
	}
	return metrics, nil
}

// CollectNamespaceMetrics collects metrics from Kubernetes namespaces.
func (nc *NamespaceCollector) CollectNamespaceMetrics(ctx context.Context) ([]models.NamespaceMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in CollectNamespaceMetrics")
		}
	}()

	namespaces, err := nc.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list namespaces")
		return []models.NamespaceMetrics{}, nil
	}

	namespaceMetrics := make([]models.NamespaceMetrics, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		logrus.WithField("namespace", ns.Name).Debug("Processing namespace")

		metrics, err := nc.collectSingleNamespaceMetrics(ctx, ns)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"namespace": ns.Name,
				"error":     err,
			}).Error("Failed to collect metrics for namespace")
			continue
		}
		namespaceMetrics = append(namespaceMetrics, metrics)
	}

	logrus.WithField("count", len(namespaceMetrics)).Debug("Completed namespace metrics collection")
	return namespaceMetrics, nil
}

// collectSingleNamespaceMetrics collects metrics from a single Kubernetes namespace.
func (nc *NamespaceCollector) collectSingleNamespaceMetrics(ctx context.Context, ns v1.Namespace) (models.NamespaceMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"namespace":  ns.Name,
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in collectSingleNamespaceMetrics")
		}
	}()

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}

	metrics := models.NamespaceMetrics{
		Name:              ns.Name,
		Status:            string(ns.Status.Phase),
		Phase:             string(ns.Status.Phase),
		CreationTimestamp: ns.CreationTimestamp.Time,
		DeletionTimestamp: nil,
		Finalizers:        ns.Finalizers,
		Labels:            ns.Labels,
		Annotations:       ns.Annotations,
	}

	// Set deletion timestamp if exists
	if ns.DeletionTimestamp != nil {
		deletionTime := ns.DeletionTimestamp.Time
		metrics.DeletionTimestamp = &deletionTime
	}

	// Convert conditions safely
	metrics.Conditions = make([]models.NamespaceCondition, 0, len(ns.Status.Conditions))
	for _, condition := range ns.Status.Conditions {
		if condition.LastTransitionTime.IsZero() {
			logrus.WithFields(logrus.Fields{
				"namespace": ns.Name,
				"condition": condition.Type,
			}).Debug("Skipping condition with zero transition time")
			continue
		}
		metrics.Conditions = append(metrics.Conditions, models.NamespaceCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: &condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	// Collect ResourceQuotas with error handling
	quotas, err := nc.clientset.CoreV1().ResourceQuotas(ns.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"namespace": ns.Name,
			"error":     err,
		}).Error("Failed to list resource quotas")
	} else {
		for _, quota := range quotas.Items {
			metrics.ResourceQuotas = append(metrics.ResourceQuotas, nc.parseResourceQuota(quota))
		}
	}

	// Collect LimitRanges with error handling
	limitRanges, err := nc.clientset.CoreV1().LimitRanges(ns.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"namespace": ns.Name,
			"error":     err,
		}).Error("Failed to list limit ranges")
	} else {
		for _, lr := range limitRanges.Items {
			metrics.LimitRanges = append(metrics.LimitRanges, nc.parseLimitRange(lr))
		}
	}

	// Collect usage metrics with error handling
	usage, err := nc.collectNamespaceUsage(ctx, ns.Name)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"namespace": ns.Name,
			"error":     err,
		}).Error("Failed to collect usage metrics")
	} else {
		metrics.Usage = usage
	}

	logrus.WithFields(logrus.Fields{
		"namespace":       ns.Name,
		"quotasCount":     len(metrics.ResourceQuotas),
		"limitsCount":     len(metrics.LimitRanges),
		"conditionsCount": len(metrics.Conditions),
	}).Debug("Collected namespace metrics")

	return metrics, nil
}

// parseResourceQuota parses metrics from a Kubernetes resource quota.
func (nc *NamespaceCollector) parseResourceQuota(quota v1.ResourceQuota) models.ResourceQuotaMetrics {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"quota":      quota.Name,
				"namespace":  quota.Namespace,
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in parseResourceQuota")
		}
	}()

	metrics := models.ResourceQuotaMetrics{
		Name: quota.Name,
	}

	// Parse basic resources
	for resourceName, hard := range quota.Status.Hard {
		used := quota.Status.Used[resourceName]
		metrics.Resources = append(metrics.Resources, models.ResourceMetric{
			ResourceName: string(resourceName),
			Hard:         hard.String(),
			Used:         used.String(),
		})
	}

	// Parse quota scopes
	if len(quota.Spec.Scopes) > 0 {
		for _, scope := range quota.Spec.Scopes {
			scopeMetric := models.QuotaScopeMetrics{
				ScopeName: string(scope),
			}
			if quota.Spec.ScopeSelector != nil && quota.Spec.ScopeSelector.MatchExpressions != nil {
				for _, expr := range quota.Spec.ScopeSelector.MatchExpressions {
					scopeMetric.MatchScopes = append(scopeMetric.MatchScopes, string(expr.Operator))
				}
			}
			metrics.Scopes = append(metrics.Scopes, scopeMetric)
		}
	}

	// Parse priority class quotas
	for resourceName, hard := range quota.Status.Hard {
		if isPriorityClassResource(string(resourceName)) {
			priorityClass := extractPriorityClass(string(resourceName))
			metrics.PriorityQuotas = append(metrics.PriorityQuotas, models.PriorityClassQuotaMetrics{
				PriorityClass: priorityClass,
				Hard:          map[string]string{string(resourceName): hard.ToUnstructured().(string)},
				Used:          map[string]string{string(resourceName): quota.Status.Used[resourceName].ToUnstructured().(string)},
			})
		}
	}

	return metrics
}

// Helper functions
func isPriorityClassResource(resource string) bool {
	return strings.HasPrefix(resource, "count/pods.") && strings.Contains(resource, "priorityclass")
}

func extractPriorityClass(resource string) string {
	parts := strings.Split(resource, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func (nc *NamespaceCollector) parseLimitRange(lr v1.LimitRange) models.LimitRangeMetrics {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"limitRange": lr.Name,
				"namespace":  lr.Namespace,
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in parseLimitRange")
		}
	}()

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

func (nc *NamespaceCollector) collectNamespaceUsage(ctx context.Context, namespace string) (models.ResourceUsage, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"namespace":  namespace,
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in collectNamespaceUsage")
		}
	}()

	usage := models.ResourceUsage{}

	pods, err := nc.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"namespace": namespace,
			"error":     err,
		}).Error("Failed to list pods")
		return usage, nil
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			usage.CPU += container.Resources.Requests.Cpu().MilliValue()
			usage.Memory += container.Resources.Requests.Memory().Value()
		}
	}

	pvcs, err := nc.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"namespace": namespace,
			"error":     err,
		}).Error("Failed to list PVCs")
		return usage, nil
	}

	for _, pvc := range pvcs.Items {
		if pvc.Spec.Resources.Requests != nil {
			if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
				usage.Storage += storage.Value()
			}
		}
	}

	usage.Pods = int64(len(pods.Items))

	logrus.WithFields(logrus.Fields{
		"namespace": namespace,
		"cpu":       usage.CPU,
		"memory":    usage.Memory,
		"storage":   usage.Storage,
		"pods":      usage.Pods,
	}).Debug("Collected namespace usage metrics")

	return usage, nil
}
