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
package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
)

// AuthResponse holds the response from the auth server
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

var tokenCache struct {
	token     string
	expiresAt time.Time
}

// GetVegaAuthToken gets an auth token from the auth server
func GetVegaAuthToken(
	ctx context.Context,
	client *http.Client,
	cfg *config.Config,
) (string, error) {
	// Check if the token is still valid
	if time.Now().Before(tokenCache.expiresAt) {
		return tokenCache.token, nil
	}

	authURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", cfg.AuthServiceURL, cfg.VegaOrgSlug)
	logrus.WithFields(logrus.Fields{
		"function": "GetVegaAuthToken",
		"clientID": cfg.VegaClientID,
		"slug":     cfg.VegaOrgSlug,
		"url":      cfg.AuthServiceURL,
		"authURL":  authURL,
	}).Debug("Getting auth token")

	form := url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_id":     []string{cfg.VegaClientID},
		"client_secret": []string{cfg.VegaClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating request for slug %s: %w", cfg.VegaOrgSlug, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Use a loop with a retry mechanism
	for i := 0; i < 3; i++ {
		resp, err := client.Do(req)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"function": "GetVegaAuthToken",
				"attempt":  i + 1,
				"error":    err,
			}).Warn("Failed to send request, retrying")

			if i == 2 {
				return "", fmt.Errorf("sending request after retries: %w", err)
			}

			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Ensure response body is closed
		defer func() {
			if err := resp.Body.Close(); err != nil {
				logrus.WithError(err).Warn("Failed to close response body")
			}
		}()

		// Handle non-2xx status codes
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			body, _ := io.ReadAll(resp.Body)
			logrus.WithFields(logrus.Fields{
				"function":   "GetVegaAuthToken",
				"statusCode": resp.StatusCode,
				"body":       string(body),
			}).Error("Received non-2xx status code")
			return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Decode the JSON response
		var authResp AuthResponse
		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			return "", fmt.Errorf("decoding response for slug %s: %w", cfg.VegaOrgSlug, err)
		}

		logrus.WithFields(logrus.Fields{
			"function":    "GetVegaAuthToken",
			"accessToken": authResp.AccessToken,
			"expiresIn":   authResp.ExpiresIn,
		}).Debug("Auth token obtained successfully")

		// Cache the token
		tokenCache.token = authResp.AccessToken
		tokenCache.expiresAt = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

		return authResp.AccessToken, nil
	}

	// Return an error if all attempts fail
	return "", fmt.Errorf("failed to obtain auth token after retries")
}
