// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/networking.go

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// NetworkingCollector collects metrics from Kubernetes networking resources.
type NetworkingCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewNetworkingCollector creates a new NetworkingCollector.
func NewNetworkingCollector(clientset *kubernetes.Clientset, cfg *config.Config) *NetworkingCollector {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NewNetworkingCollector")
		}
	}()

	logrus.Debug("Starting NetworkingCollector")
	collector := &NetworkingCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("NetworkingCollector created successfully")
	return collector
}

// CollectMetrics collects metrics from Kubernetes networking resources.
func (nc *NetworkingCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NetworkingCollector.CollectMetrics")
		}
	}()

	metrics, err := nc.CollectNetworkingMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to collect networking metrics")
		return &models.NetworkingMetrics{}, nil
	}
	return metrics, nil
}

// CollectNetworkingMetrics collects metrics from Kubernetes networking resources.
func (nc *NetworkingCollector) CollectNetworkingMetrics(ctx context.Context) (*models.NetworkingMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in CollectNetworkingMetrics")
		}
	}()

	metrics := &models.NetworkingMetrics{}

	services, err := nc.collectServiceMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to collect service metrics")
	} else {
		metrics.Services = services
	}

	ingresses, err := nc.collectIngressMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to collect ingress metrics")
	} else {
		metrics.Ingresses = ingresses
	}

	networkPolicies, err := nc.collectNetworkPolicyMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to collect network policy metrics")
	} else {
		metrics.NetworkPolicies = networkPolicies
	}

	return metrics, nil
}

func (nc *NetworkingCollector) collectServiceMetrics(ctx context.Context) ([]models.ServiceMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in collectServiceMetrics")
		}
	}()

	services, err := nc.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list services")
		return []models.ServiceMetrics{}, nil
	}

	metrics := make([]models.ServiceMetrics, 0, len(services.Items))
	for _, svc := range services.Items {
		logrus.WithFields(logrus.Fields{
			"service":   svc.Name,
			"namespace": svc.Namespace,
		}).Debug("Processing service")

		metrics = append(metrics, nc.parseServiceMetrics(svc))
	}

	logrus.WithField("count", len(metrics)).Debug("Collected service metrics")
	return metrics, nil
}

func (nc *NetworkingCollector) parseServiceMetrics(svc corev1.Service) models.ServiceMetrics {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"service":    svc.Name,
				"namespace":  svc.Namespace,
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in parseServiceMetrics")
		}
	}()

	if svc.Labels == nil {
		svc.Labels = make(map[string]string)
	}
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}

	metrics := models.ServiceMetrics{
		Name:                  svc.Name,
		Namespace:             svc.Namespace,
		Type:                  string(svc.Spec.Type),
		ClusterIP:             svc.Spec.ClusterIP,
		ExternalIP:            svc.Spec.ExternalIPs,
		Labels:                svc.Labels,
		Annotations:           svc.Annotations,
		SessionAffinity:       string(svc.Spec.SessionAffinity),
		ExternalTrafficPolicy: string(svc.Spec.ExternalTrafficPolicy),
		HealthCheckNodePort:   svc.Spec.HealthCheckNodePort,
		IPFamilies:            make([]string, 0),
	}

	// Safely handle IPFamilyPolicy
	if svc.Spec.IPFamilyPolicy != nil {
		metrics.IPFamilyPolicy = string(*svc.Spec.IPFamilyPolicy)
	}

	// Safely handle IPFamilies
	for _, family := range svc.Spec.IPFamilies {
		metrics.IPFamilies = append(metrics.IPFamilies, string(family))
	}

	// Safely handle LoadBalancerIP
	if svc.Spec.LoadBalancerIP != "" {
		metrics.LoadBalancerIP = svc.Spec.LoadBalancerIP
	}

	// Parse status safely
	metrics.Status.LoadBalancer = models.LoadBalancerStatus{
		Ingress: make([]models.LoadBalancerIngress, len(svc.Status.LoadBalancer.Ingress)),
	}

	for i, ing := range svc.Status.LoadBalancer.Ingress {
		metrics.Status.LoadBalancer.Ingress[i] = models.LoadBalancerIngress{
			IP:       ing.IP,
			Hostname: ing.Hostname,
		}
	}

	// Parse conditions safely
	for _, cond := range svc.Status.Conditions {
		if cond.LastTransitionTime.IsZero() {
			logrus.WithFields(logrus.Fields{
				"service":   svc.Name,
				"namespace": svc.Namespace,
				"condition": cond.Type,
			}).Debug("Skipping condition with zero transition time")
			continue
		}

		metrics.Status.Conditions = append(metrics.Status.Conditions, models.ServiceCondition{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			LastTransitionTime: &cond.LastTransitionTime.Time,
			Reason:             cond.Reason,
			Message:            cond.Message,
		})
	}

	// Parse ports safely
	for _, port := range svc.Spec.Ports {
		metrics.Ports = append(metrics.Ports, models.ServicePort{
			Name:       port.Name,
			Protocol:   string(port.Protocol),
			Port:       port.Port,
			TargetPort: port.TargetPort.String(),
			NodePort:   port.NodePort,
		})
	}

	metrics.Selector = svc.Spec.Selector

	logrus.WithFields(logrus.Fields{
		"service":   svc.Name,
		"namespace": svc.Namespace,
		"type":      metrics.Type,
		"ports":     len(metrics.Ports),
	}).Debug("Parsed service metrics")

	return metrics
}

