# K8s-RDiff Tasks

## Core Functionality
- [x] Project structure setup
- [x] Create basic CLI command structure
- [x] Implement resource discovery logic
- [x] Implement baseline snapshot functionality
- [x] Implement snapshot persistence
- [x] Implement comparison algorithm
- [x] Develop diff output formatting (table, JSON, YAML)
- [x] Add colorized terminal output
- [x] Support namespace filtering

## Features Based on PRD Requirements
- [x] FR-01: Baseline snapshot of API resources
- [x] FR-02: Support namespace flag for scope restriction
- [x] FR-03: Persist snapshot to temp file with timestamp
- [x] FR-04: Second snapshot capture on user cue
- [x] FR-05: Diff algorithm (Added/Removed/Changed)
- [x] FR-06: Output formats (table, JSON, YAML)
- [x] FR-07: Colorized terminal output
- [x] FR-08: Resource kind filtering
- [x] FR-09: Exit codes implementation
- [x] FR-10: Experimental watch mode

## Additional Tasks
- [x] Write comprehensive README
- [x] Add usage examples
- [x] Create installation instructions
- [x] Implement resource usage optimizations
- [ ] Write unit and integration tests
- [x] Add error handling and logging
- [x] Create release packaging

## Enhanced TUI Features
- [x] Interactive TUI implementation with Bubble Tea
- [x] Scrollable table view with proper column alignment
- [x] FR-11: Resource detail view - select a resource with up/down arrows and press enter to see detailed YAML diff
- [x] FR-12: Resource type filtering - toggle between showing all/added/removed/modified resources
- [x] FR-14: Smart default exclusions - automatically exclude noisy resources like Events, Endpoints, etc.
- [ ] FR-13: API resource selection - allow users to select which API resources to include in comparisons
- [ ] FR-15: Resource selection persistence - save resource selection preferences between runs

## Technical Enhancements
- [x] TE-01: Implement split-screen detailed diff view using a custom YAML differ with highlighted changes
- [x] TE-04: Implement resource type exclusion patterns with regex support
- [ ] TE-02: Add resource selection screen with checkbox interface for enabling/disabling resource types
- [ ] TE-03: Create configuration file for persistent settings (~/.k8s-rdiff.yaml)
- [ ] TE-05: Add keyboard shortcut hints in the UI for all interactive features
- [ ] TE-06: Add search functionality to quickly find resources in large clusters
- [ ] TE-07: Implement context-aware help system that shows relevant commands based on current view

## User Experience Improvements
- [x] UX-01: Add progress indicators for long-running operations
- [ ] UX-02: Implement resource grouping by namespace or kind
- [ ] UX-03: Add status bar with useful information (cluster name, context, etc.)
- [ ] UX-04: Create customizable theme support for different terminal color schemes
- [ ] UX-05: Add export functionality to save diff results to file
- [ ] UX-06: Implement bookmark feature for frequently monitored resources
