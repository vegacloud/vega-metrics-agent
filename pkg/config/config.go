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
	"os"
	"time"
)

// Config holds configuration parameters for the agent
type Config struct {
	AgentID                              string        `mapstructure:"agent_id"`
	VegaClientID                         string        `mapstructure:"client_id"`
	VegaClientSecret                     string        `mapstructure:"client_secret"`
	VegaClusterName                      string        `mapstructure:"cluster_name"`
	VegaOrgSlug                          string        `mapstructure:"org_slug"`
	VegaPollInterval                     time.Duration `mapstructure:"poll_interval"`
	VegaUploadRegion                     string        `mapstructure:"upload_region"`
	VegaParseMetricData                  bool          `mapstructure:"parse_metric_data"`
	MetricsCollectorAPI                  string        `mapstructure:"metrics_collector_api"`
	AuthServiceURL                       string        `mapstructure:"auth_service_url"`
	LogLevel                             string        `mapstructure:"log_level"`
	VegaInsecure                         bool          `mapstructure:"insecure"`
	VegaWorkDir                          string        `mapstructure:"work_dir"`
	VegaCollectionRetryLimit             int           `mapstructure:"collection_retry_limit"`
	VegaBearerTokenPath                  string        `mapstructure:"bearer_token_path"`
	StartCollectionNow                   bool          `mapstructure:"start_collection_now"`
	SaveLocal                            bool          `mapstructure:"save_local"`
	VegaMaxConcurrency                   int           `mapstructure:"max_concurrency"`
	VegaNamespace                        string        `mapstructure:"namespace"`
	ShouldAgentCheckIn                   bool          `mapstructure:"should_agent_check_in"`
	InformationSourcesURL                string        `mapstructure:"information_sources_url"`
	InformationSourcesFilePath           string        `mapstructure:"information_sources_file_path"`
	AutoUpdateInformationSourcesInterval time.Duration `mapstructure:"auto_update_information_sources_interval"`
}

// if DEV_MODE is true then use localhost, otherwise use the actual URL
var (
	DefaultPollInterval                         = 60 * time.Minute
	DefaultMaxConcurrency                       = 8
	DefaultS3Region                             = "us-west-2"
	DefaultBearerTokenPath                      = "/var/run/secrets/kubernetes.io/serviceaccount/token" //#nosec #nolint
	DefaultLogLevel                             = "INFO"
	DefaultVegaInsecure                         = false
	DefaultStartCollectionNow                   = false
	DefaultVegaCollectionRetryLimit             = 3
	DefaultWorkDir                              = "/tmp"
	DefaultMetricsCollectorAPI                  = getDefaultMetricsCollectorAPI()
	DefaultAuthServiceURL                       = "https://auth.vegacloud.io"
	DefaultInformationSourcesFilePath           = "./informationSources.yaml"
	DefaultInformationSourcesURL                = getDefaultInformationSourcesURL()
	DefaultAutoUpdateInformationSourcesInterval = 24 * time.Hour
	DefaultVegaNamespace                        = "default"
	DefaultShouldAgentCheckIn                   = true
	DefaultSaveLocal                            = false
)

// getDefaultMetricsCollectorAPI returns the appropriate MetricsCollectorAPI based on the DEV_MODE
func getDefaultMetricsCollectorAPI() string {
	if os.Getenv("DEV_MODE") == "true" {
		return "http://localhost:8080"
	}
	return "https://api.vegacloud.io/metrics"
}

// getDefaultInformationSourcesURL returns the appropriate InformationSourcesURL based on the DEV_MODE
func getDefaultInformationSourcesURL() string {
	if os.Getenv("DEV_MODE") == "true" {
		return "http://localhost:8080/information-sources"
	}
	return "https://api.vegacloud.io/metrics/informationSources"
}
