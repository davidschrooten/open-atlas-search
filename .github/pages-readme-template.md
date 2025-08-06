# Open Atlas Search Helm Repository

This is the official Helm repository for Open Atlas Search - a MongoDB full-text search service.

## Quick Start

### Add the Repository

```bash
helm repo add open-atlas-search https://davidschrooten.github.io/open-atlas-search
helm repo update
```

### Install the Chart

#### Standalone Mode
```bash
helm install my-search open-atlas-search/open-atlas-search \
  --set deploymentMode=standalone \
  --set image.repository=davidschrooten/open-atlas-search \
  --set image.tag=latest
```

#### Cluster Mode
```bash
helm install my-search-cluster open-atlas-search/open-atlas-search \
  --set deploymentMode=cluster \
  --set statefulSet.replicas=3 \
  --set cluster.bootstrap=true \
  --set image.repository=davidschrooten/open-atlas-search \
  --set image.tag=latest
```

## Available Charts

| Chart | Description | Latest Version |
|-------|-------------|----------------|
| open-atlas-search | MongoDB full-text search service with standalone and cluster modes | [Latest](https://github.com/davidschrooten/open-atlas-search/releases) |

## Documentation

- [Chart README](https://github.com/davidschrooten/open-atlas-search/blob/master/helm/README.md)
- [Configuration Options](https://github.com/davidschrooten/open-atlas-search/blob/master/helm/values.yaml)
- [GitHub Repository](https://github.com/davidschrooten/open-atlas-search)

## Support

For issues, questions, and contributions, please visit our [GitHub repository](https://github.com/davidschrooten/open-atlas-search).
