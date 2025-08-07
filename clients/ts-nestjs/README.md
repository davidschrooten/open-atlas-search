# Open Atlas Search NestJS Client

A NestJS module and service wrapper for the [Open Atlas Search TypeScript client](../ts/README.md), providing dependency injection, logging, and lifecycle management for NestJS applications.

## Features

- 🏗️ **NestJS Integration** - Seamless integration with NestJS dependency injection
- 🔧 **Dynamic Configuration** - Support for both synchronous and asynchronous configuration
- 📝 **Built-in Logging** - Integrated logging using NestJS Logger
- 🔄 **Lifecycle Management** - Automatic connection testing and cleanup
- 🛡️ **Type Safety** - Full TypeScript support with comprehensive type definitions
- 🧪 **Testing Support** - Easy mocking and testing utilities

## Installation

```bash
npm install oas-ts-nestjs-client
# or
yarn add oas-ts-nestjs-client
```

## Quick Start

### Basic Usage

```typescript
import { Module } from '@nestjs/common';
import { OpenAtlasSearchModule } from 'oas-ts-nestjs-client';

@Module({
  imports: [
    OpenAtlasSearchModule.forRoot({
      baseUrl: 'http://localhost:8080',
      username: 'your-username',
      password: 'your-password',
      timeout: 30000,
    }),
  ],
})
export class AppModule {}
```

### Using the Service

```typescript
import { Injectable } from '@nestjs/common';
import { OpenAtlasSearchService } from 'oas-ts-nestjs-client';

@Injectable()
export class SearchService {
  constructor(
    private readonly atlasSearch: OpenAtlasSearchService,
  ) {}

  async searchDocuments(indexName: string, query: string) {
    try {
      const results = await this.atlasSearch.simpleSearch(indexName, query, {
        size: 20,
        from: 0,
      });
      
      return results.hits.map(hit => ({
        id: hit._id,
        score: hit.score,
        ...hit.source,
      }));
    } catch (error) {
      console.error('Search failed:', error);
      throw error;
    }
  }

  async getIndexes() {
    const response = await this.atlasSearch.listIndexes();
    return response.indexes;
  }
}
```

## Configuration

### Synchronous Configuration

```typescript
import { OpenAtlasSearchModule } from 'oas-ts-nestjs-client';

@Module({
  imports: [
    OpenAtlasSearchModule.forRoot({
      baseUrl: 'http://localhost:8080',
      username: 'admin',
      password: 'password',
      timeout: 30000,
      isGlobal: true, // Makes the module globally available
      headers: {
        'X-Custom-Header': 'value',
      },
    }),
  ],
})
export class AppModule {}
```

### Asynchronous Configuration with ConfigService

```typescript
import { ConfigModule, ConfigService } from '@nestjs/config';
import { OpenAtlasSearchModule } from 'oas-ts-nestjs-client';

@Module({
  imports: [
    ConfigModule.forRoot(),
    OpenAtlasSearchModule.forRootAsync({
      imports: [ConfigModule],
      useFactory: (configService: ConfigService) => ({
        baseUrl: configService.get<string>('ATLAS_SEARCH_URL'),
        username: configService.get<string>('ATLAS_SEARCH_USERNAME'),
        password: configService.get<string>('ATLAS_SEARCH_PASSWORD'),
        timeout: configService.get<number>('ATLAS_SEARCH_TIMEOUT') || 30000,
        isGlobal: true,
      }),
      inject: [ConfigService],
    }),
  ],
})
export class AppModule {}
```

### Using a Configuration Class

```typescript
import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { 
  OpenAtlasSearchOptionsFactory,
  OpenAtlasSearchModuleOptions,
} from 'oas-ts-nestjs-client';

@Injectable()
export class OpenAtlasSearchConfigService implements OpenAtlasSearchOptionsFactory {
  constructor(private configService: ConfigService) {}

  createOpenAtlasSearchOptions(): OpenAtlasSearchModuleOptions {
    return {
      baseUrl: this.configService.get<string>('ATLAS_SEARCH_URL'),
      username: this.configService.get<string>('ATLAS_SEARCH_USERNAME'),
      password: this.configService.get<string>('ATLAS_SEARCH_PASSWORD'),
      timeout: this.configService.get<number>('ATLAS_SEARCH_TIMEOUT') || 30000,
    };
  }
}

@Module({
  imports: [
    OpenAtlasSearchModule.forRootAsync({
      useClass: OpenAtlasSearchConfigService,
    }),
  ],
  providers: [OpenAtlasSearchConfigService],
})
export class AppModule {}
```

## API Reference

### OpenAtlasSearchService

The main service provides all the functionality of the underlying TypeScript client with added NestJS features:

#### Health and Status Methods

```typescript
// Health check
const health = await service.health();

// Readiness check
const ready = await service.ready();

// List all indexes
const indexes = await service.listIndexes();

// Get index status
const status = await service.getIndexStatus('my-index');

// Get index mapping
const mapping = await service.getIndexMapping('my-index');
```

