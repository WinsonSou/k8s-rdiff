package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/winson-sou/k8s-rdiff/internal/diff"
	"github.com/winson-sou/k8s-rdiff/internal/snapshot"
)

type state int

const (
	stateReady state = iota
	stateCapturingBaseline
	stateBaselineCaptured
	stateCapturingCurrent
	stateShowingDiff
	stateShowingResourceDetail
	stateError
)

// KeyMap defines the keybindings for the application
type KeyMap struct {
	Capture     key.Binding
	Continue    key.Binding
	Back        key.Binding
	Quit        key.Binding
	ForceQuit   key.Binding
	Help        key.Binding
	ToggleView  key.Binding
	Up          key.Binding
	Down        key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Enter       key.Binding
	FilterAll   key.Binding
	FilterAdded key.Binding
	FilterRemoved key.Binding
	FilterModified key.Binding
	Escape      key.Binding
	CopyYAML    key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.ForceQuit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Capture, k.Continue, k.Back, k.Escape},
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Enter},
		{k.FilterAll, k.FilterAdded, k.FilterRemoved, k.FilterModified},
		{k.ToggleView, k.CopyYAML, k.Help, k.Quit, k.ForceQuit},
	}
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Capture: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "capture baseline"),
		),
		Continue: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "continue to capture current state"),
		),
		Back: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		ToggleView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle view (table/yaml/json)"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view resource details"),
		),
		FilterAll: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "show all resources"),
		),
		FilterAdded: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "show added resources"),
		),
		FilterRemoved: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "show removed resources"),
		),
		FilterModified: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "show modified resources"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
		CopyYAML: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy YAML to clipboard"),
		),
	}
}

type FilterType string

const (
	FilterAll      FilterType = "all"
	FilterAdded    FilterType = "added"
	FilterRemoved  FilterType = "removed"
	FilterModified FilterType = "modified"
)

type Model struct {
	state             state
	keyMap            KeyMap
	help              help.Model
	spinner           spinner.Model
	width             int
	height            int
	namespace         string
	ignorePattern     string
	kubeconfigPath    string
	baseline          *snapshot.Snapshot
	current           *snapshot.Snapshot
	diffResult        *diff.DiffResult
	diffOutput        string
	viewport          viewport.Model
	table             table.Model
	error             error
	showHelp          bool
	outputFormat      string // table, yaml, json
	selectedResource  *diff.ResourceDiff
	clusterInfo       string // Current cluster name and context
	resourceFilter    FilterType // Current resource filter
	statusMessage     string     // Message to display (e.g., "Copied to clipboard")
	statusMessageTime time.Time  // When to hide the status message
}

