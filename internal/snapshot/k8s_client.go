package snapshot

import (
	"context"
	"fmt"
	"regexp"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesClient is a client for interacting with Kubernetes
type KubernetesClient struct {
	discoveryClient *discovery.DiscoveryClient
	dynamicClient   dynamic.Interface
	config          *rest.Config
}

// NewKubernetesClient creates a new Kubernetes client
func NewKubernetesClient() (*KubernetesClient, error) {
	// Load kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
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
	
	return &KubernetesClient{
		discoveryClient: discoveryClient,
		dynamicClient:   dynamicClient,
		config:          config,
	}, nil
}

// GetAPIResources discovers all API resources available in the cluster
func (c *KubernetesClient) GetAPIResources(ignorePattern *regexp.Regexp) ([]metav1.APIResource, error) {
	// Get server preferred resources
	_, resourceList, err := c.discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get server resources: %v", err)
	}
	
	var resources []metav1.APIResource
	
	// Filter resources
	for _, list := range resourceList {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		
		for _, resource := range list.APIResources {
			// Skip resources that can't be listed
			if !containsString(resource.Verbs, "list") {
				continue
			}
			
			// Skip resources matching ignore pattern
			kind := resource.Kind
			if ignorePattern != nil && ignorePattern.MatchString(kind) {
				continue
			}
			
			// Set group version
			resource.Group = gv.Group
			resource.Version = gv.Version
			
			resources = append(resources, resource)
		}
	}
	
	return resources, nil
}

// ListResources lists all resources of the specified API resource
func (c *KubernetesClient) ListResources(resource metav1.APIResource, namespace string) ([]unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group:    resource.Group,
		Version:  resource.Version,
		Resource: resource.Name,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var list *unstructured.UnstructuredList
	var err error
	
	if resource.Namespaced && namespace != "" {
		// List resources in the specified namespace
		list, err = c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else if resource.Namespaced {
		// List resources in all namespaces
		list, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		// List cluster-scoped resources
		list, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %v", resource.Name, err)
	}
	
	return list.Items, nil
}

// Helper function to check if a string is in a slice
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
