import { OpenAtlasSearchClient, SearchRequest } from '../src';

async function basicUsageExample() {
  // Initialize the client
  const client = new OpenAtlasSearchClient({
    baseUrl: 'http://localhost:8080',
    // Optional: Add authentication if your API requires it
    // username: 'your-username',
    // password: 'your-password',
    timeout: 10000, // 10 seconds
  });

  try {
    // Check API health
    console.log('Checking API health...');
    const health = await client.health();
    console.log('Health status:', health);

    // Check API readiness
    console.log('Checking API readiness...');
    const ready = await client.ready();
    console.log('Ready status:', ready);

    // List all indexes
    console.log('Listing indexes...');
    const indexes = await client.listIndexes();
    console.log('Available indexes:', indexes);

    if (indexes.indexes.length === 0) {
      console.log('No indexes available. Make sure your API has indexes configured.');
      return;
    }

    const indexName = indexes.indexes[0].name;
    console.log(`Using index: ${indexName}`);

    // Get index status
    console.log('Getting index status...');
    const status = await client.getIndexStatus(indexName);
    console.log('Index status:', status);

    // Get index mapping
    console.log('Getting index mapping...');
    const mapping = await client.getIndexMapping(indexName);
    console.log('Index mapping:', mapping);

    // Simple text search
    console.log('Performing simple search...');
    const simpleResults = await client.simpleSearch(indexName, 'search query', {
      size: 5,
    });
    console.log('Simple search results:', simpleResults);

    // Advanced search with custom query
    console.log('Performing advanced search...');
    const searchRequest: SearchRequest = {
      query: {
        bool: {
          must: [
            {
              match: {
                title: 'example'
              }
            }
          ]
        }
      },
      facets: {
        categoryFacet: {
          type: 'terms',
          field: 'category',
          size: 10
        }
      },
      size: 10,
      from: 0,
    };

    const advancedResults = await client.search(indexName, searchRequest);
    console.log('Advanced search results:', advancedResults);

    // Get all documents (match_all)
    console.log('Getting all documents...');
    const allDocs = await client.searchAll(indexName, {
      size: 5,
    });
    console.log('All documents:', allDocs);

  } catch (error) {
    console.error('Error occurred:', error);
    
    // Handle specific API errors
    if (error.name === 'OpenAtlasSearchError') {
      console.error('API Error Response:', error.response);
      console.error('Status Code:', error.statusCode);
    }
  }
}

// Run the example
basicUsageExample().catch(console.error);
