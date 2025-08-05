package search

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"

	"github.com/davidschrooten/open-atlas-search/config"
)

// Engine manages multiple Bleve indexes
type Engine struct {
	indexes   map[string]bleve.Index
	indexPath string
	mutex     sync.RWMutex
	lastSync  map[string]time.Time // Track last sync time for each index
	syncMutex sync.RWMutex         // Separate mutex for sync times
}

// SearchResult represents search results with Atlas Search compatibility
type SearchResult struct {
	Hits     []SearchHit            `json:"hits"`
	Total    int                    `json:"total"`
	Facets   map[string]interface{} `json:"facets,omitempty"`
	MaxScore float64                `json:"maxScore"`
}

// SearchHit represents a single search result
type SearchHit struct {
	ID        string                 `json:"_id"`
	Score     float64                `json:"score"`
	Source    map[string]interface{} `json:"source"`
	Highlight map[string][]string    `json:"highlight,omitempty"`
}

// FacetRequest represents a facet aggregation request
type FacetRequest struct {
	Type  string `json:"type"`
	Field string `json:"field"`
	Size  int    `json:"size,omitempty"`
}

// SearchRequest represents a search query request
type SearchRequest struct {
	Index     string                  `json:"index"`
	Query     map[string]interface{}  `json:"query"`
	Highlight map[string]interface{}  `json:"highlight,omitempty"`
	Facets    map[string]FacetRequest `json:"facets,omitempty"`
	Size      int                     `json:"size"`
	From      int                     `json:"from"`
}

// NewEngine creates a new search engine
func NewEngine(cfg config.SearchConfig) (*Engine, error) {
	if err := os.MkdirAll(cfg.IndexPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	return &Engine{
		indexes:   make(map[string]bleve.Index),
		indexPath: cfg.IndexPath,
		lastSync:  make(map[string]time.Time),
	}, nil
}

// CreateIndex creates a new Bleve index based on configuration
func (e *Engine) CreateIndex(indexCfg config.IndexConfig) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	indexName := indexCfg.Name
	
	// In cluster mode with multiple shards, create separate indexes for each shard
	if indexCfg.Distribution.Shards > 1 {
		return e.createShardedIndex(indexCfg)
	}
	
	// Single shard index
	return e.createSingleIndex(indexCfg)
}

// createSingleIndex creates a single non-sharded index
func (e *Engine) createSingleIndex(indexCfg config.IndexConfig) error {
	indexName := indexCfg.Name
	indexPath := filepath.Join(e.indexPath, indexName)

	// Create mapping based on configuration
	indexMapping := e.createMapping(indexCfg.Definition)

	// Check if index already exists
	if _, exists := e.indexes[indexName]; exists {
		return nil // Index already exists
	}

	// Try to open existing index first
	index, err := bleve.Open(indexPath)
	if err != nil {
		// Create new index if it doesn't exist
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", indexName, err)
		}
	}

	e.indexes[indexName] = index
	return nil
}

// createShardedIndex creates multiple shard indexes for a single logical index
func (e *Engine) createShardedIndex(indexCfg config.IndexConfig) error {
	indexName := indexCfg.Name  
	
	// Create mapping based on configuration
	indexMapping := e.createMapping(indexCfg.Definition)
	
	for shard := 0; shard < indexCfg.Distribution.Shards; shard++ {
		shardName := fmt.Sprintf("%s_shard_%d", indexName, shard)
		shardPath := filepath.Join(e.indexPath, shardName)
		
		// Check if shard already exists
		if _, exists := e.indexes[shardName]; exists {
			continue // Shard already exists
		}
		
		// Try to open existing shard first
		index, err := bleve.Open(shardPath)
		if err != nil {
			// Create new shard if it doesn't exist
			index, err = bleve.New(shardPath, indexMapping)
			if err != nil {
				return fmt.Errorf("failed to create shard %s: %w", shardName, err)
			}
		}
		
		e.indexes[shardName] = index
	}
	
	return nil
}

// GetIndex returns an index by name
func (e *Engine) GetIndex(indexName string) (bleve.Index, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	index, exists := e.indexes[indexName]
	return index, exists
}

// IndexInfo represents information about an index
type IndexInfo struct {
	Name         string     `json:"name"`
	DocCount     uint64     `json:"docCount"`
	Status       string     `json:"status"`
	LastSync     *time.Time `json:"lastSync,omitempty"`
	SyncProgress string     `json:"sync_progress,omitempty"`
}

