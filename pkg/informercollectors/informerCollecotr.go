package informercollectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/config"
	"github.com/vegacloud/kubernetes/metricsagent/pkg/utils"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type InformerCollector interface {
	// CollectMetrics collects metrics and returns them as an interface{},
	CollectorFromInformer(ctx context.Context) (interface{}, error)
}

type InformationSource struct {
	Type      string   `yaml:"type"`
	Version   string   `yaml:"version"`
	Resource  string   `yaml:"resource"`
	Whitelist []string `yaml:"whitelist"`
	Blacklist []string `yaml:"blacklist"`
	Informer  cache.SharedIndexInformer
}

func FetchAndSaveInformationSources(ctx context.Context, cfg *config.Config) error {
	// Create a new HTTP client with a timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Get the auth token
	token, err := utils.GetVegaAuthToken(ctx, client, cfg)
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.InformationSourcesURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the Authorization header
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch information sources: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch information sources, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Write the data to the specified file
	err = os.WriteFile(cfg.InformationSourcesFilePath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write information sources to file: %w", err)
	}

	logrus.Debugf("Successfully fetched and saved information sources to file: %s", cfg.InformationSourcesFilePath)
	return nil
}
func UseDefaultInformationSources() error {
	// Write the default information sources to the specified file
	err := os.WriteFile(config.DefaultInformationSourcesFilePath, []byte(DefaultInformationSourcesYAML), 0600)
	if err != nil {
		return fmt.Errorf("failed to write default information sources to file: %w", err)
	}

	logrus.Debugf("Successfully wrote default information sources to file: %s", config.DefaultInformationSourcesFilePath)
	return nil
}

func LoadInformationSources(filePath string) ([]InformationSource, error) {
	// Read the YAML file
	cleanFilePath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanFilePath)

	if err != nil {
		return nil, err
	}

	// Parse the YAML into a map of slices
	var sources map[string][]InformationSource
	err = yaml.Unmarshal(data, &sources)
	if err != nil {
		return nil, err
	}
	if len(sources["informers"]) == 0 {
		return nil, errors.New("no informers found in the YAML file")
	}

	return sources["informers"], nil
}

func LoadInformers(filePath string,
	clientSet kubernetes.Interface,
	resyncInterval int,
	stopCh chan struct{}) (map[string]InformationSource, error) {
	// Read the YAML file
	sources, err := LoadInformationSources(filePath)
	if err != nil {
		return nil, err
	}

	factory := informers.NewSharedInformerFactory(clientSet, time.Duration(resyncInterval)*time.Hour)

	informationSources := make(map[string]InformationSource)

	// Iterate over the sources and create informers based on the type
	for _, source := range sources {
		var informer cache.SharedIndexInformer

		informerFactory := reflect.ValueOf(factory).MethodByName(source.Type)
		if !informerFactory.IsValid() {
			logrus.Errorf("unsupported informer type: %s", source.Type)
			continue
		}

		informerVersion := informerFactory.Call(nil)[0].MethodByName(source.Version)
		if !informerVersion.IsValid() {
			logrus.Errorf("unsupported informer version: %s", source.Version)
			continue
		}

		informerResource := informerVersion.Call(nil)[0].MethodByName(source.Resource)
		if !informerResource.IsValid() {
			logrus.Errorf("unsupported informer resource: %s", source.Resource)
			continue
		}
		informer = informerResource.Call(nil)[0].MethodByName("Informer").Call(nil)[0].Interface().(cache.SharedIndexInformer)
		source.Informer = informer
		informationSources[source.Resource] = source
	}
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	return informationSources, nil
}

// could you please update this so that it returns a interface{} instead of a string?
func GetInformerDataAsJSON(informationSource InformationSource) (interface{}, error) {
	// Filter the informer data
	filteredData, err := FilterInformerData(informationSource.Informer,
		informationSource.Whitelist,
		informationSource.Blacklist)
	if err != nil {
		return nil, fmt.Errorf("error filtering informer data: %w", err)
	}

	return filteredData, nil
}