#### Search Methods

```typescript
// Advanced search with full query DSL
const results = await service.search('my-index', {
  query: {
    bool: {
      must: [
        { match: { title: 'search terms' } },
        { range: { date: { gte: '2023-01-01' } } },
      ],
    },
  },
  size: 10,
  from: 0,
  facets: {
    categories: {
      type: 'terms',
      field: 'category',
      size: 10,
    },
  },
});

// Simple text search
const results = await service.simpleSearch('my-index', 'search query', {
  size: 20,
  from: 0,
});

// Get all documents
const allResults = await service.searchAll('my-index', {
  size: 100,
});
```

#### Utility Methods

```typescript
// Get the underlying client for advanced usage
const client = service.getClient();

// Direct client access
const customResult = await client.search('my-index', customQuery);
```

## Testing

### Mocking the Service

```typescript
import { Test, TestingModule } from '@nestjs/testing';
import { OpenAtlasSearchService } from 'oas-ts-nestjs-client';

describe('MyService', () => {
  let service: MyService;
  let mockAtlasSearch: jest.Mocked<OpenAtlasSearchService>;

  beforeEach(async () => {
    mockAtlasSearch = {
      search: jest.fn(),
      simpleSearch: jest.fn(),
      listIndexes: jest.fn(),
    } as any;

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        MyService,
        {
          provide: OpenAtlasSearchService,
          useValue: mockAtlasSearch,
        },
      ],
    }).compile();

    service = module.get<MyService>(MyService);
  });

  it('should search documents', async () => {
    mockAtlasSearch.simpleSearch.mockResolvedValue({
      hits: [{ _id: '1', score: 1.0, source: { title: 'Test' } }],
      total: 1,
      maxScore: 1.0,
    });

    const result = await service.searchDocuments('test-index', 'query');
    expect(result).toHaveLength(1);
  });
});
```

### Testing with Real Module

```typescript
import { Test, TestingModule } from '@nestjs/testing';
import { OpenAtlasSearchModule, OpenAtlasSearchService } from 'oas-ts-nestjs-client';

describe('Integration Test', () => {
  let module: TestingModule;
  let service: OpenAtlasSearchService;

  beforeEach(async () => {
    module = await Test.createTestingModule({
      imports: [
        OpenAtlasSearchModule.forRoot({
          baseUrl: 'http://localhost:8080',
          username: 'test',
          password: 'test',
        }),
      ],
    }).compile();

    service = module.get<OpenAtlasSearchService>(OpenAtlasSearchService);
  });

  afterEach(async () => {
    await module.close();
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
```

## Error Handling

The service automatically logs errors and re-throws them. All errors from the underlying client are preserved:

```typescript
import { OpenAtlasSearchError } from 'oas-ts-nestjs-client';

try {
  await service.search('non-existent-index', { query: {} });
} catch (error) {
  if (error instanceof OpenAtlasSearchError) {
    console.log(`API Error: ${error.message}`);
    console.log(`Status Code: ${error.statusCode}`);
    console.log(`Error Response:`, error.response);
  }
}
```

## Environment Variables

For production deployments, consider using environment variables:

```bash
# .env file
ATLAS_SEARCH_URL=https://your-atlas-search-api.com
ATLAS_SEARCH_USERNAME=your-username
ATLAS_SEARCH_PASSWORD=your-password
ATLAS_SEARCH_TIMEOUT=30000
```

## Global Module

To make the module available globally without importing it in every module:

```typescript
@Module({
  imports: [
    OpenAtlasSearchModule.forRoot({
      // ... configuration
      isGlobal: true,
    }),
  ],
})
export class AppModule {}
```

## Feature Modules

If you prefer not to use the global module pattern, you can use `forFeature()` in feature modules after configuring the module at the root level:

```typescript
// app.module.ts
@Module({
  imports: [
    OpenAtlasSearchModule.forRoot({
      baseUrl: 'http://localhost:8080',
      username: 'admin',
      password: 'password',
      timeout: 30000,
      // Note: isGlobal is false or omitted
    }),
  ],
})
export class AppModule {}

// feature.module.ts
@Module({
  imports: [OpenAtlasSearchModule.forFeature()],
  providers: [FeatureService],
})
export class FeatureModule {}

// feature.service.ts
@Injectable()
export class FeatureService {
  constructor(
    private readonly atlasSearch: OpenAtlasSearchService,
  ) {}
  
  async search(query: string) {
    return this.atlasSearch.simpleSearch('my-index', query);
  }
}
```

**Important**: `forFeature()` only works when the root module has already been configured with `forRoot()` or `forRootAsync()` in the same module tree. The client instance will be shared across all feature modules.

## License

MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please read the contributing guidelines and submit pull requests to the main repository.

## Related

- [Open Atlas Search TypeScript Client](../ts/README.md) - The underlying TypeScript client
- [Open Atlas Search API Documentation](../../README.md) - Main project documentation
