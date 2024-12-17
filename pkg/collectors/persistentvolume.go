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
	// "crypto/tls"
	"fmt"
	// "net/http"
	// "os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	// "k8s.io/client-go/rest"
	// "k8s.io/client-go/transport"

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

	logrus.Debug("PersistentVolumeCollector created successfully")
	return &PersistentVolumeCollector{
		clientset: clientset,
		config:    cfg,
	}

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

		pvMetrics = append(pvMetrics, metric)
	}

	return pvMetrics
}
