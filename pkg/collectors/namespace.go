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
	"fmt"
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
	// logrus.Debug("Starting NamespaceCollector")
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
	logrus.Debug("Creating new NamespaceCollector")
	return &NamespaceCollector{
		clientset: clientset,
		config:    cfg,
	}
}

// CollectMetrics collects metrics from Kubernetes namespaces.
func (nc *NamespaceCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	logrus.Debug("Collecting namespace metrics")
	return nc.CollectNamespaceMetrics(ctx)
}

// CollectNamespaceMetrics collects metrics from Kubernetes namespaces.
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

// collectSingleNamespaceMetrics collects metrics from a single Kubernetes namespace.
func (nc *NamespaceCollector) collectSingleNamespaceMetrics(
	ctx context.Context,
	ns v1.Namespace,
) (models.NamespaceMetrics, error) {
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
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

	// Convert conditions
	for _, condition := range ns.Status.Conditions {
		metrics.Conditions = append(metrics.Conditions, models.NamespaceCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: &condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
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

// parseResourceQuota parses metrics from a Kubernetes resource quota.
func (nc *NamespaceCollector) parseResourceQuota(quota v1.ResourceQuota) models.ResourceQuotaMetrics {
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
