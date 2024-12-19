// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/collector.go

// Package collectors hosts the collection functions
package collectors

import (
	"bytes"
	"context"
	"fmt"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
	"k8s.io/client-go/kubernetes"
)

// Collector defines the interface for all metric collectors
type Collector interface {
	// CollectMetrics collects metrics and returns them as an interface{},
	// which can be type-asserted to the specific metrics type for each collector
	CollectMetrics(ctx context.Context) (interface{}, error)
}

// FetchMetricsViaKubelet fetches and parses metrics from kubelet
func FetchMetricsViaKubelet(ctx context.Context, clientset *kubernetes.Clientset, nodeName, metricsPath string) (map[string]*dto.MetricFamily, error) {
	kubeletClient := clientset.CoreV1().RESTClient().Get().
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy").
		Suffix(metricsPath)

	rawMetrics, err := kubeletClient.Do(ctx).Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics: %w", err)
	}

	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(bytes.NewReader(rawMetrics))
}

// FetchRawStatsViaKubelet fetches raw stats from kubelet
func FetchRawStatsViaKubelet(ctx context.Context, clientset *kubernetes.Clientset, nodeName, statsPath string) ([]byte, error) {
	kubeletClient := clientset.CoreV1().RESTClient().Get().
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy").
		Suffix(statsPath)

	rawStats, err := kubeletClient.Do(ctx).Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	return rawStats, nil
}

// FetchRawMetricsFromKubelet fetches raw metrics from kubelet
func FetchRawMetricsFromKubelet(ctx context.Context, clientset *kubernetes.Clientset, nodeName, metricsPath string) ([]byte, error) {
	kubeletClient := clientset.CoreV1().RESTClient().Get().
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy").
		Suffix(metricsPath)

	return kubeletClient.Do(ctx).Raw()
}

// VerifyCollectorClient verifies that the collector is using the correct client
func VerifyCollectorClient(ctx context.Context, clientset *kubernetes.Clientset, namespace string, collectorName string) error {
	logger := logrus.WithField("collector", collectorName)

	if err := utils.VerifyClientIdentity(ctx, clientset, namespace); err != nil {
		logger.WithError(err).Error("Collector using incorrect client identity")
		return fmt.Errorf("%s collector: %w", collectorName, err)
	}

	logger.Debug("Verified collector client identity")
	return nil
}
