package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/winson-sou/k8s-rdiff/internal/filter"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/winson-sou/k8s-rdiff/internal/tui"
	"github.com/spf13/cobra"
)

func main() {
	var (
		namespace          string
		ignorePattern      string
		kubeconfigPath     string
		useDefaultExclusions bool
		includeSystemNamespaces bool
	)

	// Root command
	rootCmd := &cobra.Command{
		Use:   "k8s-rdiff",
		Short: "Kubernetes Resource Diff Tool",
		Long:  "Captures and compares Kubernetes resources before and after actions",
	}

	// Start command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the interactive resource diff utility",
		Run: func(cmd *cobra.Command, args []string) {
			// Convert command line options into our resource filter pattern
			var finalIgnorePattern string
			
			// Apply excluded resources
			if useDefaultExclusions {
				// We handle this directly in the snapshot.CaptureSnapshot function now
				// Using the filter.NewResourceFilter().WithNoisy() approach
				// No need to build a pattern here
			} else {
				// If not using default exclusions, use only what the user provided
				finalIgnorePattern = ignorePattern
			}

			// Display information about what's happening
			if useDefaultExclusions {
				fmt.Println("Filtering out noisy resources (events, endpoints, etc)...")
				if ignorePattern != "" {
					fmt.Printf("Also excluding resources matching pattern: %s\n", ignorePattern)
				}
			} else if ignorePattern != "" {
				fmt.Printf("Excluding only resources matching pattern: %s\n", ignorePattern)
			} else {
				fmt.Println("No resource filtering applied")
			}

			// Start the TUI application
			model := tui.New(namespace, finalIgnorePattern, kubeconfigPath)
			p := tea.NewProgram(model, tea.WithAltScreen())
			
			if _, err := p.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add flags to start command
	startCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace to monitor (empty for all namespaces)")
	startCmd.Flags().StringVarP(&ignorePattern, "ignore", "i", "", "Regex pattern to ignore additional resource kinds")
	startCmd.Flags().StringVarP(&kubeconfigPath, "kubeconfig", "k", "", "Path to kubeconfig file")
	startCmd.Flags().BoolVarP(&useDefaultExclusions, "exclude-noisy", "e", true, "Exclude noisy resources like Events, Endpoints, etc.")
	startCmd.Flags().BoolVarP(&includeSystemNamespaces, "include-system", "s", false, "Include system namespaces (kube-system, etc.)")

	// List resources command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List resource types excluded by default",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Noisy resources excluded by default:")
			fmt.Println("------------------------------------")
			for _, res := range filter.DefaultNoisyResources() {
				// Strip regex escaping for cleaner output
				cleanPattern := strings.ReplaceAll(res, "\\", "")
				fmt.Printf("  %s\n", cleanPattern)
			}
			
			fmt.Println("\nSystem namespaces (excluded with --include-system=false):")
			fmt.Println("-----------------------------------------------------")
			for _, ns := range filter.CommonSystemNamespaces() {
				fmt.Printf("  %s\n", ns)
			}
		},
	}

	// Add commands to root
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(listCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
