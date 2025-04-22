package filter

import (
	"regexp"
	"strings"
)

// ResourceFilter provides functionality for filtering Kubernetes resources
type ResourceFilter struct {
	IncludePatterns []string
	ExcludePatterns []string
	compiledFilter  *regexp.Regexp
}

// DefaultNoisyResources returns a list of regex patterns for API resources
// that frequently change but are typically not interesting to track
func DefaultNoisyResources() []string {
	return []string{
		// Event resources - very high volume and ephemerality
		"^events\\.k8s\\.io/v1/Event$",
		"^events\\.k8s\\.io/v1beta1/Event$",
		"^v1/Event$",
		
		// Endpoint resources - frequently change but are typically not interesting
		"^v1/Endpoints$",
		"^discovery\\.k8s\\.io/v1/EndpointSlice$",
		"^discovery\\.k8s\\.io/v1beta1/EndpointSlice$",
		
		// Lease resources - primarily used for leader election
		"^coordination\\.k8s\\.io/v1/Lease$",
		
		// Pod status updates - very frequent and noisy
		"^v1/Pod$",
		
		// Temporary and frequently changing objects
		"^admissionregistration\\.k8s\\.io/v1/MutatingWebhookConfiguration$",
		"^admissionregistration\\.k8s\\.io/v1/ValidatingWebhookConfiguration$",
		
		// Autoscaling resources that change frequently
		"^autoscaling/v[12]/HorizontalPodAutoscaler$",
		
		// FluxCD GitRepository status objects which change frequently
		"^source\\.toolkit\\.fluxcd\\.io/v1beta1/GitRepository$",
		"^source\\.toolkit\\.fluxcd\\.io/v1beta2/GitRepository$",
		
		// Prometheus resources that update frequently
		"^monitoring\\.coreos\\.com/v1/ServiceMonitor$",
		"^monitoring\\.coreos\\.com/v1/PodMonitor$",
		
		// Node resources that change frequently with status updates
		"^v1/Node$",
		
		// Helm releases managed by FluxCD
		"^helm\\.toolkit\\.fluxcd\\.io/v1/HelmRelease$",
		"^helm\\.toolkit\\.fluxcd\\.io/v2beta1/HelmRelease$",
		"^helm\\.toolkit\\.fluxcd\\.io/v2/HelmRelease$",
		"^helm\\.toolkit\\.fluxcd\\.io/v2beta2/HelmRelease$",
		
		// Kommander Helm release status tracking
		"^kommander\\.d2iq\\.io/v1alpha2/HelmReleaseStatus$",
		"^kommander\\.d2iq\\.io/v1alpha1/HelmReleaseStatus$",
	}
}

// CommonSystemNamespaces returns a list of system namespaces that might be excluded
func CommonSystemNamespaces() []string {
	return []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
		"flux-system",
	}
}

// NewResourceFilter creates a new resource filter
func NewResourceFilter() *ResourceFilter {
	return &ResourceFilter{
		IncludePatterns: []string{},
		ExcludePatterns: []string{},
	}
}

// WithNoisy adds the default noisy resource patterns to the exclude list
func (rf *ResourceFilter) WithNoisy() *ResourceFilter {
	rf.ExcludePatterns = append(rf.ExcludePatterns, DefaultNoisyResources()...)
	return rf
}

// WithExcludes adds custom exclude patterns
func (rf *ResourceFilter) WithExcludes(patterns []string) *ResourceFilter {
	rf.ExcludePatterns = append(rf.ExcludePatterns, patterns...)
	return rf
}

// WithIncludes adds specific include patterns (resources that should be included
// even if they match an exclude pattern)
func (rf *ResourceFilter) WithIncludes(patterns []string) *ResourceFilter {
	rf.IncludePatterns = append(rf.IncludePatterns, patterns...)
	return rf
}

// Compile prepares the filter for use
func (rf *ResourceFilter) Compile() error {
	// If no exclude patterns, there's nothing to compile
	if len(rf.ExcludePatterns) == 0 {
		rf.compiledFilter = nil
		return nil
	}
	
	// Compile all exclude patterns into a single regexp
	pattern := "(" + strings.Join(rf.ExcludePatterns, ")|(") + ")"
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	
	rf.compiledFilter = compiled
	return nil
}

// ShouldExclude determines if a resource should be excluded based on its attributes
func (rf *ResourceFilter) ShouldExclude(resourceType string) bool {
	if rf.compiledFilter == nil {
		return false
	}
	
	// Check against exclude patterns
	if rf.compiledFilter.MatchString(resourceType) {
		// Check if it matches any include patterns (which override excludes)
		for _, include := range rf.IncludePatterns {
			if regexp.MustCompile(include).MatchString(resourceType) {
				return false
			}
		}
		return true
	}
	
	return false
}
