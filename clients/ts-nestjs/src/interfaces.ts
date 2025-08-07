import { ClientConfig } from 'oas-ts-client';

/**
 * Configuration interface for the NestJS Open Atlas Search module
 * Extends the base client configuration with NestJS-specific options
 */
export interface OpenAtlasSearchModuleOptions extends ClientConfig {
  /**
   * Whether this configuration should be available globally
   * @default false
   */
  isGlobal?: boolean;
}

/**
 * Factory function type for creating OpenAtlasSearchModuleOptions
 */
export interface OpenAtlasSearchOptionsFactory {
  createOpenAtlasSearchOptions(): Promise<OpenAtlasSearchModuleOptions> | OpenAtlasSearchModuleOptions;
}

/**
 * Async configuration options for the NestJS module
 */
export interface OpenAtlasSearchModuleAsyncOptions {
  /**
   * Whether this configuration should be available globally
   * @default false
   */
  isGlobal?: boolean;

  /**
   * Dependencies that should be injected into the factory function
   */
  imports?: any[];

  /**
   * Factory function that returns the module options
   */
  useFactory?: (...args: any[]) => Promise<OpenAtlasSearchModuleOptions> | OpenAtlasSearchModuleOptions;

  /**
   * Dependencies to inject into the factory function
   */
  inject?: any[];

  /**
   * Class that implements OpenAtlasSearchOptionsFactory
   */
  useClass?: new (...args: any[]) => OpenAtlasSearchOptionsFactory;

  /**
   * Existing instance that implements OpenAtlasSearchOptionsFactory
   */
  useExisting?: OpenAtlasSearchOptionsFactory;
}
