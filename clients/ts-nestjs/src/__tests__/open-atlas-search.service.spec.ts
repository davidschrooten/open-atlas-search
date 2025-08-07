import { Test, TestingModule } from '@nestjs/testing';
import { Logger } from '@nestjs/common';
import { OpenAtlasSearchService } from '../open-atlas-search.service';
import { OPEN_ATLAS_SEARCH_CLIENT } from '../constants';
import { 
  OpenAtlasSearchClient,
  HealthResponse,
  ReadyResponse,
  ListIndexesResponse,
  SearchResult,
} from 'oas-ts-client';

describe('OpenAtlasSearchService', () => {
  let service: OpenAtlasSearchService;
  let mockClient: jest.Mocked<OpenAtlasSearchClient>;

  beforeEach(async () => {
    // Create a mock client
    mockClient = {
      health: jest.fn(),
      ready: jest.fn(),
      listIndexes: jest.fn(),
      getIndexStatus: jest.fn(),
      getIndexMapping: jest.fn(),
      search: jest.fn(),
      simpleSearch: jest.fn(),
      searchAll: jest.fn(),
    } as any;

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        OpenAtlasSearchService,
        {
          provide: OPEN_ATLAS_SEARCH_CLIENT,
          useValue: mockClient,
        },
      ],
    }).compile();

    service = module.get<OpenAtlasSearchService>(OpenAtlasSearchService);
    
    // Mock the logger to avoid console output during tests
    jest.spyOn(Logger.prototype, 'log').mockImplementation();
    jest.spyOn(Logger.prototype, 'debug').mockImplementation();
    jest.spyOn(Logger.prototype, 'warn').mockImplementation();
    jest.spyOn(Logger.prototype, 'error').mockImplementation();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('onModuleInit', () => {
    it('should test connection on initialization', async () => {
      const healthResponse: HealthResponse = {
        status: 'healthy',
        service: 'open-atlas-search',
      };
      mockClient.health.mockResolvedValue(healthResponse);

      await service.onModuleInit();

      expect(mockClient.health).toHaveBeenCalled();
      expect(Logger.prototype.log).toHaveBeenCalledWith('Initializing Open Atlas Search Service');
      expect(Logger.prototype.log).toHaveBeenCalledWith('Connected to Open Atlas Search API: open-atlas-search');
    });

    it('should handle connection failure during initialization', async () => {
      const error = new Error('Connection failed');
      mockClient.health.mockRejectedValue(error);

      await service.onModuleInit();

      expect(mockClient.health).toHaveBeenCalled();
      expect(Logger.prototype.warn).toHaveBeenCalledWith(
        'Failed to connect to Open Atlas Search API during initialization',
        error
      );
    });
  });

  describe('getClient', () => {
    it('should return the underlying client', () => {
      const client = service.getClient();
      expect(client).toBe(mockClient);
    });
  });

  describe('health', () => {
    it('should return health response', async () => {
      const healthResponse: HealthResponse = {
        status: 'healthy',
        service: 'open-atlas-search',
      };
      mockClient.health.mockResolvedValue(healthResponse);

      const result = await service.health();

      expect(result).toEqual(healthResponse);
      expect(mockClient.health).toHaveBeenCalled();
      expect(Logger.prototype.debug).toHaveBeenCalledWith('Health check successful');
    });

    it('should handle health check error', async () => {
      const error = new Error('Health check failed');
      mockClient.health.mockRejectedValue(error);

      await expect(service.health()).rejects.toThrow(error);
      expect(Logger.prototype.error).toHaveBeenCalledWith('Health check failed', error);
    });
  });

  describe('ready', () => {
    it('should return ready response', async () => {
      const readyResponse: ReadyResponse = {
        status: 'ready',
        service: 'open-atlas-search',
        checks: { elasticsearch: 'ok' },
      };
      mockClient.ready.mockResolvedValue(readyResponse);

      const result = await service.ready();

      expect(result).toEqual(readyResponse);
      expect(mockClient.ready).toHaveBeenCalled();
      expect(Logger.prototype.debug).toHaveBeenCalledWith('Readiness check successful');
    });
  });

  describe('listIndexes', () => {
    it('should return list of indexes', async () => {
      const listResponse: ListIndexesResponse = {
        indexes: [
          {
            name: 'test-index',
            docCount: 100,
            status: 'active',
          },
        ],
        total: 1,
      };
      mockClient.listIndexes.mockResolvedValue(listResponse);

      const result = await service.listIndexes();

      expect(result).toEqual(listResponse);
      expect(mockClient.listIndexes).toHaveBeenCalled();
      expect(Logger.prototype.debug).toHaveBeenCalledWith('Listed 1 indexes');
    });
  });

  describe('search', () => {
    it('should perform search and return results', async () => {
      const searchRequest = {
        query: { match: { title: 'test' } },
        size: 10,
      };
      const searchResult: SearchResult = {
        hits: [
          {
            _id: '1',
            score: 1.0,
            source: { title: 'Test Document' },
          },
        ],
        total: 1,
        maxScore: 1.0,
      };
      mockClient.search.mockResolvedValue(searchResult);

      const result = await service.search('test-index', searchRequest);

      expect(result).toEqual(searchResult);
      expect(mockClient.search).toHaveBeenCalledWith('test-index', searchRequest);
      expect(Logger.prototype.debug).toHaveBeenCalledWith(
        'Search completed for index: test-index, found 1 results'
      );
    });

    it('should handle search error', async () => {
      const error = new Error('Search failed');
      mockClient.search.mockRejectedValue(error);

      await expect(service.search('test-index', { query: {} })).rejects.toThrow(error);
      expect(Logger.prototype.error).toHaveBeenCalledWith(
        'Search failed for index: test-index',
        error
      );
    });
  });

  describe('simpleSearch', () => {
    it('should perform simple search', async () => {
      const searchResult: SearchResult = {
        hits: [],
        total: 0,
        maxScore: 0,
      };
      mockClient.simpleSearch.mockResolvedValue(searchResult);

      const result = await service.simpleSearch('test-index', 'test query');

      expect(result).toEqual(searchResult);
      expect(mockClient.simpleSearch).toHaveBeenCalledWith('test-index', 'test query', {});
      expect(Logger.prototype.debug).toHaveBeenCalledWith(
        'Simple search completed for index: test-index, query: "test query", found 0 results'
      );
    });
  });
});