// ListIndexes returns information about all indexes
func (e *Engine) ListIndexes() ([]IndexInfo, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	indexes := make([]IndexInfo, 0, len(e.indexes))

	for name, index := range e.indexes {
		docCount, err := index.DocCount()
		if err != nil {
			// If we can't get doc count, set it to 0 and continue
			docCount = 0
		}

		indexInfo := IndexInfo{
			Name:     name,
			DocCount: docCount,
			Status:   "active",
		}

		// Get last sync time if available
		e.syncMutex.RLock()
		if lastSync, exists := e.lastSync[name]; exists {
			indexInfo.LastSync = &lastSync
		}
		e.syncMutex.RUnlock()

		indexes = append(indexes, indexInfo)
	}

	return indexes, nil
}

// RemoveIndex removes an index from memory and disk
func (e *Engine) RemoveIndex(indexName string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	index, exists := e.indexes[indexName]
	if !exists {
		return fmt.Errorf("index %s not found", indexName)
	}

	// Close index
	if err := index.Close(); err != nil {
		return fmt.Errorf("failed to close index %s: %w", indexName, err)
	}

	// Remove index from the map
	delete(e.indexes, indexName)

	// Remove sync tracking
	e.syncMutex.Lock()
	delete(e.lastSync, indexName)
	e.syncMutex.Unlock()

	// Delete the index directory
	indexPath := filepath.Join(e.indexPath, indexName)
	if err := os.RemoveAll(indexPath); err != nil {
		return fmt.Errorf("failed to remove index directory %s: %w", indexPath, err)
	}

	return nil
}

// CleanupIndexes removes indexes that are no longer in the configuration
func (e *Engine) CleanupIndexes(cfg *config.Config) {
	configuredIndexes := make(map[string]bool)
	for _, indexCfg := range cfg.Indexes {
		indexName := indexCfg.Name
		configuredIndexes[indexName] = true
	}

	// Find indexes to remove
	var indexesToRemove []string
	e.mutex.RLock()
	for indexName := range e.indexes {
		if !configuredIndexes[indexName] {
			indexesToRemove = append(indexesToRemove, indexName)
		}
	}
	e.mutex.RUnlock()

	// Remove indexes (this will acquire its own locks)
	for _, indexName := range indexesToRemove {
		log.Printf("Removing index: %s", indexName)
		if err := e.removeIndexInternal(indexName); err != nil {
			log.Printf("Failed to remove index %s: %v", indexName, err)
		}
	}
}

// removeIndexInternal removes an index from memory and disk (internal method)
func (e *Engine) removeIndexInternal(indexName string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	index, exists := e.indexes[indexName]
	if !exists {
		return fmt.Errorf("index %s not found", indexName)
	}

	// Close index
	if err := index.Close(); err != nil {
		return fmt.Errorf("failed to close index %s: %w", indexName, err)
	}

	// Remove index from the map
	delete(e.indexes, indexName)

	// Remove sync tracking
	e.syncMutex.Lock()
	delete(e.lastSync, indexName)
	e.syncMutex.Unlock()

	// Delete the index directory
	indexPath := filepath.Join(e.indexPath, indexName)
	if err := os.RemoveAll(indexPath); err != nil {
		return fmt.Errorf("failed to remove index directory %s: %w", indexPath, err)
	}

	return nil
}

// IndexDocument indexes a document
func (e *Engine) IndexDocument(indexName, docID string, doc map[string]interface{}) error {
	// For sharded indexes, determine which shard to use
	shardName := e.getShardForDocument(indexName, docID)
	
	e.mutex.RLock()
	index, exists := e.indexes[shardName]
	e.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("index/shard %s not found", shardName)
	}

	return index.Index(docID, doc)
}

// IndexDocuments indexes multiple documents in a batch for better performance
func (e *Engine) IndexDocuments(indexName string, docs []DocumentBatch) error {
	e.mutex.RLock()
	index, exists := e.indexes[indexName]
	e.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("index %s not found", indexName)
	}

	// Create a batch for bulk indexing
	batch := index.NewBatch()
	for _, docBatch := range docs {
		batch.Index(docBatch.ID, docBatch.Doc)
	}

	// Execute the batch
	return index.Batch(batch)
}

// DeleteDocument removes a document from the index
func (e *Engine) DeleteDocument(indexName, docID string) error {
	e.mutex.RLock()
	index, exists := e.indexes[indexName]
	e.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("index %s not found", indexName)
	}

	return index.Delete(docID)
}

