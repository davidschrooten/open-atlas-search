import { OpenAtlasSearchClient } from '../client';
import { ClientConfig, OpenAtlasSearchError } from '../types';

// Mock fetch for testing
const mockFetch = jest.fn();
jest.spyOn(global, 'fetch').mockImplementation(mockFetch);

describe('OpenAtlasSearchClient', () => {
  let client: OpenAtlasSearchClient;
  const baseConfig: ClientConfig = {
    baseUrl: 'http://localhost:8080',
    timeout: 5000,
  };

  beforeEach(() => {
    client = new OpenAtlasSearchClient(baseConfig);
    mockFetch.mockClear();
  });

  describe('constructor', () => {
    it('should initialize with basic configuration', () => {
      expect(client).toBeInstanceOf(OpenAtlasSearchClient);
    });

    it('should remove trailing slash from baseUrl', () => {
      const clientWithSlash = new OpenAtlasSearchClient({
        ...baseConfig,
        baseUrl: 'http://localhost:8080/',
      });
      expect(clientWithSlash).toBeInstanceOf(OpenAtlasSearchClient);
    });

    it('should set up basic authentication when credentials provided', () => {
      const clientWithAuth = new OpenAtlasSearchClient({
        ...baseConfig,
        username: 'testuser',
        password: 'testpass',
      });
      expect(clientWithAuth).toBeInstanceOf(OpenAtlasSearchClient);
    });
  });

  describe('health', () => {
    it('should make GET request to /health endpoint', async () => {
      const expectedResponse = { status: 'healthy', service: 'open-atlas-search' };
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => expectedResponse,
      });

      const result = await client.health();
      
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/health',
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
        })
      );
      expect(result).toEqual(expectedResponse);
    });
  });

  describe('search', () => {
    it('should validate index name', async () => {
      await expect(client.search('', { query: { match_all: {} } }))
        .rejects.toThrow('Index name is required');
    });

    it('should validate search query', async () => {
      await expect(client.search('test-index', { query: null as any }))
        .rejects.toThrow('Search query is required');
    });

    it('should validate size parameter', async () => {
      await expect(client.search('test-index', { 
        query: { match_all: {} }, 
        size: -1 
      })).rejects.toThrow('Size parameter cannot be negative');

      await expect(client.search('test-index', { 
        query: { match_all: {} }, 
        size: 1001 
      })).rejects.toThrow('Size parameter cannot exceed 1000');
    });

    it('should validate from parameter', async () => {
      await expect(client.search('test-index', { 
        query: { match_all: {} }, 
        from: -1 
      })).rejects.toThrow('From parameter cannot be negative');
    });

    it('should make POST request to search endpoint', async () => {
      const expectedResponse = {
        hits: [],
        total: 0,
        maxScore: 0,
      };
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => expectedResponse,
      });

      const searchRequest = { query: { match_all: {} }, size: 10 };
      const result = await client.search('test-index', searchRequest);
      
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/indexes/test-index/search',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(searchRequest),
        })
      );
      expect(result).toEqual(expectedResponse);
    });
  });

  describe('simpleSearch', () => {
    it('should create proper match query', async () => {
      const expectedResponse = { hits: [], total: 0, maxScore: 0 };
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => expectedResponse,
      });

      await client.simpleSearch('test-index', 'search text');
      
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/indexes/test-index/search',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            query: {
              match: {
                _all: 'search text',
              },
            },
          }),
        })
      );
    });
  });

  describe('error handling', () => {
    it('should throw OpenAtlasSearchError for API errors', async () => {
      const errorResponse = {
        error: 'index_not_found',
        message: 'Index not found',
        code: 404,
      };

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        headers: { get: () => 'application/json' },
        json: async () => errorResponse,
      });

      await expect(client.health()).rejects.toThrow(OpenAtlasSearchError);
    });

    it('should handle timeout errors', async () => {
      const shortTimeoutClient = new OpenAtlasSearchClient({
        ...baseConfig,
        timeout: 1, // 1ms timeout
      });

      mockFetch.mockImplementationOnce(() => 
        new Promise(resolve => setTimeout(resolve, 100))
      );

      await expect(shortTimeoutClient.health())
        .rejects.toThrow(OpenAtlasSearchError);
    });
  });
});
