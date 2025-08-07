/**
 * Configuration for the Open Atlas Search client
 */
export interface ClientConfig {
  /** Base URL of the Open Atlas Search API */
  baseUrl: string;
  /** Username for basic authentication (if required) */
  username?: string;
  /** Password for basic authentication (if required) */
  password?: string;
  /** Request timeout in milliseconds (default: 30000) */
  timeout?: number;
  /** Additional headers to include in requests */
  headers?: Record<string, string>;
}

/**
 * Error response from the API
 */
export interface ErrorResponse {
  error: string;
  message: string;
  code: number;
}

/**
 * Custom error class for API errors
 */
export class OpenAtlasSearchError extends Error {
  constructor(
    message: string,
    public readonly response: ErrorResponse,
    public readonly statusCode: number
  ) {
    super(message);
    this.name = 'OpenAtlasSearchError';
  }
}

/**
 * Health check response
 */
export interface HealthResponse {
  status: 'healthy';
  service: string;
}

/**
 * Readiness check response
 */
export interface ReadyResponse {
  status: 'ready';
  service: string;
  checks: Record<string, string>;
}

/**
 * Index information
 */
export interface IndexInfo {
  name: string;
  docCount: number;
  status: string;
  lastSync?: string;
  sync_progress?: string;
}

/**
 * List indexes response
 */
export interface ListIndexesResponse {
  indexes: IndexInfo[];
  total: number;
}

/**
 * Index status response
 */
export interface IndexStatusResponse {
  service: string;
  status: string;
  index: IndexInfo;
}

/**
 * Facet request for search aggregations
 */
export interface FacetRequest {
  type: string;
  field: string;
  size?: number;
}

/**
 * Search request parameters
 */
export interface SearchRequest {
  /** Search query (Elasticsearch/MongoDB Atlas Search compatible) */
  query: Record<string, any>;
  /** Facet aggregations to compute */
  facets?: Record<string, FacetRequest>;
  /** Number of results to return (default: 10, max: 1000) */
  size?: number;
  /** Number of results to skip (for pagination) */
  from?: number;
  /** Highlight configuration */
  highlight?: Record<string, any>;
}

/**
 * Search hit/result
 */
export interface SearchHit {
  _id: string;
  score: number;
  source: Record<string, any>;
  highlight?: Record<string, string[]>;
}

/**
 * Search response
 */
export interface SearchResult {
  hits: SearchHit[];
  total: number;
  facets?: Record<string, any>;
  maxScore: number;
}

/**
 * Index mapping response
 */
export interface IndexMapping {
  [key: string]: any;
}
