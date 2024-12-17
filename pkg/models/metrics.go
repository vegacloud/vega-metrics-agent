// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// Package models provides the data models for the metrics API
// File: pkg/models/metrics.go

// Package models provides the data models for the metrics API
package models

import (
	"time"
)

// EnhancedNodeMetrics represents detailed metrics for a single node
type EnhancedNodeMetrics struct {
	Name              string                      `json:"name"`
	Capacity          ResourceMetrics             `json:"capacity"`
	Allocatable       ResourceMetrics             `json:"allocatable"`
	Usage             ResourceMetrics             `json:"usage"`
	Conditions        NodeConditions              `json:"conditions"`
	Labels            map[string]string           `json:"labels"`
	Annotations       map[string]string           `json:"annotations"`
	DetailedMetrics   NodeDetailedMetrics         `json:"detailedMetrics"`
	HardwareTopology  *HardwareTopology           `json:"hardwareTopology,omitempty"`
	PowerMetrics      *PowerMetrics               `json:"powerMetrics,omitempty"`
	Taints            []NodeTaint                 `json:"taints,omitempty"`
	Lease             *NodeLease                  `json:"lease,omitempty"`
	ExtendedResources map[string]ExtendedResource `json:"extendedResources,omitempty"`
	NodeMetrics       NodeMetrics                 `json:"nodeMetrics"`
	CPU               CPUMetrics                  `json:"cpu,omitempty"`
	Memory            MemoryMetrics               `json:"memory,omitempty"`
	Network           NetworkMetrics              `json:"network,omitempty"`
	Disk              DiskMetrics                 `json:"disk,omitempty"`
	Filesystem        FSMetrics                   `json:"filesystem,omitempty"`
	VolumeHealth      []VolumeHealthMetrics       `json:"volumeHealth,omitempty"`
	VolumeAttachments []VolumeAttachmentMetrics   `json:"volumeAttachments,omitempty"`
	NodeInfo          NodeInfo                    `json:"nodeInfo"` // Add this line
}

// EnhancedPodMetrics represents detailed metrics for a single pod
type EnhancedPodMetrics struct {
	PodMetrics          PodMetrics                  `json:"podMetrics"`
	QoSClass            string                      `json:"qosClass"`
	Conditions          PodConditions               `json:"conditions"`
	Requests            ResourceMetrics             `json:"requests"`
	Limits              ResourceMetrics             `json:"limits"`
	Containers          []ContainerMetrics          `json:"containers"`
	TotalRestarts       int32                       `json:"totalRestarts"`
	StartTime           *time.Time                  `json:"startTime,omitempty"`
	CompletionTime      *time.Time                  `json:"completionTime,omitempty"`
	Priority            *int32                      `json:"priority,omitempty"`
	PriorityClassName   string                      `json:"priorityClassName,omitempty"`
	NodeName            string                      `json:"nodeName"`
	HostIP              string                      `json:"hostIP"`
	PodIPs              []string                    `json:"podIPs"`
	NominatedNodeName   string                      `json:"nominatedNodeName,omitempty"`
	ReadinessGates      []PodReadinessGate          `json:"readinessGates,omitempty"`
	Annotations         map[string]string           `json:"annotations"`
	VolumeMounts        []VolumeMountMetrics        `json:"volumeMounts"`
	ImagePullPolicy     string                      `json:"imagePullPolicy"`
	ServiceAccountName  string                      `json:"serviceAccountName"`
	DisruptionBudget    *PodDisruptionBudgetMetrics `json:"disruptionBudget,omitempty"`
	TopologySpread      []TopologySpreadConstraint  `json:"topologySpread,omitempty"`
	Overhead            *PodOverheadMetrics         `json:"overhead,omitempty"`
	SchedulingGates     []PodSchedulingGate         `json:"schedulingGates,omitempty"`
	SecurityContext     *SecurityContextMetrics     `json:"securityContext,omitempty"`
	Affinity            *AffinityMetrics            `json:"affinity,omitempty"`
	InitContainers      []InitContainerMetrics      `json:"initContainers,omitempty"`
	EphemeralContainers []EphemeralContainerMetrics `json:"ephemeralContainers,omitempty"`
	QoSDetails          *QoSMetrics                 `json:"qosDetails,omitempty"`
}

// NodeMetrics represents the metrics for a single node
type NodeMetrics struct {
	Name              string                    `json:"name"`
	Capacity          ResourceMetrics           `json:"capacity"`
	Allocatable       ResourceMetrics           `json:"allocatable"`
	Usage             ResourceMetrics           `json:"usage"`
	CPU               CPUMetrics                `json:"cpu,omitempty"`
	Memory            MemoryMetrics             `json:"memory,omitempty"`
	Network           NetworkMetrics            `json:"network,omitempty"`
	Disk              DiskMetrics               `json:"disk,omitempty"`
	Filesystem        FSMetrics                 `json:"filesystem,omitempty"`
	VolumeHealth      []VolumeHealthMetrics     `json:"volumeHealth,omitempty"`
	VolumeAttachments []VolumeAttachmentMetrics `json:"volumeAttachments,omitempty"`
}

// ResourceMetrics represents resource usage or capacity based on kubelet /metrics/resource
type ResourceMetrics struct {
	// CPU usage in millicores (m)
	CPU int64 `json:"cpu"`
	// Memory usage in bytes
	Memory int64 `json:"memory"`
	// Storage usage in bytes
	Storage int64 `json:"storage,omitempty"`
	// Ephemeral storage usage in bytes
	EphemeralStorage int64 `json:"ephemeralStorage,omitempty"`
	// Number of pods
	Pods int64 `json:"pods,omitempty"`
	// GPU devices if available
	GPUDevices []GPUMetrics `json:"gpuDevices,omitempty"`
}

// NodeConditions represents the conditions of a node
type NodeConditions struct {
	Ready          bool `json:"ready"`
	MemoryPressure bool `json:"memoryPressure"`
	DiskPressure   bool `json:"diskPressure"`
	PIDPressure    bool `json:"pidPressure"`
}

// PodMetrics represents the metrics for a single pod
type PodMetrics struct {
	Name          string             `json:"name"`
	Namespace     string             `json:"namespace"`
	Phase         string             `json:"phase"`
	QoSClass      string             `json:"qosClass"`
	Requests      ResourceMetrics    `json:"requests"`
	Limits        ResourceMetrics    `json:"limits"`
	Usage         ResourceMetrics    `json:"usage"`
	Containers    []ContainerMetrics `json:"containers"`
	Conditions    PodConditions      `json:"conditions"`
	TotalRestarts int32              `json:"totalRestarts"`
	Labels        map[string]string  `json:"labels"`
}

// PodConditions represents the conditions of a pod
type PodConditions struct {
	PodScheduled    bool `json:"podScheduled"`
	Initialized     bool `json:"initialized"`
	ContainersReady bool `json:"containersReady"`
	Ready           bool `json:"ready"`
}

