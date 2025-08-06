package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewStateManager(t *testing.T) {
	sm := NewStateManager("/tmp/test_sync_state.json")
	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}
	if sm.filePath != "/tmp/test_sync_state.json" {
		t.Errorf("Expected filePath to be '/tmp/test_sync_state.json', got '%s'", sm.filePath)
	}
	if sm.state == nil {
		t.Error("Expected state to be initialized")
	}
	if sm.state.Collections == nil {
		t.Error("Expected Collections map to be initialized")
	}
}

func TestStateManager_SaveAndLoad(t *testing.T) {
	// Create temp file
	tempFile := filepath.Join(t.TempDir(), "test_sync_state.json")
	sm := NewStateManager(tempFile)

	// Add some test data
	testTime := time.Now().Truncate(time.Second) // Truncate for JSON precision
	testState := &CollectionState{
		LastPollTime:     testTime,
		LastSyncTime:     testTime.Add(time.Minute),
		IndexName:        "test.collection.index",
		CollectionKey:    "test.collection",
		TimestampField:   "updated_at",
		IDField:          "custom_id",
		DocumentsIndexed: 1234,
	}
	sm.UpdateCollectionState("test.collection", testState)

	// Save state
	if err := sm.Save(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatal("State file was not created")
	}

	// Create new state manager and load
	sm2 := NewStateManager(tempFile)
	if err := sm2.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify loaded data
	loadedState := sm2.GetCollectionState("test.collection")
	if loadedState == nil {
		t.Fatal("Failed to load collection state")
	}

	if !loadedState.LastPollTime.Equal(testTime) {
		t.Errorf("Expected LastPollTime %v, got %v", testTime, loadedState.LastPollTime)
	}
	if !loadedState.LastSyncTime.Equal(testTime.Add(time.Minute)) {
		t.Errorf("Expected LastSyncTime %v, got %v", testTime.Add(time.Minute), loadedState.LastSyncTime)
	}
	if loadedState.IndexName != "test.collection.index" {
		t.Errorf("Expected IndexName 'test.collection.index', got '%s'", loadedState.IndexName)
	}
	if loadedState.CollectionKey != "test.collection" {
		t.Errorf("Expected CollectionKey 'test.collection', got '%s'", loadedState.CollectionKey)
	}
	if loadedState.TimestampField != "updated_at" {
		t.Errorf("Expected TimestampField 'updated_at', got '%s'", loadedState.TimestampField)
	}
	if loadedState.IDField != "custom_id" {
		t.Errorf("Expected IDField 'custom_id', got '%s'", loadedState.IDField)
	}
	if loadedState.DocumentsIndexed != 1234 {
		t.Errorf("Expected DocumentsIndexed 1234, got %d", loadedState.DocumentsIndexed)
	}
}

func TestStateManager_LoadNonExistentFile(t *testing.T) {
	sm := NewStateManager("/tmp/non_existent_file.json")
	if err := sm.Load(); err != nil {
		t.Errorf("Expected no error when loading non-existent file, got: %v", err)
	}
}

func TestStateManager_GetCollectionState(t *testing.T) {
	sm := NewStateManager("/tmp/test.json")

	// Test non-existent collection
	state := sm.GetCollectionState("non.existent")
	if state != nil {
		t.Error("Expected nil for non-existent collection")
	}

	// Add collection and test retrieval
	testState := &CollectionState{
		CollectionKey: "test.collection",
		IDField:       "_id",
	}
	sm.UpdateCollectionState("test.collection", testState)

	state = sm.GetCollectionState("test.collection")
	if state == nil {
		t.Fatal("Expected collection state to exist")
	}
	if state.CollectionKey != "test.collection" {
		t.Errorf("Expected CollectionKey 'test.collection', got '%s'", state.CollectionKey)
	}
}

func TestStateManager_SetLastPollTime(t *testing.T) {
	sm := NewStateManager("/tmp/test.json")
	testTime := time.Now().Truncate(time.Second)

	// Set poll time for new collection
	sm.SetLastPollTime("test.collection", testTime)

	state := sm.GetCollectionState("test.collection")
	if state == nil {
		t.Fatal("Expected collection state to be created")
	}
	if !state.LastPollTime.Equal(testTime) {
		t.Errorf("Expected LastPollTime %v, got %v", testTime, state.LastPollTime)
	}

	// Update existing collection
	newTime := testTime.Add(time.Hour)
	sm.SetLastPollTime("test.collection", newTime)

	state = sm.GetCollectionState("test.collection")
	if !state.LastPollTime.Equal(newTime) {
		t.Errorf("Expected updated LastPollTime %v, got %v", newTime, state.LastPollTime)
	}
}

