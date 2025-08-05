# API Routes Documentation

## Updated API Structure

The API has been updated according to your requirements. Here are the available endpoints:

### Search Endpoints

#### `GET /indexes/{index}/search`
- **Purpose**: Search within a specific index
- **Parameters**: 
  - `{index}`: Name of the index to search
- **Request Body** (optional): JSON search request
- **Default Behavior**: 
  - If no request body provided, performs `match_all` query
  - Default pagination: 100 results per page
  - Maximum pagination: 10,000 total results
  - Maximum page size: 10,000
- **Example**:
  ```
  GET /indexes/mydb.mycollection.myindex/search
  ```

#### `GET /indexes/{index}/status`
- **Purpose**: Get status information for a specific index
- **Parameters**: 
  - `{index}`: Name of the index
- **Response**: Index-specific information including document count and last sync time
- **Example**:
  ```
  GET /indexes/mydb.mycollection.myindex/status
  ```

### General Endpoints

#### `GET /indexes`
- **Purpose**: List all available indexes
- **Response**: Array of all indexes with their information

#### `GET /health`
- **Purpose**: Basic health check
- **Response**: Simple health status

#### `GET /ready`
- **Purpose**: Readiness probe for Kubernetes
- **Response**: Detailed readiness information

## Search Behavior

### Elasticsearch-like Features
- **Empty Query Handling**: Automatically returns up to 10,000 documents when no query is provided
- **Default Pagination**: 100 results per page by default
- **Maximum Limits**: 
  - Maximum 10,000 total results retrievable
  - Maximum 10,000 results per page
  - Pagination limit: from + size ≤ 10,000

### Query Support
- **Match All**: `{"match_all": {}}`
- **Text Search**: Atlas Search compatible text queries
- **Term Queries**: Exact term matching
- **Compound Queries**: Boolean logic with must/should/mustNot
- **Wildcard Queries**: Pattern matching

## Removed Functionality
- Document indexing endpoints (handled automatically through MongoDB sync)
- Document deletion endpoints (handled automatically through MongoDB sync)

## Automatic Document Management
- Documents are automatically indexed when added to MongoDB
- Documents are automatically removed from search index when deleted from MongoDB
- No manual document management endpoints are needed

## Interface-Based Architecture
- Search engine now uses interfaces for better testability
- Comprehensive test coverage with mocking support
- Clean separation of concerns between API, search engine, and indexer components