// ClusterMetrics represents cluster-wide metrics
type ClusterMetrics struct {
	TotalCapacity     ResourceMetrics              `json:"totalCapacity"`
	TotalAllocatable  ResourceMetrics              `json:"totalAllocatable"`
	TotalRequests     ResourceMetrics              `json:"totalRequests"`
	TotalLimits       ResourceMetrics              `json:"totalLimits"`
	TotalUsage        ResourceMetrics              `json:"totalUsage"`
	NodeCount         int                          `json:"nodeCount"`
	PodCount          int                          `json:"podCount"`
	ContainerCount    int                          `json:"containerCount"`
	KubernetesVersion string                       `json:"kubernetesVersion"`
	NodeLabels        map[string]map[string]string `json:"nodeLabels"` // New field for node labels
	PodLabels         map[string]map[string]string `json:"podLabels"`  // New field for pod labels
	// ContainerLabels  map[string]map[string]string `json:"containerLabels"`
}

// PersistentVolumeMetrics represents metrics for persistent volumes
// Deprecated: Use separate PVMetric and PVCMetric instead
// type PersistentVolumeMetrics struct {
//     PVs  []PVMetric  `json:"pvs"`
//     PVCs []PVCMetric `json:"pvcs"`
// }

// PVMetric represents metrics for a single persistent volume
type PVMetric struct {
	Name               string                  `json:"name"`
	Capacity           int64                   `json:"capacity"` // in bytes
	Phase              string                  `json:"phase"`
	StorageClass       string                  `json:"storageClass"`
	BoundPVC           string                  `json:"boundPVC,omitempty"`
	Labels             map[string]string       `json:"labels"`
	AccessModes        []string                `json:"accessModes"`
	ReclaimPolicy      string                  `json:"reclaimPolicy"`
	VolumeMode         string                  `json:"volumeMode"`
	StorageProvisioner string                  `json:"storageProvisioner,omitempty"`
	MountPoint         string                  `json:"mountPoint,omitempty"`
	Status             PVStatus                `json:"status"`
	MountOptions       []string                `json:"mountOptions,omitempty"`
	VolumeBindingMode  string                  `json:"volumeBindingMode,omitempty"`
	Snapshots          []VolumeSnapshotMetrics `json:"snapshots,omitempty"`
}

// PVCMetric represents metrics for a single persistent volume claim
type PVCMetric struct {
	Name               string                  `json:"name"`
	Namespace          string                  `json:"namespace"`
	Phase              string                  `json:"phase"`
	Capacity           int64                   `json:"capacity"`         // in bytes
	RequestedStorage   int64                   `json:"requestedStorage"` // in bytes
	StorageClass       string                  `json:"storageClass"`
	BoundPV            string                  `json:"boundPV,omitempty"`
	Labels             map[string]string       `json:"labels"`
	AccessModes        []string                `json:"accessModes"`
	VolumeMode         string                  `json:"volumeMode"`
	Status             PVCStatus               `json:"status"`
	VolumeName         string                  `json:"volumeName,omitempty"`
	StorageProvisioner string                  `json:"storageProvisioner,omitempty"`
	MountOptions       []string                `json:"mountOptions,omitempty"`
	VolumeBindingMode  string                  `json:"volumeBindingMode,omitempty"`
	Expansion          *VolumeExpansionMetrics `json:"expansion,omitempty"`
}

// PVStatus represents the status of a persistent volume
type PVStatus struct {
	Phase               string     `json:"phase"`
	Message             string     `json:"message,omitempty"`
	Reason              string     `json:"reason,omitempty"`
	LastPhaseTransition *time.Time `json:"lastPhaseTransition,omitempty"`
}

// PVCStatus represents the status of a persistent volume claim
type PVCStatus struct {
	Phase               string         `json:"phase"`
	Message             string         `json:"message,omitempty"`
	Reason              string         `json:"reason,omitempty"`
	LastPhaseTransition *time.Time     `json:"lastPhaseTransition,omitempty"`
	Conditions          []PVCCondition `json:"conditions"`
}

// PVCCondition represents a condition of a PVC
type PVCCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
}

// NamespaceMetrics represents metrics for a single namespace
type NamespaceMetrics struct {
	Name              string                 `json:"name"`
	Status            string                 `json:"status"`
	Phase             string                 `json:"phase"`
	CreationTimestamp time.Time              `json:"creationTimestamp"`
	DeletionTimestamp *time.Time             `json:"deletionTimestamp,omitempty"`
	Finalizers        []string               `json:"finalizers"`
	Conditions        []NamespaceCondition   `json:"conditions"`
	ResourceQuotas    []ResourceQuotaMetrics `json:"resourceQuotas"`
	LimitRanges       []LimitRangeMetrics    `json:"limitRanges"`
	Usage             ResourceUsage          `json:"usage"`
	Labels            map[string]string      `json:"labels"`
	Annotations       map[string]string      `json:"annotations"`
}

// NamespaceCondition represents the current condition of a namespace
type NamespaceCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
}

// ResourceQuotaMetrics represents metrics for a resource quota
type ResourceQuotaMetrics struct {
	Name      string           `json:"name"`
	Resources []ResourceMetric `json:"resources"`

	// New detailed quota fields
	Scopes          []QuotaScopeMetrics         `json:"scopes,omitempty"`
	PriorityQuotas  []PriorityClassQuotaMetrics `json:"priorityQuotas,omitempty"`
	StatusHistory   []QuotaStatusHistory        `json:"statusHistory,omitempty"`
	LastUpdateTime  time.Time                   `json:"lastUpdateTime,omitempty"`
	EnforcementMode string                      `json:"enforcementMode,omitempty"`
}

// ResourceMetric represents a single resource metric
type ResourceMetric struct {
	ResourceName string `json:"resourceName"`
	Hard         string `json:"hard"`
	Used         string `json:"used"`
}

// LimitRangeMetrics represents metrics for a limit range
type LimitRangeMetrics struct {
	Name   string           `json:"name"`
	Limits []LimitRangeItem `json:"limits"`
}

// LimitRangeItem represents a single limit range item
type LimitRangeItem struct {
	Type           string            `json:"type"`
	Max            map[string]string `json:"max,omitempty"`
	Min            map[string]string `json:"min,omitempty"`
	Default        map[string]string `json:"default,omitempty"`
	DefaultRequest map[string]string `json:"defaultRequest,omitempty"`
}

// ResourceUsage represents resource usage for a namespace
type ResourceUsage struct {
	CPU     int64 `json:"cpu"`     // in millicores
	Memory  int64 `json:"memory"`  // in bytes
	Storage int64 `json:"storage"` // in bytes
	Pods    int64 `json:"pods"`
}

// WorkloadMetrics represents metrics for various workloads
type WorkloadMetrics struct {
	Deployments  []DeploymentMetrics  `json:"deployments"`
	StatefulSets []StatefulSetMetrics `json:"statefulSets"`
	DaemonSets   []DaemonSetMetrics   `json:"daemonSets"`
	Jobs         []JobMetrics         `json:"jobs"`
}

