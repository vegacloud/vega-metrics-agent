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
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sClientConfig holds the Kubernetes clientset and configuration
type K8sClientConfig struct {
	Clientset          *kubernetes.Clientset
	Config             *rest.Config
	UseInClusterConfig bool
	ClusterHostURL     string
	Namespace          string
	ClusterUID         string
	ClusterVersion     string
}

// GetClientConfig returns a Kubernetes ClientConfig using in-cluster or kubeconfig
func GetClientConfig(ctx context.Context, cfg *config.Config) (*K8sClientConfig, error) {
	logrus.Debug("GetClientConfig: Attempting to get Kubernetes client configuration")

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		logrus.Debug("GetClientConfig: Not in cluster, attempting to use kubeconfig")
		return getOutOfClusterConfig(ctx, cfg)
	}

	logrus.Info("GetClientConfig: Using in-cluster configuration")
	return createClientConfig(ctx, inClusterConfig, true, cfg)
}

func getOutOfClusterConfig(ctx context.Context, cfg *config.Config) (*K8sClientConfig, error) {
	var kubeconfigPath string

	if kc := os.Getenv("KUBECONFIG"); kc != "" {
		kubeconfigPath = kc
		logrus.Infof("getOutOfClusterConfig: Using KUBECONFIG environment variable: %s", kubeconfigPath)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getOutOfClusterConfig: unable to determine user's home directory: %v", err)
		}
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
		logrus.Infof("getOutOfClusterConfig: Using default kubeconfig path: %s", kubeconfigPath)
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("getOutOfClusterConfig: kubeconfig file not found: %s", kubeconfigPath)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("getOutOfClusterConfig: failed to build config from kubeconfig file: %v", err)
	}

	return createClientConfig(ctx, config, false, cfg)
}

func createClientConfig(
	ctx context.Context,
	config *rest.Config,
	inCluster bool,
	cfg *config.Config,
) (*K8sClientConfig, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("createClientConfig: failed to create clientset: %v", err)
	}

	clientConfig := &K8sClientConfig{
		Clientset:          clientset,
		Config:             config,
		UseInClusterConfig: inCluster,
		ClusterHostURL:     config.Host,
		Namespace:          cfg.VegaNamespace,
	}

	if err := enrichClientConfig(ctx, clientConfig); err != nil {
		return nil, err
	}

	logrus.Infof(
		"createClientConfig: Successfully created Kubernetes client configuration. Cluster URL: %s",
		clientConfig.ClusterHostURL,
	)

	return clientConfig, nil
}

func enrichClientConfig(ctx context.Context, config *K8sClientConfig) error {
	var err error

	// Get cluster UID
	config.ClusterUID, err = getNamespaceUID(ctx, config.Clientset, config.Namespace)
	if err != nil {
		return fmt.Errorf("enrichClientConfig: unable to find the namespace %s: %v", config.Namespace, err)
	}

	// Get cluster version
	config.ClusterVersion, err = getClusterVersion(config.Clientset)
	if err != nil {
		logrus.Warnf("enrichClientConfig: Unable to determine the cluster version: %v", err)
	}

	return nil
}

// getNamespaceUID retrieves the UID of a given namespace
func getNamespaceUID(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (string, error) {
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("getNamespaceUID: failed to get namespace %s: %v", namespace, err)
	}
	return string(ns.UID), nil
}

// getClusterVersion retrieves the version of the Kubernetes cluster
func getClusterVersion(clientset *kubernetes.Clientset) (string, error) {
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", fmt.Errorf("getClusterVersion: failed to get server version: %v", err)
	}
	return version.String(), nil
}