func (nc *NetworkingCollector) collectIngressMetrics(ctx context.Context) ([]models.IngressMetrics, error) {
	ingresses, err := nc.clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}
	logrus.Debugf("Successfully listed %d ingresses", len(ingresses.Items))

	metrics := make([]models.IngressMetrics, 0, len(ingresses.Items))

	for _, ing := range ingresses.Items {
		metrics = append(metrics, nc.parseIngressMetrics(ing))
	}

	return metrics, nil
}

func (nc *NetworkingCollector) parseIngressMetrics(ing networkingv1.Ingress) models.IngressMetrics {
	metrics := models.IngressMetrics{
		Name:      ing.Name,
		Namespace: ing.Namespace,
		ClassName: func() string {
			if ing.Spec.IngressClassName != nil {
				return *ing.Spec.IngressClassName
			}
			return ""
		}(),
		Labels:            ing.Labels,
		Annotations:       ing.Annotations,
		CreationTimestamp: &ing.CreationTimestamp.Time,
	}

	// Parse status
	for _, ing := range ing.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			metrics.LoadBalancerIngress = append(metrics.LoadBalancerIngress, ing.IP)
		}
		if ing.Hostname != "" {
			metrics.LoadBalancerIngress = append(metrics.LoadBalancerIngress, ing.Hostname)
		}
	}

	for _, rule := range ing.Spec.Rules {
		ingressRule := models.IngressRule{
			Host: rule.Host,
		}

		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				ingressRule.Paths = append(ingressRule.Paths, models.IngressPath{
					Path:     path.Path,
					PathType: string(*path.PathType),
					Backend: models.IngressBackend{
						Service: models.IngressServiceBackend{
							Name: path.Backend.Service.Name,
							Port: path.Backend.Service.Port.Number,
						},
					},
				})
			}
		}

		metrics.Rules = append(metrics.Rules, ingressRule)
	}

	for _, tls := range ing.Spec.TLS {
		metrics.TLS = append(metrics.TLS, models.IngressTLS{
			Hosts:      tls.Hosts,
			SecretName: tls.SecretName,
		})
	}

	logrus.Debugf("Parsed ingress metrics for ingress %s/%s", ing.Namespace, ing.Name)
	return metrics
}

func (nc *NetworkingCollector) collectNetworkPolicyMetrics(ctx context.Context) ([]models.NetworkPolicyMetrics, error) {
	networkPolicies, err := nc.clientset.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list network policies: %w", err)
	}
	logrus.Debugf("Successfully listed %d network policies", len(networkPolicies.Items))

	metrics := make([]models.NetworkPolicyMetrics, 0, len(networkPolicies.Items))

	for _, policy := range networkPolicies.Items {
		metrics = append(metrics, nc.parseNetworkPolicyMetrics(policy))
	}

	return metrics, nil
}

func (nc *NetworkingCollector) parseNetworkPolicyMetrics(policy networkingv1.NetworkPolicy) models.NetworkPolicyMetrics {
	metrics := models.NetworkPolicyMetrics{
		Name:        policy.Name,
		Namespace:   policy.Namespace,
		Labels:      policy.Labels,
		Annotations: policy.Annotations,
		PodSelector: policy.Spec.PodSelector.MatchLabels,
	}

	// Parse policy types
	for _, pType := range policy.Spec.PolicyTypes {
		metrics.PolicyTypes = append(metrics.PolicyTypes, string(pType))
	}

	// Parse ingress rules
	for _, rule := range policy.Spec.Ingress {
		ingressRule := models.NetworkPolicyIngressRule{}

		// Parse ports
		for _, port := range rule.Ports {
			ingressRule.Ports = append(ingressRule.Ports, models.NetworkPolicyPort{
				Protocol: string(*port.Protocol),
				Port:     port.Port.IntVal,
			})
		}

		// Parse from rules
		for _, from := range rule.From {
			peer := models.NetworkPolicyPeer{}
			if from.PodSelector != nil {
				peer.PodSelector = from.PodSelector.MatchLabels
			}
			if from.NamespaceSelector != nil {
				peer.NamespaceSelector = from.NamespaceSelector.MatchLabels
			}
			if from.IPBlock != nil {
				peer.IPBlock = &models.IPBlock{
					CIDR:   from.IPBlock.CIDR,
					Except: from.IPBlock.Except,
				}
			}
			ingressRule.From = append(ingressRule.From, peer)
		}

		metrics.Ingress = append(metrics.Ingress, ingressRule)
	}

	// Parse egress rules
	for _, rule := range policy.Spec.Egress {
		egressRule := models.NetworkPolicyEgressRule{}

		// Parse ports
		for _, port := range rule.Ports {
			egressRule.Ports = append(egressRule.Ports, models.NetworkPolicyPort{
				Protocol: string(*port.Protocol),
				Port:     port.Port.IntVal,
			})
		}

		// Parse to rules
		for _, to := range rule.To {
			peer := models.NetworkPolicyPeer{}
			if to.PodSelector != nil {
				peer.PodSelector = to.PodSelector.MatchLabels
			}
			if to.NamespaceSelector != nil {
				peer.NamespaceSelector = to.NamespaceSelector.MatchLabels
			}
			if to.IPBlock != nil {
				peer.IPBlock = &models.IPBlock{
					CIDR:   to.IPBlock.CIDR,
					Except: to.IPBlock.Except,
				}
			}
			egressRule.To = append(egressRule.To, peer)
		}

		metrics.Egress = append(metrics.Egress, egressRule)
	}

	return metrics
}
