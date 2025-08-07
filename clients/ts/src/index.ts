/**
 * Open Atlas Search TypeScript Client
 * 
 * A TypeScript client library for the Open Atlas Search API, providing
 * full-text search capabilities with MongoDB Atlas Search compatibility.
 */

export { OpenAtlasSearchClient } from './client';
export {
  ClientConfig,
  ErrorResponse,
  OpenAtlasSearchError,
  HealthResponse,
  ReadyResponse,
  IndexInfo,
  ListIndexesResponse,
  IndexStatusResponse,
  FacetRequest,
  SearchRequest,
  SearchHit,
  SearchResult,
  IndexMapping,
} from './types';

// Default export for convenience
export { OpenAtlasSearchClient as default } from './client';