// New returns a new instance of the application model
func New(namespace, ignorePattern, kubeconfigPath string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	h := help.New()
	h.ShowAll = true

	// Define column widths appropriate for table alignment
	columns := []table.Column{
		{Title: "OPERATION", Width: 12},
		{Title: "KIND", Width: 35},
		{Title: "NAMESPACE", Width: 20},
		{Title: "NAME", Width: 30},
		{Title: "RESOURCE VERSION", Width: 25},
		{Title: "SPEC HASH", Width: 25},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	
	t.SetStyles(tableStyles())

	return Model{
		state:          stateReady,
		keyMap:         DefaultKeyMap(),
		help:           h,
		spinner:        s,
		namespace:      namespace,
		ignorePattern:  ignorePattern,
		kubeconfigPath: kubeconfigPath,
		showHelp:       true,
		outputFormat:   "table",
		table:          t,
		resourceFilter: FilterAll,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles application updates based on messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.ForceQuit):
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Quit):
			if m.state == stateShowingResourceDetail {
				// Return to diff view when quitting from detail view
				m.state = stateShowingDiff
				return m, nil
			}
			
			if m.state != stateCapturingBaseline && m.state != stateCapturingCurrent {
				return m, tea.Quit
			}

		case key.Matches(msg, m.keyMap.Escape) && m.state == stateShowingResourceDetail:
			// Return to diff view when pressing ESC in resource detail view
			m.state = stateShowingDiff
			m.selectedResource = nil
			return m, nil

		case key.Matches(msg, m.keyMap.CopyYAML):
			if m.state == stateShowingResourceDetail && m.selectedResource != nil {
				var yamlManifest string
				
				// Determine which manifest to use based on the operation type
				switch m.selectedResource.Type {
				case diff.Added:
					if m.selectedResource.CurrentResource != nil && m.selectedResource.CurrentResource.Manifest != "" {
						yamlManifest = m.selectedResource.CurrentResource.Manifest
					}
				case diff.Removed:
					if m.selectedResource.BaselineResource != nil && m.selectedResource.BaselineResource.Manifest != "" {
						yamlManifest = m.selectedResource.BaselineResource.Manifest
					}
				case diff.Modified:
					// For modified resources, use the current (newer) state
					if m.selectedResource.CurrentResource != nil && m.selectedResource.CurrentResource.Manifest != "" {
						yamlManifest = m.selectedResource.CurrentResource.Manifest
					} else if m.selectedResource.BaselineResource != nil && m.selectedResource.BaselineResource.Manifest != "" {
						// Fallback to baseline if current manifest is unavailable
						yamlManifest = m.selectedResource.BaselineResource.Manifest
					}
				}
				
				if yamlManifest != "" {
					if err := clipboard.WriteAll(yamlManifest); err == nil {
						m.statusMessage = "✓ YAML copied to clipboard"
						m.statusMessageTime = time.Now().Add(3 * time.Second)
						return m, hideStatusMessageCmd(3)
					} else {
						m.statusMessage = "✗ Failed to copy to clipboard: " + err.Error()
						m.statusMessageTime = time.Now().Add(3 * time.Second)
						return m, hideStatusMessageCmd(3)
					}
				} else {
					m.statusMessage = "✗ No YAML manifest available to copy"
					m.statusMessageTime = time.Now().Add(3 * time.Second)
					return m, hideStatusMessageCmd(3)
				}
			}

		case key.Matches(msg, m.keyMap.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, m.keyMap.ToggleView) && m.state == stateShowingDiff:
			switch m.outputFormat {
			case "table":
				m.outputFormat = "yaml"
				cmd = m.updateDiffOutputCmd()
			case "yaml":
				m.outputFormat = "json"
				cmd = m.updateDiffOutputCmd()
			case "json":
				m.outputFormat = "table"
				cmd = m.updateDiffOutputCmd()
			}
			cmds = append(cmds, cmd)
			
		// Resource filtering keys
		case key.Matches(msg, m.keyMap.FilterAll) && m.state == stateShowingDiff:
			if m.resourceFilter != FilterAll {
				m.resourceFilter = FilterAll
				cmd = m.updateTableWithFilterCmd()
				cmds = append(cmds, cmd)
			}
			
		case key.Matches(msg, m.keyMap.FilterAdded) && m.state == stateShowingDiff:
			if m.resourceFilter != FilterAdded {
				m.resourceFilter = FilterAdded
				cmd = m.updateTableWithFilterCmd()
				cmds = append(cmds, cmd)
			}
			
		case key.Matches(msg, m.keyMap.FilterRemoved) && m.state == stateShowingDiff:
			if m.resourceFilter != FilterRemoved {
				m.resourceFilter = FilterRemoved
				cmd = m.updateTableWithFilterCmd()
				cmds = append(cmds, cmd)
			}
			
		case key.Matches(msg, m.keyMap.FilterModified) && m.state == stateShowingDiff:
			if m.resourceFilter != FilterModified {
				m.resourceFilter = FilterModified
				cmd = m.updateTableWithFilterCmd()
				cmds = append(cmds, cmd)
			}

		case key.Matches(msg, m.keyMap.Capture) && m.state == stateReady:
			m.state = stateCapturingBaseline
			cmd = m.captureBaselineCmd()
			cmds = append(cmds, cmd, m.spinner.Tick)

		case key.Matches(msg, m.keyMap.Continue) && m.state == stateBaselineCaptured:
			m.state = stateCapturingCurrent
			cmd = m.captureCurrentStateCmd()
			cmds = append(cmds, cmd, m.spinner.Tick)

		case key.Matches(msg, m.keyMap.Continue) && m.state == stateShowingDiff:
			// Move current snapshot to baseline position
			m.baseline = m.current
			m.current = nil
			m.diffResult = nil
			m.diffOutput = ""
			
			// Start capturing the new snapshot
			m.state = stateCapturingCurrent
			cmd = m.captureCurrentStateCmd()
			cmds = append(cmds, cmd, m.spinner.Tick)

		case key.Matches(msg, m.keyMap.Back):
			switch m.state {
			case stateBaselineCaptured:
				m.state = stateReady
				m.baseline = nil
			case stateShowingDiff:
				m.state = stateBaselineCaptured
				m.current = nil
				m.diffResult = nil
				m.diffOutput = ""
			case stateShowingResourceDetail:
				m.state = stateShowingDiff
				m.selectedResource = nil
			}
			
		// Add enter key to view resource details
		case key.Matches(msg, m.keyMap.Enter) && m.state == stateShowingDiff && m.outputFormat == "table":
			// Check if we have a valid selection
			if !m.table.Focused() {
				break
			}
			
			selectedRow := m.table.SelectedRow()
			if len(selectedRow) == 0 {
				break
			}
			
			// Find the selected resource
			resourceDiff := m.findSelectedResource(selectedRow)
			if resourceDiff != nil {
				m.selectedResource = resourceDiff
				m.state = stateShowingResourceDetail
				cmd = m.loadResourceDetailCmd()
				cmds = append(cmds, cmd)
			}
		}

		// Handle scrolling in the table view or viewport
		if m.state == stateShowingDiff {
			if m.outputFormat == "table" {
				// Table navigation
				switch {
				case key.Matches(msg, m.keyMap.Up):
					m.table.MoveUp(1)
					return m, nil
					
				case key.Matches(msg, m.keyMap.Down):
					m.table.MoveDown(1)
					return m, nil
					
				case key.Matches(msg, m.keyMap.PageUp):
					m.table.MoveUp(10)
					return m, nil
					
				case key.Matches(msg, m.keyMap.PageDown):
					m.table.MoveDown(10)
					return m, nil
				}
			} else {
				// Viewport navigation for yaml/json views
				switch {
				case key.Matches(msg, m.keyMap.Up):
					m.viewport.LineUp(1)
					
				case key.Matches(msg, m.keyMap.Down):
					m.viewport.LineDown(1)
					
				case key.Matches(msg, m.keyMap.PageUp):
					m.viewport.PageUp()
					
				case key.Matches(msg, m.keyMap.PageDown):
					m.viewport.PageDown()
				}
			}
		} else if m.state == stateShowingResourceDetail {
			// Viewport navigation for resource detail view
			switch {
			case key.Matches(msg, m.keyMap.Up):
				m.viewport.LineUp(1)
				
			case key.Matches(msg, m.keyMap.Down):
				m.viewport.LineDown(1)
				
			case key.Matches(msg, m.keyMap.PageUp):
				m.viewport.PageUp()
				
			case key.Matches(msg, m.keyMap.PageDown):
				m.viewport.PageDown()
			}
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		
		// Update viewport dimensions
		headerHeight := 6
		footerHeight := 2
		
		if m.showHelp {
			footerHeight = 6 // Give more space for the help view
		}
		
		viewportHeight := msg.Height - headerHeight - footerHeight
		m.viewport = viewport.New(msg.Width, viewportHeight)
		m.viewport.SetContent(m.diffOutput)
		
		// Update table dimensions
		m.table.SetHeight(viewportHeight)
		m.table.SetWidth(msg.Width)
		
		// Adjust column widths to fit the screen
		adjustedColumns := adjustColumnWidths(m.table.Columns(), msg.Width)
		m.table = table.New(
			table.WithColumns(adjustedColumns),
			table.WithFocused(true),
			table.WithHeight(viewportHeight),
		)
		m.table.SetStyles(tableStyles())
		
		// Restore table content if we're showing diff
		if m.state == stateShowingDiff && m.diffResult != nil && m.outputFormat == "table" {
			m.table.SetRows(buildTableRows(m.diffResult))
		}
		
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		
		// Update help
		m.help.Width = msg.Width

	case spinner.TickMsg:
		if m.state == stateCapturingBaseline || m.state == stateCapturingCurrent {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case baselineCapturedMsg:
		m.state = stateBaselineCaptured
		m.baseline = msg.snapshot
		if msg.err != nil {
			m.state = stateError
			m.error = msg.err
		}

	case currentStateCapturedMsg:
		if msg.err != nil {
			m.state = stateError
			m.error = msg.err
		} else {
			m.current = msg.snapshot
			m.state = stateShowingDiff
			
			// Compare snapshots
			m.diffResult = diff.Compare(m.baseline, m.current)
			cmd = m.updateDiffOutputCmd()
			cmds = append(cmds, cmd)
		}

	case diffOutputUpdatedMsg:
		m.diffOutput = msg.output
		
		// For table view, update the table with filtered resources
		if m.outputFormat == "table" {
			cmd = m.updateTableWithFilterCmd()
			cmds = append(cmds, cmd)
		} else {
			// For YAML/JSON view, update the viewport
			m.viewport.SetContent(m.diffOutput)
			m.viewport.GotoTop()
		}

	case resourceDetailLoadedMsg:
		m.viewport.SetContent(msg.output)
		m.viewport.GotoTop()

	case tableUpdatedMsg:
		m.table.SetRows(msg.rows)

	case clearStatusMessageMsg:
		m.statusMessage = ""
	}

	return m, tea.Batch(cmds...)
}

