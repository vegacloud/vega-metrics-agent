// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package config handles loading configuration using Viper
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func IsS3SafeBucketName(bueckt_name string) bool {
	// We build a regex character class that includes:
	//   - 0-9, a-z, A-Z for alphanumeric characters
	//   - !, _ , . , * , ' and ( as safe special characters.
	// The hyphen (-) is placed at the beginning to avoid confusion with a range.
	pattern := `^[-0-9A-Za-z!_.*'\(]+$`

	matched, err := regexp.MatchString(pattern, bueckt_name)
	if err != nil {
		// In case of a regex error, return false (or handle as needed)
		return false
	}
	return matched
}

// LoadConfig initializes the configuration from environment variables and command-line flags.
func LoadConfig() (*Config, error) {
	// Use the global Viper instance instead of creating a new one.
	viper.SetEnvPrefix("VEGA")
	viper.AutomaticEnv()

	// Set default values
	setDefaults()

	// Map environment variables to Viper keys
	envVars := map[string]string{
		"START_COLLECTION_NOW":  "start_collection_now",
		"SAVE_LOCAL":            "save_local",
		"AGENT_ID":              "agent_id",
		"SHOULD_AGENT_CHECK_IN": "should_agent_check_in",
		"METRICS_COLLECTOR_API": "metrics_collector_api",
		"AUTH_SERVICE_URL":      "auth_service_url",
		"LOG_LEVEL":             "log_level",
	}

	for envVar, viperKey := range envVars {
		if value := os.Getenv(envVar); value != "" {
			viper.Set(viperKey, value)
		}
	}

	// Bind environment variables
	if err := viper.BindEnv("client_id"); err != nil {
		return nil, err
	}

	// Bind command-line flags (already set in main.go)

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Parse poll_interval manually since Viper treats it as a string
	if pollIntervalStr := viper.GetString("poll_interval"); pollIntervalStr != "" {
		pollInterval, err := time.ParseDuration(pollIntervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid poll_interval: %v", err)
		}
		cfg.VegaPollInterval = pollInterval
	}

	// Validate required fields
	requiredFields := []string{cfg.VegaClientID, cfg.VegaClientSecret, cfg.VegaOrgSlug, cfg.VegaClusterName}
	for _, field := range requiredFields {
		if field == "" {
			return nil, errors.New("missing required config values: client_id, client_secret, org_slug, or cluster_name")
		}
	}

	// Validate that the cluster name is safe for use in S3 bucket names
	if !IsS3SafeBucketName(cfg.VegaClusterName) {
		return nil, fmt.Errorf("cluster_name '%s' contains invalid characters for S3 bucket names; only alphanumeric characters and the following special characters are allowed: -!_.*'(", cfg.VegaClusterName)
	}

	return &cfg, nil
}

// setDefaults sets default values for the configuration
func setDefaults() {
	viper.SetDefault("poll_interval", DefaultPollInterval)
	viper.SetDefault("log_level", DefaultLogLevel)
	viper.SetDefault("use_insecure", DefaultVegaInsecure)
	viper.SetDefault("collection_retry_limit", DefaultVegaCollectionRetryLimit)
	viper.SetDefault("upload_region", DefaultS3Region)
	viper.SetDefault("bearer_token_path", DefaultBearerTokenPath)
	viper.SetDefault("start_collection_now", DefaultStartCollectionNow)
	viper.SetDefault("save_local", DefaultSaveLocal)
	viper.SetDefault("metrics_collector_api", DefaultMetricsCollectorAPI)
	viper.SetDefault("auth_service_url", DefaultAuthServiceURL)
	viper.SetDefault("work_dir", DefaultWorkDir)
	viper.SetDefault("namespace", DefaultVegaNamespace)
	viper.SetDefault("max_concurrency", DefaultMaxConcurrency)
	viper.SetDefault("agent_id", uuid.New().String())
	viper.SetDefault("should_agent_check_in", DefaultShouldAgentCheckIn)
	viper.SetDefault("schema_version", DefaultSchemaVersion)
	viper.SetDefault("agent_version", DefaultAgentVersion)
	viper.SetDefault("qps", DefaultQPS)
	viper.SetDefault("burst", DefaultBurst)
	viper.SetDefault("timeout", DefaultTimeout)
}
