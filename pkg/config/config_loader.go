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
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// LoadConfig initializes the configuration from environment variables and command-line flags.
func LoadConfig() (*Config, error) {
	// Use the global Viper instance instead of creating a new one.
	viper.SetEnvPrefix("VEGA")
	viper.AutomaticEnv()

	// Set default values
	setDefaults()

	if os.Getenv("START_COLLECTION_NOW") == "true" {
		viper.Set("start_collection_now", true)
	}
	if os.Getenv("SAVE_LOCAL") == "true" {
		viper.Set("save_local", true)
	}
	if os.Getenv("AGENT_ID") != "" {
		viper.Set("agent_id", os.Getenv("AGENT_ID"))
	}
	if os.Getenv("SHOULD_AGENT_CHECK_IN") != "" {
		viper.Set("should_agent_check_in", os.Getenv("SHOULD_AGENT_CHECK_IN"))
	}
	if os.Getenv("METRICS_COLLECTOR_API") != "" {
		viper.Set("metrics_collector_api", os.Getenv("METRICS_COLLECTOR_API"))
	}
	if os.Getenv("AUTH_SERVICE_URL") != "" {
		viper.Set("auth_service_url", os.Getenv("AUTH_SERVICE_URL"))
	}
	// Bind environment variables
	err := viper.BindEnv("client_id")
	if err != nil {
		return nil, err
	}

	// Bind command-line flags (already set in main.go)

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Parse poll_interval manually since Viper treats it as a string
	pollIntervalStr := viper.GetString("poll_interval")
	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid poll_interval: %v", err)
	}
	cfg.VegaPollInterval = pollInterval

	// Validate required fields
	if cfg.VegaClientID == "" || cfg.VegaClientSecret == "" || cfg.VegaOrgSlug == "" || cfg.VegaClusterName == "" {
		return nil, errors.New("missing required config values: client_id, client_secret, org_slug, or cluster_name")
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
}
