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

	"github.com/vegacloud/kubernetes/metricsagent/pkg/agent"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/health"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
)

var version = "999-snapshot"

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
	rootCmd.Flags().Bool("should_agent_check_in",
		false,
		"Should agent check in (env: VEGA_SHOULD_AGENT_CHECK_IN)")
	rootCmd.Flags().Int("poll_interval",
		int(config.DefaultPollInterval.Seconds()),
		"Polling interval in seconds (env: VEGA_POLL_INTERVAL)")
	// Logging configuration
	rootCmd.Flags().String("log_level",
		config.DefaultLogLevel,
		"Log level (DEBUG, INFO, WARN, ERROR) (env: VEGA_LOG_LEVEL)")
	// Kubernetes configuration
	rootCmd.Flags().String("bearer_token_path",
		config.DefaultBearerTokenPath,
		"Bearer token path (env: VEGA_BEARER_TOKEN_PATH)")
	rootCmd.Flags().String("work_dir", config.DefaultWorkDir, "Work directory (env: VEGA_WORK_DIR)")
	rootCmd.Flags().Int("max_concurrency", config.DefaultMaxConcurrency, "Max concurrency")

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		logrus.Fatalf("Error binding flags: %v", err)
	}

	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Error executing command: %v", err)
	}
}

func runRootCmd(cmd *cobra.Command, _ []string) error {
	versionFlag, err := cmd.Flags().GetBool("version")
	if err != nil {
		return fmt.Errorf("failed to get 'version' flag: %w", err)
	}
	if versionFlag {
		logrus.Infof("Vega Kubernetes and Container Metrics Agent version: %s", version)
		return nil
	}
	requiredFlags := []string{"client_id", "client_secret", "org_slug", "cluster_name"}
	for _, flag := range requiredFlags {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			return fmt.Errorf("failed to mark flag '%s' as required: %w", flag, err)
		}
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	initializeLogging(cfg)

	logrus.WithFields(logrus.Fields{
		"version":      version,
		"client_id":    cfg.VegaClientID,
		"org_slug":     cfg.VegaOrgSlug,
		"cluster_name": cfg.VegaClusterName,
	}).Info("Starting Vega Kubernetes and Container Metrics Agent")
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("Configuration: %+v", cfg)
	}

	ctx := context.Background()
	k8sClientConfig, err := initializeKubernetesClients(ctx, cfg)
	if err != nil {
		return err
	}

	logger := logrus.WithFields(logrus.Fields{
		"client_id":    cfg.VegaClientID,
		"org_slug":     cfg.VegaOrgSlug,
		"cluster_name": cfg.VegaClusterName,
	})

	if err := startMetricsAgent(cfg, k8sClientConfig.Clientset, logger); err != nil {
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

func initializeKubernetesClients(ctx context.Context, cfg *config.Config) (*utils.K8sClientConfig, error) {
	clientConfig, err := utils.GetClientConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	return clientConfig, nil
}

func startMetricsAgent(cfg *config.Config,
	clientset kubernetes.Interface,
	logger *logrus.Entry,
) error {

	metricsAgent, err := agent.NewMetricsAgent(cfg, clientset, logger)
	if err != nil {
		return fmt.Errorf("failed to create metrics agent: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logrus.Println("Starting liveness probe server on :80")
		if err := health.ServerHealthCheck(ctx); err != nil {
			logger.Error("Health check startup failed")
			cancel()
		} else {
			logger.Debug("Health check successful")
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Infof("Received signal %s, initiating shutdown", sig)
		cancel()
	}()

	metricsAgent.Run(ctx)

	return nil
}
