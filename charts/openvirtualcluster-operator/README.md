# OpenVirtualCluster Operator Helm Chart

This Helm chart installs the OpenVirtualCluster Operator in your Kubernetes cluster.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
helm install my-release ./charts/openvirtualcluster-operator
```

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
helm delete my-release
```

## Configuration

The following table lists the configurable parameters of the OpenVirtualCluster Operator chart and their default values.

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/openvirtualcluster/openvirtualcluster-operator` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag | `latest` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `nameOverride` | Override the name of the chart | `""` |
| `fullnameOverride` | Override the full name of the chart | `""` |
| `serviceAccount.create` | If true, create a new service account | `true` |
| `serviceAccount.annotations` | Annotations for service account | `{}` |
| `serviceAccount.name` | Service account name to use, if not set and create is true, a name is generated | `""` |
| `podAnnotations` | Pod annotations | `{}` |
| `podSecurityContext` | Pod security context | `{}` |
| `securityContext` | Container security context | `{}` |
| `resources` | CPU/memory resource requests/limits | `{}` |
| `nodeSelector` | Node selector labels | `{}` |
| `tolerations` | List of tolerations | `[]` |
| `affinity` | Node affinity | `{}` |
| `operator.leaderElection` | Enable leader election | `false` |
| `operator.webhook` | Enable webhook server | `false` |
| `operator.logLevel` | Log level | `info` |
| `operator.metricsBindAddress` | Metrics bind address | `"0"` |
| `operator.secureMetrics` | Enable secure metrics | `true` |
| `operator.healthProbeBindAddress` | Health probe bind address | `":8081"` |
| `operator.enableHTTP2` | Enable HTTP/2 | `false` |
| `crds.install` | Install CRDs | `true` |
| `rbac.create` | Create RBAC resources | `true` |
| `metrics.enabled` | Enable metrics | `true` |
| `metrics.serviceMonitor.enabled` | Enable ServiceMonitor for Prometheus Operator | `false` |
| `metrics.serviceMonitor.additionalLabels` | Additional labels for ServiceMonitor | `{}` |
| `networkPolicy.enabled` | Enable NetworkPolicy | `false` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

For example:

```bash
helm install my-release \
  --set operator.leaderElection=true \
  ./charts/openvirtualcluster-operator
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example:

```bash
helm install my-release -f values.yaml ./charts/openvirtualcluster-operator
```

> **Tip**: You can use the default [values.yaml](values.yaml) 