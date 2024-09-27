// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

type NodeCollector struct {
	clientset        *kubernetes.Clientset
	metricsClientset *metricsclientset.Clientset
	config           *config.Config
	bearerToken      string
	httpClient       *http.Client
}

// NewNodeCollector initializes a new NodeCollector.
func NewNodeCollector(
	clientset *kubernetes.Clientset,
	metricsClientset *metricsclientset.Clientset,
	cfg *config.Config,
) (*NodeCollector, error) {
	nc := &NodeCollector{
		clientset:        clientset,
		metricsClientset: metricsClientset,
		config:           cfg,
	}

	// Get bearer token
	token, err := nc.getBearerToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get bearer token: %w", err)
	}
	nc.bearerToken = token
	logrus.Debug("Successfully retrieved bearer token")

	// Create HTTP client
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.VegaInsecure, //#nosec this is only off for local testing and will be true in prod.
		},
	}

	nc.httpClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
	logrus.Debug("HTTP client created successfully")

	return nc, nil
}

// CollectMetrics collects enhanced node metrics.
func (nc *NodeCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	return nc.collectEnhancedNodeMetrics(ctx)
}

// collectEnhancedNodeMetrics collects metrics for all nodes.
func (nc *NodeCollector) collectEnhancedNodeMetrics(ctx context.Context) ([]models.EnhancedNodeMetrics, error) {
	// Fetch node details
	nodes, err := nc.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	logrus.Debugf("Successfully listed %d nodes", len(nodes.Items))

	// Fetch node metrics from the Metrics Server
	nodeMetricsList, err := nc.metricsClientset.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch node metrics from metrics server: %w", err)
	}
	logrus.Debugf("Successfully fetched node metrics from metrics server")

	metricsMap := nc.mapNodeMetrics(nodeMetricsList)

	var (
		enhancedNodeMetrics []models.EnhancedNodeMetrics
		mu                  sync.Mutex
	)

	g, ctx := errgroup.WithContext(ctx)
	semaphore := make(chan struct{}, nc.config.VegaMaxConcurrency) // Configurable concurrency

	for _, node := range nodes.Items {
		node := node // capture variable
		g.Go(func() error {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return ctx.Err()
			}

			nodeMetric, found := metricsMap[node.Name]
			if !found {
				nodeMetric = metricsv1beta1.NodeMetrics{}
			}

			metrics, err := nc.collectSingleNodeMetrics(ctx, node, nodeMetric)
			if err != nil {
				logrus.Warnf("Failed to collect metrics for node %s: %v", node.Name, err)
				return nil
			}

			mu.Lock()
			enhancedNodeMetrics = append(enhancedNodeMetrics, metrics)
			mu.Unlock()
			logrus.Debugf("Successfully collected metrics for node %s", node.Name)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	logrus.Debug("Successfully collected enhanced node metrics for all nodes")
	return enhancedNodeMetrics, nil
}

// mapNodeMetrics creates a map of node metrics for easier lookup by node name.
func (nc *NodeCollector) mapNodeMetrics(
	nodeMetricsList *metricsv1beta1.NodeMetricsList,
) map[string]metricsv1beta1.NodeMetrics {
	metricsMap := make(map[string]metricsv1beta1.NodeMetrics)
	for _, nodeMetric := range nodeMetricsList.Items {
		metricsMap[nodeMetric.Name] = nodeMetric
	}
	logrus.Debug("Successfully mapped node metrics")
	return metricsMap
}

// collectSingleNodeMetrics collects metrics for a single node.
func (nc *NodeCollector) collectSingleNodeMetrics(
	ctx context.Context,
	node v1.Node,
	nodeMetrics metricsv1beta1.NodeMetrics,
) (models.EnhancedNodeMetrics, error) {
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	metrics := models.EnhancedNodeMetrics{
		Name:   node.Name,
		Labels: node.Labels,
		Capacity: models.ResourceMetrics{
			CPU:     node.Status.Capacity.Cpu().MilliValue(),
			Memory:  node.Status.Capacity.Memory().Value(),
			Pods:    node.Status.Capacity.Pods().Value(),
			Storage: node.Status.Capacity.StorageEphemeral().Value(),
		},
		Allocatable: models.ResourceMetrics{
			CPU:     node.Status.Allocatable.Cpu().MilliValue(),
			Memory:  node.Status.Allocatable.Memory().Value(),
			Pods:    node.Status.Allocatable.Pods().Value(),
			Storage: node.Status.Allocatable.StorageEphemeral().Value(),
		},
	}

	// Add node conditions
	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case v1.NodeReady:
			metrics.Conditions.Ready = condition.Status == v1.ConditionTrue
		case v1.NodeMemoryPressure:
			metrics.Conditions.MemoryPressure = condition.Status == v1.ConditionTrue
		case v1.NodeDiskPressure:
			metrics.Conditions.DiskPressure = condition.Status == v1.ConditionTrue
		case v1.NodePIDPressure:
			metrics.Conditions.PIDPressure = condition.Status == v1.ConditionTrue
		}
	}

	// Fetch resource usage from the Kubernetes Metrics API
	if nodeMetrics.Name != "" {
		metrics.Usage.CPU = nodeMetrics.Usage.Cpu().MilliValue()
		metrics.Usage.Memory = nodeMetrics.Usage.Memory().Value()
	}

	// Fetch additional usage metrics from cAdvisor
	cAdvisorUsage, err := nc.collectNodeUsageMetrics(ctx, node)
	if err != nil {
		return metrics, fmt.Errorf("failed to collect cAdvisor metrics for node %s: %w", node.Name, err)
	}

	// Combine the metrics from cAdvisor with those from Metrics API
	metrics.Usage.Storage = cAdvisorUsage.Storage

	logrus.Debugf("Successfully collected metrics for node %s", node.Name)
	return metrics, nil
}

