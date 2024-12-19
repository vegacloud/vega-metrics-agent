// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// PersistentVolumeClaimCollector collects metrics from Kubernetes persistent volume claims.
type PersistentVolumeClaimCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewPersistentVolumeClaimCollector creates a new PersistentVolumeClaimCollector.
func NewPersistentVolumeClaimCollector(clientset *kubernetes.Clientset, cfg *config.Config) *PersistentVolumeClaimCollector {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NewPersistentVolumeClaimCollector")
		}
	}()

	logrus.Debug("Creating new PersistentVolumeClaimCollector")
	collector := &PersistentVolumeClaimCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("PersistentVolumeClaimCollector created successfully")
	return collector
}

// CollectMetrics collects metrics from Kubernetes persistent volume claims.
func (pvcc *PersistentVolumeClaimCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in PersistentVolumeClaimCollector.CollectMetrics")
		}
	}()

	pvcs, err := pvcc.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list persistent volume claims")
		return []models.PVCMetric{}, nil
	}

	logrus.WithField("count", len(pvcs.Items)).Debug("Successfully listed persistent volume claims")
	metrics := pvcc.collectPVCMetrics(pvcs.Items)

	logrus.WithField("count", len(metrics)).Debug("Successfully collected PVC metrics")
	return metrics, nil
}

// collectPVCMetrics collects metrics from Kubernetes persistent volume claims.
func (pvcc *PersistentVolumeClaimCollector) collectPVCMetrics(pvcs []v1.PersistentVolumeClaim) []models.PVCMetric {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in collectPVCMetrics")
		}
	}()

	pvcMetrics := make([]models.PVCMetric, 0, len(pvcs))

	for _, claim := range pvcs {
		logrus.WithFields(logrus.Fields{
			"pvc":       claim.Name,
			"namespace": claim.Namespace,
		}).Debug("Processing persistent volume claim")

		if claim.Labels == nil {
			claim.Labels = make(map[string]string)
		}
		if claim.Annotations == nil {
			claim.Annotations = make(map[string]string)
		}

		// Safely collect access modes
		accessModes := make([]string, 0)
		for _, mode := range claim.Spec.AccessModes {
			accessModes = append(accessModes, string(mode))
		}

		// Safely collect conditions
		conditions := make([]models.PVCCondition, 0)
		for _, cond := range claim.Status.Conditions {
			if cond.LastTransitionTime.IsZero() {
				logrus.WithFields(logrus.Fields{
					"pvc":       claim.Name,
					"namespace": claim.Namespace,
					"condition": cond.Type,
				}).Debug("Skipping condition with zero transition time")
				continue
			}

			conditions = append(conditions, models.PVCCondition{
				Type:               string(cond.Type),
				Status:             string(cond.Status),
				LastTransitionTime: &cond.LastTransitionTime.Time,
				Reason:             cond.Reason,
				Message:            cond.Message,
			})
		}

		// Safely handle volume mode
		var volumeMode string
		if claim.Spec.VolumeMode != nil {
			volumeMode = string(*claim.Spec.VolumeMode)
		}

		metric := models.PVCMetric{
			Name:             claim.Name,
			Namespace:        claim.Namespace,
			Phase:            string(claim.Status.Phase),
			Capacity:         claim.Status.Capacity.Storage().Value(),
			RequestedStorage: claim.Spec.Resources.Requests.Storage().Value(),
			Labels:           claim.Labels,
			AccessModes:      accessModes,
			VolumeMode:       volumeMode,
			VolumeName:       claim.Spec.VolumeName,
			Status: models.PVCStatus{
				Phase:      string(claim.Status.Phase),
				Conditions: conditions,
			},
		}

		// Safely handle storage class
		if claim.Spec.StorageClassName != nil {
			metric.StorageClass = *claim.Spec.StorageClassName
			sc, err := pvcc.clientset.StorageV1().StorageClasses().Get(
				context.Background(),
				*claim.Spec.StorageClassName,
				metav1.GetOptions{},
			)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"pvc":          claim.Name,
					"namespace":    claim.Namespace,
					"storageClass": *claim.Spec.StorageClassName,
					"error":        err,
				}).Error("Failed to get storage class details")
			} else if sc.VolumeBindingMode != nil {
				metric.VolumeBindingMode = string(*sc.VolumeBindingMode)
			}
		}

		// Safely handle bound PV
		if claim.Spec.VolumeName != "" {
			metric.BoundPV = claim.Spec.VolumeName
			boundPV, err := pvcc.clientset.CoreV1().PersistentVolumes().Get(
				context.Background(),
				claim.Spec.VolumeName,
				metav1.GetOptions{},
			)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"pvc":       claim.Name,
					"namespace": claim.Namespace,
					"boundPV":   claim.Spec.VolumeName,
					"error":     err,
				}).Error("Failed to get bound PV details")
			} else {
				metric.MountOptions = boundPV.Spec.MountOptions
			}
		}

		// Safely handle annotations
		if ann := claim.Annotations; ann != nil {
			if provisioner, ok := ann["volume.kubernetes.io/storage-provisioner"]; ok {
				metric.StorageProvisioner = provisioner
			}
		}

		// Handle volume expansion metrics
		if claim.Spec.Resources.Requests.Storage() != nil {
			expansion := &models.VolumeExpansionMetrics{
				CurrentSize:   claim.Status.Capacity.Storage().Value(),
				RequestedSize: claim.Spec.Resources.Requests.Storage().Value(),
				InProgress:    false,
			}

			for _, condition := range claim.Status.Conditions {
				if condition.Type == v1.PersistentVolumeClaimResizing {
					expansion.InProgress = true
					expansion.LastResizeTime = &condition.LastTransitionTime.Time
					expansion.ResizeStatus = string(condition.Status)
					expansion.FailureMessage = condition.Message
					break
				}
			}

			metric.Expansion = expansion
		}

		logrus.WithFields(logrus.Fields{
			"pvc":              claim.Name,
			"namespace":        claim.Namespace,
			"phase":            metric.Phase,
			"capacity":         metric.Capacity,
			"requestedStorage": metric.RequestedStorage,
			"storageClass":     metric.StorageClass,
			"boundPV":          metric.BoundPV,
		}).Debug("Collected metrics for persistent volume claim")

		pvcMetrics = append(pvcMetrics, metric)
	}

	return pvcMetrics
}