// DeploymentMetrics represents metrics for a Kubernetes Deployment
type DeploymentMetrics struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Labels             map[string]string `json:"labels"`
	Replicas           int32             `json:"replicas"`
	ReadyReplicas      int32             `json:"readyReplicas"`
	UpdatedReplicas    int32             `json:"updatedReplicas"`
	AvailableReplicas  int32             `json:"availableReplicas"`
	CollisionCount     *int32            `json:"collisionCount"`
	Conditions         []string          `json:"conditions"`
	Generation         int64             `json:"generation"`
	ObservedGeneration int64             `json:"observedGeneration"`
}

// StatefulSetMetrics represents metrics for a Kubernetes StatefulSet
type StatefulSetMetrics struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Labels             map[string]string `json:"labels"`
	Replicas           int32             `json:"replicas"`
	ReadyReplicas      int32             `json:"readyReplicas"`
	CurrentReplicas    int32             `json:"currentReplicas"`
	UpdatedReplicas    int32             `json:"updatedReplicas"`
	AvailableReplicas  int32             `json:"availableReplicas"`
	CollisionCount     *int32            `json:"collisionCount"`
	Conditions         []string          `json:"conditions"`
	Generation         int64             `json:"generation"`
	ObservedGeneration int64             `json:"observedGeneration"`
}

// DaemonSetMetrics represents metrics for a Kubernetes DaemonSet
type DaemonSetMetrics struct {
	Name                   string               `json:"name"`
	Namespace              string               `json:"namespace"`
	Labels                 map[string]string    `json:"labels"`
	DesiredNumberScheduled int32                `json:"desiredNumberScheduled"`
	CurrentNumberScheduled int32                `json:"currentNumberScheduled"`
	NumberReady            int32                `json:"numberReady"`
	UpdatedNumberScheduled int32                `json:"updatedNumberScheduled"`
	NumberAvailable        int32                `json:"numberAvailable"`
	NumberUnavailable      int32                `json:"numberUnavailable"`
	NumberMisscheduled     int32                `json:"numberMisscheduled"`
	Generation             int64                `json:"generation"`
	ObservedGeneration     int64                `json:"observedGeneration"`
	Conditions             []DaemonSetCondition `json:"conditions"`
	Status                 DaemonSetStatus      `json:"status"`
	CreationTimestamp      *time.Time           `json:"creationTimestamp"`
	CollisionCount         *int32               `json:"collisionCount"`
	Annotations            map[string]string    `json:"annotations"`
}

// JobMetrics represents metrics for a Kubernetes Job
type JobMetrics struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Labels             map[string]string `json:"labels"`
	Active             int32             `json:"active"`
	Succeeded          int32             `json:"succeeded"`
	Failed             int32             `json:"failed"`
	Status             string            `json:"status"`
	StartTime          *time.Time        `json:"startTime,omitempty"`
	CompletionTime     *time.Time        `json:"completionTime,omitempty"`
	Duration           *time.Duration    `json:"duration,omitempty"`
	ResourceMetrics    ResourceMetrics   `json:"resourceMetrics"`
	CompletedIndexes   string            `json:"completedIndexes,omitempty"`
	Conditions         []JobCondition    `json:"conditions"`
	Generation         int64             `json:"generation"`
	ObservedGeneration int64             `json:"observedGeneration"`
	Suspended          bool              `json:"suspended"`
	CreationTime       *time.Time        `json:"creationTime"`
	Parallelism        *int32            `json:"parallelism,omitempty"`
	Completions        *int32            `json:"completions,omitempty"`
	BackoffLimit       *int32            `json:"backoffLimit,omitempty"`
	Resources          ResourceMetrics   `json:"resources,omitempty"`
}

// NetworkingMetrics represents metrics for networking resources
type NetworkingMetrics struct {
	Services        []ServiceMetrics       `json:"services"`
	Ingresses       []IngressMetrics       `json:"ingresses"`
	NetworkPolicies []NetworkPolicyMetrics `json:"networkPolicies"`
}

// ServiceMetrics represents metrics for a service
type ServiceMetrics struct {
	Name                  string            `json:"name"`
	Namespace             string            `json:"namespace"`
	Type                  string            `json:"type"`
	ClusterIP             string            `json:"clusterIP"`
	ExternalIP            []string          `json:"externalIP,omitempty"`
	LoadBalancerIP        string            `json:"loadBalancerIP,omitempty"`
	LoadBalancerIngress   []string          `json:"loadBalancerIngress,omitempty"`
	Ports                 []ServicePort     `json:"ports"`
	Selector              map[string]string `json:"selector,omitempty"`
	Labels                map[string]string `json:"labels"`
	SessionAffinity       string            `json:"sessionAffinity,omitempty"`
	ExternalTrafficPolicy string            `json:"externalTrafficPolicy,omitempty"`
	HealthCheckNodePort   int32             `json:"healthCheckNodePort,omitempty"`
	IPFamilies            []string          `json:"ipFamilies,omitempty"`
	IPFamilyPolicy        string            `json:"ipFamilyPolicy,omitempty"`
	Status                ServiceStatus     `json:"status"`
	Annotations           map[string]string `json:"annotations"`
}

// ServicePort represents a port exposed by a service
type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort string `json:"targetPort"`
	NodePort   int32  `json:"nodePort,omitempty"`
}

// IngressMetrics represents metrics for an ingress
type IngressMetrics struct {
	Name                string            `json:"name"`
	Namespace           string            `json:"namespace"`
	ClassName           string            `json:"className,omitempty"`
	Rules               []IngressRule     `json:"rules"`
	TLS                 []IngressTLS      `json:"tls,omitempty"`
	LoadBalancerIngress []string          `json:"loadBalancerIngress,omitempty"`
	Labels              map[string]string `json:"labels"`
	Annotations         map[string]string `json:"annotations"`
	Status              IngressStatus     `json:"status"`
	CreationTimestamp   *time.Time        `json:"creationTimestamp"`
}

// IngressRule represents a rule in an ingress
type IngressRule struct {
	Host  string        `json:"host,omitempty"`
	Paths []IngressPath `json:"paths"`
}

// IngressPath represents a path in an ingress rule
type IngressPath struct {
	Path     string         `json:"path"`
	PathType string         `json:"pathType"`
	Backend  IngressBackend `json:"backend"`
}

// IngressBackend represents a backend for an ingress path
type IngressBackend struct {
	Service IngressServiceBackend `json:"service"`
}

// IngressServiceBackend represents a service backend for an ingress
type IngressServiceBackend struct {
	Name string `json:"name"`
	Port int32  `json:"port"`
}

// IngressTLS represents TLS configuration for an ingress
type IngressTLS struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secretName"`
}

// CronJobMetrics represents metrics for a CronJob
type CronJobMetrics struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Schedule   string            `json:"schedule"`
	Labels     map[string]string `json:"labels"`
	Status     CronJobStatus     `json:"status"`
	Spec       CronJobSpec       `json:"spec"`
	JobMetrics []JobMetrics      `json:"jobs,omitempty"`
}

