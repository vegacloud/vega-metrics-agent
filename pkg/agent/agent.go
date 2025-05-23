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
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"crypto/tls"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/collectors"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

// MetricsAgent is the main struct for the metrics agent
type MetricsAgent struct {
	config     *config.Config
	collectors map[string]collectors.Collector
	uploader   utils.Uploader
	httpClient *http.Client
	logger     *logrus.Entry
}

// CheckinRequest represents the request structure for agent checkin operations.
type CheckinRequest struct {
	AgentID         string             `json:"agent_id"`
	ClusterName     string             `json:"cluster_name"`
	ClusterVersion  *string            `json:"cluster_version,omitempty"`
	AgentVersion    *string            `json:"agent_version,omitempty"`
	SchemaVersion   *string            `json:"schema_version,omitempty"`
	ClusterProvider *string            `json:"cluster_provider,omitempty"`
	AgentStatus     *string            `json:"agent_status,omitempty"`
	CollectorStatus *map[string]string `json:"collector_status,omitempty"`
}

// NewMetricsAgent creates a new MetricsAgent
func NewMetricsAgent(cfg *config.Config, logger *logrus.Entry) (*MetricsAgent, error) {
	logger = logger.WithField("function", "NewMetricsAgent")

	clientConfig, err := utils.GetExistingClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing client config: %w", err)
	}

	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Errorf("failed to read service account token: %w", err)
	}

	restClient := clientConfig.Clientset.CoreV1().RESTClient().(*rest.RESTClient)
	restClient.Client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.VegaInsecure}} // #nosec G402
	restClient.Client.Transport = transport.NewBearerAuthRoundTripper(string(token), restClient.Client.Transport)

	clusterVersion, err := getClusterVersion(clientConfig.Clientset)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster version: %w", err)
	}
	logger.Infof("Cluster version: %s", clusterVersion)
	logger.Debugf("Getting Cluster Provider before the function")
	clusterProvider := getClusterProvider(clientConfig.Clientset, clusterVersion, logger)
	logger.Debugf("Getting Cluster Provider after the function")
	logger.Infof("Cluster provider: %s", clusterProvider)

	cfg.ClusterVersion = clusterVersion
	cfg.ClusterProvider = clusterProvider

	collectorsMap := initializeCollectors(clientConfig.Clientset, cfg)
	logger.Debugf("loaded %v collectors", len(collectorsMap))

	uploader, err := utils.NewS3Uploader(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 uploader: %w", err)
	}

	ma := &MetricsAgent{
		config:     cfg,
		collectors: collectorsMap,
		uploader:   uploader,
		logger:     logger.WithField("component", "MetricsAgent"),
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}

	if cfg.ShouldAgentCheckIn {
		logger.Debug("Attempting to check in with the metrics server")
		if err := ma.Checkin(context.Background()); err != nil {
			logger.WithError(err).Debug("Checkin attempt failed with detailed error")
			logrus.WithError(err).Error("Failed to check in with the metrics server")
		}
	}
	return ma, nil
}