func TestStateManager_SetLastSyncTime(t *testing.T) {
	sm := NewStateManager("/tmp/test.json")
	testTime := time.Now().Truncate(time.Second)

	// Set sync time for new collection
	sm.SetLastSyncTime("test.collection", testTime)

	state := sm.GetCollectionState("test.collection")
	if state == nil {
		t.Fatal("Expected collection state to be created")
	}
	if !state.LastSyncTime.Equal(testTime) {
		t.Errorf("Expected LastSyncTime %v, got %v", testTime, state.LastSyncTime)
	}
}

func TestStateManager_IncrementDocumentsIndexed(t *testing.T) {
	sm := NewStateManager("/tmp/test.json")

	// Increment for new collection
	sm.IncrementDocumentsIndexed("test.collection", 100)

	state := sm.GetCollectionState("test.collection")
	if state == nil {
		t.Fatal("Expected collection state to be created")
	}
	if state.DocumentsIndexed != 100 {
		t.Errorf("Expected DocumentsIndexed 100, got %d", state.DocumentsIndexed)
	}

	// Increment existing collection
	sm.IncrementDocumentsIndexed("test.collection", 50)

	state = sm.GetCollectionState("test.collection")
	if state.DocumentsIndexed != 150 {
		t.Errorf("Expected DocumentsIndexed 150, got %d", state.DocumentsIndexed)
	}
}

func TestStateManager_GetAllCollectionStates(t *testing.T) {
	sm := NewStateManager("/tmp/test.json")

	// Add multiple collections
	sm.SetLastPollTime("collection1", time.Now())
	sm.SetLastPollTime("collection2", time.Now())
	sm.IncrementDocumentsIndexed("collection1", 100)

	states := sm.GetAllCollectionStates()
	if len(states) != 2 {
		t.Errorf("Expected 2 collection states, got %d", len(states))
	}

	if states["collection1"] == nil {
		t.Error("Expected collection1 to exist in states")
	}
	if states["collection2"] == nil {
		t.Error("Expected collection2 to exist in states")
	}
	if states["collection1"].DocumentsIndexed != 100 {
		t.Errorf("Expected collection1 DocumentsIndexed 100, got %d", states["collection1"].DocumentsIndexed)
	}
}

func TestStateManager_RemoveCollectionState(t *testing.T) {
	sm := NewStateManager("/tmp/test.json")

	// Add collection
	sm.SetLastPollTime("test.collection", time.Now())
	if sm.GetCollectionState("test.collection") == nil {
		t.Fatal("Expected collection to be added")
	}

	// Remove collection
	sm.RemoveCollectionState("test.collection")
	if sm.GetCollectionState("test.collection") != nil {
		t.Error("Expected collection to be removed")
	}
}

func TestStateManager_ConcurrentAccess(t *testing.T) {
	sm := NewStateManager("/tmp/test.json")
	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines performing concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			collectionKey := fmt.Sprintf("collection%d", id)

			for j := 0; j < numOperations; j++ {
				sm.SetLastPollTime(collectionKey, time.Now())
				sm.IncrementDocumentsIndexed(collectionKey, 1)
				sm.GetCollectionState(collectionKey)
			}
		}(i)
	}

	wg.Wait()

	// Verify all collections exist and have correct document counts
	states := sm.GetAllCollectionStates()
	if len(states) != numGoroutines {
		t.Errorf("Expected %d collections, got %d", numGoroutines, len(states))
	}

	for i := 0; i < numGoroutines; i++ {
		collectionKey := fmt.Sprintf("collection%d", i)
		state := states[collectionKey]
		if state == nil {
			t.Errorf("Expected collection %s to exist", collectionKey)
			continue
		}
		if state.DocumentsIndexed != numOperations {
			t.Errorf("Expected collection %s to have %d documents, got %d",
				collectionKey, numOperations, state.DocumentsIndexed)
		}
	}
}

func TestStateManager_AtomicSave(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test_atomic_save.json")
	sm := NewStateManager(tempFile)

	// Add test data
	sm.SetLastPollTime("test.collection", time.Now())

	// Save and verify temp file doesn't exist after successful save
	if err := sm.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	tempTempFile := tempFile + ".tmp"
	if _, err := os.Stat(tempTempFile); !os.IsNotExist(err) {
		t.Error("Temporary file should not exist after successful save")
	}

	// Verify main file exists
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("Main state file should exist after save")
	}
}
