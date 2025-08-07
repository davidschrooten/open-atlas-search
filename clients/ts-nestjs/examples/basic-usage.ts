/**
 * Basic NestJS Application Example
 * 
 * This example demonstrates how to set up and use the Open Atlas Search NestJS client
 * in a basic NestJS application.
 */

import { Module, Injectable, Controller, Get, Query, Post, Body } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { 
  OpenAtlasSearchModule, 
  OpenAtlasSearchService,
  SearchRequest,
  SearchResult,
} from 'oas-ts-nestjs-client';

// Configuration
const config = {
  openAtlasSearch: {
    baseUrl: process.env.ATLAS_SEARCH_URL || 'http://localhost:8080',
    username: process.env.ATLAS_SEARCH_USERNAME,
    password: process.env.ATLAS_SEARCH_PASSWORD,
    timeout: 30000,
  },
};

// Search Service
@Injectable()
export class SearchService {
  constructor(private readonly atlasSearch: OpenAtlasSearchService) {}

  async searchDocuments(indexName: string, query: string, options: {
    size?: number;
    from?: number;
  } = {}) {
    const results = await this.atlasSearch.simpleSearch(indexName, query, options);
    
    return {
      documents: results.hits.map(hit => ({
        id: hit._id,
        score: hit.score,
        ...hit.source,
      })),
      total: results.total,
      maxScore: results.maxScore,
    };
  }

  async advancedSearch(indexName: string, searchRequest: SearchRequest) {
    const results = await this.atlasSearch.search(indexName, searchRequest);
    
    return {
      documents: results.hits.map(hit => ({
        id: hit._id,
        score: hit.score,
        data: hit.source,
        highlight: hit.highlight,
      })),
      total: results.total,
      maxScore: results.maxScore,
      facets: results.facets,
    };
  }

  async getAvailableIndexes() {
    const response = await this.atlasSearch.listIndexes();
    return response.indexes.map(index => ({
      name: index.name,
      documentCount: index.docCount,
      status: index.status,
      lastSync: index.lastSync,
    }));
  }

  async checkHealth() {
    try {
      const health = await this.atlasSearch.health();
      const ready = await this.atlasSearch.ready();
      return {
        healthy: health.status === 'healthy',
        ready: ready.status === 'ready',
        service: health.service,
        checks: ready.checks,
      };
    } catch (error) {
      return {
        healthy: false,
        ready: false,
        error: error.message,
      };
    }
  }
}

// REST API Controller
@Controller('search')
export class SearchController {
  constructor(private readonly searchService: SearchService) {}

  @Get('health')
  async getHealth() {
    return this.searchService.checkHealth();
  }

  @Get('indexes')
  async getIndexes() {
    return this.searchService.getAvailableIndexes();
  }

  @Get(':index')
  async search(
    @Query('q') query: string,
    @Query('size') size = '10',
    @Query('from') from = '0',
    @Query('index') indexName: string,
  ) {
    if (!query) {
      throw new Error('Query parameter "q" is required');
    }

    return this.searchService.searchDocuments(indexName, query, {
      size: parseInt(size, 10),
      from: parseInt(from, 10),
    });
  }

  @Post(':index/search')
  async advancedSearch(
    @Query('index') indexName: string,
    @Body() searchRequest: SearchRequest,
  ) {
    return this.searchService.advancedSearch(indexName, searchRequest);
  }
}

// Main Application Module
@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      load: [() => config],
    }),
    OpenAtlasSearchModule.forRootAsync({
      imports: [ConfigModule],
      useFactory: (configService: ConfigService) => {
        return {
          ...configService.get('openAtlasSearch'),
          isGlobal: true,
        };
      },
      inject: [ConfigService],
    }),
  ],
  controllers: [SearchController],
  providers: [SearchService],
})
export class AppModule {}

// Bootstrap function
async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  
  // Enable CORS for development
  app.enableCors();
  
  // Global prefix
  app.setGlobalPrefix('api');
  
  await app.listen(3000);
  console.log('Application is running on: http://localhost:3000');
  
  // Test the connection on startup
  try {
    const searchService = app.get(SearchService);
    const health = await searchService.checkHealth();
    console.log('Open Atlas Search connection:', health);
  } catch (error) {
    console.error('Failed to connect to Open Atlas Search:', error.message);
  }
}

// Run the application
if (require.main === module) {
  bootstrap();
}

export { AppModule, SearchService, SearchController };
