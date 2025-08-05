package indexer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/david/open-atlas-search/config"
	"github.com/david/open-atlas-search/internal/mongodb"
	"github.com/david/open-atlas-search/internal/search"
	syncstate "github.com/david/open-atlas-search/internal/sync"
)

// Service manages indexing operations
type Service struct {
	mongoClient      *mongodb.Client
	searchEngine     *search.Engine
	config           *config.Config
	wg               sync.WaitGroup
	stopCh           chan struct{}
	syncStateManager *syncstate.StateManager
	saveStateCh      chan struct{} // Channel to trigger state saving
	// Performance optimization fields
	workQueue       chan IndexingJob
	workerPool      []chan IndexingJob
	bulkBuffer      map[string][]search.DocumentBatch
	bulkBufferMutex sync.RWMutex
}

// IndexingJob represents a document indexing job
type IndexingJob struct {
	IndexName     string
	CollectionKey string
	Documents     []search.DocumentBatch
}

// NewService creates a new indexer service
func NewService(mongoClient *mongodb.Client, searchEngine *search.Engine, cfg *config.Config) (*Service, error) {
	// Initialize sync state manager
	syncStateManager := syncstate.NewStateManager(cfg.Search.SyncStatePath)
	if err := syncStateManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load sync state: %w", err)
	}

	service := &Service{
		mongoClient:      mongoClient,
		searchEngine:     searchEngine,
		config:           cfg,
		stopCh:           make(chan struct{}),
		syncStateManager: syncStateManager,
		saveStateCh:      make(chan struct{}, 1),
	}

	// Create indexes based on configuration
	for _, indexCfg := range cfg.Indexes {
		if err := searchEngine.CreateIndex(indexCfg); err != nil {
			return nil, fmt.Errorf("failed to create index %s: %w", indexCfg.Name, err)
		}
	}

	// Validate and setup timestamp fields
	if err := service.setupTimestampFields(); err != nil {
		return nil, fmt.Errorf("failed to setup timestamp fields: %w", err)
	}

	// Cleanup indexes that are no longer in configuration
	searchEngine.CleanupIndexes(cfg)

	return service, nil
}

// setupTimestampFields validates and sets up timestamp fields for each collection
func (s *Service) setupTimestampFields() error {
	for _, indexCfg := range s.config.Indexes {
		timestampField := indexCfg.TimestampField
		if timestampField == "" {
			timestampField = "updated_at" // Default timestamp field
		}

		// Skip _id field validation
		if timestampField == "_id" {
			continue
		}

		// Check if timestamp field exists
		exists, err := s.mongoClient.CheckTimestampField(indexCfg.Collection, timestampField)
		if err != nil {
			return fmt.Errorf("failed to check timestamp field %s in collection %s: %w", timestampField, indexCfg.Collection, err)
		}

		if !exists {
			// Ask user if they want to add the timestamp field
			log.Printf("Timestamp field '%s' not found in collection '%s'", timestampField, indexCfg.Collection)
			log.Printf("Do you want to add '%s' field to all documents in collection '%s'? This will set the field to current timestamp for existing documents. (y/N)", timestampField, indexCfg.Collection)
			
			var response string
			fmt.Scanln(&response)
			
			if response == "y" || response == "Y" || response == "yes" || response == "Yes" {
				log.Printf("Adding '%s' field to collection '%s'...", timestampField, indexCfg.Collection)
				if err := s.mongoClient.AddTimestampField(indexCfg.Collection, timestampField); err != nil {
					return fmt.Errorf("failed to add timestamp field: %w", err)
				}
			} else {
				log.Printf("Skipping timestamp field setup for collection '%s'. Using _id field for polling.", indexCfg.Collection)
				// Update the configuration to use _id field
				for i := range s.config.Indexes {
					if s.config.Indexes[i].Collection == indexCfg.Collection {
						s.config.Indexes[i].TimestampField = "_id"
					}
				}
			}
		}
	}
	return nil
}

