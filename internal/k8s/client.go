package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/winson-sou/k8s-rdiff/internal/filter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// Resource represents a Kubernetes resource
type Resource struct {
	ApiVersion         string                 `json:"apiVersion"`
	Kind               string                 `json:"kind"`
	Metadata           Metadata               `json:"metadata"`
	Spec               map[string]interface{} `json:"spec,omitempty"`
	Status             map[string]interface{} `json:"status,omitempty"`
	AdditionalData     map[string]interface{} `json:"-"`
}

// Metadata contains resource metadata
type Metadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	UID               string            `json:"uid"`
	ResourceVersion   string            `json:"resourceVersion"`
	CreationTimestamp string            `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
}

// Client is a client for interacting with Kubernetes
type Client struct {
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath string) (*Client, error) {
	// Use the provided kubeconfig path or determine the default
	if kubeconfigPath == "" {
		kubeconfigEnv := os.Getenv("KUBECONFIG")
		if kubeconfigEnv != "" {
			kubeconfigPath = kubeconfigEnv
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get user home directory: %v", err)
			}
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// Build configuration from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig %s: %v", kubeconfigPath, err)
	}

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	return &Client{
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
	}, nil
}

// DiscoverResources discovers all API resources available in the cluster
func (c *Client) DiscoverResources(resourceFilter *filter.ResourceFilter) ([]string, error) {
	// Get server API resources
	_, apiResources, err := c.discoveryClient.ServerGroupsAndResources()
	if err != nil {
		// Handle partial discovery errors
		if !discovery.IsGroupDiscoveryFailedError(err) {
			return nil, fmt.Errorf("failed to discover API resources: %v", err)
		}
		// Continue with partial results if some groups failed
		fmt.Fprintf(os.Stderr, "Warning: partial discovery failure: %v\n", err)
	}
	
	resourceTypes := []string{}
	
	// Process each resource
	for _, resourceList := range apiResources {
		gv := resourceList.GroupVersion
		for _, r := range resourceList.APIResources {
			// Skip resources that can't be listed
			if !containsString(r.Verbs, "list") {
				continue
			}
			
			// Create resource type string
			resourceType := fmt.Sprintf("%s/%s", gv, r.Kind)
			
			// Skip resources that should be excluded by our filter
			if resourceFilter != nil && resourceFilter.ShouldExclude(resourceType) {
				fmt.Fprintf(os.Stderr, "Ignoring resource type: %s (matched exclusion pattern)\n", resourceType)
				continue
			}
			
			resourceTypes = append(resourceTypes, resourceType)
		}
	}
	
	return resourceTypes, nil
}

// ListResources lists all resources of the specified type in the given namespace
func (c *Client) ListResources(resourceType string, namespace string) ([]Resource, error) {
	// Parse resource type to get group, version, and kind
	parts := strings.Split(resourceType, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resource type format: %s", resourceType)
	}
	
	kind := parts[len(parts)-1]
	groupVersion := strings.Join(parts[:len(parts)-1], "/")
	
	// Find the resource in the API server
	resourceList, err := c.discoveryClient.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources for %s: %v", groupVersion, err)
	}
	
	var resource metav1.APIResource
	var foundResource bool
	
	for _, r := range resourceList.APIResources {
		if r.Kind == kind {
			resource = r
			foundResource = true
			break
		}
	}
	
	if !foundResource {
		return nil, fmt.Errorf("resource not found: %s", resourceType)
	}
	
	// Create group version resource
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid group version: %s", groupVersion)
	}
	
	gvr := schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource.Name,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// List the resources
	var list *unstructured.UnstructuredList
	if resource.Namespaced && namespace != "" {
		list, err = c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else if resource.Namespaced {
		list, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		// Cluster-scoped resources
		list, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %v", err)
	}
	
	// Convert to our Resource type
	var resources []Resource
	for _, item := range list.Items {
		// Extract spec and status safely
		var spec map[string]interface{}
		if specObj, ok := item.Object["spec"]; ok {
			if specMap, ok := specObj.(map[string]interface{}); ok {
				spec = specMap
			}
		}
		
		var status map[string]interface{}
		if statusObj, ok := item.Object["status"]; ok {
			if statusMap, ok := statusObj.(map[string]interface{}); ok {
				status = statusMap
			}
		}
		
		// Create resource
		resource := Resource{
			ApiVersion: item.GetAPIVersion(),
			Kind:       item.GetKind(),
			Metadata: Metadata{
				Name:              item.GetName(),
				Namespace:         item.GetNamespace(),
				UID:               string(item.GetUID()),
				ResourceVersion:   item.GetResourceVersion(),
				CreationTimestamp: item.GetCreationTimestamp().String(),
			},
			Spec:   spec,
			Status: status,
		}
		
		resources = append(resources, resource)
	}
	
	return resources, nil
}

// DefaultExcludedResourceTypes returns a list of noisy resources that should be excluded by default
// DEPRECATED: Use filter.DefaultNoisyResources() instead
func DefaultExcludedResourceTypes() []string {
	return filter.DefaultNoisyResources()
}

// CalculateSpecHash calculates a hash for the resource spec
func CalculateSpecHash(spec interface{}) (string, error) {
	// Convert spec to JSON
	data, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	
	// In a real implementation, we should use a proper hash function
	// For now we'll return a prefix of the JSON string for demonstration
	if len(data) > 20 {
		return string(data[:20]) + "...", nil
	}
	return string(data), nil
}

// Helper function
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
