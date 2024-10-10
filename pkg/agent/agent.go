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

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/informercollectors"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
)

// MetricsAgent is the main struct for the metrics agent
type MetricsAgent struct {
	config     *config.Config
	collectors map[string]informercollectors.InformationSource
	uploader   utils.Uploader
	httpClient *http.Client
	logger     *logrus.Entry
	clientSet  kubernetes.Interface
	stopCh     chan struct{}
}

// NewMetricsAgent creates a new MetricsAgent
func NewMetricsAgent(cfg *config.Config,
	clientSet kubernetes.Interface,
	logger *logrus.Entry,
) (*MetricsAgent, error) {
	logger = logger.WithField("function", "NewMetricsAgent")
	stopCh := make(chan struct{})
	err := informercollectors.FetchAndSaveInformationSources(context.Background(), cfg)
	if err != nil {
		err = informercollectors.UseDefaultInformationSources()
		if err != nil {
			panic(fmt.Sprintf("failed to load default information sources: %v", err))
		}
	}
	informationSources, err := informercollectors.LoadInformers(cfg.InformationSourcesFilePath,
		clientSet, int(cfg.VegaPollInterval), stopCh)
	if err != nil {
		logrus.Fatalf("Error loading information sources: %v", err)
	}
	// Initialize the uploader
	uploader, err := utils.NewS3Uploader(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 uploader: %w", err)
	}
	ma := &MetricsAgent{
		config:     cfg,
		collectors: informationSources,
		uploader:   uploader,
		logger:     logger.WithField("component", "MetricsAgent"),
		httpClient: &http.Client{
			Timeout: 90 * time.Second, // Set a timeout for HTTP requests
		},
		clientSet: clientSet,
		stopCh:    stopCh,
	}
	if cfg.ShouldAgentCheckIn {
		logrus.Info("Checking in with the metrics server")
		if err := ma.Checkin(context.Background()); err != nil {
			logrus.WithError(err).Error("Failed to check in with the metrics server")
		}
	}
	return ma, nil
}

func (ma *MetricsAgent) RefetchAndReloadInformationSources(ctx context.Context, clientSet kubernetes.Interface) error {
	ma.logger.Info("Refetching and reloading information sources")
	err := informercollectors.FetchAndSaveInformationSources(ctx, ma.config)
	if err != nil {
		return fmt.Errorf("error fetching and saving information sources: %w", err)
	}
	close(ma.stopCh)
	// Refetch the information sources

	// Reload the information sources
	stopCh := make(chan struct{})
	informationSources, err := informercollectors.LoadInformers(ma.config.InformationSourcesFilePath,
		clientSet, int(ma.config.VegaPollInterval), stopCh)
	if err != nil {
		return fmt.Errorf("error reloading information sources: %w", err)
	}

	// Update the collectors with the new information sources
	ma.collectors = informationSources
	ma.stopCh = stopCh
	ma.logger.Info("Successfully re-fetched and reloaded information sources")
	return nil
}

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

	// Track the last time information sources were updated
	lastUpdate := time.Now()

	// Run the metrics collection in a loop
	for {
		select {
		case <-ctx.Done():
			ma.logger.Info("Stopping metrics agent")
			return
		case <-ticker.C:
			// Check if it's time to refetch and reload the information sources
			if time.Since(lastUpdate) > ma.config.AutoUpdateInformationSourcesInterval {
				ma.logger.Info("Updating information sources")
				if err := ma.RefetchAndReloadInformationSources(ctx, ma.clientSet); err != nil {
					ma.logger.WithError(err).Errorf(
						"Failed to refetch and reload information sources continuing with the "+
							"current information sources, will retry update in %v",
						ma.config.AutoUpdateInformationSourcesInterval)
				} else {
					lastUpdate = time.Now()
				}
			}

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
	// Collect metrics from each information source concurrently
	for name, infoSource := range ma.collectors {
		wg.Add(1)
		go func(name string, infoSource informercollectors.InformationSource) {
			defer wg.Done()

			// Acquire a slot in the semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release the slot when done

			ma.logger.WithField("informationSource", name).Debug("Collecting metrics")
			collectedMetrics, err := informercollectors.GetInformerDataAsJSON(infoSource)
			if err != nil {
				ma.logger.WithField("informationSource", name).WithError(err).Error("Failed to collect metrics")
				mu.Lock()
				combinedErrors = errors.Join(combinedErrors, fmt.Errorf("informationSource %s: %w", name, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			metrics[name] = collectedMetrics
			mu.Unlock()
		}(name, infoSource)
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