// Start begins the indexing process
func (s *Service) Start(ctx context.Context) error {
	log.Println("Starting indexer service...")

	// Start periodic state saving
	s.wg.Add(1)
	go s.syncStateManager.StartPeriodicSave(30*time.Second, s.stopCh, &s.wg)

	// Start initial bulk indexing for each configured index
	for _, indexCfg := range s.config.Indexes {
		s.wg.Add(1)
		go s.performInitialIndexing(ctx, indexCfg)

		s.wg.Add(1)
		go s.pollForChanges(ctx, indexCfg)
	}

	// Start flush routine
	s.wg.Add(1)
	go s.flushRoutine(ctx)

	return nil
}

// Stop stops the indexing service
func (s *Service) Stop() {
	log.Println("Stopping indexer service...")
	close(s.stopCh)
	s.wg.Wait()

	// Final save of sync state
	if err := s.syncStateManager.Save(); err != nil {
		log.Printf("Failed to save sync state during shutdown: %v", err)
	} else {
		log.Println("Sync state saved successfully")
	}

	log.Println("Indexer service stopped")
}

// performInitialIndexing performs bulk indexing of existing documents
func (s *Service) performInitialIndexing(ctx context.Context, indexCfg config.IndexConfig) {
	defer s.wg.Done()

	log.Printf("Starting initial indexing for %s.%s", indexCfg.Database, indexCfg.Collection)

	indexName := indexCfg.Name
	collectionKey := fmt.Sprintf("%s.%s", indexCfg.Database, indexCfg.Collection)

	// Set initial sync status to in_progress
	s.syncStateManager.SetSyncStatus(collectionKey, syncstate.SyncStatusInProgress)
	s.syncStateManager.SetProgress(collectionKey, "0%")

	// Get total document count for progress calculation
	totalDocs, err := s.mongoClient.CountDocuments(indexCfg.Collection, bson.M{})
	if err != nil {
		log.Printf("Failed to count documents in %s: %v", indexCfg.Collection, err)
		// Set progress to not_available if we can't count
		s.syncStateManager.SetProgress(collectionKey, "not_available")
	} else {
		s.syncStateManager.SetTotalDocuments(collectionKey, totalDocs)
	}

	// Get cursor for all documents
	cursor, err := s.mongoClient.FindDocuments(indexCfg.Collection, bson.M{}, 0)
	if err != nil {
		log.Printf("Failed to get documents for initial indexing: %v", err)
		s.syncStateManager.SetSyncStatus(collectionKey, syncstate.SyncStatusIdle)
		return
	}
	defer cursor.Close(ctx)

	count := 0
	batch := make([]map[string]interface{}, 0, s.config.Search.BatchSize)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Failed to decode document: %v", err)
			continue
		}

		// Convert ObjectID to string for indexing, but support other ID types
		if id, ok := doc["_id"].(primitive.ObjectID); ok {
			doc["_id"] = id.Hex()
		} else {
			// Keep other ID types as-is (string, int, etc.)
			doc["_id"] = fmt.Sprintf("%v", doc["_id"])
		}

		batch = append(batch, doc)

		if len(batch) >= s.config.Search.BatchSize {
			s.indexBatch(indexName, batch)
			batch = batch[:0] // Reset slice
			count += s.config.Search.BatchSize
			// Update progress during initial indexing
			s.syncStateManager.IncrementDocumentsIndexed(collectionKey, int64(s.config.Search.BatchSize))
			s.syncStateManager.UpdateProgress(collectionKey)
		}

		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		default:
		}
	}

	// Index remaining documents
	if len(batch) > 0 {
		s.indexBatch(indexName, batch)
		count += len(batch)
		// Update progress for remaining documents
		s.syncStateManager.IncrementDocumentsIndexed(collectionKey, int64(len(batch)))
		s.syncStateManager.UpdateProgress(collectionKey)
	}

	log.Printf("Initial indexing completed for %s.%s: %d documents indexed", 
		indexCfg.Database, indexCfg.Collection, count)

	// Set final status to idle after completion
	s.syncStateManager.SetSyncStatus(collectionKey, syncstate.SyncStatusIdle)
	s.syncStateManager.SetProgress(collectionKey, "100%")

	// Update the last sync time for the index after initial indexing
	s.searchEngine.UpdateLastSync(indexName, time.Now())
}

