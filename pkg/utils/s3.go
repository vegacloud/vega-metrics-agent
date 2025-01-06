// Package utils provides utility functions for the agent
// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// Package utils provides utility functions for the agent
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
)

// Uploader defines the interface for uploading data
type Uploader interface {
	UploadMetrics(ctx context.Context, metrics map[string]interface{}) error
}

// S3Uploader implements the Uploader interface for uploading data to S3
type S3Uploader struct {
	config    *config.Config
	s3Session *session.Session
	client    *http.Client // HTTP client for making requests
}

// NewS3Uploader creates a new S3Uploader instance
func NewS3Uploader(cfg *config.Config) (*S3Uploader, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.VegaUploadRegion),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 session: %w", err)
	}

	// Create a new HTTP client with a timeout
	client := &http.Client{Timeout: 10 * time.Second}

	return &S3Uploader{
		config:    cfg,
		s3Session: sess,
		client:    client,
	}, nil
}

// getPresignedURL generates a presigned URL for uploading to S3
func (u *S3Uploader) getPresignedURL(ctx context.Context, clusterName, filename string) (string, error) {
	logrus.Debugf("getPresignedURL: Generating presigned URL for object key: %s", filename)

	// Get the auth token using the refactored GetAuthToken function
	token, err := GetVegaAuthToken(ctx, u.client, u.config)
	if err != nil {
		return "", fmt.Errorf("getPresignedURL: failed to get auth token: %w", err)
	}

	url := fmt.Sprintf(
		"%s/generate-presigned-url?cluster_name=%s&object_key=%s&slug=%s",
		u.config.MetricsCollectorAPI,
		clusterName,
		filename,
		u.config.VegaOrgSlug)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("getPresignedURL: error creating request: %w", err)
	}
	// Use bearer token for authentication
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := u.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("getPresignedURL: error making request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("getPresignedURL: API returned non-200 status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("getPresignedURL: error reading response body: %w", err)
	}

	var result struct {
		PresignedURL string `json:"presigned_url"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("getPresignedURL: error unmarshaling response: %w", err)
	}
	return result.PresignedURL, nil
}

func (u *S3Uploader) uploadToS3(ctx context.Context, presignedURL string, data []byte) error {
	client := &http.Client{
		Timeout: time.Minute * 5, // Increased timeout for large uploads
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		req, err := http.NewRequestWithContext(ctx, "PUT", presignedURL, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("uploadToS3: failed to create request: %w", err)
		}

		req.ContentLength = int64(len(data))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			logrus.Warnf("uploadToS3: Failed to upload to S3, attempt %d: %v", i+1, err)
			time.Sleep(2 * time.Second) // Backoff before retrying
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("uploadToS3: S3 upload failed with status %d: %s", resp.StatusCode, string(body))
			logrus.Warnf("uploadToS3: Failed to upload to S3, attempt %d: %v", i+1, lastErr)
			time.Sleep(2 * time.Second) // Backoff before retrying
			continue
		}

		return nil
	}

	return lastErr
}

// MetricsUpload is the struct for the metrics upload
type MetricsUpload struct {
	AgentVersion  string      `json:"agentVersion"`
	SchemaVersion string      `json:"schemaVersion"`
	Items         interface{} `json:"items"`
}

// UploadMetrics uploads metrics to S3
func (u *S3Uploader) UploadMetrics(ctx context.Context, metrics map[string]interface{}) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(metrics))
	concurrencyLimit := u.config.VegaMaxConcurrency
	semaphore := make(chan struct{}, concurrencyLimit)

	for collectorName, data := range metrics {
		wg.Add(1)
		go func(collectorName string, data interface{}) {
			defer wg.Done()

			// Acquire a slot in the semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release the slot when done
			metricsUpload := MetricsUpload{
				AgentVersion:  u.config.AgentVersion,
				SchemaVersion: u.config.SchemaVersion,
				Items:         data,
			}
			// Marshal the metrics data to JSON
			jsonData, err := json.Marshal(metricsUpload)
			if err != nil {
				errChan <- fmt.Errorf("failed to marshal metrics for collector %s: %w", collectorName, err)
				return
			}

			filename := fmt.Sprintf("%s_%sUTC.json", collectorName, time.Now().Format("2006-01-02_15:04"))

			// Generate a presigned URL for the file
			presignedURL, err := u.getPresignedURL(ctx, u.config.VegaClusterName, filename)
			if err != nil {
				errChan <- fmt.Errorf("failed to generate presigned URL for collector %s: %w", collectorName, err)
				return
			}

			// Upload the JSON file to S3 using the presigned URL
			if err := u.uploadToS3(ctx, presignedURL, jsonData); err != nil {
				errChan <- fmt.Errorf("failed to upload metrics for collector %s: %w", collectorName, err)
				return
			}

			logrus.Infof("Successfully uploaded metrics for collector %s to S3", collectorName)
		}(collectorName, data)
	}

	wg.Wait()
	close(errChan)

	// Check if any errors occurred during the upload process
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
