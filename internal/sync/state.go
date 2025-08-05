package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// CollectionState represents the sync state for a single collection
type CollectionState struct {
	LastPollTime     time.Time `json:"lastPollTime"`
	LastSyncTime     time.Time `json:"lastSyncTime"`
	IndexName        string    `json:"indexName"`
	CollectionKey    string    `json:"collectionKey"`
	TimestampField   string    `json:"timestampField"`
	IDField          string    `json:"idField"`
	DocumentsIndexed int64     `json:"documentsIndexed"`
}

// SyncState manages persistent state for all collections
type SyncState struct {
	Collections map[string]*CollectionState `json:"collections"`
	LastSaved   time.Time                   `json:"lastSaved"`
}

// StateManager handles loading and saving sync state
type StateManager struct {
	filePath string
	state    *SyncState
	mutex    sync.RWMutex
}

// NewStateManager creates a new sync state manager
func NewStateManager(filePath string) *StateManager {
	return &StateManager{
		filePath: filePath,
		state: &SyncState{
			Collections: make(map[string]*CollectionState),
		},
	}
}

// Load loads the sync state from disk
func (sm *StateManager) Load() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if file exists
	if _, err := os.Stat(sm.filePath); os.IsNotExist(err) {
		log.Printf("Sync state file not found, starting fresh: %s", sm.filePath)
		return nil
	}

	// Read file
	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read sync state file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, sm.state); err != nil {
		return fmt.Errorf("failed to parse sync state file: %w", err)
	}

	log.Printf("Loaded sync state for %d collections from %s", len(sm.state.Collections), sm.filePath)
	return nil
}

// Save saves the current sync state to disk
func (sm *StateManager) Save() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.state.LastSaved = time.Now()

	// Marshal to JSON
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync state: %w", err)
	}

	// Write to temporary file first
	tempFile := sm.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp sync state file: %w", err)
	}

	// Atomic move
	if err := os.Rename(tempFile, sm.filePath); err != nil {
		return fmt.Errorf("failed to move sync state file: %w", err)
	}

	return nil
}

// GetCollectionState gets the sync state for a collection
func (sm *StateManager) GetCollectionState(collectionKey string) *CollectionState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if state, exists := sm.state.Collections[collectionKey]; exists {
		return state
	}
	return nil
}

// UpdateCollectionState updates the sync state for a collection
func (sm *StateManager) UpdateCollectionState(collectionKey string, state *CollectionState) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.state.Collections[collectionKey] = state
}

// SetLastPollTime updates the last poll time for a collection
func (sm *StateManager) SetLastPollTime(collectionKey string, pollTime time.Time) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if state, exists := sm.state.Collections[collectionKey]; exists {
		state.LastPollTime = pollTime
	} else {
		sm.state.Collections[collectionKey] = &CollectionState{
			CollectionKey: collectionKey,
			LastPollTime:  pollTime,
		}
	}
}

// SetLastSyncTime updates the last sync time for a collection
func (sm *StateManager) SetLastSyncTime(collectionKey string, syncTime time.Time) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if state, exists := sm.state.Collections[collectionKey]; exists {
		state.LastSyncTime = syncTime
	} else {
		sm.state.Collections[collectionKey] = &CollectionState{
			CollectionKey: collectionKey,
			LastSyncTime:  syncTime,
		}
	}
}

// IncrementDocumentsIndexed increments the documents indexed counter
func (sm *StateManager) IncrementDocumentsIndexed(collectionKey string, count int64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if state, exists := sm.state.Collections[collectionKey]; exists {
		state.DocumentsIndexed += count
	} else {
		sm.state.Collections[collectionKey] = &CollectionState{
			CollectionKey:    collectionKey,
			DocumentsIndexed: count,
		}
	}
}

// GetAllCollectionStates returns all collection states
func (sm *StateManager) GetAllCollectionStates() map[string]*CollectionState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*CollectionState)
	for key, state := range sm.state.Collections {
		// Deep copy the state
		stateCopy := *state
		result[key] = &stateCopy
	}
	return result
}

// RemoveCollectionState removes a collection state (for cleanup)
func (sm *StateManager) RemoveCollectionState(collectionKey string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	delete(sm.state.Collections, collectionKey)
}

// StartPeriodicSave starts a goroutine that periodically saves state
func (sm *StateManager) StartPeriodicSave(interval time.Duration, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sm.Save(); err != nil {
				log.Printf("Failed to save sync state: %v", err)
			}
		case <-stopCh:
			// Final save before stopping
			if err := sm.Save(); err != nil {
				log.Printf("Failed to save sync state on shutdown: %v", err)
			}
			return
		}
	}
}