// CronJobStatus represents the status of a CronJob
type CronJobStatus struct {
	LastScheduleTime   *time.Time `json:"lastScheduleTime,omitempty"`
	LastSuccessfulTime *time.Time `json:"lastSuccessfulTime,omitempty"`
	NextScheduledTime  *time.Time `json:"nextScheduledTime,omitempty"`
	Active             int        `json:"active"`
	SuccessRate        float64    `json:"successRate"`
}

// CronJobSpec represents the specification of a CronJob
type CronJobSpec struct {
	Suspend                    bool   `json:"suspend"`
	Concurrency                string `json:"concurrencyPolicy"`
	StartingDeadlineSeconds    int64  `json:"startingDeadlineSeconds"`
	SuccessfulJobsHistoryLimit int32  `json:"successfulJobsHistoryLimit"`
	FailedJobsHistoryLimit     int32  `json:"failedJobsHistoryLimit"`
}

// HPAMetrics represents metrics for a horizontal pod autoscaler
type HPAMetrics struct {
	Name                            string            `json:"name"`
	Namespace                       string            `json:"namespace"`
	ScaleTargetRef                  ScaleTargetRef    `json:"scaleTargetRef"`
	MinReplicas                     *int32            `json:"minReplicas,omitempty"`
	MaxReplicas                     int32             `json:"maxReplicas"`
	CurrentReplicas                 int32             `json:"currentReplicas"`
	DesiredReplicas                 int32             `json:"desiredReplicas"`
	CurrentCPUUtilizationPercentage *int32            `json:"currentCPUUtilizationPercentage,omitempty"`
	TargetCPUUtilizationPercentage  *int32            `json:"targetCPUUtilizationPercentage,omitempty"`
	LastScaleTime                   *time.Time        `json:"lastScaleTime,omitempty"`
	ObservedGeneration              *int64            `json:"observedGeneration,omitempty"`
	Status                          HPAStatus         `json:"status"`
	Labels                          map[string]string `json:"labels"`
	Annotations                     map[string]string `json:"annotations"`
}

// ScaleTargetRef represents the target reference for scaling
type ScaleTargetRef struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion"`
}

// HPAStatus represents the status of the HPA
type HPAStatus struct {
	CurrentReplicas                 int32      `json:"currentReplicas"`
	DesiredReplicas                 int32      `json:"desiredReplicas"`
	CurrentCPUUtilizationPercentage *int32     `json:"currentCPUUtilizationPercentage,omitempty"`
	LastScaleTime                   *time.Time `json:"lastScaleTime,omitempty"`
}

// ReplicaSetMetrics represents metrics for a replica set
type ReplicaSetMetrics struct {
	Name                 string            `json:"name"`
	Namespace            string            `json:"namespace"`
	Replicas             int32             `json:"replicas"`
	ReadyReplicas        int32             `json:"readyReplicas"`
	AvailableReplicas    int32             `json:"availableReplicas"`
	CurrentReplicas      int32             `json:"currentReplicas"`
	FullyLabeledReplicas int32             `json:"fullyLabeledReplicas"`
	ObservedGeneration   int64             `json:"observedGeneration"`
	Conditions           []RSCondition     `json:"conditions"`
	Labels               map[string]string `json:"labels"`
	Annotations          map[string]string `json:"annotations"`
	CreationTimestamp    *time.Time        `json:"creationTimestamp"`
}

// RSCondition represents a condition of a ReplicaSet
type RSCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
}