// Search performs a search query
func (e *Engine) Search(req SearchRequest) (*SearchResult, error) {
	e.mutex.RLock()
	index, exists := e.indexes[req.Index]
	e.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s not found", req.Index)
	}

	// Convert query to Bleve query
	bleveQuery, err := e.convertQuery(req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to convert query: %w", err)
	}

	// Create search request
	searchReq := bleve.NewSearchRequest(bleveQuery)
	searchReq.Size = req.Size
	searchReq.From = req.From

	// Include all stored fields in results
	searchReq.Fields = []string{"*"}
	searchReq.IncludeLocations = false // We don't need location info

	// Add highlighting if requested
	if req.Highlight != nil {
		e.addHighlighting(searchReq, req.Highlight)
	}

	// Add facets if requested
	if req.Facets != nil {
		e.addFacets(searchReq, req.Facets)
	}

	// Execute search
	searchResult, err := index.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to our result format
	return e.convertSearchResult(searchResult), nil
}

// Close closes all indexes
func (e *Engine) Close() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var errors []error
	for name, index := range e.indexes {
		if err := index.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close index %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing indexes: %v", errors)
	}

	return nil
}

// createMapping creates a Bleve mapping from configuration
func (e *Engine) createMapping(def config.IndexDefinition) mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	if def.Mappings.Dynamic {
		indexMapping.DefaultMapping.Dynamic = true
		// Enable storing all fields by default for dynamic mapping
		indexMapping.StoreDynamic = true
	}

	// Configure field mappings
	for _, fieldCfg := range def.Mappings.Fields {
		fieldMapping := e.createFieldMapping(fieldCfg)
		indexMapping.DefaultMapping.AddFieldMappingsAt(fieldCfg.Name, fieldMapping)
	}

	return indexMapping
}

// createFieldMapping creates a field mapping from configuration
func (e *Engine) createFieldMapping(cfg config.FieldConfig) *mapping.FieldMapping {
	fieldMapping := bleve.NewTextFieldMapping()

	switch cfg.Type {
	case "text":
		fieldMapping = bleve.NewTextFieldMapping()
	case "keyword":
		fieldMapping = bleve.NewKeywordFieldMapping()
	case "numeric":
		fieldMapping = bleve.NewNumericFieldMapping()
	case "date":
		fieldMapping = bleve.NewDateTimeFieldMapping()
	case "boolean":
		fieldMapping = bleve.NewBooleanFieldMapping()
	}

	if cfg.Analyzer != "" {
		fieldMapping.Analyzer = cfg.Analyzer
	}

	// Always store field values so they can be retrieved in search results
	fieldMapping.Store = true

	return fieldMapping
}

// convertQuery converts Atlas Search query to Bleve query
func (e *Engine) convertQuery(atlasQuery map[string]interface{}) (query.Query, error) {
	if compound, ok := atlasQuery["compound"]; ok {
		return e.convertCompoundQuery(compound.(map[string]interface{}))
	}

	if text, ok := atlasQuery["text"]; ok {
		return e.convertTextQuery(text.(map[string]interface{}))
	}

	if term, ok := atlasQuery["term"]; ok {
		return e.convertTermQuery(term.(map[string]interface{}))
	}

	if wildcard, ok := atlasQuery["wildcard"]; ok {
		return e.convertWildcardQuery(wildcard.(map[string]interface{}))
	}

	// Handle match_all query (Elasticsearch-like)
	if _, ok := atlasQuery["match_all"]; ok {
		return bleve.NewMatchAllQuery(), nil
	}

	// Default to match all query
	return bleve.NewMatchAllQuery(), nil
}

// convertCompoundQuery converts compound queries
func (e *Engine) convertCompoundQuery(compound map[string]interface{}) (query.Query, error) {
	boolQuery := bleve.NewBooleanQuery()

	if must, ok := compound["must"]; ok {
		mustQueries := must.([]interface{})
		for _, q := range mustQueries {
			subQuery, err := e.convertQuery(q.(map[string]interface{}))
			if err != nil {
				return nil, err
			}
			boolQuery.AddMust(subQuery)
		}
	}

	if should, ok := compound["should"]; ok {
		shouldQueries := should.([]interface{})
		for _, q := range shouldQueries {
			subQuery, err := e.convertQuery(q.(map[string]interface{}))
			if err != nil {
				return nil, err
			}
			boolQuery.AddShould(subQuery)
		}
	}

	if mustNot, ok := compound["mustNot"]; ok {
		mustNotQueries := mustNot.([]interface{})
		for _, q := range mustNotQueries {
			subQuery, err := e.convertQuery(q.(map[string]interface{}))
			if err != nil {
				return nil, err
			}
			boolQuery.AddMustNot(subQuery)
		}
	}

	return boolQuery, nil
}

