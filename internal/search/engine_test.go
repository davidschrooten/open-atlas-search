package search

import (
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"

	"github.com/david/open-atlas-search/config"
)

func TestNewEngine(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.SearchConfig{
		IndexPath: tempDir,
	}

	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}
	if engine.indexPath != tempDir {
		t.Errorf("Expected indexPath %s, got %s", tempDir, engine.indexPath)
	}
	if engine.indexes == nil {
		t.Error("Expected indexes map to be initialized")
	}
	if engine.lastSync == nil {
		t.Error("Expected lastSync map to be initialized")
	}
}

func TestEngine_UpdateLastSync(t *testing.T) {
	engine := &Engine{
		lastSync: make(map[string]time.Time),
	}

	testTime := time.Now().Truncate(time.Second)
	indexName := "test.index"

	engine.UpdateLastSync(indexName, testTime)

	// Get the sync time
	engine.syncMutex.RLock()
	syncTime, exists := engine.lastSync[indexName]
	engine.syncMutex.RUnlock()

	if !exists {
		t.Error("Expected sync time to be set")
	}
	if !syncTime.Equal(testTime) {
		t.Errorf("Expected sync time %v, got %v", testTime, syncTime)
	}
}

func TestEngine_ListIndexes(t *testing.T) {
	engine := &Engine{
		indexes:  make(map[string]bleve.Index),
		lastSync: make(map[string]time.Time),
	}

	// Test empty indexes
	indexes, err := engine.ListIndexes()
	if err != nil {
		t.Fatalf("Failed to list indexes: %v", err)
	}
	if len(indexes) != 0 {
		t.Errorf("Expected 0 indexes, got %d", len(indexes))
	}

	// Note: Testing with actual Bleve indexes would require more complex setup
	// This test focuses on the basic structure and empty case
}

func TestEngine_ConvertSearchResult(t *testing.T) {
	engine := &Engine{}

	// Create a mock Bleve search result
	mockResult := &bleve.SearchResult{
		Total:    5,
		MaxScore: 1.5,
		Hits: []*search.DocumentMatch{
			{
				ID:    "doc1",
				Score: 1.5,
				Fields: map[string]interface{}{
					"title": "Test Document",
					"content": "This is a test",
				},
				Fragments: map[string][]string{
					"content": {"This is a <mark>test</mark>"},
				},
			},
			{
				ID:    "doc2",
				Score: 1.2,
				Fields: map[string]interface{}{
					"title": "Another Document",
				},
			},
		},
		Facets: nil,
	}

	result := engine.convertSearchResult(mockResult)

	// Verify basic properties
	if result.Total != 5 {
		t.Errorf("Expected total 5, got %d", result.Total)
	}
	if result.MaxScore != 1.5 {
		t.Errorf("Expected max score 1.5, got %f", result.MaxScore)
	}
	if len(result.Hits) != 2 {
		t.Errorf("Expected 2 hits, got %d", len(result.Hits))
	}

	// Verify first hit
	if result.Hits[0].ID != "doc1" {
		t.Errorf("Expected first hit ID 'doc1', got '%s'", result.Hits[0].ID)
	}
	if result.Hits[0].Score != 1.5 {
		t.Errorf("Expected first hit score 1.5, got %f", result.Hits[0].Score)
	}
	if result.Hits[0].Source["title"] != "Test Document" {
		t.Errorf("Expected first hit title 'Test Document', got '%v'", result.Hits[0].Source["title"])
	}

	// Verify highlighting
	if result.Hits[0].Highlight == nil {
		t.Error("Expected highlighting to be present")
	} else if len(result.Hits[0].Highlight["content"]) != 1 {
		t.Errorf("Expected 1 highlight fragment, got %d", len(result.Hits[0].Highlight["content"]))
	} else if result.Hits[0].Highlight["content"][0] != "This is a <mark>test</mark>" {
		t.Errorf("Expected highlight fragment 'This is a <mark>test</mark>', got '%s'", result.Hits[0].Highlight["content"][0])
	}

	// Verify second hit (no highlighting)
	if result.Hits[1].ID != "doc2" {
		t.Errorf("Expected second hit ID 'doc2', got '%s'", result.Hits[1].ID)
	}
	if result.Hits[1].Highlight != nil {
		t.Error("Expected no highlighting for second hit")
	}
}

func TestEngine_ConvertTextQuery(t *testing.T) {
	engine := &Engine{}

	// Test text query with path
	textQuery := map[string]interface{}{
		"query": "test search",
		"path":  "content",
	}

	query, err := engine.convertTextQuery(textQuery)
	if err != nil {
		t.Fatalf("Failed to convert text query: %v", err)
	}

	if query == nil {
		t.Fatal("Expected query to be created")
	}

	// Test text query without path
	textQueryNoPath := map[string]interface{}{
		"query": "test search",
	}

	query2, err := engine.convertTextQuery(textQueryNoPath)
	if err != nil {
		t.Fatalf("Failed to convert text query without path: %v", err)
	}

	if query2 == nil {
		t.Fatal("Expected query to be created")
	}
}

func TestEngine_ConvertTermQuery(t *testing.T) {
	engine := &Engine{}

	termQuery := map[string]interface{}{
		"value": "exact_value",
		"path":  "status",
	}

	query, err := engine.convertTermQuery(termQuery)
	if err != nil {
		t.Fatalf("Failed to convert term query: %v", err)
	}

	if query == nil {
		t.Fatal("Expected query to be created")
	}
}

func TestEngine_ConvertWildcardQuery(t *testing.T) {
	engine := &Engine{}

	wildcardQuery := map[string]interface{}{
		"value": "test*",
		"path":  "title",
	}

	query, err := engine.convertWildcardQuery(wildcardQuery)
	if err != nil {
		t.Fatalf("Failed to convert wildcard query: %v", err)
	}

	if query == nil {
		t.Fatal("Expected query to be created")
	}
}