// Helper function to build table rows from diff result
func buildTableRows(diffResult *diff.DiffResult) []table.Row {
	var rows []table.Row
	
	// Add rows for added resources
	for _, res := range diffResult.Added {
		rows = append(rows, table.Row{
			"Added",
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			res.Resource.ResourceVersion,
			res.Resource.SpecHash,
		})
	}
	
	// Add rows for removed resources
	for _, res := range diffResult.Removed {
		rows = append(rows, table.Row{
			"Removed",
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			res.Resource.ResourceVersion,
			res.Resource.SpecHash,
		})
	}
	
	// Add rows for modified resources
	for _, res := range diffResult.Modified {
		rows = append(rows, table.Row{
			"Modified",
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			fmt.Sprintf("%s → %s", res.OldResourceVersion, res.NewResourceVersion),
			fmt.Sprintf("%s → %s", res.OldSpecHash, res.NewSpecHash),
		})
	}
	
	return rows
}

// Helper function to adjust column widths based on available space
func adjustColumnWidths(columns []table.Column, totalWidth int) []table.Column {
	// Define minimum widths for each column
	minWidths := []int{
		10,  // Operation
		20,  // Kind
		15,  // Namespace
		15,  // Name
		15,  // Resource Version
		15,  // Spec Hash
	}
	
	// Define flex factors (how much each column can grow)
	flexFactors := []int{
		1,  // Operation
		3,  // Kind
		2,  // Namespace
		3,  // Name
		2,  // Resource Version
		2,  // Spec Hash
	}
	
	// Calculate total minimum width
	totalMinWidth := 0
	for _, w := range minWidths {
		totalMinWidth += w
	}
	
	// Calculate total flex factor
	totalFlex := 0
	for _, f := range flexFactors {
		totalFlex += f
	}
	
	// Calculate extra space to distribute
	extraSpace := totalWidth - totalMinWidth - 5 // Account for borders and padding
	if extraSpace < 0 {
		extraSpace = 0
	}
	
	// Distribute extra space according to flex factors
	adjustedColumns := make([]table.Column, len(columns))
	for i, col := range columns {
		if i < len(minWidths) {
			flexSpace := 0
			if totalFlex > 0 {
				flexSpace = (extraSpace * flexFactors[i]) / totalFlex
			}
			adjustedColumns[i] = table.Column{
				Title: col.Title,
				Width: minWidths[i] + flexSpace,
			}
		} else {
			adjustedColumns[i] = col
		}
	}
	
	return adjustedColumns
}

