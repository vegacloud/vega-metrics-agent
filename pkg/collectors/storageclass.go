// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package collectors

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/models"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
)

// StorageClassCollector collects metrics from Kubernetes storage classes
type StorageClassCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewStorageClassCollector creates a new StorageClassCollector
func NewStorageClassCollector(clientset *kubernetes.Clientset, cfg *config.Config) *StorageClassCollector {
	logrus.Debug("StorageClassCollector created successfully")
	return &StorageClassCollector{
		clientset: clientset,
		config:    cfg,
	}

}

// CollectMetrics collects metrics from Kubernetes storage classes
func (sc *StorageClassCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Recovered from panic in StorageClassCollector: %v", r)
		}
	}()

	storageClasses, err := sc.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list storage classes")
		return []models.StorageClassMetrics{}, nil
	}

	metrics := make([]models.StorageClassMetrics, 0, len(storageClasses.Items))
	for _, storageClass := range storageClasses.Items {
		metric := sc.parseStorageClassMetrics(ctx, storageClass)
		metrics = append(metrics, metric)
	}

	logrus.Debugf("Successfully collected metrics for %d storage classes", len(metrics))
	return metrics, nil
}

func (sc *StorageClassCollector) parseStorageClassMetrics(ctx context.Context, storageClass storagev1.StorageClass) models.StorageClassMetrics {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"storageClass": storageClass.Name,
				"panic":        r,
			}).Error("Recovered from panic while parsing storage class metrics")
		}
	}()

	if storageClass.Labels == nil {
		storageClass.Labels = make(map[string]string)
	}
	if storageClass.Annotations == nil {
		storageClass.Annotations = make(map[string]string)
	}

	metrics := models.StorageClassMetrics{
		Name:              storageClass.Name,
		Provisioner:       storageClass.Provisioner,
		ReclaimPolicy:     "", // Will be set below if not nil
		VolumeBindingMode: "", // Will be set below if not nil
		Labels:            storageClass.Labels,
		Annotations:       storageClass.Annotations,
		Parameters:        storageClass.Parameters,
		MountOptions:      storageClass.MountOptions,
		CreationTimestamp: &storageClass.CreationTimestamp.Time,
	}

	// Safely set ReclaimPolicy
	if storageClass.ReclaimPolicy != nil {
		metrics.ReclaimPolicy = string(*storageClass.ReclaimPolicy)
	}

	// Safely set VolumeBindingMode
	if storageClass.VolumeBindingMode != nil {
		metrics.VolumeBindingMode = string(*storageClass.VolumeBindingMode)
	}

	// Safely set AllowVolumeExpansion
	metrics.AllowVolumeExpansion = storageClass.AllowVolumeExpansion != nil && *storageClass.AllowVolumeExpansion

	// Check if this is the default storage class
	metrics.IsDefault = false
	for key, value := range storageClass.Annotations {
		if (key == "storageclass.kubernetes.io/is-default-class" ||
			key == "storageclass.beta.kubernetes.io/is-default-class") &&
			value == "true" {
			metrics.IsDefault = true
			break
		}
	}

	// Collect CSI driver metrics with better error logging
	if csiDriver, err := sc.collectCSIDriverMetrics(ctx, storageClass.Provisioner); err != nil {
		logrus.WithFields(logrus.Fields{
			"storageClass": storageClass.Name,
			"provisioner":  storageClass.Provisioner,
			"error":        err,
		}).Error("Failed to collect CSI driver metrics")
	} else {
		metrics.CSIDriver = csiDriver
	}

	// Collect storage pool metrics with better error logging
	if storagePools, err := sc.collectStoragePoolMetrics(ctx, storageClass.Name); err != nil {
		logrus.WithFields(logrus.Fields{
			"storageClass": storageClass.Name,
			"error":        err,
		}).Error("Failed to collect storage pool metrics")
	} else {
		metrics.StoragePools = storagePools
	}

	// Collect capacity metrics with better error logging
	if capacityMetrics, err := sc.collectCapacityMetrics(ctx, storageClass.Name); err != nil {
		logrus.WithFields(logrus.Fields{
			"storageClass": storageClass.Name,
			"error":        err,
		}).Error("Failed to collect capacity metrics")
	} else {
		metrics.TotalCapacity = capacityMetrics.TotalCapacity
		metrics.AllocatedCapacity = capacityMetrics.AllocatedCapacity
		metrics.AvailableCapacity = capacityMetrics.AvailableCapacity
		metrics.CapacityUtilization = capacityMetrics.CapacityUtilization
		metrics.ProvisionedPVCs = capacityMetrics.ProvisionedPVCs
		metrics.ProvisioningRate = capacityMetrics.ProvisioningRate
	}

	logrus.Debugf("Parsed metrics for storage class %s", storageClass.Name)
	return metrics
}

