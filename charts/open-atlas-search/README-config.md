# Open Atlas Search Helm Chart Configuration

## Configuration Updates

This document describes the recent improvements to the Helm chart configuration.

### Issues Fixed

1. **Index Configuration Support**: The `indexes` configuration from the application config.yaml can now be configured through Helm values.
2. **Config File Mounting**: Fixed the ConfigMap mounting issue where the config file was not found by the application.

### New Configuration Options

#### Enhanced MongoDB Configuration
```yaml
config:
  mongodb:
    uri: "mongodb://mongodb:27017"
    database: "production"
    username: ""          # Optional MongoDB username
    password: ""          # Optional MongoDB password
    timeout: 30           # Connection timeout in seconds
```

#### Enhanced Search Configuration
```yaml
config:
  search:
    index_path: "/data/indexes"
    batch_size: 1000                    # Batch size for indexing operations
    flush_interval: 30                  # How often to flush indexes (seconds)
    sync_state_path: "/data/sync_state.json"  # Path to sync state file
```

#### Search Indexes Configuration
You can now configure search indexes directly through values.yaml:

```yaml
indexes:
  - name: "tags"                        # Index name
    database: "production"              # Source database
    collection: "tags"                  # Source collection
    distribution:
      replicas: 1                       # Number of replicas
      shards: 1                         # Number of shards
    definition:
      mappings:
        dynamic: true                   # Allow dynamic field mapping
        fields:
          - name: "tag_name_search"     # Field name in index
            field: "tag_name"           # Source field from MongoDB
            type: "text"                # Field type: text, keyword, numeric, date
            analyzer: "standard"        # Text analyzer (for text fields)
          - name: "source"
            field: "source"
            type: "keyword"
            facet: true                 # Enable faceted search
```

### File Structure Updates

#### New Files
- `README-config.md` - This configuration documentation

#### Modified Files
- `values.yaml` - Comprehensive configuration file with all options including MongoDB, search, authentication, clustering, and indexes
- `templates/configmap.yaml` - Updated to include all configuration options
- `templates/deployment.yaml` - Fixed config file path mounting
- `templates/statefulset.yaml` - Fixed config file path mounting

#### Removed Files
- `values-with-auth.yaml` - Merged into main values.yaml
- `values-with-indexes.yaml` - Merged into main values.yaml

### Usage Examples

#### Basic deployment (default configuration):
```bash
helm install my-search ./charts/open-atlas-search
```

#### Enable authentication:
```bash
helm install my-search ./charts/open-atlas-search \
  --set authentication.enabled=true \
  --set authentication.username=admin \
  --set authentication.password=secret123
```

#### Deploy with custom configuration file:
```bash
# Create your custom values file
cat > my-values.yaml << EOF
# Enable authentication
authentication:
  enabled: true
  username: "admin"
  password: "secret123"

# Custom MongoDB configuration
config:
  mongodb:
    uri: "mongodb://my-mongo:27017"
    database: "my_database"

# Define search indexes
indexes:
  - name: "products"
    database: "my_database"
    collection: "products"
    distribution:
      replicas: 1
      shards: 1
    definition:
      mappings:
        dynamic: true
        fields:
          - name: "title"
            field: "title"
            type: "text"
            analyzer: "standard"
          - name: "category"
            field: "category"
            type: "keyword"
            facet: true
EOF

helm install my-search ./charts/open-atlas-search --values my-values.yaml
```

#### Cluster deployment:
```bash
# Deploy a 3-node cluster
helm install my-search ./charts/open-atlas-search \
  --set deploymentMode=cluster \
  --set statefulSet.replicas=3 \
  --set cluster.bootstrap=true  # Only for the first deployment
```

### Configuration Validation

You can validate your Helm templates before deployment:

```bash
# Test template rendering
helm template test-release ./charts/open-atlas-search --values your-values.yaml

# Dry run deployment
helm install test-release ./charts/open-atlas-search --values your-values.yaml --dry-run
```

### Troubleshooting

#### Config File Not Found
If you still encounter "config file not found" errors, ensure:
1. The ConfigMap is being created (`kubectl get configmap`)
2. The volume is mounted correctly in the pod (`kubectl describe pod <pod-name>`)
3. The CONFIG_PATH environment variable is set to `/app/config/config.yaml`

#### Index Configuration Issues
- Ensure your index definitions match the expected YAML structure
- Validate that database and collection names are correct
- Check that field mappings correspond to actual fields in your MongoDB collections
