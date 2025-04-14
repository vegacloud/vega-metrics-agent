// Package utils provides utility functions for the agent
// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package utils

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	metricsClientInstance *metricsv.Clientset
	metricsClientOnce     sync.Once
)

// MetricsClientConfig holds the Kubernetes metrics clientset
type MetricsClientConfig struct {
	MetricsClientset *metricsv.Clientset
}

// GetMetricsClient returns a metrics client for the metrics.k8s.io API
func GetMetricsClient(config *rest.Config) (*metricsv.Clientset, error) {
	var err error
	metricsClientOnce.Do(func() {
		metricsClientInstance, err = metricsv.NewForConfig(config)
		if err != nil {
			logrus.WithError(err).Error("Failed to create metrics client")
		}
		logrus.Debug("Metrics client created successfully")
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}

	return metricsClientInstance, nil
}

// GetNodeMetricsFromAPI fetches node metrics using the metrics.k8s.io API
func GetNodeMetricsFromAPI(ctx context.Context, clientset *kubernetes.Clientset, config *rest.Config, nodeName string) (*metricsapi.NodeMetrics, error) {
	metricsClient, err := GetMetricsClient(config)
	if err != nil {
		return nil, err
	}

	// If nodeName is empty, we want to get metrics for all nodes
	if nodeName == "" {
		logrus.Debug("No node name specified, returning nil")
		return nil, nil
	}

	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for node %s: %w", nodeName, err)
	}

	return nodeMetrics, nil
}

// GetAllNodeMetricsFromAPI fetches metrics for all nodes using the metrics.k8s.io API
func GetAllNodeMetricsFromAPI(ctx context.Context, clientset *kubernetes.Clientset, config *rest.Config) (*metricsapi.NodeMetricsList, error) {
	metricsClient, err := GetMetricsClient(config)
	if err != nil {
		return nil, err
	}

	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list node metrics: %w", err)
	}

	return nodeMetrics, nil
}

// GetPodMetricsFromAPI fetches pod metrics using the metrics.k8s.io API
func GetPodMetricsFromAPI(ctx context.Context, clientset *kubernetes.Clientset, config *rest.Config, namespace, podName string) (*metricsapi.PodMetrics, error) {
	metricsClient, err := GetMetricsClient(config)
	if err != nil {
		return nil, err
	}

	// If podName is empty, we want to get metrics for all pods in the namespace
	if podName == "" {
		logrus.Debug("No pod name specified, returning nil")
		return nil, nil
	}

	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for pod %s/%s: %w", namespace, podName, err)
	}

	return podMetrics, nil
}

// GetAllPodMetricsFromAPI fetches metrics for all pods using the metrics.k8s.io API
func GetAllPodMetricsFromAPI(ctx context.Context, clientset *kubernetes.Clientset, config *rest.Config, namespace string) (*metricsapi.PodMetricsList, error) {
	metricsClient, err := GetMetricsClient(config)
	if err != nil {
		return nil, err
	}

	// If namespace is empty, get metrics for pods in all namespaces
	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pod metrics for namespace %s: %w", namespace, err)
	}

	return podMetrics, nil
}