func (sc *StorageClassCollector) collectCSIDriverMetrics(ctx context.Context, provisioner string) (*models.CSIDriverMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"provisioner": provisioner,
				"panic":       r,
			}).Error("Recovered from panic while collecting CSI driver metrics")
		}
	}()

	csiDrivers, err := sc.clientset.StorageV1().CSIDrivers().List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("Failed to list CSI drivers: %v", err)
		return &models.CSIDriverMetrics{
			Name:      provisioner,
			Available: false,
		}, nil
	}

	for _, driver := range csiDrivers.Items {
		if driver.Name == provisioner {
			metrics := &models.CSIDriverMetrics{
				Name:               driver.Name,
				Available:          true,
				VolumeSnapshotting: driver.Spec.VolumeLifecycleModes != nil,
				VolumeCloning:      driver.Spec.VolumeLifecycleModes != nil,
				VolumeExpansion:    driver.Spec.RequiresRepublish != nil && *driver.Spec.RequiresRepublish,
				NodePluginPods:     make(map[string]string),
				ControllerPods:     make(map[string]string),
			}

			if pods, err := sc.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", driver.Name),
			}); err == nil {
				for _, pod := range pods.Items {
					if pod.Spec.NodeName == "" {
						continue
					}
					if strings.Contains(pod.Name, "node") {
						metrics.NodePluginPods[pod.Spec.NodeName] = string(pod.Status.Phase)
					} else if strings.Contains(pod.Name, "controller") {
						metrics.ControllerPods[pod.Namespace] = string(pod.Status.Phase)
					}
				}
			}

			return metrics, nil
		}
	}

	// Return empty metrics if CSI driver not found
	return &models.CSIDriverMetrics{
		Name:      provisioner,
		Available: false,
	}, nil
}

func (sc *StorageClassCollector) collectStoragePoolMetrics(ctx context.Context, storageClassName string) ([]models.StoragePoolMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"storageClass": storageClassName,
				"panic":        r,
				"stacktrace":   string(debug.Stack()),
			}).Error("Recovered from panic while collecting storage pool metrics")
		}
	}()

	pvs, err := sc.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"storageClass": storageClassName,
			"error":        err,
		}).Error("Failed to list PVs for storage pools")
		return []models.StoragePoolMetrics{}, nil
	}

	poolMap := make(map[string]*models.StoragePoolMetrics)
	for _, pv := range pvs.Items {
		if pv.Spec.StorageClassName != storageClassName {
			continue
		}

		if pv.Spec.CSI == nil {
			logrus.WithFields(logrus.Fields{
				"storageClass": storageClassName,
				"pvName":       pv.Name,
			}).Debug("Skipping PV without CSI spec")
			continue
		}

		poolName := pv.Labels["storage-pool"]
		if poolName == "" {
			poolName = "default"
		}

		if pool, exists := poolMap[poolName]; exists {
			pool.UsedCapacity += pv.Spec.Capacity.Storage().Value()
			pool.VolumeCount++
		} else {
			poolMap[poolName] = &models.StoragePoolMetrics{
				Name:         poolName,
				Provider:     pv.Spec.CSI.Driver,
				StorageClass: storageClassName,
				UsedCapacity: pv.Spec.Capacity.Storage().Value(),
				VolumeCount:  1,
			}
		}
	}

	pools := make([]models.StoragePoolMetrics, 0, len(poolMap))
	for _, pool := range poolMap {
		if pool.TotalCapacity > 0 {
			pool.UtilizationPct = float64(pool.UsedCapacity) / float64(pool.TotalCapacity) * 100
			pool.AvailableSpace = pool.TotalCapacity - pool.UsedCapacity
		} else {
			logrus.WithFields(logrus.Fields{
				"storageClass": storageClassName,
				"poolName":     pool.Name,
			}).Debug("Pool has zero total capacity")
			pool.UtilizationPct = 0
			pool.AvailableSpace = 0
		}
		pools = append(pools, *pool)
	}

	return pools, nil
}

func (sc *StorageClassCollector) collectCapacityMetrics(ctx context.Context, storageClassName string) (*models.StorageClassMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"storageClass": storageClassName,
				"panic":        r,
				"stacktrace":   string(debug.Stack()),
			}).Error("Recovered from panic while collecting capacity metrics")
		}
	}()

	metrics := &models.StorageClassMetrics{}

	pvcs, err := sc.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"storageClass": storageClassName,
			"error":        err,
		}).Error("Failed to list PVCs for capacity metrics")
		return &models.StorageClassMetrics{}, nil
	}

	var totalRequested int64
	provisionedCount := 0
	for _, pvc := range pvcs.Items {
		if pvc.Spec.StorageClassName == nil {
			logrus.WithFields(logrus.Fields{
				"pvcName":      pvc.Name,
				"pvcNamespace": pvc.Namespace,
			}).Debug("Skipping PVC without storage class name")
			continue
		}

		if *pvc.Spec.StorageClassName == storageClassName {
			if pvc.Spec.Resources.Requests != nil {
				storage := pvc.Spec.Resources.Requests.Storage()
				if storage != nil {
					totalRequested += storage.Value()
				} else {
					logrus.WithFields(logrus.Fields{
						"pvcName":      pvc.Name,
						"pvcNamespace": pvc.Namespace,
						"storageClass": storageClassName,
					}).Debug("PVC has nil storage request")
				}
			}

			if pvc.Status.Phase == v1.ClaimBound {
				provisionedCount++
			}
		}
	}

	metrics.AllocatedCapacity = totalRequested
	metrics.ProvisionedPVCs = utils.SafeInt32Conversion(provisionedCount)

	logrus.WithFields(logrus.Fields{
		"storageClass":     storageClassName,
		"totalRequested":   totalRequested,
		"provisionedCount": provisionedCount,
	}).Debug("Collected capacity metrics")

	return metrics, nil
}
