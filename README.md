# OpenVC - VirtualCluster Operator

This is a Kubernetes operator for managing VirtualClusters using the [loft vcluster](https://www.vcluster.com/) Helm chart.

## Overview

The OpenVC operator provides a declarative way to manage VirtualClusters within your Kubernetes environment. It uses Helm to deploy and manage the lifecycle of VirtualClusters based on the `VirtualCluster` custom resource.

## Features

- Create, update, and delete VirtualClusters using Kubernetes CRDs
- Declaratively configure VirtualClusters using spec.values (directly corresponds to the Helm chart values)
- Automated lifecycle management with finalizers to ensure proper cleanup
- Based on Helm v3 and the official vcluster Helm chart (v0.24.1)

## Getting Started

### Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

### Installation

1. Install CRDs:

```bash
kubectl apply -f https://raw.githubusercontent.com/prakashmishra1598/openvc/main/config/crd/bases/core.openvc.dev_virtualclusters.yaml
```

2. Deploy the operator:

```bash
kubectl apply -f https://raw.githubusercontent.com/prakashmishra1598/openvc/main/config/default
```

### Creating a VirtualCluster

Create a VirtualCluster by applying a YAML manifest:

```yaml
apiVersion: core.openvc.dev/v1alpha1
kind: VirtualCluster
metadata:
  name: sample-vcluster
  namespace: default
spec:
  values:
    service:
      type: ClusterIP
    sync:
      ingresses:
        enabled: true
    storage:
      persistence: false
    telemetry:
      disabled: true
```

Apply the manifest:

```bash
kubectl apply -f virtualcluster.yaml
```

### Accessing the VirtualCluster

You can access your VirtualCluster using the vcluster CLI:

```bash
vcluster connect sample-vcluster -n default
```

Or you can get the kubeconfig locally:

```bash
kubectl get secret sample-vcluster-kubeconfig -n default -o jsonpath="{.data.config}" | base64 --decode > kc.yaml
kubectl --kubeconfig=vc-kc.yaml get pods -A
```

## Configuration

The `spec.values` field in the VirtualCluster CR directly maps to the values.yaml of the vcluster Helm chart. For all available configuration options, refer to the [vcluster documentation](https://www.vcluster.com/docs/architecture/configuration).

## Development

### Building the Operator

```bash
# Build the operator
make build

# Run the operator locally
make run
```

### Building the Docker Image

```bash
# Build the Docker image
make docker-build

# Push the Docker image
make docker-push
```

## License

This project is licensed under the Apache License 2.0.