// ReplicationControllerMetrics represents metrics for a replication controller
type ReplicationControllerMetrics struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"readyReplicas"`
	AvailableReplicas int32             `json:"availableReplicas"`
	Labels            map[string]string `json:"labels"`
	// Adding missing fields
	ObservedGeneration   int64             `json:"observedGeneration"`
	FullyLabeledReplicas int32             `json:"fullyLabeledReplicas"`
	Conditions           []RCCondition     `json:"conditions"`
	Annotations          map[string]string `json:"annotations"`
	CreationTimestamp    *time.Time        `json:"creationTimestamp"`
}

// RCCondition represents a condition of a ReplicationController
type RCCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
}

// NodeDetailedMetrics represents detailed metrics collected for a node
type NodeDetailedMetrics struct {
	CPU        CPUMetrics     `json:"cpu"`
	Memory     MemoryMetrics  `json:"memory"`
	Network    NetworkMetrics `json:"network"`
	DiskIO     DiskMetrics    `json:"diskIO"`
	Filesystem FSMetrics      `json:"filesystem"`
	Process    ProcessMetrics `json:"process"`
	Runtime    RuntimeMetrics `json:"runtime"`
	System     SystemMetrics  `json:"system"`
	Volume     VolumeMetrics  `json:"volume"`
	Container  ContainerStats `json:"container"`
}

// CPUMetrics represents CPU metrics from cadvisor
type CPUMetrics struct {
	// Usage values in nanoseconds
	UsageTotal     uint64            `json:"usageTotal"`
	UsageUser      uint64            `json:"usageUser"`
	UsageSystem    uint64            `json:"usageSystem"`
	SystemTime     uint64            `json:"systemTime"`
	UserTime       uint64            `json:"userTime"`
	LoadAverage    LoadAverageStats  `json:"loadAverage"`
	SchedulerStats SchedulerMetrics  `json:"schedulerStats"`
	Throttling     ThrottlingMetrics `json:"throttling"`
	// Per-core CPU metrics
	PerCoreUsage map[string]uint64 `json:"perCoreUsage,omitempty"`

	// Add new fields for detailed CPU metrics
	CPUPressure float64    `json:"cpuPressure,omitempty"`
	SchedStats  SchedStats `json:"schedStats,omitempty"`
}

// LoadAverageStats represents CPU load averages
type LoadAverageStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// SchedulerMetrics represents CPU scheduler statistics
type SchedulerMetrics struct {
	RunQueueLength   uint64 `json:"runQueueLength"`
	ContextSwitches  uint64 `json:"contextSwitches"`
	ProcessesCreated uint64 `json:"processesCreated"`
	ProcessesRunning uint64 `json:"processesRunning"`
	ProcessesBlocked uint64 `json:"processesBlocked"`
}

// ThrottlingMetrics represents CPU throttling information
type ThrottlingMetrics struct {
	Periods          uint64 `json:"periods"`
	ThrottledPeriods uint64 `json:"throttledPeriods"`
	ThrottledTime    uint64 `json:"throttledTime"`
}

// MemoryMetrics represents memory statistics
type MemoryMetrics struct {
	// All values in bytes
	Total           uint64  `json:"total"`
	Available       uint64  `json:"available"`
	Used            uint64  `json:"used"`
	WorkingSet      uint64  `json:"workingSet"`
	RSS             uint64  `json:"rss"`
	Cache           uint64  `json:"cache"`
	Swap            uint64  `json:"swap"`
	PageFaults      uint64  `json:"pageFaults"`
	KernelUsage     uint64  `json:"kernelUsage,omitempty"`
	OOMEvents       uint64  `json:"oomEvents"`
	OOMKills        uint64  `json:"oomKills"`
	PressureLevel   float64 `json:"pressureLevel"`
	SwapInBytes     uint64  `json:"swapInBytes"`
	SwapOutBytes    uint64  `json:"swapOutBytes"`
	MajorPageFaults uint64  `json:"majorPageFaults"`
	MinorPageFaults uint64  `json:"minorPageFaults"`
}

// NetworkMetrics represents network statistics
// Note: Metric availability depends on CNI plugin and network configuration
type NetworkMetrics struct {
	// Basic network metrics - generally available
	Summary NetworkSummary `json:"summary"`

	// Detailed interface metrics - availability varies by platform and CNI
	Interfaces []InterfaceStats `json:"interfaces,omitempty"`
}

// InterfaceStats represents per-interface network statistics
type InterfaceStats struct {
	InterfaceName string `json:"interfaceName"`
	RxBytes       uint64 `json:"rxBytes"`
	TxBytes       uint64 `json:"txBytes"`
	RxPackets     uint64 `json:"rxPackets"`
	TxPackets     uint64 `json:"txPackets"`
	RxErrors      uint64 `json:"rxErrors"`
	TxErrors      uint64 `json:"txErrors"`
	RxDropped     uint64 `json:"rxDropped"`
	TxDropped     uint64 `json:"txDropped"`
}

// NetworkSummary represents aggregated network statistics
type NetworkSummary struct {
	RxBytesTotal          uint64
	TxBytesTotal          uint64
	ContainerRxBytesTotal uint64
	ContainerTxBytesTotal uint64
}

// FSMetrics represents filesystem metrics
type FSMetrics struct {
	// Capacity and usage information
	TotalBytes     uint64 `json:"totalBytes"`
	UsedBytes      uint64 `json:"usedBytes"`
	AvailableBytes uint64 `json:"availableBytes"`
	// Inode information
	TotalInodes uint64 `json:"totalInodes"`
	UsedInodes  uint64 `json:"usedInodes"`
	FreeInodes  uint64 `json:"freeInodes"`
	// Per-device statistics
	DeviceStats map[string]FSStats `json:"deviceStats"`
	// Add inode metrics
	Inodes InodeStats `json:"inodes"`
}

// FSStats represents per-device filesystem statistics
type FSStats struct {
	Major       uint64 `json:"major"`
	Minor       uint64 `json:"minor"`
	ReadOps     uint64 `json:"readOps"`
	WriteOps    uint64 `json:"writeOps"`
	ReadBytes   uint64 `json:"readBytes"`
	WriteBytes  uint64 `json:"writeBytes"`
	ReadTimeMs  uint64 `json:"readTimeMs"`
	WriteTimeMs uint64 `json:"writeTimeMs"`
	IoTimeMs    uint64 `json:"ioTimeMs"`
}

// ContainerStats represents container-specific metrics
type ContainerStats struct {
	RunningCount int                         `json:"runningCount"`
	TotalCount   int                         `json:"totalCount"`
	PerContainer map[string]ContainerMetrics `json:"perContainer"`
}

// GPUMetrics represents GPU device metrics if available
type GPUMetrics struct {
	DeviceID    string          `json:"deviceId"`
	MemoryTotal uint64          `json:"memoryTotal"`
	MemoryUsed  uint64          `json:"memoryUsed"`
	OptMetrics  OptionalMetrics `json:"optionalMetrics,omitempty"`
}

// OptionalMetrics represents optional metrics for GPU devices
type OptionalMetrics struct {
	DutyCycle   float64 `json:"dutyCycle"`
	Temperature float64 `json:"temperature"`
	PowerUsage  float64 `json:"powerUsage"`
}

// ProcessMetrics represents process-related metrics
type ProcessMetrics struct {
	ProcessCount int `json:"processCount"`
	ThreadCount  int `json:"threadCount,omitempty"`
	FDCount      int `json:"fdCount,omitempty"`
}

// RuntimeMetrics represents container runtime metrics
type RuntimeMetrics struct {
	Operations map[string]RuntimeOperation `json:"operations"`
}

// RuntimeOperation represents a runtime operation metric
type RuntimeOperation struct {
	Type   string `json:"type"`
	Count  uint64 `json:"count"`
	Errors uint64 `json:"errors"`
}

// ImageStats represents container image statistics
type ImageStats struct {
	// Total number of images
	TotalCount int `json:"totalCount"`
	// Total size of all images in bytes
	TotalSizeBytes uint64 `json:"totalSizeBytes"`
}

// ContainerMetrics represents metrics for a single container
// Note: Some metrics may not be available depending on the container runtime
type ContainerMetrics struct {
	// Required fields - always available
	Name      string    `json:"name"`
	ID        string    `json:"id"`
	State     string    `json:"state"`
	StartTime time.Time `json:"startTime"`

	// Basic resource usage - generally available
	UsageNanos int64 `json:"usageNanos"` // CPU usage in nanoseconds
	UsageBytes int64 `json:"usageBytes"` // Memory usage in bytes

	// Detailed metrics - availability varies by runtime and platform
	CPU struct {
		UsageTotal  uint64 `json:"usageTotal"`
		UsageUser   uint64 `json:"usageUser,omitempty"`
		UsageSystem uint64 `json:"usageSystem,omitempty"`
		Throttling  *struct {
			Periods          uint64 `json:"periods"`
			ThrottledPeriods uint64 `json:"throttledPeriods"`

			ThrottledTime uint64 `json:"throttledTime"`
		} `json:"throttling,omitempty"`
	} `json:"cpu"`

	// Optional fields - may not be available in all environments
	LastTerminationReason string          `json:"lastTerminationReason,omitempty"`
	BlockIO               *BlockIOMetrics `json:"blockIO,omitempty"`
	RestartCount          int32           `json:"restartCount"`
	Ready                 bool            `json:"ready"`
	Memory                MemoryMetrics   `json:"memory"`
}

// BlockIOMetrics represents block I/O metrics
type BlockIOMetrics struct {
	// Read operations
	ReadOps uint64 `json:"readOps"`
	// Write operations
	WriteOps uint64 `json:"writeOps"`
	// Bytes read
	ReadBytes uint64 `json:"readBytes"`
	// Bytes written
	WriteBytes uint64 `json:"writeBytes"`
}

// DiskMetrics represents disk I/O metrics
type DiskMetrics struct {
	ReadBytes  int64           // Total bytes read across all devices
	WriteBytes int64           // Total bytes written across all devices
	Devices    []NodeDiskStats // Per-device statistics
}

// ServiceStatus represents the status of a Kubernetes service
type ServiceStatus struct {
	LoadBalancer LoadBalancerStatus `json:"loadBalancer"`
	Conditions   []ServiceCondition `json:"conditions"`
}

// ServiceCondition represents the current condition of a Kubernetes service
type ServiceCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
}

// LoadBalancerStatus represents the status of a Kubernetes load balancer
type LoadBalancerStatus struct {
	Ingress []LoadBalancerIngress `json:"ingress,omitempty"`
}

// LoadBalancerIngress represents the ingress point for a Kubernetes load balancer
type LoadBalancerIngress struct {
	IP       string `json:"ip,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

