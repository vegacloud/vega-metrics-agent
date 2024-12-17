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

// PersistentVolumeClaimCollector collects metrics from Kubernetes persistent volume claims.
type PersistentVolumeClaimCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewPersistentVolumeClaimCollector creates a new PersistentVolumeClaimCollector.
func NewPersistentVolumeClaimCollector(clientset *kubernetes.Clientset, cfg *config.Config) *PersistentVolumeClaimCollector {
	// logrus.Debug("Starting PersistentVolumeClaimCollector")
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
	logrus.Debug("PersistentVolumeClaimCollector created successfully")
	return &PersistentVolumeClaimCollector{
		clientset: clientset,
		config:    cfg,
	}

}

// CollectMetrics collects metrics from Kubernetes persistent volume claims.
func (pvcc *PersistentVolumeClaimCollector) CollectMetrics(ctx context.Context) (interface{}, error) {
	pvcs, err := pvcc.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volume claims: %w", err)
	}
	logrus.Debugf("Successfully listed %d persistent volume claims", len(pvcs.Items))

	metrics := pvcc.collectPVCMetrics(pvcs.Items)
	logrus.Debug("Successfully collected persistent volume claim metrics")
	return metrics, nil
}

// collectPVCMetrics collects metrics from Kubernetes persistent volume claims.
func (pvcc *PersistentVolumeClaimCollector) collectPVCMetrics(pvcs []v1.PersistentVolumeClaim) []models.PVCMetric {
	pvcMetrics := make([]models.PVCMetric, 0, len(pvcs))

	for _, claim := range pvcs {
		if claim.Labels == nil {
			claim.Labels = make(map[string]string)
		}

		accessModes := make([]string, 0)
		for _, mode := range claim.Spec.AccessModes {
			accessModes = append(accessModes, string(mode))
		}

		conditions := make([]models.PVCCondition, 0)
		for _, cond := range claim.Status.Conditions {
			conditions = append(conditions, models.PVCCondition{
				Type:               string(cond.Type),
				Status:             string(cond.Status),
				Reason:             cond.Reason,
				Message:            cond.Message,
				LastTransitionTime: &cond.LastTransitionTime.Time,
			})
		}

		metric := models.PVCMetric{
			Name:             claim.Name,
			Namespace:        claim.Namespace,
			Phase:            string(claim.Status.Phase),
			Capacity:         claim.Status.Capacity.Storage().Value(),
			RequestedStorage: claim.Spec.Resources.Requests.Storage().Value(),
			Labels:           claim.Labels,
			AccessModes:      accessModes,
			VolumeMode:       string(*claim.Spec.VolumeMode),
			VolumeName:       claim.Spec.VolumeName,
			Status: models.PVCStatus{
				Phase:      string(claim.Status.Phase),
				Conditions: conditions,
			},
		}

		if claim.Spec.StorageClassName != nil {
			metric.StorageClass = *claim.Spec.StorageClassName
			if sc, err := pvcc.clientset.StorageV1().StorageClasses().Get(context.Background(), *claim.Spec.StorageClassName, metav1.GetOptions{}); err == nil {
				metric.VolumeBindingMode = string(*sc.VolumeBindingMode)
			}
		}

		if claim.Spec.VolumeName != "" {
			metric.BoundPV = claim.Spec.VolumeName
			if boundPV, err := pvcc.clientset.CoreV1().PersistentVolumes().Get(context.Background(), claim.Spec.VolumeName, metav1.GetOptions{}); err == nil {
				metric.MountOptions = boundPV.Spec.MountOptions
			}
		}

		if ann := claim.Annotations; ann != nil {
			if provisioner, ok := ann["volume.kubernetes.io/storage-provisioner"]; ok {
				metric.StorageProvisioner = provisioner
			}
		}

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

		pvcMetrics = append(pvcMetrics, metric)
	}

	return pvcMetrics
}
