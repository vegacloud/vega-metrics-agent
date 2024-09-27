// Package agent provides the main struct and methods for the metrics agent
// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/collectors"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
)

// MetricsAgent is the main struct for the metrics agent
type MetricsAgent struct {
	config     *config.Config
	collectors map[string]collectors.Collector
	uploader   utils.Uploader
	httpClient *http.Client
	logger     *logrus.Entry
}

// NewMetricsAgent creates a new MetricsAgent
func NewMetricsAgent(cfg *config.Config,
	clientset kubernetes.Interface,
	metricsClientset metricsclientset.Interface,
	logger *logrus.Entry,
	k8sConfig utils.K8sClientConfig,
) (*MetricsAgent, error) {
	logger = logger.WithField("function", "NewMetricsAgent")

	k8sClientset, ok := clientset.(*kubernetes.Clientset)
	if !ok {
		return nil, fmt.Errorf("invalid clientset type: expected *kubernetes.Clientset, got %T", clientset)
	}
	// Create an empty map to hold the collectors
	collectorsMap := make(map[string]collectors.Collector)
	metricsClientsetImpl, ok := metricsClientset.(*metricsclientset.Clientset)
	if !ok {
		return nil, fmt.Errorf(
			"invalid metrics clientset type: expected *metricsclientset.Clientset, got %T",
			metricsClientset,
		)
	}

	// Initialize the NodeCollector
	nodeCollector, err := collectors.NewNodeCollector(k8sClientset, metricsClientsetImpl, cfg)
	if err != nil {
		logger.Fatalf("Failed to create NodeCollector: %v", err)
	}
	collectorsMap["node"] = nodeCollector

	// Initialize the PodCollector (assuming it doesn't return an error)
	podCollector := collectors.NewPodCollector(k8sClientset, metricsClientsetImpl, cfg)
	collectorsMap["pod"] = podCollector

	// Initialize the ClusterCollector (assuming it doesn't return an error)
	clusterCollector := collectors.NewClusterCollector(k8sClientset, cfg, k8sConfig)
	collectorsMap["cluster"] = clusterCollector

	// Continue initializing other collectors similarly
	collectorsMap["pv"] = collectors.NewPersistentVolumeCollector(k8sClientset, cfg)
	collectorsMap["namespace"] = collectors.NewNamespaceCollector(k8sClientset, cfg)
	collectorsMap["workload"] = collectors.NewWorkloadCollector(k8sClientset, cfg)
	collectorsMap["network"] = collectors.NewNetworkingCollector(k8sClientset, cfg)
	collectorsMap["job"] = collectors.NewJobCollector(k8sClientset, cfg)
	collectorsMap["cronjob"] = collectors.NewCronJobCollector(k8sClientset, cfg)
	collectorsMap["HPA"] = collectors.NewHPACollector(k8sClientset, cfg)
	collectorsMap["replicationController"] = collectors.NewReplicationControllerCollector(k8sClientset, cfg)

	collectorsMap["replicasets"] = collectors.NewReplicaSetCollector(k8sClientset, cfg)
	logger.Debugf("loaded %v collectors", len(collectorsMap))

	// Initialize the uploader
	uploader, err := utils.NewS3Uploader(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 uploader: %w", err)
	}
	ma := &MetricsAgent{
		config:     cfg,
		collectors: collectorsMap,
		uploader:   uploader,
		logger:     logger.WithField("component", "MetricsAgent"),
		httpClient: &http.Client{
			Timeout: 90 * time.Second, // Set a timeout for HTTP requests
		},
	}
	if cfg.ShouldAgentCheckIn {
		logrus.Info("Checking in with the metrics server")
		if err := ma.Checkin(context.Background()); err != nil {
			logrus.WithError(err).Error("Failed to check in with the metrics server")
		}
	}
	return ma, nil
}