// convertTextQuery converts text search queries
func (e *Engine) convertTextQuery(textQuery map[string]interface{}) (query.Query, error) {
	queryText := textQuery["query"].(string)

	if path, ok := textQuery["path"]; ok {
		field := path.(string)
		matchQuery := bleve.NewMatchQuery(queryText)
		matchQuery.SetField(field)
		return matchQuery, nil
	}

	return bleve.NewQueryStringQuery(queryText), nil
}

// convertTermQuery converts term queries
func (e *Engine) convertTermQuery(termQuery map[string]interface{}) (query.Query, error) {
	value := termQuery["value"].(string)
	path := termQuery["path"].(string)

	termQueryObj := bleve.NewTermQuery(value)
	termQueryObj.SetField(path)
	return termQueryObj, nil
}

// convertWildcardQuery converts wildcard queries
func (e *Engine) convertWildcardQuery(wildcardQuery map[string]interface{}) (query.Query, error) {
	value := wildcardQuery["value"].(string)
	path := wildcardQuery["path"].(string)

	wildcardQueryObj := bleve.NewWildcardQuery(value)
	wildcardQueryObj.SetField(path)
	return wildcardQueryObj, nil
}

// addHighlighting adds highlighting to search request
func (e *Engine) addHighlighting(searchReq *bleve.SearchRequest, highlight map[string]interface{}) {
	searchReq.Highlight = bleve.NewHighlight()
	if fields, ok := highlight["fields"]; ok {
		for _, field := range fields.([]interface{}) {
			searchReq.Highlight.AddField(field.(string))
		}
	}
}

// addFacets adds facets to search request
func (e *Engine) addFacets(searchReq *bleve.SearchRequest, facets map[string]FacetRequest) {
	for name, facet := range facets {
		var facetReq *bleve.FacetRequest

		switch facet.Type {
		case "terms":
			facetReq = bleve.NewFacetRequest(facet.Field, facet.Size)
		case "numeric":
			facetReq = bleve.NewFacetRequest(facet.Field, facet.Size)
		case "date":
			facetReq = bleve.NewFacetRequest(facet.Field, facet.Size)
		}

		if facetReq != nil {
			searchReq.AddFacet(name, facetReq)
		}
	}
}

// convertSearchResult converts Bleve search result to our format
func (e *Engine) convertSearchResult(result *bleve.SearchResult) *SearchResult {
	hits := make([]SearchHit, len(result.Hits))

	for i, hit := range result.Hits {
		// Convert fields to source document
		source := make(map[string]interface{})
		for field, value := range hit.Fields {
			source[field] = value
		}

		hits[i] = SearchHit{
			ID:     hit.ID,
			Score:  hit.Score,
			Source: source,
		}

		// Add highlighting if available
		if len(hit.Fragments) > 0 {
			hits[i].Highlight = hit.Fragments
		}
	}

	searchResult := &SearchResult{
		Hits:     hits,
		Total:    int(result.Total),
		MaxScore: result.MaxScore,
	}

	// Add facets if available
	if len(result.Facets) > 0 {
		searchResult.Facets = make(map[string]interface{})
		for name, facet := range result.Facets {
			buckets := make([]map[string]interface{}, 0)

			if facet.Terms != nil {
				for _, term := range facet.Terms.Terms() {
					buckets = append(buckets, map[string]interface{}{
						"key":   term.Term,
						"count": term.Count,
					})
				}
			}

			facetData := map[string]interface{}{
				"buckets": buckets,
			}

			searchResult.Facets[name] = facetData
		}
	}

	return searchResult
}

// UpdateLastSync updates the last sync time for an index
func (e *Engine) UpdateLastSync(indexName string, syncTime time.Time) {
	e.syncMutex.Lock()
	defer e.syncMutex.Unlock()
	e.lastSync[indexName] = syncTime
}

// GetIndexMapping returns the mapping configuration for an index
func (e *Engine) GetIndexMapping(indexName string) (map[string]interface{}, error) {
	e.mutex.RLock()
	_, exists := e.indexes[indexName]
	e.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s not found", indexName)
	}

	// Return basic mapping info
	// For a more complete implementation, you'd need to store the original config
	// or parse the bleve mapping structure more carefully
	result := map[string]interface{}{
		"name":    indexName,
		"type":    "bleve",
		"status":  "active",
		"message": "Mapping details available through Bleve index introspection",
	}

	return result, nil
}

// getShardForDocument determines which shard a document should be indexed to
func (e *Engine) getShardForDocument(indexName, docID string) string {
	// Check if this is a sharded index by looking for shard indexes
	shardCount := 0
	e.mutex.RLock()
	for name := range e.indexes {
		if len(name) > len(indexName) && name[:len(indexName)] == indexName && name[len(indexName):len(indexName)+7] == "_shard_" {
			shardCount++
		}
	}
	e.mutex.RUnlock()

	// If no shards found, use the index name directly
	if shardCount == 0 {
		return indexName
	}

	// Use consistent hashing to determine shard
	hash := fnv32(docID)
	shardNum := int(hash) % shardCount
	return fmt.Sprintf("%s_shard_%d", indexName, shardNum)
}

