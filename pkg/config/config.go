// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package config provides configuration parameters for the agent
package config

import (
	"time"
)

// Config holds configuration parameters for the agent
type Config struct {
	AgentID                  string        `mapstructure:"agent_id"`
	VegaClientID             string        `mapstructure:"client_id"`
	VegaClientSecret         string        `mapstructure:"client_secret"`
	VegaClusterName          string        `mapstructure:"cluster_name"`
	VegaOrgSlug              string        `mapstructure:"org_slug"`
	VegaPollInterval         time.Duration `mapstructure:"poll_interval"`
	VegaUploadRegion         string        `mapstructure:"upload_region"`
	VegaParseMetricData      bool          `mapstructure:"parse_metric_data"`
	VegaInsecure             bool          `mapstructure:"insecure"`
	VegaWorkDir              string        `mapstructure:"work_dir"`
	VegaCollectionRetryLimit int           `mapstructure:"collection_retry_limit"`
	VegaBearerTokenPath      string        `mapstructure:"bearer_token_path"`
	VegaNamespace            string        `mapstructure:"namespace"`
	MetricsCollectorAPI      string        `mapstructure:"metrics_collector_api"`
	AuthServiceURL           string        `mapstructure:"auth_service_url"`
	StartCollectionNow       bool          `mapstructure:"start_collection_now"`
	SaveLocal                bool          `mapstructure:"save_local"`
	VegaMaxConcurrency       int           `mapstructure:"max_concurrency"`
	ShouldAgentCheckIn       bool          `mapstructure:"should_agent_check_in"`
	LogLevel                 string        `mapstructure:"log_level"`
}

// Default configuration values
const (
	DefaultPollInterval             = "60m"
	DefaultMaxConcurrency           = 8
	DefaultS3Region                 = "us-west-2"
	DefaultBearerTokenPath          = "/var/run/secrets/kubernetes.io/serviceaccount/token" //#nosec #nolint
	DefaultLogLevel                 = "INFO"
	DefaultVegaInsecure             = false
	DefaultStartCollectionNow       = false
	DefaultVegaCollectionRetryLimit = 3
	DefaultWorkDir                  = "/tmp"
	DefaultMetricsCollectorAPI      = "https://api.vegacloud.io/metrics"
	DefaultAuthServiceURL           = "https://auth.vegacloud.io"
	DefaultVegaNamespace            = "vegacloud"
	DefaultShouldAgentCheckIn       = true
	DefaultSaveLocal                = false
)