// pollForChanges polls MongoDB for new/updated documents since last poll
func (s *Service) pollForChanges(ctx context.Context, indexCfg config.IndexConfig) {
	defer s.wg.Done()

	log.Printf("Starting polling for changes on %s.%s", indexCfg.Database, indexCfg.Collection)

	indexName := indexCfg.Name
	collectionKey := fmt.Sprintf("%s.%s", indexCfg.Database, indexCfg.Collection)

	// Get timestamp field for this collection
	timestampField := indexCfg.TimestampField
	if timestampField == "" {
		timestampField = "updated_at"
	}

	// Get ID field for this collection
	idField := indexCfg.IDField
	if idField == "" {
		idField = "_id"
	}

	// Initialize or restore collection state
	collectionState := s.syncStateManager.GetCollectionState(collectionKey)
	if collectionState == nil {
		// Get the timestamp of the most recent document as starting point
		lastTimestamp, err := s.mongoClient.GetLastDocumentTimestamp(indexCfg.Collection, timestampField)
		if err != nil {
			log.Printf("Failed to get last document timestamp for %s: %v", collectionKey, err)
			// Start from current time if we can't get last document timestamp
			lastTimestamp = time.Now()
		}

		collectionState = &syncstate.CollectionState{
			LastPollTime:   lastTimestamp,
			IndexName:      indexName,
			CollectionKey:  collectionKey,
			TimestampField: timestampField,
			IDField:        idField,
		}
		s.syncStateManager.UpdateCollectionState(collectionKey, collectionState)
		log.Printf("Initialized collection state for %s, starting from %v", collectionKey, lastTimestamp)
	} else {
		log.Printf("Restored collection state for %s, resuming from %v", collectionKey, collectionState.LastPollTime)
	}

	// Poll interval (configurable per index, defaulting to 5 seconds)
	pollInterval := 5 * time.Second
	if indexCfg.PollInterval > 0 {
		pollInterval = time.Duration(indexCfg.PollInterval) * time.Second
	} else if s.config.Search.FlushInterval > 0 {
		// Use flush interval as a basis, but make polling more frequent
		pollInterval = time.Duration(s.config.Search.FlushInterval/2) * time.Second
		if pollInterval < time.Second {
			pollInterval = time.Second
		}
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performPoll(ctx, indexCfg)

		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		}
	}
}

// performPoll performs a single polling operation to check for new documents
func (s *Service) performPoll(ctx context.Context, indexCfg config.IndexConfig) {
	indexName := indexCfg.Name
	collectionKey := fmt.Sprintf("%s.%s", indexCfg.Database, indexCfg.Collection)

	// Get current collection state
	collectionState := s.syncStateManager.GetCollectionState(collectionKey)
	if collectionState == nil {
		log.Printf("No collection state found for %s, skipping poll", collectionKey)
		return
	}

	lastPoll := collectionState.LastPollTime
	timestampField := collectionState.TimestampField
	idField := collectionState.IDField

	// Find documents created/updated since last poll
	cursor, err := s.mongoClient.FindDocumentsSince(indexCfg.Collection, timestampField, lastPoll, int64(s.config.Search.BatchSize))
	if err != nil {
		log.Printf("Failed to poll for changes in %s: %v", collectionKey, err)
		return
	}
	defer cursor.Close(ctx)

	count := 0
	batch := make([]map[string]interface{}, 0, s.config.Search.BatchSize)
	newestTimestamp := lastPoll

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Failed to decode document: %v", err)
			continue
		}

		// Track the newest timestamp based on the configured field
		if timestampField == "" || timestampField == "_id" {
			// Use ObjectID timestamp
			if id, ok := doc["_id"].(primitive.ObjectID); ok {
				docTimestamp := id.Timestamp()
				if docTimestamp.After(newestTimestamp) {
					newestTimestamp = docTimestamp
				}
			}
		} else {
			// Use custom timestamp field
			if timestampVal, exists := doc[timestampField]; exists {
				if docTimestamp, err := s.mongoClient.ParseTimestamp(timestampVal); err == nil {
					if docTimestamp.After(newestTimestamp) {
						newestTimestamp = docTimestamp
					}
				}
			}
		}

		// Handle configurable ID field - convert to string for indexing
		if idVal, exists := doc[idField]; exists {
			if id, ok := idVal.(primitive.ObjectID); ok {
				doc[idField] = id.Hex()
			} else {
				// Keep other ID types as-is (string, int, etc.)
				doc[idField] = fmt.Sprintf("%v", idVal)
			}
			// Always ensure _id is set for search indexing
			if idField != "_id" {
				doc["_id"] = doc[idField]
			}
		} else {
			log.Printf("Document missing ID field '%s', skipping", idField)
			continue
		}

		batch = append(batch, doc)
		count++

		if len(batch) >= s.config.Search.BatchSize {
			s.indexBatch(indexName, batch)
			batch = batch[:0] // Reset slice
		}

		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		default:
		}
	}

	// Index remaining documents
	if len(batch) > 0 {
		s.indexBatch(indexName, batch)
	}

	// Update state with new poll time and document count
	if count > 0 {
		s.syncStateManager.SetLastPollTime(collectionKey, newestTimestamp)
		s.syncStateManager.IncrementDocumentsIndexed(collectionKey, int64(count))
		log.Printf("Polled %d new/updated documents from %s using timestamp field '%s'", count, collectionKey, timestampField)
	}

	// Always update the last sync time for the index (even if no new documents)
	s.syncStateManager.SetLastSyncTime(collectionKey, time.Now())
	s.searchEngine.UpdateLastSync(indexName, time.Now())
}