// Run starts the metrics collection and upload process
func (ma *MetricsAgent) Run(ctx context.Context) {
	// Check if we should start collection immediately
	if ma.config.StartCollectionNow {
		ma.logger.Info("Starting metrics collection immediately as per configuration")
		if err := ma.collectAndUploadMetrics(ctx); err != nil {
			ma.logger.WithError(err).Error("Failed to collect and upload metrics")
		}
	} else {
		// Calculate the time to the next upcoming half-hour
		now := time.Now()
		nextHalfHour := now.Truncate(30 * time.Minute).Add(30 * time.Minute)
		timeUntilNextHalfHour := time.Until(nextHalfHour)

		// Log the time until the first execution
		ma.logger.Infof("Waiting %v until next half-hour to start metrics collection", timeUntilNextHalfHour)

		// Wait until the next half-hour
		select {
		case <-time.After(timeUntilNextHalfHour):
		case <-ctx.Done():
			ma.logger.Info("Stopping metrics agent before first run")
			return
		}

		// Collect and upload metrics at the next half-hour mark
		if err := ma.collectAndUploadMetrics(ctx); err != nil {
			ma.logger.WithError(err).Error("Failed to collect and upload metrics")
		}
	}

	// Start the ticker for regular intervals after the first execution
	ticker := time.NewTicker(ma.config.VegaPollInterval)
	defer ticker.Stop()

	// Run the metrics collection in a loop
	for {
		select {
		case <-ctx.Done():
			ma.logger.Info("Stopping metrics agent")
			return
		case <-ticker.C:
			if err := ma.collectAndUploadMetrics(ctx); err != nil {
				ma.logger.WithError(err).Error("Failed to collect and upload metrics")
			}
		}
	}
}

func (ma *MetricsAgent) collectAndUploadMetrics(ctx context.Context) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	metrics := make(map[string]interface{})
	var combinedErrors error

	// Create a buffered channel to limit the number of concurrent collectors
	concurrencyLimit := ma.config.VegaMaxConcurrency
	semaphore := make(chan struct{}, concurrencyLimit)
	startTime := time.Now()
	if ma.config.ShouldAgentCheckIn {
		go func() {
			logrus.Info("Checking in with the metrics server")
			if err := ma.Checkin(ctx); err != nil {
				logrus.WithError(err).Error("Failed to check in with the metrics server")
			}
		}()
	}
	// Collect metrics from each collector concurrently
	for name, collector := range ma.collectors {
		wg.Add(1)
		go func(name string, collector collectors.Collector) {
			defer wg.Done()

			// Acquire a slot in the semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release the slot when done

			ma.logger.WithField("collector", name).Info("Collecting metrics")
			collectedMetrics, err := collector.CollectMetrics(ctx)
			if err != nil {
				ma.logger.WithField("collector", name).WithError(err).Error("Failed to collect metrics")
				mu.Lock()
				combinedErrors = errors.Join(combinedErrors, fmt.Errorf("collector %s: %w", name, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			metrics[name] = collectedMetrics
			mu.Unlock()
		}(name, collector)
	}
	wg.Wait()

	if err := ma.uploader.UploadMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to upload metrics: %w", err)
	}

	if combinedErrors != nil {
		ma.logger.WithError(combinedErrors).Error("Failed to collect and upload metrics")
		return combinedErrors
	}

	ma.logger.Debugf("Successfully collected and uploaded metrics, in %v", time.Since(startTime))
	return nil
}

// Checkin calls the /agents/checkin endpoint on the metrics server
// with the AgentId, VegaOrgSlug, VegaClientID, and VegaClusterName
func (ma *MetricsAgent) Checkin(ctx context.Context) error {
	// Prepare the URL for the check-in endpoint
	checkinURL := ma.config.MetricsCollectorAPI + "/agents/checkin"

	// Prepare the payload with the required fields
	payload := map[string]string{
		"agent_id":     ma.config.AgentID,
		"org_slug":     ma.config.VegaOrgSlug,
		"client_id":    ma.config.VegaClientID,
		"cluster_name": ma.config.VegaClusterName,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal check-in payload: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, checkinURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create check-in request: %w", err)
	}

	// Set the appropriate headers
	req.Header.Set("Content-Type", "application/json")

	// Get the auth token
	token, err := utils.GetVegaAuthToken(ctx,
		ma.httpClient,
		ma.config)
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	// Set the Authorization header
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	resp, err := ma.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send check-in request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("check-in request failed with status %d: %s", resp.StatusCode, string(body))
	}

	ma.logger.Debug("Successfully checked in with the metrics server")
	return nil
}
