# Open Atlas Search

A drop-in replacement for MongoDB Atlas Search functionality built in Go. This application provides full-text search, faceted search, and real-time indexing capabilities for self-managed MongoDB deployments.

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
- **Change Streams**: Real-time monitoring of document changes

## Installation

1. Clone the repository:
```bash
git clone https://github.com/david/open-atlas-search.git
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

Perform searches using HTTP POST to `/search`:

```bash
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "index": "myapp_products_default",
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

## Environment Variables

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

## Performance Tuning

- Adjust `batch_size` for bulk indexing performance
- Set `flush_interval` based on your consistency requirements
- Use appropriate field types (keyword vs text) for better performance
- Adjust polling interval based on your real-time requirements

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details
