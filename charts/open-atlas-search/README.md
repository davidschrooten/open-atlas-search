# Open Atlas Search Helm Chart

A Helm chart for deploying Open Atlas Search - a MongoDB full-text search service that supports both standalone and cluster deployments.

## Features

- **Dual Deployment Modes**: Supports both standalone (Deployment) and cluster (StatefulSet) modes
- **Persistent Storage**: Configurable persistent storage for search indexes and cluster data
- **Auto-scaling**: Horizontal Pod Autoscaler support for standalone mode
- **Ingress**: Built-in ingress configuration with TLS support
- **Clustering**: Full Raft consensus-based clustering with automatic service discovery
- **Configuration Management**: Flexible configuration through ConfigMaps and environment variables

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- PV provisioner support in the underlying infrastructure (for persistent storage)

## Installation

### Adding the Helm Repository

```bash
# Add the repository
helm repo add open-atlas-search https://davidschrooten.github.io/open-atlas-search
helm repo update
```

### Quick Start - Standalone Mode

Deploy a standalone instance (single or multiple replicas without clustering):

```bash
helm install my-search open-atlas-search/open-atlas-search \
  --set deploymentMode=standalone \
  --set image.repository=davidschrooten/open-atlas-search \
  --set image.tag=latest \
  --set ingress.hosts[0].host=search.yourdomain.com \
  --set config.mongodb.uri=mongodb://your-mongodb:27017
```

### Quick Start - Cluster Mode

Deploy a 3-node cluster:

```bash
helm install my-search-cluster open-atlas-search/open-atlas-search \
  --set deploymentMode=cluster \
  --set statefulSet.replicas=3 \
  --set cluster.bootstrap=true \
  --set image.repository=davidschrooten/open-atlas-search \
  --set image.tag=latest \
  --set ingress.hosts[0].host=search.yourdomain.com \
  --set config.mongodb.uri=mongodb://your-mongodb:27017
```

## Configuration

### Deployment Modes

The chart supports two deployment modes controlled by the `deploymentMode` parameter:

#### Standalone Mode (`deploymentMode: "standalone"`)

- Uses Kubernetes **Deployment**
- Suitable for simple scaling scenarios
- Supports Horizontal Pod Autoscaler
- Uses shared PVC for persistence
- No clustering features

#### Cluster Mode (`deploymentMode: "cluster"`)

- Uses Kubernetes **StatefulSet**
- Each pod gets its own persistent storage
- Includes **headless service** for service discovery
- Supports Raft consensus clustering
- Automatic peer discovery within the cluster

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `deploymentMode` | Deployment mode: "standalone" or "cluster" | `standalone` |
| `replicaCount` | Number of replicas for standalone mode | `3` |
| `statefulSet.replicas` | Number of replicas for cluster mode | `3` |
| `image.repository` | Image repository | `davidschrooten/open-atlas-search` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |

### Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.server.port` | Server port | `3000` |
| `config.mongodb.uri` | MongoDB connection URI | `mongodb://mongodb:27017` |
| `config.search.index_path` | Path for search indexes | `/data/indexes` |

### Cluster Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `cluster.raftPort` | Raft communication port | `7946` |
| `cluster.grpcPort` | gRPC communication port | `50051` |
| `cluster.raftDir` | Raft data directory | `/data/raft` |
| `cluster.dataDir` | Cluster data directory | `/data/cluster` |
| `cluster.bootstrap` | Bootstrap first node | `false` |
| `cluster.joinAddr` | List of existing nodes to join | `[]` |
| `cluster.bindAddr` | Bind address for cluster communication | `0.0.0.0:7946` |

### Storage Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.enabled` | Enable persistent storage | `true` |
| `persistence.storageClass` | StorageClass for PVC | `""` |
| `persistence.accessMode` | Access mode for PVC | `ReadWriteOnce` |
| `persistence.size` | Size of persistent volume | `10Gi` |
| `persistence.mountPath` | Mount path for persistent storage | `/data` |

### Service & Ingress Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `80` |
| `service.targetPort` | Target port | `3000` |
| `ingress.enabled` | Enable ingress | `true` |
| `ingress.className` | Ingress class name | `nginx` |
| `ingress.hosts[0].host` | Hostname | `search.example.com` |

## Usage Examples

### Example 1: Standalone Deployment with Custom Configuration

