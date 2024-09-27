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
package models

import (
	"time"
)

// EnhancedNodeMetrics represents detailed metrics for a single node
type EnhancedNodeMetrics struct {
	Name        string            `json:"name"`
	Capacity    ResourceMetrics   `json:"capacity"`
	Allocatable ResourceMetrics   `json:"allocatable"`
	Usage       ResourceMetrics   `json:"usage"`
	Conditions  NodeConditions    `json:"conditions"`
	Labels      map[string]string `json:"labels"`
}

// EnhancedPodMetrics represents detailed metrics for a single pod
type EnhancedPodMetrics struct {
	PodMetrics    PodMetrics         `json:"podMetrics"`
	QoSClass      string             `json:"qosClass"`
	Conditions    PodConditions      `json:"conditions"`
	Requests      ResourceMetrics    `json:"requests"`
	Limits        ResourceMetrics    `json:"limits"`
	Containers    []ContainerMetrics `json:"containers"`
	TotalRestarts int32              `json:"totalRestarts"`
}

// NodeMetrics represents the metrics for a single node
type NodeMetrics struct {
	Name        string          `json:"name"`
	Capacity    ResourceMetrics `json:"capacity"`
	Allocatable ResourceMetrics `json:"allocatable"`
	Usage       ResourceMetrics `json:"usage"`
	Conditions  NodeConditions  `json:"conditions"`
}

