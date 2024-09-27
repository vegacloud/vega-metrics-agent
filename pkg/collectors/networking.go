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
package collectors

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

type NetworkingCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewNetworkingCollector(clientset *kubernetes.Clientset, cfg *config.Config) *NetworkingCollector {
	collector := &NetworkingCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("NetworkingCollector created successfully")
	return collector
}

func (nc *NetworkingCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := nc.CollectNetworkingMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected networking metrics")
	return metrics, nil
}

func (nc *NetworkingCollector) CollectNetworkingMetrics(ctx context.Context) (*models.NetworkingMetrics, error) {
	metrics := &models.NetworkingMetrics{}

	var err error
	metrics.Services, err = nc.collectServiceMetrics(ctx)
	if err != nil {
		logrus.Warnf("Failed to collect service metrics: %v", err)
	} else {
		logrus.Debug("Successfully collected service metrics")
	}

	metrics.Ingresses, err = nc.collectIngressMetrics(ctx)
	if err != nil {
		logrus.Warnf("Failed to collect ingress metrics: %v", err)
	} else {
		logrus.Debug("Successfully collected ingress metrics")
	}

	return metrics, nil
}

func (nc *NetworkingCollector) collectServiceMetrics(ctx context.Context) ([]models.ServiceMetrics, error) {
	services, err := nc.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	logrus.Debugf("Successfully listed %d services", len(services.Items))

	metrics := make([]models.ServiceMetrics, 0, len(services.Items))

	for _, svc := range services.Items {
		metrics = append(metrics, nc.parseServiceMetrics(svc))
	}

	return metrics, nil
}

func (nc *NetworkingCollector) parseServiceMetrics(svc corev1.Service) models.ServiceMetrics {
	if svc.Labels == nil {
		svc.Labels = make(map[string]string)
	}
	metrics := models.ServiceMetrics{
		Name:       svc.Name,
		Namespace:  svc.Namespace,
		Type:       string(svc.Spec.Type),
		ClusterIP:  svc.Spec.ClusterIP,
		ExternalIP: svc.Spec.ExternalIPs,
		Labels:     svc.Labels,
	}

	if svc.Spec.LoadBalancerIP != "" {
		metrics.LoadBalancerIP = svc.Spec.LoadBalancerIP
	}

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			metrics.LoadBalancerIngress = append(metrics.LoadBalancerIngress, ingress.IP)
		}
		if ingress.Hostname != "" {
			metrics.LoadBalancerIngress = append(metrics.LoadBalancerIngress, ingress.Hostname)
		}
	}

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

	logrus.Debugf("Parsed service metrics for service %s/%s", svc.Namespace, svc.Name)
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
	if ing.Labels == nil {
		ing.Labels = make(map[string]string)
	}
	metrics := models.IngressMetrics{
		Name:      ing.Name,
		Namespace: ing.Namespace,
		ClassName: func() string {
			if ing.Spec.IngressClassName != nil {
				return *ing.Spec.IngressClassName
			}
			return ""
		}(),
		Labels: ing.Labels,
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

	for _, ing := range ing.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			metrics.LoadBalancerIngress = append(metrics.LoadBalancerIngress, ing.IP)
		}
		if ing.Hostname != "" {
			metrics.LoadBalancerIngress = append(metrics.LoadBalancerIngress, ing.Hostname)
		}
	}

	logrus.Debugf("Parsed ingress metrics for ingress %s/%s", ing.Namespace, ing.Name)
	return metrics
}
