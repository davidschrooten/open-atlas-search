# Open Atlas Search

[![Build and Publish Docker Image](https://github.com/davidschrooten/open-atlas-search/actions/workflows/docker.yml/badge.svg)](https://github.com/davidschrooten/open-atlas-search/actions/workflows/docker.yml)
[![Test](https://github.com/davidschrooten/open-atlas-search/actions/workflows/test.yml/badge.svg)](https://github.com/davidschrooten/open-atlas-search/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/davidschrooten/open-atlas-search)](https://goreportcard.com/report/github.com/davidschrooten/open-atlas-search)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A replacement for MongoDB Atlas Search functionality built in Go. Provides full-text search, faceted search, and real-time indexing capabilities for self-managed MongoDB deployments.

**Key Benefits:**
- No MongoDB change streams required - uses timestamp-based polling
- Atlas Search compatible functionality-wise
- Self-hosted alternative to MongoDB Atlas Search
- Easily deployable with Helm
- Low memory footprint, stellar performance

## API Routes

The API has been designed to match MongoDB Atlas Search functionality. Below are the main routes:

### POST /indexes/{index}/search
- **Purpose**: Search within a specific index
- **Parameters**: `{index}`: Name of the index
- **Request Body**: JSON search request with query, facets, size, and from parameters

### GET /indexes/{index}/status
- **Purpose**: Get status information for a specific index

### GET /indexes/{index}/mapping
- **Purpose**: Retrieve the mapping of a specific index

### GET /indexes
- **Purpose**: List all available indexes

### GET /health
- **Purpose**: Basic health check

### GET /ready
- **Purpose**: Readiness probe for comprehensive startup verification

## Features

- **Full-text Search**: Powered by Bleve search engine
- **Faceted Search**: Support for term, numeric, date, and boolean facets
- **Real-time Indexing**: Polling-based approach compatible with standalone MongoDB
- **Atlas Search Compatible**: Similar API and query syntax
- **Configuration-driven**: Define indexes like MongoDB Atlas Search
- **High Performance**: Goroutine-based concurrent processing
- **RESTful API**: HTTP endpoints for search operations

## Architecture

- **Cobra CLI**: Command-line interface with multiple commands
- **Viper Configuration**: YAML-based configuration management
- **Chi Router**: Lightweight HTTP router for API endpoints
- **Bleve Search**: Full-text search and indexing engine
- **MongoDB Driver**: Official Go driver for MongoDB connectivity
- **Timestamp Polling**: Real-time monitoring of document changes via timestamp queries

## Installation

1. Clone the repository:
```bash
git clone https://github.com/davidschrooten/open-atlas-search.git
cd open-atlas-search
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o open-atlas-search
```

## Configuration

Create a `config.yaml` file with your settings:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

mongodb:
  uri: "mongodb://localhost:27017"
  database: "myapp"
  username: ""
  password: ""
  timeout: 30

search:
  index_path: "./indexes"
  batch_size: 1000
  flush_interval: 30

indexes:
  - name: "default"
    database: "myapp"
    collection: "products"
    timestamp_field: "updated_at"  # Optional: custom timestamp field for polling (default: "updated_at")
    poll_interval: 5               # Optional: polling interval in seconds (default: 5)
    definition:
      mappings:
        dynamic: true
        fields:
          name:
            type: "text"
            analyzer: "standard"
          category:
            type: "keyword"
            facet: true
          price:
            type: "numeric"
            facet: true
```

## Usage

### Start the Server

```bash
./open-atlas-search server
```

Or with custom config:

```bash
./open-atlas-search server --config /path/to/config.yaml
```

### Search API

Perform searches using HTTP POST to `/indexes/{index}/search`:

```bash
curl -X POST http://localhost:8080/indexes/default/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "text": {
        "query": "laptop",
        "path": "name"
      }
    },
    "facets": {
      "categories": {
        "type": "terms",
        "field": "category",
        "size": 10
      }
    },
    "size": 20,
    "from": 0
  }'
```

### Query Types

#### Text Search
```json
{
  "text": {
    "query": "search terms",
    "path": "field_name"
  }
}
```

#### Term Search
```json
{
  "term": {
    "path": "field_name",
    "value": "exact_value"
  }
}
```

#### Compound Search
```json
{
  "compound": {
    "must": [
      {"text": {"query": "laptop", "path": "name"}},
      {"term": {"path": "category", "value": "electronics"}}
    ],
    "should": [
      {"term": {"path": "brand", "value": "apple"}}
    ]
  }
}
```

#### Wildcard Search
```json
{
  "wildcard": {
    "path": "field_name",
    "value": "pattern*"
  }
}
```

### Faceted Search

Request facets alongside search results:

```json
{
  "query": {"text": {"query": "laptop", "path": "name"}},
  "facets": {
    "price_ranges": {
      "type": "numeric",
      "field": "price",
      "size": 5
    },
    "categories": {
      "type": "terms",
      "field": "category",
      "size": 10
    }
  }
}
```

## Persistent Sync State

The sync state is saved to disk, allowing the application to resume indexing from the last checkpoint after restarts or crashes.

### Configuration

```yaml
search:
  index_path: "./indexes"
  sync_state_path: "./sync_state.json"
```

### Sync State File Format

The sync state file stores the last poll timestamp for collections, enabling seamless recovery.

You can override configuration using environment variables with the `OAS_` prefix:

```bash
export OAS_MONGODB_URI="mongodb://user:pass@localhost:27017"
export OAS_SERVER_PORT=9090
export OAS_SEARCH_BATCH_SIZE=2000
```

## Index Management

Indexes are automatically created and maintained based on your configuration. The system will:

1. Create Bleve indexes on startup
2. Perform initial bulk indexing of existing documents
3. Poll MongoDB for new/updated documents at regular intervals
4. Handle document insertions, updates, and deletions

## Field Types

Supported field types in index definitions:

- `text`: Full-text searchable fields
- `keyword`: Exact-match fields, good for faceting
- `numeric`: Numeric values with range search support
- `date`: Date/datetime fields
- `boolean`: Boolean values

## Kubernetes Deployment

For Kubernetes deployment with Bitnami MongoDB:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: open-atlas-search
spec:
  replicas: 1
  selector:
    matchLabels:
      app: open-atlas-search
  template:
    metadata:
      labels:
        app: open-atlas-search
    spec:
      containers:
      - name: open-atlas-search
        image: open-atlas-search:latest
        ports:
        - containerPort: 8080
        env:
        - name: OAS_MONGODB_URI
          value: "mongodb://mongodb-service:27017"
        - name: OAS_MONGODB_DATABASE
          value: "myapp"
        volumeMounts:
        - name: config
          mountPath: /etc/config
        - name: indexes
          mountPath: /app/indexes
      volumes:
      - name: config
        configMap:
          name: open-atlas-search-config
      - name: indexes
        persistentVolumeClaim:
          claimName: open-atlas-search-indexes
```

## Configuration Options

```yaml
search:
  index_path: "./indexes"
  batch_size: 1000
  flush_interval: 30
  sync_state_path: "./sync_state.json"
  worker_count: 4          # Number of concurrent workers
  bulk_indexing: true      # Enable bulk indexing
```

## Performance Tuning

- Adjust `batch_size` for bulk indexing performance
- Set `flush_interval` based on your consistency requirements
- Use appropriate field types (`keyword` vs `text`) for better performance
- Configure polling intervals based on real-time requirements
- Tune `worker_count` for concurrent processing

## Health Checks

### Health and Readiness Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 15
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
  successThreshold: 1
```

## Status Monitoring

The status endpoint provides detailed information about synchronization across all indexes:

### Example Response

```json
{
  "service": "open-atlas-search",
  "status": "running",
  "indexes": [
    {
      "name": "products",
      "docCount": 1500,
      "status": "active",
      "lastSync": "2025-07-31T18:57:24Z"
    }
  ]
}
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details