// initializeCollectors initializes the collectors map
func initializeCollectors(clientset *kubernetes.Clientset, cfg *config.Config) map[string]collectors.Collector {
	return map[string]collectors.Collector{
		"cluster":               collectors.NewClusterCollector(clientset, cfg),
		"namespace":             collectors.NewNamespaceCollector(clientset, cfg),
		"node":                  collectors.NewNodeCollector(clientset, cfg),
		"pod":                   collectors.NewPodCollector(clientset, cfg),
		"pv":                    collectors.NewPersistentVolumeCollector(clientset, cfg),
		"pvc":                   collectors.NewPersistentVolumeClaimCollector(clientset, cfg),
		"workload":              collectors.NewWorkloadCollector(clientset, cfg),
		"daemonset":             collectors.NewDaemonSetCollector(clientset, cfg),
		"network":               collectors.NewNetworkingCollector(clientset, cfg),
		"job":                   collectors.NewJobCollector(clientset, cfg),
		"cronjob":               collectors.NewCronJobCollector(clientset, cfg),
		"hpa":                   collectors.NewHPACollector(clientset, cfg),
		"replicationcontroller": collectors.NewReplicationControllerCollector(clientset, cfg),
		"storageclass":          collectors.NewStorageClassCollector(clientset, cfg),
		"replicaset":            collectors.NewReplicaSetCollector(clientset, cfg),
	}
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
			logrus.Info("Starting Kubernetes Data Collection")
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
	const (
		statusError = "Error"
		statusOk    = "Ok"
	)

	checkinURL := ma.config.MetricsCollectorAPI + "/agents/checkin"
	collectorStatus := make(map[string]string)
	hasError := false

	for name, collector := range ma.collectors {
		_, err := collector.CollectMetrics(ctx)
		if err != nil {
			collectorStatus[name] = statusError
			hasError = true
		} else {
			collectorStatus[name] = statusOk
		}
	}

	agentStatus := statusOk
	if hasError {
		agentStatus = statusError
	}
	ma.logger.WithFields(logrus.Fields{
		"collector_status_length": len(collectorStatus),
		"agent_status":            agentStatus,
	}).Debug("Collector and Agent status")

	payload := CheckinRequest{
		AgentID:         ma.config.AgentID,
		ClusterName:     ma.config.VegaClusterName,
		ClusterVersion:  &ma.config.ClusterVersion,
		AgentVersion:    &ma.config.AgentVersion,
		SchemaVersion:   &ma.config.SchemaVersion,
		ClusterProvider: &ma.config.ClusterProvider,
		AgentStatus:     &agentStatus,
		CollectorStatus: &collectorStatus,
	}

	// log payloads agent id
	ma.logger.WithField("agent_id", ma.config.AgentID).Debug("Payload")
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal check-in payload: %w", err)
	}
	ma.logger.WithField("payload", string(payloadBytes)).Debug("Payload")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, checkinURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create check-in request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	token, err := utils.GetVegaAuthToken(ctx, ma.httpClient, ma.config)
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ma.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send check-in request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Warn("Failed to close response body in agent Checkin")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("check-in request failed with status %d: %s", resp.StatusCode, string(body))
	}

	ma.logger.Debug("Successfully checked in with the metrics server")
	return nil
}

// getClusterVersion retrieves the Kubernetes cluster version using the clientset
func getClusterVersion(clientset *kubernetes.Clientset) (string, error) {
	versionInfo, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get server version: %w", err)
	}
	return versionInfo.String(), nil
}

var eksPattern = regexp.MustCompile(`(?i)eks|aws|amazon`)

func isEKS(clusterVersion string, logger *logrus.Entry) bool {
	logger.Debugf("Checking if cluster is EKS")
	return eksPattern.MatchString(clusterVersion)
}

func isAKS(clientset *kubernetes.Clientset, logger *logrus.Entry) bool {
	logger.Debugf("Checking if cluster is AKS")
	var aksAzurePattern = regexp.MustCompile(`(?i)aks|azure`)
	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false
	}

	// Create a channel to receive results
	results := make(chan bool, len(pods.Items))

	// Process each pod in a separate goroutine
	for _, pod := range pods.Items {
		go func(pod corev1.Pod) {
			labels := pod.GetLabels()

			// Fast path check
			if managedBy, exists := labels["kubernetes.azure.com/managedby"]; exists {
				if strings.EqualFold(managedBy, "aks") {
					results <- true
					return
				}
			}

			// Slower path check
			for key, value := range labels {
				combined := key + " " + value
				if aksAzurePattern.MatchString(combined) {
					results <- true
					return
				}
			}

			results <- false
		}(pod)
	}

	// Collect results
	for range pods.Items {
		if <-results {
			return true
		}
	}

	return false
}

func isGKE(clientset *kubernetes.Clientset, logger *logrus.Entry) bool {
	logger.Debugf("Checking if cluster is GKE")
	discoveryClient := clientset.Discovery()
	apiGroups, err := discoveryClient.ServerGroups()
	if err != nil {
		return false
	}
	for _, group := range apiGroups.Groups {
		if strings.Contains(group.Name, "gke") {
			return true
		}
	}
	return false
}

// getClusterProvider determines the cloud provider based on API groups
func getClusterProvider(clientset *kubernetes.Clientset, clusterVersion string, logger *logrus.Entry) string {
	logger.Debugf("Getting Cluster Provider")

	if isEKS(clusterVersion, logger) {
		logger.Debug("Cluster is EKS")
		return "AWS"
	} else if isAKS(clientset, logger) {
		logger.Debug("Cluster is AKS")

		return "AZURE"
	} else if isGKE(clientset, logger) {
		logger.Debug("Cluster is GKE")
		return "GCP"
	}

	return "UNKNOWN"
}
