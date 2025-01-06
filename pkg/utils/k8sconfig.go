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
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
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

var (
	instance *K8sClientConfig
	once     sync.Once
)

// logCurrentIdentity attempts to get and log the current service account identity
func logCurrentIdentity(ctx context.Context, clientset *kubernetes.Clientset) {
	logger := logrus.WithField("function", "logCurrentIdentity")

	// Try to get self review
	selfReview := &authenticationv1.SelfSubjectReview{}
	result, err := clientset.AuthenticationV1().SelfSubjectReviews().Create(ctx, selfReview, metav1.CreateOptions{})
	if err != nil {
		logger.WithError(err).Error("Failed to get self subject review")
		return
	}

	logger.WithFields(logrus.Fields{
		"username": result.Status.UserInfo.Username,
		"uid":      result.Status.UserInfo.UID,
		"groups":   result.Status.UserInfo.Groups,
	}).Info("Current identity")

	// Try to get self access review
	accessReview := &authorizationv1.SelfSubjectRulesReview{
		Spec: authorizationv1.SelfSubjectRulesReviewSpec{
			Namespace: "default",
		},
	}
	accessResult, err := clientset.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, accessReview, metav1.CreateOptions{})
	if err != nil {
		logger.WithError(err).Error("Failed to get self subject rules review")
		return
	}

	logger.WithField("rules", accessResult.Status.ResourceRules).Debug("Current permissions")
}

// GetClientConfig returns a Kubernetes ClientConfig using in-cluster or kubeconfig
func GetClientConfig(ctx context.Context, cfg *config.Config) (*K8sClientConfig, error) {
	var err error
	once.Do(func() {
		logrus.Debug("GetClientConfig: Attempting to get Kubernetes client configuration")

		inClusterConfig, inClusterErr := rest.InClusterConfig()
		if inClusterErr != nil {
			logrus.Debug("GetClientConfig: Not in cluster, attempting to use kubeconfig")
			instance, err = getOutOfClusterConfig(ctx, cfg)
			return
		}

		// Explicitly set the bearer token from the specified path
		token, tokenErr := os.ReadFile(cfg.VegaBearerTokenPath)
		if tokenErr != nil {
			err = fmt.Errorf("failed to read service account token from %s: %w", cfg.VegaBearerTokenPath, tokenErr)
			return
		}

		// Override the token in the config
		inClusterConfig.BearerToken = string(token)
		inClusterConfig.BearerTokenFile = cfg.VegaBearerTokenPath
		inClusterConfig.QPS = cfg.QPS
		inClusterConfig.Burst = cfg.Burst
		inClusterConfig.Timeout = cfg.Timeout

		logrus.WithFields(logrus.Fields{
			"token_path": cfg.VegaBearerTokenPath,
			"token_len":  len(string(token)),
			"qps":        cfg.QPS,
			"burst":      cfg.Burst,
			"timeout":    cfg.Timeout,
		}).Debug("Using explicit service account token")

		instance, err = createClientConfig(ctx, inClusterConfig, true, cfg)
		if err != nil {
			return
		}

		// Log the current identity
		logCurrentIdentity(ctx, instance.Clientset)
	})

	return instance, nil
}

// GetExistingClientConfig returns the existing client config without creating a new one
func GetExistingClientConfig() (*K8sClientConfig, error) {
	if instance == nil {
		return nil, fmt.Errorf("kubernetes client configuration has not been initialized")
	}
	return instance, nil
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
	logrus.Debugf("Attempting to get namespace %s", namespace)

	_, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		logrus.Debugf("Error reading service account token: %v", err)
	} else {
		logrus.Debugf("Service account token exists and is readable")
	}

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

// VerifyClientIdentity checks if the client is using the expected service account
func VerifyClientIdentity(ctx context.Context, clientset *kubernetes.Clientset, expectedNamespace string) error {
	logger := logrus.WithField("function", "VerifyClientIdentity")

	selfReview := &authenticationv1.SelfSubjectReview{}
	result, err := clientset.AuthenticationV1().SelfSubjectReviews().Create(ctx, selfReview, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to get self subject review: %w", err)
	}

	// Log the identity information
	logger.WithFields(logrus.Fields{
		"username": result.Status.UserInfo.Username,
		"uid":      result.Status.UserInfo.UID,
		"groups":   result.Status.UserInfo.Groups,
	}).Info("Current client identity")

	// Verify the service account namespace
	if !strings.Contains(result.Status.UserInfo.Username, expectedNamespace) {
		return fmt.Errorf("client is not using the expected service account in namespace %s", expectedNamespace)
	}

	return nil
}