// IngressStatus represents the status of a Kubernetes ingress
type IngressStatus struct {
	LoadBalancer LoadBalancerStatus `json:"loadBalancer"`
}

// IngressCondition represents the current condition of a Kubernetes ingress
type IngressCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
}

// SystemMetrics represents system-level metrics
// Note: Availability of these metrics depends on the host system configuration
type SystemMetrics struct {
	UptimeSeconds   float64 `json:"uptimeSeconds"`
	NumCPUs         int64   `json:"numCPUs"`
	NumProcs        uint64  `json:"numProcs"`
	BootTimeSeconds float64 `json:"bootTimeSeconds"`

	// Platform-specific fields - may not be available in all environments
	KernelVersion string `json:"kernelVersion,omitempty"`
	OSVersion     string `json:"osVersion,omitempty"`
	// These IDs may not be available on all platforms
	MachineID  string `json:"machineID,omitempty"`
	SystemUUID string `json:"systemUUID,omitempty"`
	BootID     string `json:"bootID,omitempty"`
}

// KubeletMetrics represents kubelet-related metrics
// Note: Metric availability depends on Kubelet version and configuration
type KubeletMetrics struct {
	// Basic metrics - generally available
	PodStartLatency LatencyMetric `json:"podStartLatency"`

	// Advanced metrics - may not be available in all configurations
	OperationLatency  map[string]LatencyMetric   `json:"operationLatency,omitempty"`
	CGroupManager     *CGroupManagerMetrics      `json:"cgroupManager,omitempty"`
	PLEGLatency       *LatencyMetric             `json:"plegLatency,omitempty"`
	RuntimeOperations map[string]RuntimeOpMetric `json:"runtimeOperations,omitempty"`
	EvictionStats     *EvictionStats             `json:"evictionStats,omitempty"`
}

// LatencyMetric represents latency metrics
type LatencyMetric struct {
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
	P99   float64 `json:"p99"`
	Count int64   `json:"count"`
}

// CGroupManagerMetrics represents cgroup manager metrics
type CGroupManagerMetrics struct {
	Operations uint64        `json:"operations"`
	Errors     uint64        `json:"errors"`
	Latency    LatencyMetric `json:"latency"`
}

// RuntimeOpMetric represents runtime operation metrics
type RuntimeOpMetric struct {
	Count    uint64  `json:"count"`
	Errors   uint64  `json:"errors"`
	Duration float64 `json:"duration"`
}

// EvictionStats represents eviction statistics
type EvictionStats struct {
	ThresholdMet      bool       `json:"thresholdMet"`
	HardEvictionCount uint64     `json:"hardEvictionCount"`
	SoftEvictionCount uint64     `json:"softEvictionCount"`
	LastEvictionTime  *time.Time `json:"lastEvictionTime,omitempty"`
}

// VolumeMetrics represents volume-related metrics
type VolumeMetrics struct {
	AttachDetachCount  uint64                   `json:"attachDetachCount"`
	InUseCount         uint64                   `json:"inUseCount"`
	OperationLatency   map[string]LatencyMetric `json:"operationLatency"`
	TotalCapacityBytes uint64                   `json:"totalCapacityBytes"`
	AvailableBytes     uint64                   `json:"availableBytes"`
	UsedBytes          uint64                   `json:"usedBytes"`
}

// PodReadinessGate represents a readiness gate for a pod
type PodReadinessGate struct {
	ConditionType string `json:"conditionType"`
	Status        bool   `json:"status"`
}

// JobCondition represents a condition of a Job
type JobCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	LastProbeTime      *time.Time `json:"lastProbeTime,omitempty"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
}

// DaemonSetStatus represents the status of a DaemonSet
type DaemonSetStatus struct {
	ObservedGeneration int64 `json:"observedGeneration"`
}

// DaemonSetCondition represents a condition of a DaemonSet
type DaemonSetCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
}

// NodeInfo represents node information
type NodeInfo struct {
	Architecture            string `json:"architecture"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
	KernelVersion           string `json:"kernelVersion"`
	OSImage                 string `json:"osImage"`
	KubeletVersion          string `json:"kubeletVersion"`
}

// NodeNetworkStats represents network metrics for a node
type NodeNetworkStats struct {
	InterfaceName string `json:"interfaceName"`
	RxBytes       uint64 `json:"rxBytes"`
	RxPackets     uint64 `json:"rxPackets"`
	RxErrors      uint64 `json:"rxErrors"`
	RxDropped     uint64 `json:"rxDropped"`
	TxBytes       uint64 `json:"txBytes"`
	TxPackets     uint64 `json:"txPackets"`
	TxErrors      uint64 `json:"txErrors"`
	TxDropped     uint64 `json:"txDropped"`
}

// NodeDiskStats represents disk metrics for a node
type NodeDiskStats struct {
	Device         string `json:"device"`
	ReadOps        uint64 `json:"readOps"`
	WriteOps       uint64 `json:"writeOps"`
	ReadBytes      uint64 `json:"readBytes"`
	WriteBytes     uint64 `json:"writeBytes"`
	ReadLatency    uint64 `json:"readLatency"`
	WriteLatency   uint64 `json:"writeLatency"`
	IoInProgress   uint64 `json:"ioInProgress"`
	IoTime         uint64 `json:"ioTime"`
	WeightedIoTime uint64 `json:"weightedIoTime"`
}

// InodeStats represents inode metrics
type InodeStats struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

// VolumeMountMetrics represents volume mount metrics
type VolumeMountMetrics struct {
	Name             string `json:"name"`
	MountPath        string `json:"mountPath"`
	ReadOnly         bool   `json:"readOnly,omitempty"`
	SubPath          string `json:"subPath,omitempty"`
	SubPathExpr      string `json:"subPathExpr,omitempty"`
	MountPropagation string `json:"mountPropagation,omitempty"`
}

// NetworkPolicyMetrics represents metrics for networking resources
type NetworkPolicyMetrics struct {
	Name        string                     `json:"name"`
	Namespace   string                     `json:"namespace"`
	Labels      map[string]string          `json:"labels"`
	Annotations map[string]string          `json:"annotations"`
	PodSelector map[string]string          `json:"podSelector"`
	Ingress     []NetworkPolicyIngressRule `json:"ingress"`
	Egress      []NetworkPolicyEgressRule  `json:"egress"`
	PolicyTypes []string                   `json:"policyTypes"`
}

// NetworkPolicyIngressRule represents an ingress rule in a network policy
type NetworkPolicyIngressRule struct {
	Ports []NetworkPolicyPort `json:"ports"`
	From  []NetworkPolicyPeer `json:"from"`
}