func FilterInformerData(informer cache.SharedIndexInformer,
	whitelist, blacklist []string) ([]map[string]interface{}, error) {
	// Get the list of resources from the informer
	resourceList := informer.GetIndexer().List()
	filteredData := make([]map[string]interface{}, 0, len(resourceList))

	for _, resource := range resourceList {
		// Convert resource to a map for easier manipulation
		resourceMap := make(map[string]interface{})
		resourceBytes, err := json.Marshal(resource)
		if err != nil {
			return nil, errors.New("error: unable to marshal resource to JSON")
		}
		err = json.Unmarshal(resourceBytes, &resourceMap)
		if err != nil {
			return nil, errors.New("error: unable to unmarshal resource JSON to map")
		}

		// Apply whitelist and blacklist
		filteredResource := applyFilters(resourceMap, whitelist, blacklist)
		filteredData = append(filteredData, filteredResource)
	}

	return filteredData, nil
}

func applyFilters(resource map[string]interface{}, whitelist, blacklist []string) map[string]interface{} {
	filteredResource := make(map[string]interface{})

	// If whitelist contains "*", include all fields except those in the blacklist
	if len(whitelist) == 1 && whitelist[0] == "*" {
		for key, value := range resource {
			if !contains(blacklist, key) {
				filteredResource[key] = value
			}
		}
		return filteredResource
	}

	// Include fields from the whitelist
	for _, key := range whitelist {
		if value, exists := resource[key]; exists {
			filteredResource[key] = value
		}
	}

	// Exclude fields from the blacklist
	for _, key := range blacklist {
		delete(filteredResource, key)
	}

	return filteredResource
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

const ( // this should put us in parity with Apptio
	DefaultInformationSourcesYAML = `
# informers.yaml

informers:
  - type: Core
    version: V1
    resource: Pods
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration",
      "spec.containers.env",
      "spec.containers.command",
      "spec.containers.args",
      "spec.containers.imagePullPolicy",
      "spec.containers.livenessProbe",
      "spec.containers.startupProbe",
      "spec.containers.readinessProbe",
      "spec.containers.terminationMessagePath",
      "spec.containers.terminationMessagePolicy",
      "spec.containers.securityContext",
      "spec.initContainers.env",
      "spec.initContainers.command",
      "spec.initContainers.args",
      "spec.initContainers.imagePullPolicy",
      "spec.initContainers.livenessProbe",
      "spec.initContainers.startupProbe",
      "spec.initContainers.readinessProbe",
      "spec.initContainers.terminationMessagePath",
      "spec.initContainers.terminationMessagePolicy",
      "spec.initContainers.securityContext"
    ]
  - type: Apps
    version: V1
    resource: Deployments
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration",
      "spec.template",
      "spec.replicas",
      "spec.strategy",
      "spec.minReadySeconds",
      "spec.revisionHistoryLimit",
      "spec.progressDeadlineSeconds"
    ]
  - type: Batch
    version: V1
    resource: Jobs
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration",
      "spec.template",
      "spec.parallelism",
      "spec.completions",
      "spec.activeDeadlineSeconds",
      "spec.backoffLimit",
      "spec.manualSelector",
      "spec.ttlSecondsAfterFinished",
      "spec.completionMode",
      "spec.suspend"
    ]
  - type: Core
    version: V1
    resource: Services
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration",
      "spec.ports",
      "spec.clusterIP",
      "spec.clusterIPs",
      "spec.type",
      "spec.externalIPs",
      "spec.sessionAffinity",
      "spec.loadBalancerIP",
      "spec.loadBalancerSourceRanges",
      "spec.externalName",
      "spec.externalTrafficPolicy",
      "spec.healthCheckNodePort",
      "spec.sessionAffinityConfig",
      "spec.ipFamilies",
      "spec.ipFamilyPolicy",
      "spec.allocateLoadBalancerNodePorts",
      "spec.loadBalancerClass",
      "spec.internalTrafficPolicy"
    ]
  - type: Core
    version: V1
    resource: Nodes
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration"
    ]
  - type: Apps
    version: V1
    resource: ReplicaSets
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration",
      "spec.replicas",
      "spec.template",
      "spec.minReadySeconds"
    ]
  - type: Core
    version: V1
    resource: PersistentVolumes
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration"
    ]
  - type: Core
    version: V1
    resource: PersistentVolumeClaims
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration"
    ]
  - type: Core
    version: V1
    resource: Namespaces
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields"
    ]
  - type: Batch
    version: V1
    resource: CronJobs
    whitelist: ["*"]
    blacklist: [
      "metadata.managedFields",
      "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration",
      "spec"
    ]
	`
)
