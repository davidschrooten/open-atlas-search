#!/usr/bin/env node

/**
 * Build verification script for the Open Atlas Search NestJS Client
 * 
 * This script verifies that the client can be imported and instantiated correctly.
 */

import { OpenAtlasSearchModule, OpenAtlasSearchService } from './src/index';
import { Test } from '@nestjs/testing';

async function testBuild() {
  console.log('🧪 Testing Open Atlas Search NestJS Client build...');
  
  try {
    // Test that we can import the module
    console.log('✓ Module import successful');
    
    // Test that we can create a testing module
    const module = await Test.createTestingModule({
      imports: [
        OpenAtlasSearchModule.forRoot({
          baseUrl: 'http://localhost:8080',
          username: 'test',
          password: 'test',
        }),
      ],
    }).compile();
    
    console.log('✓ Module compilation successful');
    
    // Test that we can get the service
    const service = module.get<OpenAtlasSearchService>(OpenAtlasSearchService);
    console.log('✓ Service instantiation successful');
    
    // Test that service has expected methods
    const expectedMethods = [
      'health',
      'ready', 
      'listIndexes',
      'getIndexStatus',
      'getIndexMapping',
      'search',
      'simpleSearch',
      'searchAll',
      'getClient',
    ];
    
    for (const method of expectedMethods) {
      if (typeof service[method] !== 'function') {
        throw new Error(`Service missing method: ${method}`);
      }
    }
    
    console.log('✓ All service methods present');
    
    // Clean up
    await module.close();
    
    console.log('🎉 Build verification successful!');
    
  } catch (error) {
    console.error('❌ Build verification failed:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  testBuild();
}