// Helper method to find the selected resource
func (m Model) findSelectedResource(selectedRow table.Row) *diff.ResourceDiff {
	if len(selectedRow) < 6 || m.diffResult == nil {
		return nil
	}
	
	operation := selectedRow[0]
	kind := selectedRow[1]
	namespace := selectedRow[2]
	name := selectedRow[3]
	
	var resources []diff.ResourceDiff
	
	switch operation {
	case "Added":
		resources = m.diffResult.Added
	case "Removed":
		resources = m.diffResult.Removed
	case "Modified":
		resources = m.diffResult.Modified
	default:
		return nil
	}
	
	for i, res := range resources {
		if res.Resource.GroupVersionKind == kind && 
		   res.Resource.Namespace == namespace && 
		   res.Resource.Name == name {
			// Return a pointer to the actual resource in the diff result
			switch operation {
			case "Added":
				return &m.diffResult.Added[i]
			case "Removed":
				return &m.diffResult.Removed[i]
			case "Modified":
				return &m.diffResult.Modified[i]
			}
		}
	}
	
	return nil
}

// Command to load resource detail
func (m Model) loadResourceDetailCmd() tea.Cmd {
	return func() tea.Msg {
		if m.selectedResource == nil {
			return nil
		}
		
		var detailOutput strings.Builder
		
		detailOutput.WriteString(fmt.Sprintf("Kind: %s\n", m.selectedResource.Resource.GroupVersionKind))
		detailOutput.WriteString(fmt.Sprintf("Name: %s\n", m.selectedResource.Resource.Name))
		detailOutput.WriteString(fmt.Sprintf("Namespace: %s\n", m.selectedResource.Resource.Namespace))
		detailOutput.WriteString("---\n\n")
		
		// If it's an added or removed resource, just show the manifest
		if m.selectedResource.IsPresentInBaseline() && m.selectedResource.IsPresentInCurrent() {
			// It's a modified resource, show a diff
			detailOutput.WriteString("## YAML Diff (- old, + new)\n\n")
			
			// Get old and new manifests
			oldManifest := m.selectedResource.BaselineResource.Manifest
			newManifest := m.selectedResource.CurrentResource.Manifest
			
			// Generate a YAML diff
			yamlDiff := generateYAMLDiff(oldManifest, newManifest)
			detailOutput.WriteString(yamlDiff)
		} else if m.selectedResource.IsPresentInBaseline() {
			// Removed resource
			detailOutput.WriteString("## Removed Resource (Baseline Manifest)\n\n")
			detailOutput.WriteString(m.selectedResource.BaselineResource.Manifest)
		} else {
			// Added resource
			detailOutput.WriteString("## Added Resource (Current Manifest)\n\n")
			detailOutput.WriteString(m.selectedResource.CurrentResource.Manifest)
		}
		
		return resourceDetailLoadedMsg{output: detailOutput.String()}
	}
}

