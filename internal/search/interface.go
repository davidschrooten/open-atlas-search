package search

import (
	"time"

	"github.com/davidschrooten/open-atlas-search/config"
)

// SearchEngine defines the interface for search engine operations
// This interface allows for easy mocking and testing
type SearchEngine interface {
	// Index management
	CreateIndex(indexCfg config.IndexConfig) error
	ListIndexes() ([]IndexInfo, error)
	RemoveIndex(indexName string) error
	CleanupIndexes(cfg *config.Config)

	// Document operations
	IndexDocument(indexName, docID string, doc map[string]interface{}) error
	IndexDocuments(indexName string, docs []DocumentBatch) error // Bulk indexing
	DeleteDocument(indexName, docID string) error

	// Search operations
	Search(req SearchRequest) (*SearchResult, error)

	// Mapping operations
	GetIndexMapping(indexName string) (map[string]interface{}, error)

	// Sync tracking
	UpdateLastSync(indexName string, syncTime time.Time)

	// Lifecycle
	Close() error
}

// DocumentBatch represents a document for bulk indexing
type DocumentBatch struct {
	ID  string                 `json:"id"`
	Doc map[string]interface{} `json:"doc"`
}