```yaml
# values-standalone.yaml
deploymentMode: standalone
replicaCount: 2

image:
  repository: myregistry/open-atlas-search
  tag: "v1.2.0"

config:
  mongodb:
    uri: "mongodb://mongo-cluster:27017/searchdb"
  search:
    index_path: "/data/search-indexes"

persistence:
  enabled: true
  storageClass: "fast-ssd"
  size: 50Gi

resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "1000m"

ingress:
  hosts:
    - host: search.mycompany.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: search-tls
      hosts:
        - search.mycompany.com
```

Deploy with:
```bash
helm install my-search open-atlas-search/open-atlas-search -f values-standalone.yaml
```

### Example 2: Cluster Deployment with 5 Nodes

```yaml
# values-cluster.yaml
deploymentMode: cluster
statefulSet:
  replicas: 5

cluster:
  bootstrap: true  # Only set true for initial deployment
  raftPort: 7946
  grpcPort: 50051

persistence:
  enabled: true
  size: 100Gi
  storageClass: "premium-rwo"

resources:
  requests:
    memory: "2Gi"
    cpu: "1000m"
  limits:
    memory: "4Gi"
    cpu: "2000m"

# Anti-affinity to spread pods across nodes
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app.kubernetes.io/name
            operator: In
            values:
            - open-atlas-search
        topologyKey: kubernetes.io/hostname
```

Deploy with:
```bash
helm install search-cluster open-atlas-search/open-atlas-search -f values-cluster.yaml
```

### Example 3: Joining an Existing Cluster

To add nodes to an existing cluster:

```yaml
# values-join-cluster.yaml
deploymentMode: cluster
statefulSet:
  replicas: 3

cluster:
  bootstrap: false  # Don't bootstrap, join existing cluster
  joinAddr:
    - "search-cluster-open-atlas-search-0.search-cluster-open-atlas-search-headless.default.svc.cluster.local:7946"
    - "search-cluster-open-atlas-search-1.search-cluster-open-atlas-search-headless.default.svc.cluster.local:7946"
```

## Monitoring and Operations

### Health Checks

The chart includes built-in health checks:

- **Liveness Probe**: `/health` endpoint
- **Readiness Probe**: `/ready` endpoint

### Cluster Operations

#### Viewing Cluster Status

```bash
# Port-forward to a cluster node
kubectl port-forward search-cluster-open-atlas-search-0 3000:3000

# Check cluster status (assuming your API supports this)
curl http://localhost:3000/cluster/status
```

#### Scaling the Cluster

```bash
# Scale up the cluster
helm upgrade search-cluster open-atlas-search/open-atlas-search \
  --set statefulSet.replicas=5

# Scale down the cluster  
helm upgrade search-cluster open-atlas-search/open-atlas-search \
  --set statefulSet.replicas=3
```

#### Rolling Updates

```bash
# Update image version
helm upgrade search-cluster open-atlas-search/open-atlas-search \
  --set image.tag=v1.3.0
```

## Troubleshooting

### Common Issues

1. **Pods stuck in Pending state**
   - Check if PVC can be provisioned
   - Verify StorageClass exists
   - Check node resources

2. **Cluster nodes can't join**
   - Verify headless service is created
   - Check network policies
   - Confirm `joinAddr` values are correct

3. **High memory usage**
   - Adjust `resources.limits.memory`
   - Consider reducing `persistence.size` if using memory-based storage
   - Check MongoDB connection pool settings

### Debugging Commands

```bash
# Check pod logs
kubectl logs -f search-cluster-open-atlas-search-0

# Check service discovery
kubectl get svc search-cluster-open-atlas-search-headless

# Describe StatefulSet
kubectl describe statefulset search-cluster-open-atlas-search

# Check PVC status
kubectl get pvc

# View ConfigMap
kubectl get configmap search-cluster-open-atlas-search-config -o yaml
```

## Uninstalling

```bash
# Uninstall the release
helm uninstall my-search

# Optional: Clean up PVCs (data will be lost!)
kubectl delete pvc -l app.kubernetes.io/instance=my-search
```

## Values Reference

For a complete list of configurable values, see the [values.yaml](values.yaml) file or use:

```bash
helm show values open-atlas-search/open-atlas-search
```

## Contributing

Please refer to the main repository for contribution guidelines.

## License

This chart is licensed under the same license as the Open Atlas Search project.
