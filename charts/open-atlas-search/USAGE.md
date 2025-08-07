# Open Atlas Search - Quick Usage Guide

This guide shows common deployment scenarios for the Open Atlas Search Helm chart.

## Prerequisites

- Kubernetes cluster
- Helm 3.x installed
- MongoDB instance (can be deployed separately)

## Quick Start Deployments

### 1. Basic Deployment (Default Configuration)

Deploy with default settings - no authentication, no custom indexes:

```bash
helm install my-search ./charts/open-atlas-search
```

This creates:
- 3 replica deployment
- ClusterIP service on port 80
- Persistent volume for indexes (10Gi)
- Ingress with default hostname

### 2. Development Setup with Authentication

Enable authentication for development/testing:

```bash
helm install my-search ./charts/open-atlas-search \
  --set authentication.enabled=true \
  --set authentication.username=admin \
  --set authentication.password=secret123
```

### 3. Production Setup with Custom Configuration

Create a production values file:

```bash
cat > production-values.yaml << EOF
# Production configuration
replicaCount: 5

# Custom domain and TLS
ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: search.mycompany.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: mycompany-search-tls
      hosts:
        - search.mycompany.com

# Authentication with existing secret (recommended)
authentication:
  enabled: true
  existingSecret: "search-auth-secret"

# Custom MongoDB connection
config:
  mongodb:
    uri: "mongodb://mongo.mycompany.com:27017"
    database: "search_production"

# Define search indexes
indexes:
  - name: "products"
    database: "search_production"
    collection: "products"
    distribution:
      replicas: 2
      shards: 3
    definition:
      mappings:
        dynamic: false
        fields:
          - name: "title"
            field: "title"
            type: "text"
            analyzer: "standard"
          - name: "description"
            field: "description"
            type: "text"
            analyzer: "standard"
          - name: "category"
            field: "category"
            type: "keyword"
            facet: true
          - name: "price"
            field: "price"
            type: "numeric"
            facet: true
          - name: "brand"
            field: "brand"
            type: "keyword"
            facet: true

# Resource limits for production
resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 1000m
    memory: 2Gi

# Larger persistent storage
persistence:
  size: 100Gi
  storageClass: "fast-ssd"
EOF

# Deploy with production configuration
helm install my-search ./charts/open-atlas-search --values production-values.yaml
```

### 4. Cluster Mode Deployment

Deploy a 3-node search cluster:

```bash
# First deployment (bootstrap node)
helm install search-cluster ./charts/open-atlas-search \
  --set deploymentMode=cluster \
  --set statefulSet.replicas=3 \
  --set cluster.bootstrap=true

# After the first deployment is ready, you can scale or join additional clusters
```

### 5. Minimal Resource Setup

For development/testing with minimal resources:

```bash
helm install my-search ./charts/open-atlas-search \
  --set replicaCount=1 \
  --set persistence.size=1Gi \
  --set resources.limits.cpu=500m \
  --set resources.limits.memory=512Mi \
  --set resources.requests.cpu=250m \
  --set resources.requests.memory=256Mi \
  --set ingress.enabled=false
```

## Common Configuration Examples

### Enable Authentication with Existing Secret

First create the secret:
```bash
kubectl create secret generic search-auth --from-literal=username=admin --from-literal=password=mysecretpassword
```

Then deploy:
```bash
helm install my-search ./charts/open-atlas-search \
  --set authentication.enabled=true \
  --set authentication.existingSecret=search-auth
```

### Configure Multiple Indexes

```bash
cat > indexes-values.yaml << EOF
indexes:
  - name: "users"
    database: "myapp"
    collection: "users"
    distribution:
      replicas: 1
      shards: 1
    definition:
      mappings:
        dynamic: true
        fields:
          - name: "username"
            field: "username"
            type: "keyword"
          - name: "email"
            field: "email"
            type: "keyword"
          - name: "fullname"
            field: "fullname"
            type: "text"
            analyzer: "standard"
          
  - name: "posts"
    database: "myapp"
    collection: "posts"
    distribution:
      replicas: 1
      shards: 2
    definition:
      mappings:
        dynamic: false
        fields:
          - name: "title"
            field: "title"
            type: "text"
            analyzer: "standard"
          - name: "content"
            field: "content"
            type: "text"
            analyzer: "standard"
          - name: "tags"
            field: "tags"
            type: "keyword"
            facet: true
          - name: "created_at"
            field: "created_at"
            type: "date"
EOF

helm install my-search ./charts/open-atlas-search --values indexes-values.yaml
```

## Verification and Monitoring

### Check Deployment Status
```bash
# Check pods
kubectl get pods -l app.kubernetes.io/name=open-atlas-search

# Check services
kubectl get svc -l app.kubernetes.io/name=open-atlas-search

# Check ingress
kubectl get ingress -l app.kubernetes.io/name=open-atlas-search

# Check configmap
kubectl get configmap -l app.kubernetes.io/name=open-atlas-search
```

### View Configuration
```bash
# View the generated config
kubectl get configmap <release-name>-open-atlas-search-config -o yaml

# Check environment variables in pods
kubectl describe pod <pod-name>
```

### Test the Service
```bash
# Port forward to test locally
kubectl port-forward svc/<release-name>-open-atlas-search 8080:80

# Test health endpoint
curl http://localhost:8080/health

# Test with authentication (if enabled)
curl -u admin:secret123 http://localhost:8080/health
```

## Upgrading

```bash
# Upgrade with new values
helm upgrade my-search ./charts/open-atlas-search --values new-values.yaml

# Upgrade with inline overrides
helm upgrade my-search ./charts/open-atlas-search \
  --set image.tag=v2.0.0 \
  --set replicaCount=5
```

## Uninstalling

```bash
# Uninstall the release
helm uninstall my-search

# Note: PVCs are not automatically deleted
# Delete them manually if needed:
kubectl delete pvc -l app.kubernetes.io/name=open-atlas-search
```
