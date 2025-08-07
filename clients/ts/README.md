# Open Atlas Search TypeScript Client

A TypeScript client library for the Open Atlas Search API, providing full-text search capabilities with MongoDB Atlas Search compatibility.

## Features

- **Full TypeScript Support**: Complete type definitions for all API endpoints and responses
- **Authentication**: Built-in support for HTTP Basic Authentication
- **Error Handling**: Structured error handling with detailed error responses
- **Request Timeout**: Configurable request timeouts with proper cancellation
- **Comprehensive API Coverage**: All Open Atlas Search API endpoints supported
- **Convenience Methods**: Helper methods for common search patterns

## Installation

```bash
npm install @open-atlas-search/client
```

Or with yarn:

```bash
yarn add @open-atlas-search/client
```

## Quick Start

```typescript
import { OpenAtlasSearchClient } from '@open-atlas-search/client';

const client = new OpenAtlasSearchClient({
  baseUrl: 'http://localhost:8080',
  // Optional authentication
  username: 'your-username',
  password: 'your-password',
});

// Check API health
const health = await client.health();
console.log(health); // { status: 'healthy', service: 'open-atlas-search' }

// List available indexes
const indexes = await client.listIndexes();
console.log(indexes);

// Perform a simple text search
const results = await client.simpleSearch('my-index', 'search query');
console.log(results.hits);
```

## Configuration

The client accepts a configuration object with the following options:

```typescript
interface ClientConfig {
  baseUrl: string;           // Base URL of the Open Atlas Search API
  username?: string;         // Username for basic authentication (optional)
  password?: string;         // Password for basic authentication (optional)
  timeout?: number;          // Request timeout in milliseconds (default: 30000)
  headers?: Record<string, string>; // Additional headers to include in requests
}
```

## API Methods

### Health and Status

#### `health(): Promise<HealthResponse>`
Check the health status of the API.

```typescript
const health = await client.health();
// Returns: { status: 'healthy', service: 'open-atlas-search' }
```

#### `ready(): Promise<ReadyResponse>`
Check the readiness status of the API.

```typescript
const ready = await client.ready();
// Returns: { status: 'ready', service: 'open-atlas-search', checks: {...} }
```

### Index Management

#### `listIndexes(): Promise<ListIndexesResponse>`
List all available indexes.

```typescript
const indexes = await client.listIndexes();
// Returns: { indexes: [...], total: number }
```

#### `getIndexStatus(indexName: string): Promise<IndexStatusResponse>`
Get the status of a specific index.

```typescript
const status = await client.getIndexStatus('my-index');
// Returns: { service: string, status: string, index: IndexInfo }
```

#### `getIndexMapping(indexName: string): Promise<IndexMapping>`
Get the mapping (schema) of a specific index.

```typescript
const mapping = await client.getIndexMapping('my-index');
// Returns the index mapping object
```

### Search Operations

#### `search(indexName: string, searchRequest: SearchRequest): Promise<SearchResult>`
Perform a search with full control over the search query.

```typescript
const searchRequest: SearchRequest = {
  query: {
    bool: {
      must: [
        { match: { title: 'example' } }
      ]
    }
  },
  facets: {
    categoryFacet: {
      type: 'terms',
      field: 'category',
      size: 10
    }
  },
  size: 10,
  from: 0,
};

const results = await client.search('my-index', searchRequest);
```

#### `simpleSearch(indexName: string, query: string, options?): Promise<SearchResult>`
Perform a simple text search.

```typescript
const results = await client.simpleSearch('my-index', 'search query', {
  size: 20,
  from: 0,
  facets: {
    categoryFacet: { type: 'terms', field: 'category' }
  }
});
```

#### `searchAll(indexName: string, options?): Promise<SearchResult>`
Get all documents from an index (match_all query).

```typescript
const results = await client.searchAll('my-index', {
  size: 100,
  from: 0
});
```

## Search Query Format

The search queries follow Elasticsearch/MongoDB Atlas Search format. Here are some examples:

### Simple Match Query
```typescript
{
  query: {
    match: {
      title: 'search text'
    }
  }
}
```

### Boolean Query
```typescript
{
  query: {
    bool: {
      must: [
        { match: { title: 'example' } }
      ],
      filter: [
        { term: { status: 'published' } }
      ]
    }
  }
}
```

### Range Query
```typescript
{
  query: {
    range: {
      publishDate: {
        gte: '2023-01-01',
        lte: '2023-12-31'
      }
    }
  }
}
```

### Match All Query
```typescript
{
  query: {
    match_all: {}
  }
}
```

## Facets (Aggregations)

You can request facet aggregations along with your search:

```typescript
const searchRequest: SearchRequest = {
  query: { match_all: {} },
  facets: {
    categoryFacet: {
      type: 'terms',
      field: 'category',
      size: 10
    },
    tagsFacet: {
      type: 'terms',
      field: 'tags',
      size: 5
    }
  }
};
```

## Error Handling

The client provides structured error handling with the `OpenAtlasSearchError` class:

```typescript
try {
  const results = await client.search('my-index', searchRequest);
} catch (error) {
  if (error instanceof OpenAtlasSearchError) {
    console.log('API Error:', error.message);
    console.log('Error Code:', error.response.error);
    console.log('Status Code:', error.statusCode);
  } else {
    console.log('Network or other error:', error.message);
  }
}
```

## Response Types

### SearchResult
```typescript
interface SearchResult {
  hits: SearchHit[];
  total: number;
  facets?: Record<string, any>;
  maxScore: number;
}
```

### SearchHit
```typescript
interface SearchHit {
  _id: string;
  score: number;
  source: Record<string, any>;
  highlight?: Record<string, string[]>;
}
```

### IndexInfo
```typescript
interface IndexInfo {
  name: string;
  docCount: number;
  status: string;
  lastSync?: string;
  sync_progress?: string;
}
```

## Development

To build the client:

```bash
npm run build
```

To run tests:

```bash
npm test
```

To run in development mode with watch:

```bash
npm run dev
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
