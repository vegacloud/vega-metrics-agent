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

	snapshots "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

// PersistentVolumeCollector collects metrics from Kubernetes persistent volumes.
type PersistentVolumeCollector struct {
	clientset      *kubernetes.Clientset
	snapshotClient *snapshots.Clientset
	config         *config.Config
}

// NewPersistentVolumeCollector creates a new PersistentVolumeCollector.
func NewPersistentVolumeCollector(clientset *kubernetes.Clientset, cfg *config.Config) *PersistentVolumeCollector {
	collector := &PersistentVolumeCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("PersistentVolumeCollector created successfully")
	return collector
}

// CollectMetrics collects metrics from Kubernetes persistent volumes.
func (pvc *PersistentVolumeCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	pvs, err := pvc.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volumes: %w", err)
	}
	logrus.Debugf("Successfully listed %d persistent volumes", len(pvs.Items))

	metrics := pvc.collectPVMetrics(pvs.Items)
	logrus.Debug("Successfully collected persistent volume metrics")
	return metrics, nil
}

// collectPVMetrics collects metrics from Kubernetes persistent volumes.
func (pvc *PersistentVolumeCollector) collectPVMetrics(pvs []v1.PersistentVolume) []models.PVMetric {
	pvMetrics := make([]models.PVMetric, 0, len(pvs))

	for _, pv := range pvs {
		if pv.Labels == nil {
			pv.Labels = make(map[string]string)
		}

		accessModes := make([]string, 0)
		for _, mode := range pv.Spec.AccessModes {
			accessModes = append(accessModes, string(mode))
		}

		metric := models.PVMetric{
			Name:          pv.Name,
			Capacity:      pv.Spec.Capacity.Storage().Value(),
			Phase:         string(pv.Status.Phase),
			StorageClass:  pv.Spec.StorageClassName,
			Labels:        pv.Labels,
			AccessModes:   accessModes,
			ReclaimPolicy: string(pv.Spec.PersistentVolumeReclaimPolicy),
			VolumeMode:    string(*pv.Spec.VolumeMode),
			Status: models.PVStatus{
				Phase:   string(pv.Status.Phase),
				Message: pv.Status.Message,
				Reason:  pv.Status.Reason,
			},
			MountOptions: pv.Spec.MountOptions,
		}

		if sc, err := pvc.clientset.StorageV1().StorageClasses().Get(context.Background(), pv.Spec.StorageClassName, metav1.GetOptions{}); err == nil {
			metric.VolumeBindingMode = string(*sc.VolumeBindingMode)
		}

		if pv.Spec.ClaimRef != nil {
			metric.BoundPVC = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}

		if ann := pv.Annotations; ann != nil {
			if provisioner, ok := ann["pv.kubernetes.io/provisioned-by"]; ok {
				metric.StorageProvisioner = provisioner
			}
		}

		// Collect snapshot metrics
		snapshots, err := pvc.collectVolumeSnapshots(context.Background(), pv.Name)
		if err != nil {
			logrus.Warnf("Failed to collect snapshots for PV %s: %v", pv.Name, err)
		} else {
			metric.Snapshots = snapshots
		}

		pvMetrics = append(pvMetrics, metric)
	}

	return pvMetrics
}

func (pvc *PersistentVolumeCollector) collectVolumeSnapshots(ctx context.Context, pvName string) ([]models.VolumeSnapshotMetrics, error) {
	snapshots, err := pvc.snapshotClient.SnapshotV1().VolumeSnapshots("").List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("source-pv=%s", pvName),
	})
	if err != nil {
		return nil, err
	}

	var metrics []models.VolumeSnapshotMetrics
	for _, snap := range snapshots.Items {
		metric := models.VolumeSnapshotMetrics{
			Name:         snap.Name,
			Namespace:    snap.Namespace,
			SourcePVName: pvName,
			CreationTime: snap.CreationTimestamp.Time,
			ReadyToUse:   *snap.Status.ReadyToUse,
		}

		if snap.Status.RestoreSize != nil {
			metric.RestoreSize = snap.Status.RestoreSize.Value()
		}
		if snap.DeletionTimestamp != nil {
			metric.DeletionTime = &snap.DeletionTimestamp.Time
		}
		if snap.Status.Error != nil && snap.Status.Error.Message != nil {
			metric.Error = *snap.Status.Error.Message
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}
