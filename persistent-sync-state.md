# Configurable ID Field and Persistent Sync State

This document describes the enhanced features for configurable ID fields and persistent sync state that allow the indexing service to resume seamlessly after restarts.

## Features Added

### 1. Configurable ID Field

You can now specify a custom field name to use as the document ID for indexing, instead of being limited to MongoDB's `_id` field.

**Configuration:**
```yaml
indexes:
  - name: "search_index"
    database: "myapp"
    collection: "users"
    id_field: "user_id"  # Custom ID field (defaults to "_id")
    timestamp_field: "updated_at"
    definition:
      mappings:
        dynamic: true
```

**Benefits:**
- Use business-specific IDs (e.g., `user_id`, `product_code`, `order_number`)
- Support existing data models with custom primary keys
- Maintain search index consistency with application logic

### 2. Persistent Sync State

The service now saves its synchronization state to disk, allowing it to resume indexing from where it left off after restarts, crashes, or deployments.

**Configuration:**
```yaml
search:
  index_path: "./indexes"
  batch_size: 1000
  flush_interval: 30
  sync_state_path: "./sync_state.json"  # Path for sync state persistence
```

**Sync State Contents:**
- Last poll timestamp for each collection
- Last successful sync timestamp
- Total documents indexed
- Collection-specific configuration (ID field, timestamp field)

## How It Works

### Startup Process

1. **Load Sync State**: Service attempts to load existing sync state from disk
2. **Resume Indexing**: For each collection, resumes from the last known timestamp
3. **Fresh Start**: If no sync state exists, starts from the most recent document timestamp

### Runtime Operations

1. **Periodic State Saving**: Sync state is automatically saved every 30 seconds
2. **Incremental Updates**: Document counts and timestamps are updated as indexing progresses
3. **Graceful Shutdown**: Final sync state is saved when the service stops

### Recovery Scenarios

**Service Restart:**
```
2025-07-31T19:08:12Z [INFO] Loading sync state...
2025-07-31T19:08:12Z [INFO] Loaded sync state for 3 collections from ./sync_state.json
2025-07-31T19:08:12Z [INFO] Restored collection state for myapp.users, resuming from 2025-07-31T19:05:45Z
2025-07-31T19:08:12Z [INFO] Restored collection state for myapp.orders, resuming from 2025-07-31T19:06:12Z
```

**Crash Recovery:**
- Service automatically resumes from the last saved sync state
- At most 30 seconds of indexing progress may be lost (configurable)
- No manual intervention required

## Configuration Examples

### Example 1: E-commerce with Custom IDs
```yaml
indexes:
  - name: "product_search"
    database: "ecommerce"
    collection: "products"
    id_field: "sku"           # Use SKU as document ID
    timestamp_field: "modified_at"
    definition:
      mappings:
        dynamic: true
        fields:
          name:
            type: "text"
            analyzer: "standard"
          price:
            type: "numeric"

  - name: "order_search"
    database: "ecommerce"
    collection: "orders"
    id_field: "order_number"  # Use order number as document ID
    timestamp_field: "updated_at"
    definition:
      mappings:
        dynamic: true
```

### Example 2: Multi-tenant with ObjectIDs
```yaml
indexes:
  - name: "tenant_data"
    database: "saas"
    collection: "tenant_records"
    # id_field omitted - defaults to "_id"
    timestamp_field: "_id"    # Use ObjectID timestamp
    definition:
      mappings:
        dynamic: true
```

## Sync State File Format

The sync state is stored as JSON:

```json
{
  "collections": {
    "myapp.users": {
      "lastPollTime": "2025-07-31T19:08:12.123Z",
      "lastSyncTime": "2025-07-31T19:08:15.456Z",
      "indexName": "myapp.users.search_index",
      "collectionKey": "myapp.users",
      "timestampField": "updated_at",
      "idField": "user_id",
      "documentsIndexed": 15420
    },
    "myapp.orders": {
      "lastPollTime": "2025-07-31T19:08:10.789Z",
      "lastSyncTime": "2025-07-31T19:08:15.456Z",
      "indexName": "myapp.orders.search_index",
      "collectionKey": "myapp.orders",
      "timestampField": "created_at",
      "idField": "order_id",
      "documentsIndexed": 8756
    }
  },
  "lastSaved": "2025-07-31T19:08:15.456Z"
}
```

## Monitoring and Troubleshooting

### Log Messages

**Successful Restoration:**
```
Restored collection state for myapp.users, resuming from 2025-07-31T19:05:45Z
Polled 15 new/updated documents from myapp.users using timestamp field 'updated_at'
```

**New Collection Initialization:**
```
Initialized collection state for myapp.products, starting from 2025-07-31T19:08:12Z
```

**State Persistence:**
```
Sync state saved successfully
```

### Health Monitoring

Use the status endpoint to monitor sync progress:

```bash
curl http://localhost:8080/status | jq '.indexes[] | {name, lastSync, docCount}'
```

### Manual Recovery

If needed, you can manually edit or delete the sync state file:

```bash
# View current sync state
cat sync_state.json | jq

# Reset sync state (forces full re-index on next start)
rm sync_state.json

# Edit specific collection state
jq '.collections["myapp.users"].lastPollTime = "2025-07-31T18:00:00Z"' sync_state.json > sync_state_new.json
mv sync_state_new.json sync_state.json
```

## Best Practices

### ID Field Selection
- **Use business IDs** when they are unique and stable
- **Stick with `_id`** for maximum compatibility
- **Avoid changing ID fields** after initial indexing

### State File Management
- **Backup sync state** before major deployments
- **Monitor disk space** for the sync state file location
- **Use absolute paths** in containerized environments

### Recovery Planning
- **Test recovery scenarios** in development
- **Monitor logs** for successful state restoration
- **Set up alerts** for sync state save failures

## Performance Considerations

### Memory Usage
- Sync state is kept in memory and periodically saved
- Memory usage is minimal (typically < 1MB per 1000 collections)

### Disk I/O
- State saves are atomic (write to temp file, then rename)
- Configurable save interval balances performance vs. recovery time
- State file size grows linearly with number of collections

### Network Impact
- No additional MongoDB queries for state management
- Existing timestamp-based polling remains unchanged

## Migration Guide

### From Previous Versions

1. **Add ID field configuration** (optional):
   ```yaml
   # Before
   indexes:
     - name: "search_index"
       collection: "users"
   
   # After
   indexes:
     - name: "search_index"
       collection: "users"
       id_field: "user_id"  # Add if using custom IDs
   ```

2. **Configure sync state path** (recommended):
   ```yaml
   search:
     sync_state_path: "/var/lib/search/sync_state.json"
   ```

3. **Update deployment scripts** to preserve sync state:
   ```bash
   # Backup sync state before deployment
   cp sync_state.json sync_state.backup
   
   # Deploy new version
   kubectl apply -f deployment.yaml
   
   # Verify sync state restoration
   kubectl logs deployment/open-atlas-search | grep "Restored collection state"
   ```

### Breaking Changes
- None - all new features are backward compatible
- Default behavior remains unchanged if new fields are omitted
