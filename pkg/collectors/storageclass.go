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
	// logrus.Debug("Starting StorageClassCollector")
	// if token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
	// 	clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport = &http.Transport{
	// 		TLSClientConfig: &tls.Config{
	// 			InsecureSkipVerify: cfg.VegaInsecure,
	// 		},
	// 	}
	// 	clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport = transport.NewBearerAuthRoundTripper(
	// 		string(token),
	// 		clientset.CoreV1().RESTClient().(*rest.RESTClient).Client.Transport,
	// 	)
	// }
	logrus.Debug("StorageClassCollector created successfully")
	return &StorageClassCollector{
		clientset: clientset,
		config:    cfg,
	}

}

// CollectMetrics collects metrics from Kubernetes storage classes
func (sc *StorageClassCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	storageClasses, err := sc.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list storage classes: %w", err)
	}

	metrics := make([]models.StorageClassMetrics, 0, len(storageClasses.Items))
	for _, storageClass := range storageClasses.Items {
		metrics = append(metrics, sc.parseStorageClassMetrics(ctx, storageClass))
	}

	logrus.Debugf("Successfully collected metrics for %d storage classes", len(metrics))
	return metrics, nil
}

func (sc *StorageClassCollector) parseStorageClassMetrics(ctx context.Context, storageClass storagev1.StorageClass) models.StorageClassMetrics {
	if storageClass.Labels == nil {
		storageClass.Labels = make(map[string]string)
	}
	if storageClass.Annotations == nil {
		storageClass.Annotations = make(map[string]string)
	}

	metrics := models.StorageClassMetrics{
		Name:                 storageClass.Name,
		Provisioner:          storageClass.Provisioner,
		ReclaimPolicy:        string(*storageClass.ReclaimPolicy),
		VolumeBindingMode:    string(*storageClass.VolumeBindingMode),
		AllowVolumeExpansion: storageClass.AllowVolumeExpansion != nil && *storageClass.AllowVolumeExpansion,
		Labels:               storageClass.Labels,
		Annotations:          storageClass.Annotations,
		Parameters:           storageClass.Parameters,
		MountOptions:         storageClass.MountOptions,
		CreationTimestamp:    &storageClass.CreationTimestamp.Time,
	}

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

	// Collect CSI driver metrics - continue even if this fails
	if csiDriver, err := sc.collectCSIDriverMetrics(ctx, storageClass.Provisioner); err != nil {
		logrus.Debugf("Storage class %s: CSI driver metrics collection skipped: %v", storageClass.Name, err)
	} else {
		metrics.CSIDriver = csiDriver
	}

	// Collect storage pool metrics - continue even if this fails
	if storagePools, err := sc.collectStoragePoolMetrics(ctx, storageClass.Name); err != nil {
		logrus.Debugf("Storage class %s: Storage pool metrics collection skipped: %v", storageClass.Name, err)
	} else {
		metrics.StoragePools = storagePools
	}

	// Collect capacity metrics - continue even if this fails
	if capacityMetrics, err := sc.collectCapacityMetrics(ctx, storageClass.Name); err != nil {
		logrus.Debugf("Storage class %s: Capacity metrics collection skipped: %v", storageClass.Name, err)
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
	csiDrivers, err := sc.clientset.StorageV1().CSIDrivers().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CSI drivers: %w", err)
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

			// Try to get CSI pods, but continue if we can't
			if pods, err := sc.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", driver.Name),
			}); err == nil {
				for _, pod := range pods.Items {
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
	pvs, err := sc.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return []models.StoragePoolMetrics{}, fmt.Errorf("failed to list PVs: %w", err)
	}

	poolMap := make(map[string]*models.StoragePoolMetrics)
	for _, pv := range pvs.Items {
		if pv.Spec.StorageClassName == storageClassName {
			poolName := pv.Labels["storage-pool"] // Adjust label key based on your CSI driver
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
	}

	pools := make([]models.StoragePoolMetrics, 0, len(poolMap))
	for _, pool := range poolMap {
		if pool.TotalCapacity > 0 {
			pool.UtilizationPct = float64(pool.UsedCapacity) / float64(pool.TotalCapacity) * 100
		}
		pool.AvailableSpace = pool.TotalCapacity - pool.UsedCapacity
		pools = append(pools, *pool)
	}

	return pools, nil
}

func (sc *StorageClassCollector) collectCapacityMetrics(ctx context.Context, storageClassName string) (*models.StorageClassMetrics, error) {
	metrics := &models.StorageClassMetrics{}

	// Get PVCs using this storage class
	pvcs, err := sc.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list PVCs: %w", err)
	}

	var totalRequested int64
	provisionedCount := 0
	for _, pvc := range pvcs.Items {
		if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == storageClassName {
			totalRequested += pvc.Spec.Resources.Requests.Storage().Value()
			if pvc.Status.Phase == v1.ClaimBound {
				provisionedCount++
			}
		}
	}

	metrics.AllocatedCapacity = totalRequested
	metrics.ProvisionedPVCs = utils.SafeInt32Conversion(provisionedCount)

	return metrics, nil
}
