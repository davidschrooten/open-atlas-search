/**
 * Open Atlas Search NestJS Client
 * 
 * A NestJS module and service wrapper for the Open Atlas Search TypeScript client,
 * providing dependency injection, logging, and lifecycle management.
 */

// Core module and service
export { OpenAtlasSearchModule } from './open-atlas-search.module';
export { OpenAtlasSearchService } from './open-atlas-search.service';

// Configuration interfaces
export {
  OpenAtlasSearchModuleOptions,
  OpenAtlasSearchModuleAsyncOptions,
  OpenAtlasSearchOptionsFactory,
} from './interfaces';

// Constants for dependency injection
export {
  OPEN_ATLAS_SEARCH_MODULE_OPTIONS,
  OPEN_ATLAS_SEARCH_CLIENT,
} from './constants';

// Re-export everything from the base client for convenience
export * from 'oas-ts-client';

// Default export for convenience
export { OpenAtlasSearchModule as default } from './open-atlas-search.module';