// NetworkPolicyEgressRule represents an egress rule in a network policy
type NetworkPolicyEgressRule struct {
	Ports []NetworkPolicyPort `json:"ports"`
	To    []NetworkPolicyPeer `json:"to"`
}

// NetworkPolicyPort represents a port in a network policy
type NetworkPolicyPort struct {
	Protocol string `json:"protocol"`
	Port     int32  `json:"port"`
}

// NetworkPolicyPeer represents a peer in a network policy
type NetworkPolicyPeer struct {
	PodSelector       map[string]string `json:"podSelector,omitempty"`
	NamespaceSelector map[string]string `json:"namespaceSelector,omitempty"`
	IPBlock           *IPBlock          `json:"ipBlock,omitempty"`
}

// IPBlock represents an IP block in a network policy
type IPBlock struct {
	CIDR   string   `json:"cidr"`
	Except []string `json:"except,omitempty"`
}

// StorageClassMetrics represents metrics for storage classes
type StorageClassMetrics struct {
	Name                 string            `json:"name"`
	Provisioner          string            `json:"provisioner"`
	ReclaimPolicy        string            `json:"reclaimPolicy"`
	VolumeBindingMode    string            `json:"volumeBindingMode"`
	AllowVolumeExpansion bool              `json:"allowVolumeExpansion"`
	Labels               map[string]string `json:"labels"`
	Annotations          map[string]string `json:"annotations"`
	IsDefault            bool              `json:"isDefault"`
	Parameters           map[string]string `json:"parameters"`
	MountOptions         []string          `json:"mountOptions,omitempty"`
	CreationTimestamp    *time.Time        `json:"creationTimestamp"`

	// New fields for CSI and capacity metrics
	CSIDriver           *CSIDriverMetrics    `json:"csiDriver,omitempty"`
	StoragePools        []StoragePoolMetrics `json:"storagePools,omitempty"`
	TotalCapacity       int64                `json:"totalCapacity,omitempty"`
	AllocatedCapacity   int64                `json:"allocatedCapacity,omitempty"`
	AvailableCapacity   int64                `json:"availableCapacity,omitempty"`
	CapacityUtilization float64              `json:"capacityUtilization,omitempty"`
	ProvisionedPVCs     int32                `json:"provisionedPVCs,omitempty"`
	ProvisioningRate    float64              `json:"provisioningRate,omitempty"`
}

// HardwareTopology represents the hardware topology of a node
type HardwareTopology struct {
	Sockets int64 `json:"sockets,omitempty"`
	Cores   int64 `json:"cores,omitempty"`
	Threads int64 `json:"threads,omitempty"`
	// NUMA topology
	NUMANodes []NUMANode `json:"numaNodes,omitempty"`
}

// NUMANode represents NUMA node information
type NUMANode struct {
	ID       int32    `json:"id"`
	Memory   uint64   `json:"memory,omitempty"`   // Memory in bytes
	CPUList  []int32  `json:"cpuList,omitempty"`  // List of CPU IDs
	Distance []uint32 `json:"distance,omitempty"` // NUMA distance array
}

// PowerMetrics represents power management information
type PowerMetrics struct {
	CurrentWatts float64 `json:"currentWatts,omitempty"`
	MaxWatts     float64 `json:"maxWatts,omitempty"`
	// Power capping information
	PowerCap     *float64 `json:"powerCap,omitempty"`
	PowerProfile string   `json:"powerProfile,omitempty"`
}

// NodeTaint represents a Kubernetes node taint
type NodeTaint struct {
	Key       string     `json:"key"`
	Value     string     `json:"value,omitempty"`
	Effect    string     `json:"effect"`
	TimeAdded *time.Time `json:"timeAdded,omitempty"`
}

// NodeLease represents node lease information
type NodeLease struct {
	HolderIdentity       string     `json:"holderIdentity"`
	LeaseDurationSeconds int32      `json:"leaseDurationSeconds"`
	AcquireTime          *time.Time `json:"acquireTime,omitempty"`
	RenewTime            *time.Time `json:"renewTime,omitempty"`
}

// ExtendedResource represents custom resource information
type ExtendedResource struct {
	Name        string `json:"name"`
	Capacity    string `json:"capacity"`
	Allocatable string `json:"allocatable"`
	Used        string `json:"used,omitempty"`
}

// PodDisruptionBudgetMetrics represents PDB metrics
type PodDisruptionBudgetMetrics struct {
	MinAvailable       string `json:"minAvailable,omitempty"`
	MaxUnavailable     string `json:"maxUnavailable,omitempty"`
	CurrentHealthy     int32  `json:"currentHealthy"`
	DesiredHealthy     int32  `json:"desiredHealthy"`
	DisruptionsAllowed int32  `json:"disruptionsAllowed"`
	ExpectedPods       int32  `json:"expectedPods"`
}

// TopologySpreadConstraint represents pod topology spread constraints
type TopologySpreadConstraint struct {
	MaxSkew           int32  `json:"maxSkew"`
	TopologyKey       string `json:"topologyKey"`
	WhenUnsatisfiable string `json:"whenUnsatisfiable"`
	LabelSelector     string `json:"labelSelector,omitempty"`
	MinDomains        *int32 `json:"minDomains,omitempty"`
}

// PodOverheadMetrics represents pod overhead metrics
type PodOverheadMetrics struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// PodSchedulingGate represents pod scheduling gates
type PodSchedulingGate struct {
	Name   string     `json:"name"`
	Active bool       `json:"active"`
	Since  *time.Time `json:"since,omitempty"`
}

// SecurityContextMetrics represents security context information
type SecurityContextMetrics struct {
	RunAsUser      *int64            `json:"runAsUser,omitempty"`
	RunAsGroup     *int64            `json:"runAsGroup,omitempty"`
	FSGroup        *int64            `json:"fsGroup,omitempty"`
	RunAsNonRoot   *bool             `json:"runAsNonRoot,omitempty"`
	SELinuxOptions map[string]string `json:"seLinuxOptions,omitempty"`
	Sysctls        []string          `json:"sysctls,omitempty"`
	SeccompProfile string            `json:"seccompProfile,omitempty"`
}

// AffinityMetrics represents pod affinity/anti-affinity rules
type AffinityMetrics struct {
	NodeAffinity    []NodeAffinityTerm `json:"nodeAffinity,omitempty"`
	PodAffinity     []PodAffinityTerm  `json:"podAffinity,omitempty"`
	PodAntiAffinity []PodAffinityTerm  `json:"podAntiAffinity,omitempty"`
}

// NodeAffinityTerm represents a node affinity requirement
type NodeAffinityTerm struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// PodAffinityTerm represents a pod affinity/anti-affinity requirement
type PodAffinityTerm struct {
	LabelSelector string   `json:"labelSelector"`
	Namespaces    []string `json:"namespaces,omitempty"`
	TopologyKey   string   `json:"topologyKey"`
}

