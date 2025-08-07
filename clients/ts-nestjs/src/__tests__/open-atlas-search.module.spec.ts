import { Test, TestingModule } from '@nestjs/testing';
import { OpenAtlasSearchModule } from '../open-atlas-search.module';
import { OpenAtlasSearchService } from '../open-atlas-search.service';
import { 
  OPEN_ATLAS_SEARCH_MODULE_OPTIONS,
  OPEN_ATLAS_SEARCH_CLIENT 
} from '../constants';
import { OpenAtlasSearchModuleOptions } from '../interfaces';

describe('OpenAtlasSearchModule', () => {
  const mockConfig: OpenAtlasSearchModuleOptions = {
    baseUrl: 'http://localhost:8080',
    username: 'test',
    password: 'test',
  };

  describe('forRoot', () => {
    let module: TestingModule;

    beforeEach(async () => {
      module = await Test.createTestingModule({
        imports: [OpenAtlasSearchModule.forRoot(mockConfig)],
      }).compile();
    });

    afterEach(async () => {
      await module.close();
    });

    it('should provide OpenAtlasSearchService', () => {
      const service = module.get<OpenAtlasSearchService>(OpenAtlasSearchService);
      expect(service).toBeDefined();
      expect(service).toBeInstanceOf(OpenAtlasSearchService);
    });

    it('should provide the client instance', () => {
      const client = module.get(OPEN_ATLAS_SEARCH_CLIENT);
      expect(client).toBeDefined();
    });

    it('should provide the module options', () => {
      const options = module.get(OPEN_ATLAS_SEARCH_MODULE_OPTIONS);
      expect(options).toEqual(mockConfig);
    });
  });

  describe('forRootAsync with useFactory', () => {
    let module: TestingModule;

    beforeEach(async () => {
      module = await Test.createTestingModule({
        imports: [
          OpenAtlasSearchModule.forRootAsync({
            useFactory: () => mockConfig,
          }),
        ],
      }).compile();
    });

    afterEach(async () => {
      await module.close();
    });

    it('should provide OpenAtlasSearchService', () => {
      const service = module.get<OpenAtlasSearchService>(OpenAtlasSearchService);
      expect(service).toBeDefined();
      expect(service).toBeInstanceOf(OpenAtlasSearchService);
    });

    it('should provide the client instance', () => {
      const client = module.get(OPEN_ATLAS_SEARCH_CLIENT);
      expect(client).toBeDefined();
    });
  });


  describe('forRootAsync error handling', () => {
    it('should throw error when no configuration method is provided', () => {
      expect(() => {
        OpenAtlasSearchModule.forRootAsync({});
      }).toThrow('Invalid OpenAtlasSearchModuleAsyncOptions: must provide useFactory, useClass, or useExisting');
    });
  });

  describe('global module', () => {
    let module: TestingModule;

    beforeEach(async () => {
      module = await Test.createTestingModule({
        imports: [
          OpenAtlasSearchModule.forRoot({
            ...mockConfig,
            isGlobal: true,
          }),
        ],
      }).compile();
    });

    afterEach(async () => {
      await module.close();
    });

    it('should create a global module when isGlobal is true', () => {
      const service = module.get<OpenAtlasSearchService>(OpenAtlasSearchService);
      expect(service).toBeDefined();
    });
  });
});