type resourceDetailLoadedMsg struct {
	output string
}

// generateYAMLDiff creates a simple text diff between two YAML documents
func generateYAMLDiff(oldYAML, newYAML string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldYAML, newYAML, false)
	
	var result strings.Builder
	
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			// Split the lines to add - prefix
			lines := strings.Split(diff.Text, "\n")
			for _, line := range lines {
				if line != "" {
					result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("- %s\n", line)))
				}
			}
		case diffmatchpatch.DiffInsert:
			// Split the lines to add + prefix
			lines := strings.Split(diff.Text, "\n")
			for _, line := range lines {
				if line != "" {
					result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(fmt.Sprintf("+ %s\n", line)))
				}
			}
		case diffmatchpatch.DiffEqual:
			// Add context lines without prefix
			result.WriteString(diff.Text)
		}
	}
	
	return result.String()
}

// New command to update table with filtered resources
func (m Model) updateTableWithFilterCmd() tea.Cmd {
	return func() tea.Msg {
		if m.diffResult == nil {
			return nil
		}
		
		var rows []table.Row
		
		// Filter resources based on the current filter
		switch m.resourceFilter {
		case FilterAll:
			rows = buildTableRows(m.diffResult)
		case FilterAdded:
			rows = buildTableRowsForAddedOnly(m.diffResult)
		case FilterRemoved:
			rows = buildTableRowsForRemovedOnly(m.diffResult)
		case FilterModified:
			rows = buildTableRowsForModifiedOnly(m.diffResult)
		}
		
		return tableUpdatedMsg{rows: rows}
	}
}

