// Package main is the entrypoint to the Vega Metrics Agent
// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/agent"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/health"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
)

func main() {
	time.Local = time.UTC

	rootCmd := &cobra.Command{
		Use:   "vega-metrics-agent",
		Short: "A metrics agent for collecting Kubernetes node and pod metrics and sending to Vega Cloud",
		RunE:  runRootCmd,
	}

	rootCmd.Flags().BoolP("version", "v", false, "Print the version and exit")

	// Vega configuration
	rootCmd.Flags().String("client_id", "", "Client ID (env: VEGA_CLIENT_ID)")
	rootCmd.Flags().String("client_secret", "", "Client Secret (env: VEGA_CLIENT_SECRET)")
	rootCmd.Flags().String("org_slug", "", "Organization slug (env: VEGA_ORG_SLUG)")
	rootCmd.Flags().String("cluster_name", "", "Cluster name (env: VEGA_CLUSTER_NAME)")
	rootCmd.Flags().Bool("insecure",
		config.DefaultVegaInsecure,
		"Insecure TLS (env: VEGA_INSECURE)")
	rootCmd.Flags().String("poll_interval",
		string(config.DefaultPollInterval),
		"Polling interval in minutes (env: VEGA_POLL_INTERVAL)")
	// Logging configuration
	rootCmd.Flags().String("log_level",
		config.DefaultLogLevel,
		"Log level (DEBUG, INFO, WARN, ERROR) (env: VEGA_LOG_LEVEL)")
	// Kubernetes configuration
	rootCmd.Flags().String("bearer_token_path",
		config.DefaultBearerTokenPath,
		"Bearer token path (env: VEGA_BEARER_TOKEN_PATH)")
	rootCmd.Flags().String("work_dir", config.DefaultWorkDir, "Work directory (env: VEGA_WORK_DIR)")
	rootCmd.Flags().Int("max_concurrency", config.DefaultMaxConcurrency, "Max concurrency (env: VEGA_MAX_CONCURRENCY)")

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		logrus.Fatalf("Error binding flags: %v", err)
	}

	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Error executing command: %v", err)
	}
}

func runRootCmd(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	versionFlag, err := cmd.Flags().GetBool("version")
	if err != nil {
		return fmt.Errorf("failed to get 'version' flag: %w", err)
	}
	if versionFlag {
		logrus.Infof("Vega Kubernetes and Container Metrics Agent version: %s, Schema version: %s", cfg.AgentVersion, cfg.SchemaVersion)
		return nil
	}
	requiredFlags := []string{"client_id", "client_secret", "org_slug", "cluster_name"}
	for _, flag := range requiredFlags {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			return fmt.Errorf("failed to mark flag '%s' as required: %w", flag, err)
		}
	}

	initializeLogging(cfg)

	logrus.WithFields(logrus.Fields{
		"version":      cfg.AgentVersion,
		"client_id":    cfg.VegaClientID,
		"org_slug":     cfg.VegaOrgSlug,
		"cluster_name": cfg.VegaClusterName,
	}).Info("Starting Vega Kubernetes and Container Metrics Agent")
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("Configuration: %+v", cfg)
	}

	logger := logrus.WithFields(logrus.Fields{
		"client_id":    cfg.VegaClientID,
		"org_slug":     cfg.VegaOrgSlug,
		"cluster_name": cfg.VegaClusterName,
	})

	if err := startMetricsAgent(cfg, logger); err != nil {
		return err
	}

	return nil
}

