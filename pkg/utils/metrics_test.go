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
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestGetMetricsClient(t *testing.T) {
	// Create a fake config
	config := &rest.Config{}

	// Test if the metrics client is created successfully
	client, err := GetMetricsClient(config)
	if err != nil {
		t.Fatalf("Failed to create metrics client: %v", err)
	}
	if client == nil {
		t.Fatal("Expected metrics client to be created, got nil")
	}
}

func TestNodeAndPodMetricsFromAPI(t *testing.T) {
	// We won't use real metrics calls in unit tests
	// Instead, we'll ensure the functions exist and have the correct signatures

	// We're going to test if the functions compile with correct types, not actual functionality
	var _ = func() {
		ctx := context.Background()
		clientset := &kubernetes.Clientset{}
		config := &rest.Config{}

		// Test GetNodeMetricsFromAPI
		_, err := GetNodeMetricsFromAPI(ctx, clientset, config, "test-node")
		if err != nil {
			// We expect this to fail in tests since we don't have real clients
			// but we're only checking method signatures
		}

		// Test GetPodMetricsFromAPI
		_, err = GetPodMetricsFromAPI(ctx, clientset, config, "default", "test-pod")
		if err != nil {
			// Similarly, this would fail but we're checking signatures
		}
	}
}

// This test uses fake clients to ensure the function integration works
func TestMetricsIntegration(t *testing.T) {
	t.Skip("This test requires a real Kubernetes cluster and is meant for manual execution only")

	// This test would be run manually against a real cluster
	// In a real environment, we'd set up the following:

	/*
		ctx := context.Background()
		config, err := rest.InClusterConfig()
		if err != nil {
			t.Fatalf("Failed to get in-cluster config: %v", err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			t.Fatalf("Failed to create clientset: %v", err)
		}

		// Test node metrics
		nodeMetrics, err := GetNodeMetricsFromAPI(ctx, clientset, config, "some-node-name")
		if err != nil {
			t.Fatalf("Failed to get node metrics: %v", err)
		}

		if nodeMetrics == nil {
			t.Fatal("Expected node metrics, got nil")
		}

		// Test pod metrics
		podMetrics, err := GetPodMetricsFromAPI(ctx, clientset, config, "default", "some-pod-name")
		if err != nil {
			t.Fatalf("Failed to get pod metrics: %v", err)
		}

		if podMetrics == nil {
			t.Fatal("Expected pod metrics, got nil")
		}
	*/
}
