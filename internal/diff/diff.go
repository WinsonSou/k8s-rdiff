package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/winson-sou/k8s-rdiff/internal/snapshot"
	"gopkg.in/yaml.v2"
)

// DiffType represents the type of difference
type DiffType string

const (
	Added    DiffType = "Added"
	Removed  DiffType = "Removed"
	Modified DiffType = "Modified"
)

// ResourceDiff represents a difference in a resource
type ResourceDiff struct {
	Type              DiffType                `json:"type"`
	Resource          snapshot.ResourceInfo   `json:"resource"`
	OldResourceVersion string                 `json:"oldResourceVersion,omitempty"`
	NewResourceVersion string                 `json:"newResourceVersion,omitempty"`
	OldSpecHash       string                  `json:"oldSpecHash,omitempty"`
	NewSpecHash       string                  `json:"newSpecHash,omitempty"`
	BaselineResource  *snapshot.ResourceInfo  `json:"-"` // Not included in JSON/YAML output
	CurrentResource   *snapshot.ResourceInfo  `json:"-"` // Not included in JSON/YAML output
}

// IsPresentInBaseline returns true if the resource exists in the baseline
func (r *ResourceDiff) IsPresentInBaseline() bool {
	return r.Type == Modified || r.Type == Removed
}

// IsPresentInCurrent returns true if the resource exists in the current state
func (r *ResourceDiff) IsPresentInCurrent() bool {
	return r.Type == Modified || r.Type == Added
}

// DiffResult contains all differences between snapshots
type DiffResult struct {
	Added    []ResourceDiff `json:"added"`
	Removed  []ResourceDiff `json:"removed"`
	Modified []ResourceDiff `json:"modified"`
}

// IsEmpty checks if there are any differences
func (d *DiffResult) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Modified) == 0
}

// Compare compares two snapshots and returns the differences
func Compare(baseline, current *snapshot.Snapshot) *DiffResult {
	result := &DiffResult{
		Added:    []ResourceDiff{},
		Removed:  []ResourceDiff{},
		Modified: []ResourceDiff{},
	}

	// Find added and modified resources
	for key, res := range current.Resources {
		if baseRes, exists := baseline.Resources[key]; !exists {
			// Resource was added
			resCopy := res
			result.Added = append(result.Added, ResourceDiff{
				Type:            Added,
				Resource:        res,
				CurrentResource: &resCopy,
			})
		} else if res.ResourceVersion != baseRes.ResourceVersion || res.SpecHash != baseRes.SpecHash {
			// Resource was modified
			resCopy := res
			baseResCopy := baseRes
			result.Modified = append(result.Modified, ResourceDiff{
				Type:              Modified,
				Resource:          res,
				OldResourceVersion: baseRes.ResourceVersion,
				NewResourceVersion: res.ResourceVersion,
				OldSpecHash:       baseRes.SpecHash,
				NewSpecHash:       res.SpecHash,
				BaselineResource:  &baseResCopy,
				CurrentResource:   &resCopy,
			})
		}
	}

	// Find removed resources
	for key, res := range baseline.Resources {
		if _, exists := current.Resources[key]; !exists {
			// Resource was removed
			resCopy := res
			result.Removed = append(result.Removed, ResourceDiff{
				Type:             Removed,
				Resource:         res,
				BaselineResource: &resCopy,
			})
		}
	}

	return result
}

// DisplayDiff outputs the diff result in the specified format
func DisplayDiff(diff *DiffResult, format string) {
	switch strings.ToLower(format) {
	case "json":
		OutputJSON(diff, os.Stdout)
	case "yaml":
		OutputYAML(diff, os.Stdout)
	default: // "table" is the default
		OutputTable(diff, os.Stdout)
	}
}

// OutputTable outputs the diff as a table
func OutputTable(diff *DiffResult, writer io.Writer) {
	// Initialize tabwriter
	w := tabwriter.NewWriter(writer, 0, 0, 3, ' ', tabwriter.TabIndent)

	// Use color for better readability
	addColor := color.New(color.FgGreen).SprintFunc()
	removeColor := color.New(color.FgRed).SprintFunc()
	modifyColor := color.New(color.FgYellow).SprintFunc()

	// Print header
	fmt.Fprintln(w, "OPERATION\tKIND\tNAMESPACE\tNAME\tRESOURCE VERSION\tSPEC HASH")

	// Print added resources
	for _, res := range diff.Added {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			addColor("Added"),
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			res.Resource.ResourceVersion,
			res.Resource.SpecHash,
		)
	}

	// Print removed resources
	for _, res := range diff.Removed {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			removeColor("Removed"),
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			res.Resource.ResourceVersion,
			res.Resource.SpecHash,
		)
	}

	// Print modified resources
	for _, res := range diff.Modified {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s -> %s\t%s -> %s\n",
			modifyColor("Modified"),
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			res.OldResourceVersion,
			res.NewResourceVersion,
			res.OldSpecHash,
			res.NewSpecHash,
		)
	}

	w.Flush()
}

// OutputJSON outputs the diff as JSON
func OutputJSON(diff *DiffResult, writer io.Writer) {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.Encode(diff)
}

// OutputYAML outputs the diff as YAML
func OutputYAML(diff *DiffResult, writer io.Writer) {
	data, err := yaml.Marshal(diff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to YAML: %v\n", err)
		return
	}
	
	writer.Write(data)
}

// outputTable is maintained for backward compatibility
func outputTable(diff *DiffResult, writer io.Writer) {
	OutputTable(diff, writer)
}

// outputJSON is maintained for backward compatibility
func outputJSON(diff *DiffResult, writer io.Writer) {
	OutputJSON(diff, writer)
}

// outputYAML is maintained for backward compatibility
func outputYAML(diff *DiffResult, writer io.Writer) {
	OutputYAML(diff, writer)
}
