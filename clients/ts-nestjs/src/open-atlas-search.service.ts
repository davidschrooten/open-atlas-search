import { Injectable, Inject, OnModuleInit, OnModuleDestroy, Logger } from '@nestjs/common';
import { 
  OpenAtlasSearchClient,
  HealthResponse,
  ReadyResponse,
  ListIndexesResponse,
  IndexStatusResponse,
  SearchRequest,
  SearchResult,
  IndexMapping,
} from 'oas-ts-client';
import { OPEN_ATLAS_SEARCH_CLIENT } from './constants';

/**
 * NestJS service wrapper for Open Atlas Search client
 * 
 * This service provides a NestJS-friendly interface to the Open Atlas Search API
 * with dependency injection support, logging integration, and lifecycle hooks.
 */
@Injectable()
export class OpenAtlasSearchService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(OpenAtlasSearchService.name);

  constructor(
    @Inject(OPEN_ATLAS_SEARCH_CLIENT)
    private readonly client: OpenAtlasSearchClient,
  ) {}

  async onModuleInit() {
    this.logger.log('Initializing Open Atlas Search Service');
    
    try {
      // Test connection on initialization
      const health = await this.client.health();
      this.logger.log(`Connected to Open Atlas Search API: ${health.service}`);
    } catch (error) {
      this.logger.warn('Failed to connect to Open Atlas Search API during initialization', error);
    }
  }

  async onModuleDestroy() {
    this.logger.log('Destroying Open Atlas Search Service');
  }

  /**
   * Get the underlying client instance
   * Useful for advanced use cases or direct API access
   */
  getClient(): OpenAtlasSearchClient {
    return this.client;
  }

  /**
   * Check the health status of the API
   * @returns Promise<HealthResponse>
   */
  async health(): Promise<HealthResponse> {
    try {
      const result = await this.client.health();
      this.logger.debug('Health check successful');
      return result;
    } catch (error) {
      this.logger.error('Health check failed', error);
      throw error;
    }
  }

  /**
   * Check the readiness status of the API
   * @returns Promise<ReadyResponse>
   */
  async ready(): Promise<ReadyResponse> {
    try {
      const result = await this.client.ready();
      this.logger.debug('Readiness check successful');
      return result;
    } catch (error) {
      this.logger.error('Readiness check failed', error);
      throw error;
    }
  }

  /**
   * List all available indexes
   * @returns Promise<ListIndexesResponse>
   */
  async listIndexes(): Promise<ListIndexesResponse> {
    try {
      const result = await this.client.listIndexes();
      this.logger.debug(`Listed ${result.total} indexes`);
      return result;
    } catch (error) {
      this.logger.error('Failed to list indexes', error);
      throw error;
    }
  }

  /**
   * Get the status of a specific index
   * @param indexName - Name of the index
   * @returns Promise<IndexStatusResponse>
   */
  async getIndexStatus(indexName: string): Promise<IndexStatusResponse> {
    try {
      const result = await this.client.getIndexStatus(indexName);
      this.logger.debug(`Retrieved status for index: ${indexName}`);
      return result;
    } catch (error) {
      this.logger.error(`Failed to get status for index: ${indexName}`, error);
      throw error;
    }
  }

  /**
   * Get the mapping (schema) of a specific index
   * @param indexName - Name of the index
   * @returns Promise<IndexMapping>
   */
  async getIndexMapping(indexName: string): Promise<IndexMapping> {
    try {
      const result = await this.client.getIndexMapping(indexName);
      this.logger.debug(`Retrieved mapping for index: ${indexName}`);
      return result;
    } catch (error) {
      this.logger.error(`Failed to get mapping for index: ${indexName}`, error);
      throw error;
    }
  }

  /**
   * Search documents in an index
   * @param indexName - Name of the index to search
   * @param searchRequest - Search parameters
   * @returns Promise<SearchResult>
   */
  async search(indexName: string, searchRequest: SearchRequest): Promise<SearchResult> {
    try {
      const result = await this.client.search(indexName, searchRequest);
      this.logger.debug(
        `Search completed for index: ${indexName}, found ${result.total} results`
      );
      return result;
    } catch (error) {
      this.logger.error(`Search failed for index: ${indexName}`, error);
      throw error;
    }
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
    try {
      const result = await this.client.simpleSearch(indexName, query, options);
      this.logger.debug(
        `Simple search completed for index: ${indexName}, query: "${query}", found ${result.total} results`
      );
      return result;
    } catch (error) {
      this.logger.error(`Simple search failed for index: ${indexName}, query: "${query}"`, error);
      throw error;
    }
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
    try {
      const result = await this.client.searchAll(indexName, options);
      this.logger.debug(
        `Search all completed for index: ${indexName}, found ${result.total} results`
      );
      return result;
    } catch (error) {
      this.logger.error(`Search all failed for index: ${indexName}`, error);
      throw error;
    }
  }
}
