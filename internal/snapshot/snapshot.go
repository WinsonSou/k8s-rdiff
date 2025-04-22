package snapshot

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/winson-sou/k8s-rdiff/internal/filter"
	internal_k8s "github.com/winson-sou/k8s-rdiff/internal/k8s"
	"gopkg.in/yaml.v2"
)

// ResourceInfo represents the metadata for a Kubernetes resource
type ResourceInfo struct {
	GroupVersionKind  string `json:"groupVersionKind"`
	Namespace         string `json:"namespace"`
	Name              string `json:"name"`
	UID               string `json:"uid"`
	ResourceVersion   string `json:"resourceVersion"`
	CreationTimestamp string `json:"creationTimestamp"`
	SpecHash          string `json:"specHash"`
	Manifest          string `json:"manifest,omitempty"` // YAML representation of the resource
}

// Snapshot represents a collection of resources at a point in time
type Snapshot struct {
	Timestamp time.Time               `json:"timestamp"`
	Namespace string                  `json:"namespace"`
	Resources map[string]ResourceInfo `json:"resources"` // Key: GVK|NS|Name
}

// CaptureSnapshot captures all resources in the specified namespace
func CaptureSnapshot(namespace, ignoreKindRegex, kubeconfigPath string) (*Snapshot, error) {
	// Create Kubernetes client
	client, err := internal_k8s.NewClient(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Create and compile the resource filter
	resourceFilter := filter.NewResourceFilter().WithNoisy()

	// Add custom exclusion patterns if provided
	if ignoreKindRegex != "" {
		resourceFilter.WithExcludes([]string{ignoreKindRegex})
	}

	if err := resourceFilter.Compile(); err != nil {
		return nil, fmt.Errorf("failed to compile resource filter: %v", err)
	}

	snapshot := &Snapshot{
		Timestamp: time.Now().UTC(),
		Namespace: namespace,
		Resources: make(map[string]ResourceInfo),
	}

	// Discover API resources
	resourceTypes, err := client.DiscoverResources(resourceFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to discover API resources: %v", err)
	}

	// Capture resources for each resource type
	for _, resourceType := range resourceTypes {
		resources, err := client.ListResources(resourceType, namespace)
		if err != nil {
			// Just log the error and continue with other resources
			fmt.Fprintf(os.Stderr, "Warning: failed to list %s: %v\n", resourceType, err)
			continue
		}

		for _, resource := range resources {
			// Generate a unique key for the resource
			gvk := fmt.Sprintf("%s/%s", resource.ApiVersion, resource.Kind)

			// Extract resource manifest for YAML diffing later
			resourceData, err := json.Marshal(resource)

			// Create resource info
			resourceInfo := ResourceInfo{
				GroupVersionKind:  gvk,
				Namespace:         resource.Metadata.Namespace,
				Name:              resource.Metadata.Name,
				UID:               resource.Metadata.UID,
				ResourceVersion:   resource.Metadata.ResourceVersion,
				CreationTimestamp: resource.Metadata.CreationTimestamp,
				SpecHash:          resource.Metadata.ResourceVersion, // Fall back to resource version if hash fails
			}

			// Calculate spec hash
			if resource.Spec != nil {
				if hash, err := internal_k8s.CalculateSpecHash(resource.Spec); err == nil {
					resourceInfo.SpecHash = hash
				}
			}

			// Add YAML manifest if available
			if err == nil {
				var obj interface{}
				_ = json.Unmarshal(resourceData, &obj)
				if yamlData, err := yaml.Marshal(obj); err == nil {
					resourceInfo.Manifest = string(yamlData)
				}
			}

			// Add to snapshot
			key := fmt.Sprintf("%s|%s|%s", gvk, resource.Metadata.Namespace, resource.Metadata.Name)
			snapshot.Resources[key] = resourceInfo
		}
	}

	return snapshot, nil
}

// SaveToFile persists the snapshot to a temporary file
func (s *Snapshot) SaveToFile() error {
	// Create a temporary directory or use XDG_RUNTIME_DIR
	tempDir := os.TempDir()

	// Create a unique filename based on timestamp
	timestamp := s.Timestamp.Format("20060102-150405")
	namespace := s.Namespace
	if namespace == "" {
		namespace = "all-namespaces"
	}
	filename := filepath.Join(tempDir, fmt.Sprintf("k8s-rdiff-%s-%s.json", namespace, timestamp))

	// Marshal snapshot to JSON
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %v", err)
	}

	// Write snapshot to file
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot: %v", err)
	}

	return nil
}

// LoadFromFile loads a snapshot from a file
func LoadFromFile(filename string) (*Snapshot, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot: %v", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %v", err)
	}

	return &snapshot, nil
}

// CalculateSpecHash computes a hash for the resource spec
func CalculateSpecHash(spec interface{}) (string, error) {
	// Marshal spec to JSON
	data, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal spec: %v", err)
	}

	// Calculate MD5 hash
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:]), nil
}