// getBearerToken retrieves the token from a file or environment variable.
func (nc *NodeCollector) getBearerToken() (string, error) {
	// First, try to read from file
	if nc.config.VegaBearerTokenPath != "" {
		tokenBytes, err := os.ReadFile(nc.config.VegaBearerTokenPath)
		if err == nil {
			token := strings.TrimSpace(string(tokenBytes))
			logrus.Debug("Successfully read Service Account bearer token from file")
			return token, nil
		}
		logrus.Printf("Failed to read bearer token from file, defaulting to BEARER_TOKEN environment variable: %v", err)
	}

	// If file read failed or no file path was provided, try environment variable
	token := strings.TrimSpace(os.Getenv("BEARER_TOKEN"))
	if token != "" {
		logrus.Debug("Successfully read bearer token from environment variable")
		return token, nil
	}

	// If both file and environment variable failed, return an error
	return "", errors.New("bearer token not found in file or environment")
}

// collectNodeUsageMetrics collects cAdvisor metrics for a node.
func (nc *NodeCollector) collectNodeUsageMetrics(ctx context.Context, node v1.Node) (models.ResourceMetrics, error) {
	var nodeAddress string
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeInternalIP {
			nodeAddress = address.Address
			break
		}
	}
	if nodeAddress == "" {
		return models.ResourceMetrics{}, fmt.Errorf("no valid IP found for node %s", node.Name)
	}

	metricsURL := fmt.Sprintf("https://%s:%d/metrics/cadvisor", nodeAddress, 10250)

	req, err := http.NewRequestWithContext(ctx, "GET", metricsURL, nil)
	if err != nil {
		return models.ResourceMetrics{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Add the bearer token to the request header
	req.Header.Add("Authorization", "Bearer "+nc.bearerToken)

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		return models.ResourceMetrics{}, fmt.Errorf("failed to fetch cAdvisor metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.ResourceMetrics{}, fmt.Errorf("cAdvisor returned non-200 status: %d", resp.StatusCode)
	}

	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return models.ResourceMetrics{}, fmt.Errorf("failed to parse cAdvisor metrics: %w", err)
	}

	logrus.Debugf("Successfully fetched and parsed cAdvisor metrics for node %s", node.Name)
	return nc.extractNodeUsageMetrics(metricFamilies), nil
}

// extractNodeUsageMetrics extracts usage metrics from metric families.
func (nc *NodeCollector) extractNodeUsageMetrics(metricFamilies map[string]*dto.MetricFamily) models.ResourceMetrics {
	usage := models.ResourceMetrics{}

	if cpuMetric, ok := metricFamilies["node_cpu_seconds_total"]; ok {
		usage.CPU = nc.parseCPUUsage(cpuMetric)
	}

	if memoryMetric, ok := metricFamilies["node_memory_Active_bytes"]; ok {
		usage.Memory = nc.parseMemoryUsage(memoryMetric)
	}

	if storageMetric, ok := metricFamilies["node_filesystem_size_bytes"]; ok {
		usage.Storage = nc.parseStorageUsage(storageMetric)
	}

	logrus.Debug("Successfully extracted node usage metrics")
	return usage
}

// parseCPUUsage parses CPU usage from the metric family.
func (nc *NodeCollector) parseCPUUsage(family *dto.MetricFamily) int64 {
	var totalCPUUsage float64
	for _, metric := range family.Metric {
		mode := ""
		for _, label := range metric.Label {
			if label.GetName() == "mode" {
				mode = label.GetValue()
				break
			}
		}
		if mode == "user" || mode == "system" {
			if metric.Counter != nil && metric.Counter.Value != nil {
				totalCPUUsage += metric.Counter.GetValue()
			}
		}
	}
	logrus.Debug("Successfully parsed CPU usage metrics")
	return int64(totalCPUUsage * 1000) // Convert to millicores
}

// parseMemoryUsage parses memory usage from the metric family.
func (nc *NodeCollector) parseMemoryUsage(family *dto.MetricFamily) int64 {
	if len(family.Metric) > 0 && family.Metric[0].Gauge != nil && family.Metric[0].Gauge.Value != nil {
		logrus.Debug("Successfully parsed memory usage metrics")
		return int64(family.Metric[0].Gauge.GetValue())
	}
	return 0
}

// parseStorageUsage parses storage usage from the metric family.
func (nc *NodeCollector) parseStorageUsage(family *dto.MetricFamily) int64 {
	var totalStorage int64
	for _, metric := range family.Metric {
		device := ""
		for _, label := range metric.Label {
			if label.GetName() == "mountpoint" {
				device = label.GetValue()
				break
			}
		}
		if device == "/" {
			if metric.Gauge != nil && metric.Gauge.Value != nil {
				totalStorage += int64(metric.Gauge.GetValue())
			}
			break
		}
	}
	logrus.Debug("Successfully parsed storage usage metrics")
	return totalStorage
}
