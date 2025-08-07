import { DynamicModule, Module, Provider } from '@nestjs/common';
import { OpenAtlasSearchClient } from 'oas-ts-client';
import { OpenAtlasSearchService } from './open-atlas-search.service';
import {
  OpenAtlasSearchModuleOptions,
  OpenAtlasSearchModuleAsyncOptions,
  OpenAtlasSearchOptionsFactory,
} from './interfaces';
import {
  OPEN_ATLAS_SEARCH_MODULE_OPTIONS,
  OPEN_ATLAS_SEARCH_CLIENT,
} from './constants';

@Module({})
export class OpenAtlasSearchModule {
  /**
   * Register the module synchronously with provided configuration
   * @param options - Configuration options
   * @returns DynamicModule
   */
  static forRoot(options: OpenAtlasSearchModuleOptions): DynamicModule {
    const providers: Provider[] = [
      {
        provide: OPEN_ATLAS_SEARCH_MODULE_OPTIONS,
        useValue: options,
      },
      {
        provide: OPEN_ATLAS_SEARCH_CLIENT,
        useFactory: (config: OpenAtlasSearchModuleOptions) => {
          return new OpenAtlasSearchClient(config);
        },
        inject: [OPEN_ATLAS_SEARCH_MODULE_OPTIONS],
      },
      OpenAtlasSearchService,
    ];

    return {
      module: OpenAtlasSearchModule,
      global: options.isGlobal || false,
      providers,
      exports: [OpenAtlasSearchService, OPEN_ATLAS_SEARCH_CLIENT],
    };
  }

  /**
   * Register the module asynchronously with dynamic configuration
   * @param options - Async configuration options
   * @returns DynamicModule
   */
  static forRootAsync(options: OpenAtlasSearchModuleAsyncOptions): DynamicModule {
    const providers: Provider[] = [
      ...this.createAsyncProviders(options),
      {
        provide: OPEN_ATLAS_SEARCH_CLIENT,
        useFactory: (config: OpenAtlasSearchModuleOptions) => {
          return new OpenAtlasSearchClient(config);
        },
        inject: [OPEN_ATLAS_SEARCH_MODULE_OPTIONS],
      },
      OpenAtlasSearchService,
    ];

    return {
      module: OpenAtlasSearchModule,
      global: options.isGlobal || false,
      imports: options.imports || [],
      providers,
      exports: [OpenAtlasSearchService, OPEN_ATLAS_SEARCH_CLIENT],
    };
  }

  /**
   * Create providers for async configuration
   * @param options - Async configuration options
   * @returns Provider[]
   */
  private static createAsyncProviders(
    options: OpenAtlasSearchModuleAsyncOptions,
  ): Provider[] {
    if (options.useFactory) {
      return [
        {
          provide: OPEN_ATLAS_SEARCH_MODULE_OPTIONS,
          useFactory: options.useFactory,
          inject: options.inject || [],
        },
      ];
    }

    if (options.useClass) {
      return [
        {
          provide: OPEN_ATLAS_SEARCH_MODULE_OPTIONS,
          useFactory: async (optionsFactory: OpenAtlasSearchOptionsFactory) =>
            optionsFactory.createOpenAtlasSearchOptions(),
          inject: [options.useClass],
        },
        {
          provide: options.useClass,
          useClass: options.useClass,
        },
      ];
    }

    if (options.useExisting) {
      return [
        {
          provide: OPEN_ATLAS_SEARCH_MODULE_OPTIONS,
          useFactory: async (optionsFactory: OpenAtlasSearchOptionsFactory) =>
            optionsFactory.createOpenAtlasSearchOptions(),
          inject: [options.useExisting as any],
        },
      ];
    }

    throw new Error(
      'Invalid OpenAtlasSearchModuleAsyncOptions: must provide useFactory, useClass, or useExisting',
    );
  }

  /**
   * For importing in feature modules when the root module is already configured
   * This provides access to the service without reconfiguring the client
   * @returns DynamicModule
   */
  static forFeature(): DynamicModule {
    return {
      module: OpenAtlasSearchModule,
      providers: [OpenAtlasSearchService],
      exports: [OpenAtlasSearchService],
    };
  }
}
