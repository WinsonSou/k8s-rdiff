# Kubernetes Resource-Diff Dialog Utility (k8s-rdiff)

A dialog-based CLI utility that captures the Kubernetes cluster state *before* and *after* an arbitrary user action, then highlights only the delta.

## Overview

Operations teams often need to understand precisely **what changed** inside a Kubernetes cluster after running administrative commands, CI/CD pipelines, or Helm upgrades. While traditional methods like `kubectl get all` list all resources, they lack state comparison and generate noisy output.

The **Resource-Diff Dialog Utility** provides an interactive workflow that:
1. Captures a baseline snapshot of resources
2. Waits for you to execute any command or action
3. Captures a second snapshot 
4. Shows only what changed (added, removed, modified)

## Features

- Dialog-based CLI workflow
- Support for both cluster-wide and namespace-scoped operation
- Multiple output formats (table, JSON, YAML)
- Colorized diff output for quick scanning
- Resource filtering capabilities
- Meaningful exit codes for automation

## Installation

### Prerequisites

- Go 1.18 or later
- Access to a Kubernetes cluster
- kubectl configured with appropriate permissions

### Building from Source

```bash
# Clone the repository
git clone https://github.com/winson-sou/k8s-rdiff.git
cd k8s-rdiff

# Build the binary
go build -o k8s-rdiff ./cmd/k8s-rdiff

# Move to a directory in your PATH (optional)
sudo mv k8s-rdiff /usr/local/bin/
```

## Usage

### Basic Usage

```bash
# Start a diff dialog for all namespaces
k8s-rdiff start

# Start a diff dialog for a specific namespace
k8s-rdiff start --namespace mynamespace
```

### Example Workflow

1. Start the diff dialog:
   ```
   $ k8s-rdiff start --namespace myapp
   Capturing baseline... done!
   Baseline captured at 2025-04-18T10:02:17Z
   Execute your command(s) and type 'continue' when done.
   >
   ```

2. Run your commands in another terminal, such as:
   ```
   $ helm upgrade myrelease mychart
   ```

3. Return to the k8s-rdiff dialog and type `continue`:
   ```
   > continue
   Capturing current state... done!
   
   OPERATION  KIND               NAMESPACE  NAME                   RESOURCE VERSION      SPEC HASH
   Added      apps/v1/Deployment myapp      new-deployment         123456               a1b2c3d4e5f6...
   Modified   v1/ConfigMap       myapp      my-config              123123 -> 123789     abc123 -> def456
   Removed    v1/Pod             myapp      old-pod-abcd1234       987654               f6e5d4c3b2a1...
   ```

### Output Formats

```bash
# Output as JSON
k8s-rdiff start --output json

# Output as YAML
k8s-rdiff start --output yaml
```

### Resource Filtering

```bash
# Ignore specific kinds of resources
k8s-rdiff start --ignore-kind "^events|^endpoints"
```

## Exit Codes

- **0**: No changes detected
- **2**: Changes detected
- **3+**: Error occurred

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
