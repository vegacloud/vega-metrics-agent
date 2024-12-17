// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/persistentvolume.go

// Package collectors hosts the collection functions
package collectors

import (
	"context"
	"fmt"
	"runtime/debug"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// PersistentVolumeCollector collects metrics from Kubernetes persistent volumes.
type PersistentVolumeCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewPersistentVolumeCollector creates a new PersistentVolumeCollector.
func NewPersistentVolumeCollector(clientset *kubernetes.Clientset, cfg *config.Config) *PersistentVolumeCollector {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in NewPersistentVolumeCollector")
		}
	}()

	logrus.Debug("Creating new PersistentVolumeCollector")
	collector := &PersistentVolumeCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("PersistentVolumeCollector created successfully")
	return collector
}

// CollectMetrics collects metrics from Kubernetes persistent volumes.
func (pvc *PersistentVolumeCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in PersistentVolumeCollector.CollectMetrics")
		}
	}()

	pvs, err := pvc.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list persistent volumes")
		return []models.PVMetric{}, nil
	}

	logrus.WithField("count", len(pvs.Items)).Debug("Successfully listed persistent volumes")
	metrics := pvc.collectPVMetrics(pvs.Items)

	logrus.WithField("count", len(metrics)).Debug("Successfully collected persistent volume metrics")
	return metrics, nil
}

// collectPVMetrics collects metrics from Kubernetes persistent volumes.
func (pvc *PersistentVolumeCollector) collectPVMetrics(pvs []v1.PersistentVolume) []models.PVMetric {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":      r,
				"stacktrace": string(debug.Stack()),
			}).Error("Recovered from panic in collectPVMetrics")
		}
	}()

	pvMetrics := make([]models.PVMetric, 0, len(pvs))

	for _, pv := range pvs {
		logrus.WithFields(logrus.Fields{
			"pv": pv.Name,
		}).Debug("Processing persistent volume")

		if pv.Labels == nil {
			pv.Labels = make(map[string]string)
		}
		if pv.Annotations == nil {
			pv.Annotations = make(map[string]string)
		}

		// Safely collect access modes
		accessModes := make([]string, 0)
		for _, mode := range pv.Spec.AccessModes {
			accessModes = append(accessModes, string(mode))
		}

		// Safely handle volume mode
		var volumeMode string
		if pv.Spec.VolumeMode != nil {
			volumeMode = string(*pv.Spec.VolumeMode)
		}

		metric := models.PVMetric{
			Name:          pv.Name,
			Capacity:      pv.Spec.Capacity.Storage().Value(),
			Phase:         string(pv.Status.Phase),
			StorageClass:  pv.Spec.StorageClassName,
			Labels:        pv.Labels,
			AccessModes:   accessModes,
			ReclaimPolicy: string(pv.Spec.PersistentVolumeReclaimPolicy),
			VolumeMode:    volumeMode,
			Status: models.PVStatus{
				Phase:   string(pv.Status.Phase),
				Message: pv.Status.Message,
				Reason:  pv.Status.Reason,
			},
			MountOptions: pv.Spec.MountOptions,
		}

		// Safely get storage class details
		if pv.Spec.StorageClassName != "" {
			sc, err := pvc.clientset.StorageV1().StorageClasses().Get(
				context.Background(),
				pv.Spec.StorageClassName,
				metav1.GetOptions{},
			)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"pv":           pv.Name,
					"storageClass": pv.Spec.StorageClassName,
					"error":        err,
				}).Error("Failed to get storage class details")
			} else if sc.VolumeBindingMode != nil {
				metric.VolumeBindingMode = string(*sc.VolumeBindingMode)
			}
		}

		// Safely handle claim reference
		if pv.Spec.ClaimRef != nil {
			metric.BoundPVC = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}

		// Safely handle annotations
		if ann := pv.Annotations; ann != nil {
			if provisioner, ok := ann["pv.kubernetes.io/provisioned-by"]; ok {
				metric.StorageProvisioner = provisioner
			}
		}

		logrus.WithFields(logrus.Fields{
			"pv":           pv.Name,
			"phase":        metric.Phase,
			"capacity":     metric.Capacity,
			"storageClass": metric.StorageClass,
			"boundPVC":     metric.BoundPVC,
		}).Debug("Collected metrics for persistent volume")

		pvMetrics = append(pvMetrics, metric)
	}

	return pvMetrics
}
