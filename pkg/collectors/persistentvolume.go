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
package collectors

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
)

type PersistentVolumeCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewPersistentVolumeCollector(clientset *kubernetes.Clientset, cfg *config.Config) *PersistentVolumeCollector {
	collector := &PersistentVolumeCollector{
		clientset: clientset,
		config:    cfg,
	}
	logrus.Debug("PersistentVolumeCollector created successfully")
	return collector
}

func (pvc *PersistentVolumeCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	metrics, err := pvc.CollectPersistentVolumeMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Successfully collected persistent volume metrics")
	return metrics, nil
}

func (pvc *PersistentVolumeCollector) CollectPersistentVolumeMetrics(
	ctx context.Context,
) (*models.PersistentVolumeMetrics, error) {
	pvs, err := pvc.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volumes: %w", err)
	}
	logrus.Debugf("Successfully listed %d persistent volumes", len(pvs.Items))

	pvcs, err := pvc.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volume claims: %w", err)
	}
	logrus.Debugf("Successfully listed %d persistent volume claims", len(pvcs.Items))

	metrics := &models.PersistentVolumeMetrics{
		PVs:  pvc.collectPVMetrics(pvs.Items),
		PVCs: pvc.collectPVCMetrics(pvcs.Items),
	}

	logrus.Debug("Successfully calculated persistent volume metrics")
	return metrics, nil
}

func (pvc *PersistentVolumeCollector) collectPVMetrics(pvs []v1.PersistentVolume) []models.PVMetric {

	pvMetrics := make([]models.PVMetric, 0, len(pvs))

	for _, pv := range pvs {
		if pv.Labels == nil {
			pv.Labels = make(map[string]string)
		}
		metric := models.PVMetric{
			Name:         pv.Name,
			Capacity:     pv.Spec.Capacity.Storage().Value(),
			Phase:        string(pv.Status.Phase),
			StorageClass: pv.Spec.StorageClassName,
			Labels:       pv.Labels,
		}

		if pv.Spec.ClaimRef != nil {
			metric.BoundPVC = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}

		pvMetrics = append(pvMetrics, metric)
	}

	logrus.Debugf("Collected metrics for %d persistent volumes", len(pvMetrics))
	return pvMetrics
}

func (pvc *PersistentVolumeCollector) collectPVCMetrics(pvcs []v1.PersistentVolumeClaim) []models.PVCMetric {

	pvcMetrics := make([]models.PVCMetric, 0, len(pvcs))

	for _, pvc := range pvcs {
		if pvc.Labels == nil {
			pvc.Labels = make(map[string]string)
		}
		metric := models.PVCMetric{
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
			Phase:     string(pvc.Status.Phase),
			Capacity:  pvc.Status.Capacity.Storage().Value(),
			Labels:    pvc.Labels,
		}

		if pvc.Spec.StorageClassName != nil {
			metric.StorageClass = *pvc.Spec.StorageClassName
		}

		if pvc.Spec.VolumeName != "" {
			metric.BoundPV = pvc.Spec.VolumeName
		}

		pvcMetrics = append(pvcMetrics, metric)
	}

	logrus.Debugf("Collected metrics for %d persistent volume claims", len(pvcMetrics))
	return pvcMetrics
}