// InitContainerMetrics represents init container specific metrics
type InitContainerMetrics struct {
	ContainerMetrics
	CompletionTime *time.Time `json:"completionTime,omitempty"`
	Duration       float64    `json:"duration,omitempty"`
}

// EphemeralContainerMetrics represents ephemeral container metrics
type EphemeralContainerMetrics struct {
	ContainerMetrics
	TargetContainerName string     `json:"targetContainerName"`
	StartTime           *time.Time `json:"startTime,omitempty"`
}

// QoSMetrics represents detailed QoS metrics
type QoSMetrics struct {
	Class            string  `json:"class"`
	CPUGuaranteed    bool    `json:"cpuGuaranteed"`
	MemoryGuaranteed bool    `json:"memoryGuaranteed"`
	BurstableLimit   float64 `json:"burstableLimit,omitempty"`
}

// CSIDriverMetrics represents CSI driver metrics
type CSIDriverMetrics struct {
	Name               string            `json:"name"`
	Version            string            `json:"version,omitempty"`
	Available          bool              `json:"available"`
	VolumeSnapshotting bool              `json:"volumeSnapshotting,omitempty"`
	VolumeCloning      bool              `json:"volumeCloning,omitempty"`
	VolumeExpansion    bool              `json:"volumeExpansion,omitempty"`
	NodePluginPods     map[string]string `json:"nodePluginPods,omitempty"`
	ControllerPods     map[string]string `json:"controllerPods,omitempty"`
	OperationStats     map[string]int64  `json:"operationStats,omitempty"`
}

// StoragePoolMetrics represents storage pool metrics
type StoragePoolMetrics struct {
	Name           string  `json:"name"`
	Provider       string  `json:"provider"`
	TotalCapacity  int64   `json:"totalCapacity"`
	UsedCapacity   int64   `json:"usedCapacity"`
	AvailableSpace int64   `json:"availableSpace"`
	VolumeCount    int32   `json:"volumeCount,omitempty"`
	UtilizationPct float64 `json:"utilizationPct,omitempty"`
	Health         string  `json:"health,omitempty"`
	StorageClass   string  `json:"storageClass"`
}

// VolumeHealthMetrics represents volume health status
type VolumeHealthMetrics struct {
	VolumeName     string    `json:"volumeName"`
	PodName        string    `json:"podName,omitempty"`
	Namespace      string    `json:"namespace,omitempty"`
	State          string    `json:"state"`
	LastCheckTime  time.Time `json:"lastCheckTime"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	RepairRequired bool      `json:"repairRequired,omitempty"`
	IoPerformance  string    `json:"ioPerformance,omitempty"`
	FsIntegrity    bool      `json:"fsIntegrity,omitempty"`
}

// VolumeAttachmentMetrics represents volume attachment status
type VolumeAttachmentMetrics struct {
	VolumeName      string    `json:"volumeName"`
	PodName         string    `json:"podName,omitempty"`
	Namespace       string    `json:"namespace,omitempty"`
	AttachTime      time.Time `json:"attachTime"`
	DevicePath      string    `json:"devicePath,omitempty"`
	AttachmentState string    `json:"attachmentState"`
	ReadOnly        bool      `json:"readOnly"`
	MountPoint      string    `json:"mountPoint,omitempty"`
	AttachError     string    `json:"attachError,omitempty"`
}

// NodeVolumeMetrics represents node volume metrics
type NodeVolumeMetrics struct {
	// Existing fields
	InUseCount       uint64                   `json:"inUseCount"`
	OperationLatency map[string]LatencyMetric `json:"operationLatency,omitempty"`

	// New fields
	VolumeHealth      []VolumeHealthMetrics     `json:"volumeHealth,omitempty"`
	VolumeAttachments []VolumeAttachmentMetrics `json:"volumeAttachments,omitempty"`
	AttachLimit       int32                     `json:"attachLimit,omitempty"`
	AttachCount       int32                     `json:"attachCount"`
	AttachErrors      uint64                    `json:"attachErrors,omitempty"`
	DetachErrors      uint64                    `json:"detachErrors,omitempty"`
}

// VolumeSnapshotMetrics represents snapshot information for a volume
type VolumeSnapshotMetrics struct {
	Name         string     `json:"name"`
	Namespace    string     `json:"namespace,omitempty"`
	SourcePVName string     `json:"sourcePVName"`
	CreationTime time.Time  `json:"creationTime"`
	Size         int64      `json:"size"`
	ReadyToUse   bool       `json:"readyToUse"`
	RestoreSize  int64      `json:"restoreSize,omitempty"`
	DeletionTime *time.Time `json:"deletionTime,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// VolumeExpansionMetrics represents volume expansion metrics
type VolumeExpansionMetrics struct {
	CurrentSize    int64      `json:"currentSize"`
	RequestedSize  int64      `json:"requestedSize"`
	InProgress     bool       `json:"inProgress"`
	LastResizeTime *time.Time `json:"lastResizeTime,omitempty"`
	ResizeStatus   string     `json:"resizeStatus,omitempty"`
	FailureMessage string     `json:"failureMessage,omitempty"`
}

// QuotaScopeMetrics represents quota scope information
type QuotaScopeMetrics struct {
	ScopeName   string            `json:"scopeName"`
	Resources   []string          `json:"resources,omitempty"`
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
	MatchScopes []string          `json:"matchScopes,omitempty"`
}

// PriorityClassQuotaMetrics represents quota metrics for priority classes
type PriorityClassQuotaMetrics struct {
	PriorityClass string            `json:"priorityClass"`
	Hard          map[string]string `json:"hard,omitempty"`
	Used          map[string]string `json:"used,omitempty"`
}

// QuotaStatusHistory represents historical quota status
type QuotaStatusHistory struct {
	Timestamp time.Time `json:"timestamp"`
	Resource  string    `json:"resource"`
	Hard      string    `json:"hard"`
	Used      string    `json:"used"`
}

// CustomResourceQuota represents quotas for custom resources
// type CustomResourceQuota struct {
// 	Group    string `json:"group"`
// 	Resource string `json:"resource"`
// 	Version  string `json:"version"`
// 	Hard     string `json:"hard,omitempty"`
// 	Used     string `json:"used,omitempty"`
// }

// SchedStats represents scheduler statistics
type SchedStats struct {
	RunQueueLength   uint64 `json:"runQueueLength"`
	WaitingProcesses uint64 `json:"waitingProcesses"`
	SleepingTasks    uint64 `json:"sleepingTasks"`
}

// PSIMetrics represents metrics for the container runtime
type PSIMetrics struct {
	CPU struct {
		Some float64 `json:"some"`
		Full float64 `json:"full"`
	} `json:"cpu"`
	Memory struct {
		Some float64 `json:"some"`
		Full float64 `json:"full"`
	} `json:"memory"`
	IO struct {
		Some float64 `json:"some"`
		Full float64 `json:"full"`
	} `json:"io"`
}