func initializeLogging(cfg *config.Config) {
	logLevel, err := logrus.ParseLevel(strings.ToLower(cfg.LogLevel))
	if err != nil {
		logrus.Warnf("Invalid log level %s, defaulting to INFO", cfg.LogLevel)
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

func initializeKubernetesClients(ctx context.Context, cfg *config.Config, logger *logrus.Entry) (*kubernetes.Clientset, *utils.K8sClientConfig, error) {
	logger.Debug("Starting Kubernetes client initialization...")

	// Get Kubernetes client configuration
	logger.Debug("Getting Kubernetes client configuration...")
	// Add functionality to verify we can access the service token
	token, err := os.ReadFile(cfg.VegaBearerTokenPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read service token: %w", err)
	}
	logger.WithField("token", string(token)).Debug("Successfully read service token")

	k8sConfig, err := utils.GetClientConfig(ctx, cfg)
	if err != nil {
		logger.WithError(err).Error("Failed to get Kubernetes client configuration")
		return nil, nil, fmt.Errorf("failed to get Kubernetes client config: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"host":              k8sConfig.Config.Host,
		"bearer_token_path": k8sConfig.Config.BearerTokenFile,
		"qps":               k8sConfig.Config.QPS,
		"burst":             k8sConfig.Config.Burst,
		"timeout":           k8sConfig.Config.Timeout,
	}).Debug("Successfully obtained Kubernetes client configuration")

	return k8sConfig.Clientset, k8sConfig, nil
}

func startMetricsAgent(cfg *config.Config, logger *logrus.Entry) error {
	logger.Debug("Initializing metrics agent...")

	// Configure TLS transport
	logger.Debug("Configuring TLS transport...")
	transport, err := rest.TransportFor(&rest.Config{
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: cfg.VegaInsecure,
		},
	})
	if err != nil {
		logger.WithError(err).Error("Failed to create TLS transport")
		return fmt.Errorf("failed to create transport: %w", err)
	}
	logger.WithField("insecure", cfg.VegaInsecure).Debug("Successfully created TLS transport")

	// Initialize Kubernetes clients
	clientset, k8sConfig, err := initializeKubernetesClients(context.Background(), cfg, logger)
	if err != nil {
		return err
	}

	// Add detailed k8s config logging here
	logger.WithFields(logrus.Fields{
		"cluster_host_url":      k8sConfig.ClusterHostURL,
		"cluster_uid":           k8sConfig.ClusterUID,
		"cluster_version":       k8sConfig.ClusterVersion,
		"namespace":             k8sConfig.Namespace,
		"use_in_cluster_config": k8sConfig.UseInClusterConfig,
		"qps":                   k8sConfig.Config.QPS,
		"burst":                 k8sConfig.Config.Burst,
		"timeout":               k8sConfig.Config.Timeout,
		"bearer_token_file":     k8sConfig.Config.BearerTokenFile,
		"ca_file":               k8sConfig.Config.CAFile,
		"server_name":           k8sConfig.Config.ServerName,
		"insecure_skip_verify":  k8sConfig.Config.Insecure,
	}).Info("Kubernetes configuration details")

	// Configure TLS for the REST client
	if client, ok := clientset.CoreV1().RESTClient().(*rest.RESTClient); ok {
		logger.Debug("Configuring TLS for kubelet client")
		client.Client.Transport = transport
		logger.WithFields(logrus.Fields{
			"insecure":       cfg.VegaInsecure,
			"transport_type": fmt.Sprintf("%T", transport),
		}).Debug("Successfully configured TLS for kubelet client")
	} else {
		logger.Warn("Unable to configure TLS: unexpected REST client type")
	}

	// Create metrics agent
	logger.Debug("Creating metrics agent...")
	// os.Exit(1)
	metricsAgent, err := agent.NewMetricsAgent(cfg, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to create metrics agent")
		return fmt.Errorf("failed to create metrics agent: %w", err)
	}
	logger.Debug("Successfully created metrics agent")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health check server
	go func() {
		logger.Info("Starting liveness probe server on :80")
		if err := health.ServerHealthCheck(ctx); err != nil {
			logger.WithError(err).Error("Health check startup failed")
			cancel()
		} else {
			logger.Info("Health check successful")
		}
	}()

	// Handle signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.WithField("signal", sig).Info("Received signal, initiating shutdown")
		cancel()
	}()

	logger.Info("Starting metrics agent main loop")
	metricsAgent.Run(ctx)

	return nil
}