// SearchSharded performs a search across all shards of an index
func (e *Engine) SearchSharded(req SearchRequest) (*SearchResult, error) {
	// Find all shards for this index
	shards := e.getShardsForIndex(req.Index)
	
	if len(shards) == 0 {
		// No shards found, try direct index search
		return e.Search(req)
	}

	// Search all shards in parallel
	type shardResult struct {
		result *SearchResult
		err    error
	}

	resultChan := make(chan shardResult, len(shards))

	for _, shardName := range shards {
		go func(shard string) {
			shardReq := req
			shardReq.Index = shard
			result, err := e.Search(shardReq)
			resultChan <- shardResult{result: result, err: err}
		}(shardName)
	}

	// Collect and merge results
	allHits := []SearchHit{}
	allFacets := make(map[string]interface{})
	totalCount := 0
	maxScore := float64(0)

	for i := 0; i < len(shards); i++ {
		shardRes := <-resultChan
		if shardRes.err != nil {
			log.Printf("Error searching shard: %v", shardRes.err)
			continue
		}

		allHits = append(allHits, shardRes.result.Hits...)
		totalCount += shardRes.result.Total
		if shardRes.result.MaxScore > maxScore {
			maxScore = shardRes.result.MaxScore
		}

		// Merge facets (simple aggregation)
		for name, facet := range shardRes.result.Facets {
			if facetData, ok := facet.(map[string]interface{}); ok {
				if buckets, ok := facetData["buckets"].([]map[string]interface{}); ok {
					if existingFacet, exists := allFacets[name]; exists {
						// Merge buckets
						if existingData, ok := existingFacet.(map[string]interface{}); ok {
							if existingBuckets, ok := existingData["buckets"].([]map[string]interface{}); ok {
								allFacets[name] = map[string]interface{}{
									"buckets": e.mergeFacetBuckets(existingBuckets, buckets),
								}
							}
						}
					} else {
						allFacets[name] = facet
					}
				}
			}
		}
	}

	// Sort hits by score and apply pagination
	e.sortHitsByScore(allHits)

	// Apply pagination
	from := req.From
	size := req.Size
	if size == 0 {
		size = 10 // Default size
	}

	if from >= len(allHits) {
		allHits = []SearchHit{}
	} else {
		end := from + size
		if end > len(allHits) {
			end = len(allHits)
		}
		allHits = allHits[from:end]
	}

	return &SearchResult{
		Hits:     allHits,
		Total:    totalCount,
		Facets:   allFacets,
		MaxScore: maxScore,
	}, nil
}

// getShardsForIndex returns all shard names for a given index
func (e *Engine) getShardsForIndex(indexName string) []string {
	var shards []string
	e.mutex.RLock()
	for name := range e.indexes {
		if len(name) > len(indexName) && name[:len(indexName)] == indexName && name[len(indexName):len(indexName)+7] == "_shard_" {
			shards = append(shards, name)
		}
	}
	e.mutex.RUnlock()
	return shards
}

// mergeFacetBuckets merges two sets of facet buckets
func (e *Engine) mergeFacetBuckets(buckets1, buckets2 []map[string]interface{}) []map[string]interface{} {
	bucketMap := make(map[string]int)
	for _, bucket := range buckets1 {
		if key, ok := bucket["key"].(string); ok {
			if count, ok := bucket["count"].(int); ok {
				bucketMap[key] = count
			}
		}
	}

	for _, bucket := range buckets2 {
		if key, ok := bucket["key"].(string); ok {
			if count, ok := bucket["count"].(int); ok {
				bucketMap[key] += count
			}
		}
	}

	var mergedBuckets []map[string]interface{}
	for key, count := range bucketMap {
		mergedBuckets = append(mergedBuckets, map[string]interface{}{
			"key":   key,
			"count": count,
		})
	}

	return mergedBuckets
}

// sortHitsByScore sorts search hits by score in descending order
func (e *Engine) sortHitsByScore(hits []SearchHit) {
	for i := 0; i < len(hits)-1; i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[i].Score < hits[j].Score {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
}

// fnv32 implements a simple 32-bit FNV-1a hash
func fnv32(data string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)

	hash := uint32(offset32)
	for _, b := range []byte(data) {
		hash ^= uint32(b)
		hash *= prime32
	}
	return hash
}