type tableUpdatedMsg struct {
	rows []table.Row
}

// Helper functions to build filtered table rows
func buildTableRowsForAddedOnly(diffResult *diff.DiffResult) []table.Row {
	var rows []table.Row
	
	// Add rows for added resources only
	for _, res := range diffResult.Added {
		rows = append(rows, table.Row{
			"Added",
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			res.Resource.ResourceVersion,
			res.Resource.SpecHash,
		})
	}
	
	return rows
}

func buildTableRowsForRemovedOnly(diffResult *diff.DiffResult) []table.Row {
	var rows []table.Row
	
	// Add rows for removed resources only
	for _, res := range diffResult.Removed {
		rows = append(rows, table.Row{
			"Removed",
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			res.Resource.ResourceVersion,
			res.Resource.SpecHash,
		})
	}
	
	return rows
}

func buildTableRowsForModifiedOnly(diffResult *diff.DiffResult) []table.Row {
	var rows []table.Row
	
	// Add rows for modified resources only
	for _, res := range diffResult.Modified {
		rows = append(rows, table.Row{
			"Modified",
			res.Resource.GroupVersionKind,
			res.Resource.Namespace,
			res.Resource.Name,
			fmt.Sprintf("%s → %s", res.OldResourceVersion, res.NewResourceVersion),
			fmt.Sprintf("%s → %s", res.OldSpecHash, res.NewSpecHash),
		})
	}
	
	return rows
}

// View renders the current UI
func (m Model) View() string {
	var s strings.Builder

	// Header based on current state
	switch m.state {
	case stateReady:
		s.WriteString("◆ Kubernetes Resource-Diff Utility\n\n")
		s.WriteString("Press 'c' to capture baseline snapshot\n\n")

	case stateCapturingBaseline:
		s.WriteString("◆ Kubernetes Resource-Diff Utility\n\n")
		s.WriteString(fmt.Sprintf("%s Capturing baseline snapshot... Please wait\n\n", m.spinner.View()))

	case stateBaselineCaptured:
		s.WriteString("◆ Kubernetes Resource-Diff Utility\n\n")
		captureTime := m.baseline.Timestamp.Format(time.RFC3339)
		s.WriteString(fmt.Sprintf("✅ Baseline captured at %s\n", captureTime))
		
		if m.namespace != "" {
			s.WriteString(fmt.Sprintf("   Namespace: %s\n\n", m.namespace))
		} else {
			s.WriteString("   All namespaces\n\n")
		}
		
		s.WriteString("Press 'c' to continue and capture current state\n\n")

	case stateCapturingCurrent:
		s.WriteString("◆ Kubernetes Resource-Diff Utility\n\n")
		s.WriteString(fmt.Sprintf("%s Capturing current state... Please wait\n\n", m.spinner.View()))

	case stateShowingDiff:
		s.WriteString("◆ Kubernetes Resource-Diff Utility\n\n")
		
		// Summary info
		baselineTime := m.baseline.Timestamp.Format(time.RFC3339)
		currentTime := m.current.Timestamp.Format(time.RFC3339)
		
		// Extract the namespace information from the baseline snapshot
		namespace := m.baseline.Namespace
		if namespace == "" {
			namespace = "all namespaces"
		}
		
		s.WriteString(fmt.Sprintf("Baseline: %s\n", baselineTime))
		s.WriteString(fmt.Sprintf("Current:  %s\n", currentTime))
		s.WriteString(fmt.Sprintf("Namespace: %s\n", namespace))
		
		if m.clusterInfo != "" {
			s.WriteString(fmt.Sprintf("Cluster: %s\n", m.clusterInfo))
		}
		
		s.WriteString(fmt.Sprintf("Filter: %s\n\n", m.resourceFilter))
		
		// Display resources based on output format
		if m.outputFormat == "table" {
			s.WriteString(m.table.View())
			
			// Add counts at the bottom
			total := len(m.diffResult.Added) + len(m.diffResult.Removed) + len(m.diffResult.Modified)
			countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
			s.WriteString("\n" + countStyle.Render(fmt.Sprintf(
				"Total: %d resources | Added: %d | Removed: %d | Modified: %d",
				total, len(m.diffResult.Added), len(m.diffResult.Removed), len(m.diffResult.Modified),
			)))
			
			// Hint for continuing to next snapshot
			s.WriteString("\n" + countStyle.Render("Press 'c' to capture a new snapshot (will compare against current state)"))
		} else {
			// YAML or JSON view
			s.WriteString(m.viewport.View())
		}
		
		// Show hints based on current mode
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
		if m.outputFormat == "table" {
			s.WriteString("\n" + hintStyle.Render("Use up/down arrows to select a resource and press Enter to view details"))
			s.WriteString("\n" + hintStyle.Render("Press 0-3 to filter resources (0=all, 1=added, 2=removed, 3=modified)"))
		}

	case stateShowingResourceDetail:
		// Show resource details
		s.WriteString(fmt.Sprintf("Resource Detail: %s/%s\n\n", 
			m.selectedResource.Resource.GroupVersionKind, m.selectedResource.Resource.Name))
		
		s.WriteString(m.viewport.View())
		
		// Show status message if present
		if m.statusMessage != "" {
			statusStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true).
				Padding(0, 1)
			
			s.WriteString("\n" + statusStyle.Render(m.statusMessage))
		}
		
		// Back hint
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
		s.WriteString("\n" + hintStyle.Render("Press 'b' or 'esc' or 'q' to go back to diff view, 'y' to copy YAML to clipboard"))
	case stateError:
		s.WriteString("⚠️ Error\n\n")
		s.WriteString(fmt.Sprintf("%v\n\n", m.error))
		s.WriteString("Press 'q' to quit or 'b' to go back\n")
	}

	// Footer
	var footer strings.Builder
	if m.showHelp {
		footer.WriteString(m.help.View(m.keyMap))
	} else {
		footer.WriteString("Press ? for help")
	}

	return lipgloss.JoinVertical(lipgloss.Left, s.String(), footer.String())
}