// indexBatch indexes a batch of documents using bulk operations for better performance
func (s *Service) indexBatch(indexName string, batch []map[string]interface{}) {
	if s.config.Search.BulkIndexing {
		// Use bulk indexing for better performance
		s.indexBatchBulk(indexName, batch)
	} else {
		// Use individual indexing for compatibility
		s.indexBatchIndividual(indexName, batch)
	}
}

// indexBatchBulk indexes documents using bulk operations for optimal performance
func (s *Service) indexBatchBulk(indexName string, batch []map[string]interface{}) {
	docs := make([]search.DocumentBatch, 0, len(batch))
	for _, doc := range batch {
		if idVal, ok := doc["_id"]; ok {
			docID := fmt.Sprintf("%v", idVal)
			docs = append(docs, search.DocumentBatch{
				ID:  docID,
				Doc: doc,
			})
		}
	}

	if len(docs) > 0 {
		if err := s.searchEngine.IndexDocuments(indexName, docs); err != nil {
			log.Printf("Failed to bulk index %d documents: %v", len(docs), err)
			// Fallback to individual indexing on error
			s.indexBatchIndividual(indexName, batch)
		}
	}
}

// indexBatchIndividual indexes documents one by one (fallback method)
func (s *Service) indexBatchIndividual(indexName string, batch []map[string]interface{}) {
	for _, doc := range batch {
		if idVal, ok := doc["_id"]; ok {
			docID := fmt.Sprintf("%v", idVal)
			if err := s.searchEngine.IndexDocument(indexName, docID, doc); err != nil {
				log.Printf("Failed to index document %s: %v", docID, err)
			}
		}
	}
}

// flushRoutine periodically flushes indexes
func (s *Service) flushRoutine(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(s.config.Search.FlushInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Note: Bleve automatically handles flushing, but we could add custom logic here
			log.Println("Periodic flush completed")

		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		}
	}
}

// GetIndexStats returns statistics about an index
func (s *Service) GetIndexStats(indexName string) (map[string]interface{}, error) {
	index, exists := s.searchEngine.GetIndex(indexName)
	if !exists {
		return nil, fmt.Errorf("index %s not found", indexName)
	}

	docCount, err := index.DocCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	stats := map[string]interface{}{
		"name":      indexName,
		"docCount":  docCount,
		"status":    "active",
	}

	return stats, nil
}

// GetSyncStates returns the synchronization states for all collections
func (s *Service) GetSyncStates() map[string]*syncstate.CollectionState {
	if s.syncStateManager == nil {
		return nil
	}

	return s.syncStateManager.GetAllCollectionStates()
}