// ResourceMetrics represents resource usage or capacity
type ResourceMetrics struct {
	CPU     int64 `json:"cpu"`    // in millicores
	Memory  int64 `json:"memory"` // in bytes
	Pods    int64 `json:"pods"`
	Storage int64 `json:"storage"` // in bytes
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

// ContainerMetrics represents the metrics for a single container
type ContainerMetrics struct {
	Name           string `json:"name"`
	RestartCount   int32  `json:"restartCount"`
	Ready          bool   `json:"ready"`
	State          string `json:"state"`
	LastTerminated string `json:"lastTerminated,omitempty"`
	CPU            int64  `json:"cpu"`    // in millicores
	Memory         int64  `json:"memory"` // in bytes
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
type PersistentVolumeMetrics struct {
	PVs  []PVMetric  `json:"pvs"`
	PVCs []PVCMetric `json:"pvcs"`
}

// PVMetric represents metrics for a single persistent volume
type PVMetric struct {
	Name         string            `json:"name"`
	Capacity     int64             `json:"capacity"` // in bytes
	Phase        string            `json:"phase"`
	StorageClass string            `json:"storageClass"`
	BoundPVC     string            `json:"boundPVC,omitempty"`
	Labels       map[string]string `json:"labels"`
}

// PVCMetric represents metrics for a single persistent volume claim
type PVCMetric struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Phase        string            `json:"phase"`
	Capacity     int64             `json:"capacity"` // in bytes
	StorageClass string            `json:"storageClass"`
	BoundPV      string            `json:"boundPV,omitempty"`
	Labels       map[string]string `json:"labels"`
}

// NamespaceMetrics represents metrics for a single namespace
type NamespaceMetrics struct {
	Name           string                 `json:"name"`
	Status         string                 `json:"status"`
	ResourceQuotas []ResourceQuotaMetrics `json:"resourceQuotas"`
	LimitRanges    []LimitRangeMetrics    `json:"limitRanges"`
	Usage          ResourceUsage          `json:"usage"`
	Labels         map[string]string      `json:"labels"`
}

// ResourceQuotaMetrics represents metrics for a resource quota
type ResourceQuotaMetrics struct {
	Name      string           `json:"name"`
	Resources []ResourceMetric `json:"resources"`
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

// DeploymentMetrics represents metrics for a deployment
type DeploymentMetrics struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"readyReplicas"`
	UpdatedReplicas   int32             `json:"updatedReplicas"`
	AvailableReplicas int32             `json:"availableReplicas"`
	Labels            map[string]string `json:"labels"`
}

// StatefulSetMetrics represents metrics for a stateful set
type StatefulSetMetrics struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Replicas        int32             `json:"replicas"`
	ReadyReplicas   int32             `json:"readyReplicas"`
	CurrentReplicas int32             `json:"currentReplicas"`
	UpdatedReplicas int32             `json:"updatedReplicas"`
	Labels          map[string]string `json:"labels"`
}

// DaemonSetMetrics represents metrics for a daemon set
type DaemonSetMetrics struct {
	Name                   string            `json:"name"`
	Namespace              string            `json:"namespace"`
	DesiredNumberScheduled int32             `json:"desiredNumberScheduled"`
	CurrentNumberScheduled int32             `json:"currentNumberScheduled"`
	NumberReady            int32             `json:"numberReady"`
	UpdatedNumberScheduled int32             `json:"updatedNumberScheduled"`
	NumberAvailable        int32             `json:"numberAvailable"`
	Labels                 map[string]string `json:"labels"`
}

// JobMetrics represents metrics for a job
type JobMetrics struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Completions    *int32            `json:"completions,omitempty"`
	Parallelism    *int32            `json:"parallelism,omitempty"`
	Active         int32             `json:"active"`
	Succeeded      int32             `json:"succeeded"`
	Failed         int32             `json:"failed"`
	StartTime      time.Time         `json:"startTime,omitempty"`
	CompletionTime time.Time         `json:"completionTime,omitempty"`
	Labels         map[string]string `json:"labels"`
}

// NetworkingMetrics represents metrics for networking resources
type NetworkingMetrics struct {
	Services  []ServiceMetrics `json:"services"`
	Ingresses []IngressMetrics `json:"ingresses"`
}

// ServiceMetrics represents metrics for a service
type ServiceMetrics struct {
	Name                string            `json:"name"`
	Namespace           string            `json:"namespace"`
	Type                string            `json:"type"`
	ClusterIP           string            `json:"clusterIP"`
	ExternalIP          []string          `json:"externalIP,omitempty"`
	LoadBalancerIP      string            `json:"loadBalancerIP,omitempty"`
	LoadBalancerIngress []string          `json:"loadBalancerIngress,omitempty"`
	Ports               []ServicePort     `json:"ports"`
	Selector            map[string]string `json:"selector,omitempty"`
	Labels              map[string]string `json:"labels"`
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

// CronJobMetrics represents metrics for a cron job
type CronJobMetrics struct {
	Name             string            `json:"name"`
	Namespace        string            `json:"namespace"`
	Schedule         string            `json:"schedule"`
	Suspend          bool              `json:"suspend"`
	ActiveJobs       int               `json:"activeJobs"`
	LastScheduleTime *time.Time        `json:"lastScheduleTime,omitempty"`
	Labels           map[string]string `json:"labels"`
}

// HPAMetrics represents metrics for a horizontal pod autoscaler
type HPAMetrics struct {
	Name                            string            // Name of the HPA
	Namespace                       string            // Namespace of the HPA
	CurrentReplicas                 int32             // Current number of replicas
	DesiredReplicas                 int32             // Desired number of replicas
	CurrentCPUUtilizationPercentage *int32            // Current CPU utilization percentage, if available
	TargetCPUUtilizationPercentage  *int32            // Target CPU utilization percentage, if specified
	Labels                          map[string]string `json:"labels"`
}

// ReplicaSetMetrics represents metrics for a replica set
type ReplicaSetMetrics struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"readyReplicas"`
	AvailableReplicas int32             `json:"availableReplicas"`
	Labels            map[string]string `json:"labels"`
}

// ReplicationControllerMetrics represents metrics for a replication controller
type ReplicationControllerMetrics struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"readyReplicas"`
	AvailableReplicas int32             `json:"availableReplicas"`
	Labels            map[string]string `json:"labels"`
}
