import fetch, { Response } from 'node-fetch';
import {
  ClientConfig,
  ErrorResponse,
  OpenAtlasSearchError,
  HealthResponse,
  ReadyResponse,
  ListIndexesResponse,
  IndexStatusResponse,
  SearchRequest,
  SearchResult,
  IndexMapping,
} from './types';

/**
 * Open Atlas Search TypeScript Client
 * 
 * Provides a TypeScript interface to the Open Atlas Search API
 * with support for authentication, error handling, and all API endpoints.
 */
export class OpenAtlasSearchClient {
  private readonly baseUrl: string;
  private readonly timeout: number;
  private readonly headers: Record<string, string>;

  constructor(config: ClientConfig) {
    this.baseUrl = config.baseUrl.replace(/\/$/, ''); // Remove trailing slash
    this.timeout = config.timeout || 30000; // Default 30 seconds
    
    this.headers = {
      'Content-Type': 'application/json',
      'User-Agent': 'open-atlas-search-client-ts/1.0.0',
      ...config.headers,
    };

    // Add basic authentication if credentials are provided
    if (config.username && config.password) {
      const credentials = Buffer.from(`${config.username}:${config.password}`).toString('base64');
      this.headers['Authorization'] = `Basic ${credentials}`;
    }
  }

  /**
   * Make an HTTP request to the API
   */
  private async request<T>(
    method: 'GET' | 'POST',
    path: string,
    body?: any
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response: Response = await fetch(url, {
        method,
        headers: this.headers,
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      const contentType = response.headers.get('content-type');
      const isJson = contentType && contentType.includes('application/json');

      if (!response.ok) {
        if (isJson) {
          const errorData: ErrorResponse = await response.json() as ErrorResponse;
          throw new OpenAtlasSearchError(
            errorData.message || `HTTP ${response.status}: ${response.statusText}`,
            errorData,
            response.status
          );
        } else {
          const errorText = await response.text();
          throw new OpenAtlasSearchError(
            `HTTP ${response.status}: ${errorText || response.statusText}`,
            {
              error: 'http_error',
              message: errorText || response.statusText,
              code: response.status,
            },
            response.status
          );
        }
      }

      if (isJson) {
        return await response.json() as T;
      } else {
        // Handle non-JSON responses
        const text = await response.text();
        return text as unknown as T;
      }
    } catch (error) {
      clearTimeout(timeoutId);
      
      if (error instanceof OpenAtlasSearchError) {
        throw error;
      }
      
      // Type guard for error with name property (AbortError)
      if (error && typeof error === 'object' && 'name' in error && error.name === 'AbortError') {
        throw new OpenAtlasSearchError(
          `Request timeout after ${this.timeout}ms`,
          {
            error: 'timeout',
            message: `Request timeout after ${this.timeout}ms`,
            code: 408,
          },
          408
        );
      }

      // Type guard for error with message property
      const errorMessage = error && typeof error === 'object' && 'message' in error 
        ? String(error.message) 
        : 'Unknown error occurred';

      throw new OpenAtlasSearchError(
        errorMessage,
        {
          error: 'network_error',
          message: errorMessage,
          code: 0,
        },
        0
      );
    }
  }

  /**
   * Check the health status of the API
   * @returns Promise<HealthResponse>
   */
  async health(): Promise<HealthResponse> {
    return this.request<HealthResponse>('GET', '/health');
  }

  /**
   * Check the readiness status of the API
   * @returns Promise<ReadyResponse>
   */
  async ready(): Promise<ReadyResponse> {
    return this.request<ReadyResponse>('GET', '/ready');
  }

  /**
   * List all available indexes
   * @returns Promise<ListIndexesResponse>
   */
  async listIndexes(): Promise<ListIndexesResponse> {
    return this.request<ListIndexesResponse>('GET', '/indexes');
  }

  /**
   * Get the status of a specific index
   * @param indexName - Name of the index
   * @returns Promise<IndexStatusResponse>
   */
  async getIndexStatus(indexName: string): Promise<IndexStatusResponse> {
    if (!indexName || indexName.trim() === '') {
      throw new Error('Index name is required');
    }
    return this.request<IndexStatusResponse>('GET', `/indexes/${encodeURIComponent(indexName)}/status`);
  }

  /**
   * Get the mapping (schema) of a specific index
   * @param indexName - Name of the index
   * @returns Promise<IndexMapping>
   */
  async getIndexMapping(indexName: string): Promise<IndexMapping> {
    if (!indexName || indexName.trim() === '') {
      throw new Error('Index name is required');
    }
    return this.request<IndexMapping>('GET', `/indexes/${encodeURIComponent(indexName)}/mapping`);
  }

  /**
   * Search documents in an index
   * @param indexName - Name of the index to search
   * @param searchRequest - Search parameters
   * @returns Promise<SearchResult>
   */
  async search(indexName: string, searchRequest: SearchRequest): Promise<SearchResult> {
    if (!indexName || indexName.trim() === '') {
      throw new Error('Index name is required');
    }
    
    if (!searchRequest.query) {
      throw new Error('Search query is required');
    }

    // Validate size and from parameters
    if (searchRequest.size !== undefined && searchRequest.size < 0) {
      throw new Error('Size parameter cannot be negative');
    }
    
    if (searchRequest.from !== undefined && searchRequest.from < 0) {
      throw new Error('From parameter cannot be negative');
    }
    
    if (searchRequest.size !== undefined && searchRequest.size > 1000) {
      throw new Error('Size parameter cannot exceed 1000');
    }

    return this.request<SearchResult>('POST', `/indexes/${encodeURIComponent(indexName)}/search`, searchRequest);
  }

  /**
   * Convenience method for simple text search
   * @param indexName - Name of the index to search
   * @param query - Simple text query
   * @param options - Additional search options
   * @returns Promise<SearchResult>
   */
  async simpleSearch(
    indexName: string,
    query: string,
    options: {
      size?: number;
      from?: number;
      facets?: Record<string, { type: string; field: string; size?: number }>;
    } = {}
  ): Promise<SearchResult> {
    const searchRequest: SearchRequest = {
      query: {
        match: {
          _all: query,
        },
      },
      size: options.size,
      from: options.from,
      facets: options.facets,
    };

    return this.search(indexName, searchRequest);
  }

  /**
   * Convenience method for match-all search (get all documents)
   * @param indexName - Name of the index to search
   * @param options - Additional search options
   * @returns Promise<SearchResult>
   */
  async searchAll(
    indexName: string,
    options: {
      size?: number;
      from?: number;
      facets?: Record<string, { type: string; field: string; size?: number }>;
    } = {}
  ): Promise<SearchResult> {
    const searchRequest: SearchRequest = {
      query: {
        match_all: {},
      },
      size: options.size,
      from: options.from,
      facets: options.facets,
    };

    return this.search(indexName, searchRequest);
  }
}
