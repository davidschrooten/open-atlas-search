// Simple test to verify TypeScript compilation
import { OpenAtlasSearchClient, OpenAtlasSearchError } from './src/index';

// This should compile without errors
const client = new OpenAtlasSearchClient({
  baseUrl: 'http://localhost:8080',
  timeout: 5000,
});

// Type checking - these should all be properly typed
async function testTypes() {
  try {
    const health = await client.health();
    const indexes = await client.listIndexes();
    const status = await client.getIndexStatus('test');
    const mapping = await client.getIndexMapping('test');
    
    const searchResult = await client.search('test', {
      query: { match_all: {} },
      size: 10
    });
    
    const simpleResult = await client.simpleSearch('test', 'query');
    const allResult = await client.searchAll('test');
    
    console.log('All types compile correctly!');
  } catch (error) {
    if (error instanceof OpenAtlasSearchError) {
      console.log('Caught OpenAtlasSearchError:', error.message);
    }
  }
}

export { testTypes };