// Helper to create styled table
func tableStyles() table.Styles {
	s := table.DefaultStyles()
	
	// Style the header
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	
	// Style the selected row
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	
	// Add colors for different operation types
	s.Cell = s.Cell.
		PaddingLeft(1).
		PaddingRight(1)
	
	return s
}

// Message types
type baselineCapturedMsg struct {
	snapshot *snapshot.Snapshot
	err      error
}

type currentStateCapturedMsg struct {
	snapshot *snapshot.Snapshot
	err      error
}

type diffOutputUpdatedMsg struct {
	output string
}

type clearStatusMessageMsg struct{}

// Command to hide status messages after a delay
func hideStatusMessageCmd(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(time.Time) tea.Msg {
		return clearStatusMessageMsg{}
	})
}

// Commands
func (m Model) captureBaselineCmd() tea.Cmd {
	return func() tea.Msg {
		snapshot, err := snapshot.CaptureSnapshot(m.namespace, m.ignorePattern, m.kubeconfigPath)
		return baselineCapturedMsg{snapshot: snapshot, err: err}
	}
}

func (m Model) captureCurrentStateCmd() tea.Cmd {
	return func() tea.Msg {
		snapshot, err := snapshot.CaptureSnapshot(m.namespace, m.ignorePattern, m.kubeconfigPath)
		return currentStateCapturedMsg{snapshot: snapshot, err: err}
	}
}

func (m Model) updateDiffOutputCmd() tea.Cmd {
	return func() tea.Msg {
		var output strings.Builder
		
		switch m.outputFormat {
		case "json":
			diff.OutputJSON(m.diffResult, &output)
		case "yaml":
			diff.OutputYAML(m.diffResult, &output)
		default:
			// Table output is handled by the table component
			if m.diffResult.IsEmpty() {
				output.WriteString("No differences detected")
			}
		}
		
		return diffOutputUpdatedMsg{output: output.String()}
	}
}
