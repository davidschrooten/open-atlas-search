package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davidschrooten/open-atlas-search/config"
	"github.com/davidschrooten/open-atlas-search/internal/indexer"
	"github.com/davidschrooten/open-atlas-search/internal/search"
)

// mockSearchEngine implements a basic mock for testing
type mockSearchEngine struct {
	indexes   []search.IndexInfo
	searchErr error
}

func (m *mockSearchEngine) ListIndexes() ([]search.IndexInfo, error) {
	return m.indexes, nil
}

func (m *mockSearchEngine) Search(req search.SearchRequest) (*search.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return &search.SearchResult{
		Hits: []search.SearchHit{
			{
				ID:    "test1",
				Score: 1.0,
				Source: map[string]interface{}{
					"title": "Test Document",
				},
			},
		},
		Total:    1,
		MaxScore: 1.0,
	}, nil
}

func (m *mockSearchEngine) IndexDocument(indexName, docID string, doc map[string]interface{}) error {
	return nil
}

func (m *mockSearchEngine) DeleteDocument(indexName, docID string) error {
	return nil
}

func (m *mockSearchEngine) CreateIndex(indexCfg config.IndexConfig) error {
	return nil
}

func (m *mockSearchEngine) RemoveIndex(indexName string) error {
	return nil
}

func (m *mockSearchEngine) CleanupIndexes(cfg *config.Config) {}

func (m *mockSearchEngine) UpdateLastSync(indexName string, syncTime time.Time) {}

func (m *mockSearchEngine) Close() error {
	return nil
}

func (m *mockSearchEngine) GetIndexMapping(indexName string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"name":   indexName,
		"type":   "mock",
		"status": "active",
	}, nil
}

func (m *mockSearchEngine) IndexDocuments(indexName string, docs []search.DocumentBatch) error {
	return nil
}

func TestServer_handleHealth(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}
}

func TestServer_handleReady_MissingIndexer(t *testing.T) {
	cfg := &config.Config{
		Indexes: []config.IndexConfig{
			{Name: "test_index"},
		},
	}

	mockEngine := &mockSearchEngine{
		indexes: []search.IndexInfo{
			{
				Name:     "test.collection.index",
				DocCount: 100,
				Status:   "active",
				LastSync: &[]time.Time{time.Now()}[0],
			},
		},
	}

	// This test verifies that missing indexerService causes a 503 response
	server := &Server{
		searchEngine:   mockEngine,
		indexerService: nil, // This should cause the ready check to fail
		config:         cfg,
	}

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestServer_handleReady(t *testing.T) {
	cfg := &config.Config{
		Indexes: []config.IndexConfig{
			{Name: "test_index"},
		},
	}

	mockEngine := &mockSearchEngine{
		indexes: []search.IndexInfo{
			{
				Name:     "test.collection.index",
				DocCount: 100,
				Status:   "active",
				LastSync: &[]time.Time{time.Now()}[0],
			},
		},
	}

	// Create a non-nil indexerService for this test (we can use an empty struct)
	server := &Server{
		searchEngine:   mockEngine,
		indexerService: &indexer.Service{}, // Non-nil service
		config:         cfg,
	}

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%v'", response["status"])
	}

	checks, ok := response["checks"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected checks to be present")
	}

	if checks["searchEngine"] != "ok" {
		t.Errorf("Expected searchEngine check to be 'ok', got '%v'", checks["searchEngine"])
	}
}

func TestServer_handleReady_NotReady(t *testing.T) {
	server := &Server{
		searchEngine: nil, // Simulate uninitialized engine
		config:       &config.Config{},
	}

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestServer_handleListIndexes(t *testing.T) {
	mockEngine := &mockSearchEngine{
		indexes: []search.IndexInfo{
			{
				Name:     "test.collection.index",
				DocCount: 100,
				Status:   "active",
				LastSync: &[]time.Time{time.Now()}[0],
			},
			{
				Name:     "test.collection2.index",
				DocCount: 200,
				Status:   "active",
			},
		},
	}

	server := &Server{
		searchEngine: mockEngine,
	}

	req := httptest.NewRequest("GET", "/indexes", nil)
	w := httptest.NewRecorder()

	server.handleListIndexes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	indexes, ok := response["indexes"].([]interface{})
	if !ok {
		t.Fatal("Expected indexes to be an array")
	}

	if len(indexes) != 2 {
		t.Errorf("Expected 2 indexes, got %d", len(indexes))
	}

	total, ok := response["total"].(float64)
	if !ok {
		t.Fatal("Expected total to be a number")
	}

	if int(total) != 2 {
		t.Errorf("Expected total 2, got %d", int(total))
	}
}

func TestServer_handleSearch(t *testing.T) {
	mockEngine := &mockSearchEngine{}

	server := &Server{
		searchEngine: mockEngine,
		config:       &config.Config{},
	}
	mockEngine.indexes = []search.IndexInfo{
		{
			Name:     "test.index",
			DocCount: 1,
			Status:   "active",
		},
	}
	router := server.Router()

	searchReq := search.SearchRequest{
		Query: map[string]interface{}{
			"text": map[string]interface{}{
				"query": "test",
				"path":  "content",
			},
		},
		Size: 10,
		From: 0,
	}

	reqBody, _ := json.Marshal(searchReq)
	req := httptest.NewRequest("POST", "/indexes/test.index/search", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response search.SearchResult
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Total != 1 {
		t.Errorf("Expected total 1, got %d", response.Total)
	}

	if len(response.Hits) != 1 {
		t.Errorf("Expected 1 hit, got %d", len(response.Hits))
	}

	if response.Hits[0].ID != "test1" {
		t.Errorf("Expected hit ID 'test1', got '%s'", response.Hits[0].ID)
	}
}

func TestServer_handleSearch_EmptyQuery(t *testing.T) {
	mockEngine := &mockSearchEngine{}

	server := &Server{
		searchEngine: mockEngine,
		config:       &config.Config{},
	}
	mockEngine.indexes = []search.IndexInfo{
		{
			Name:     "test.index",
			DocCount: 1,
			Status:   "active",
		},
	}
	router := server.Router()

	// Test with empty query body
	emptyReq := map[string]interface{}{}
	reqBody, _ := json.Marshal(emptyReq)
	req := httptest.NewRequest("POST", "/indexes/test.index/search", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response search.SearchResult
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return default 100 results with match_all query
	if response.Total != 1 {
		t.Errorf("Expected total 1, got %d", response.Total)
	}
}

func TestServer_handleStatus_WithIndex(t *testing.T) {
	mockEngine := &mockSearchEngine{
		indexes: []search.IndexInfo{
			{
				Name:     "test.collection.index",
				DocCount: 100,
				Status:   "active",
				LastSync: &[]time.Time{time.Now()}[0],
			},
		},
	}

	server := &Server{
		searchEngine: mockEngine,
	}
	router := server.Router()

	req := httptest.NewRequest("GET", "/indexes/test.collection.index/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["service"] != "open-atlas-search" {
		t.Errorf("Expected service 'open-atlas-search', got '%v'", response["service"])
	}

	if response["status"] != "running" {
		t.Errorf("Expected status 'running', got '%v'", response["status"])
	}

	// Check that it returns specific index info
	index, ok := response["index"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected index to be present")
	}

	if index["name"] != "test.collection.index" {
		t.Errorf("Expected index name 'test.collection.index', got '%v'", index["name"])
	}
}

func TestServer_Authentication_Disabled(t *testing.T) {
	mockEngine := &mockSearchEngine{
		indexes: []search.IndexInfo{
			{
				Name:     "test.index",
				DocCount: 1,
				Status:   "active",
			},
		},
	}

	// Server without auth config (username and password empty)
	server := &Server{
		searchEngine: mockEngine,
		config: &config.Config{
			Server: config.ServerConfig{
				Host:     "0.0.0.0",
				Port:     8080,
				Username: "",
				Password: "",
			},
		},
	}
	router := server.Router()

	// Request without auth header should succeed when auth is disabled
	req := httptest.NewRequest("GET", "/indexes", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d when auth is disabled, got %d", http.StatusOK, w.Code)
	}
}

func TestServer_Authentication_Enabled_NoAuth(t *testing.T) {
	mockEngine := &mockSearchEngine{
		indexes: []search.IndexInfo{
			{
				Name:     "test.index",
				DocCount: 1,
				Status:   "active",
			},
		},
	}

	// Server with auth config
	server := &Server{
		searchEngine: mockEngine,
		config: &config.Config{
			Server: config.ServerConfig{
				Host:     "0.0.0.0",
				Port:     8080,
				Username: "admin",
				Password: "secret",
			},
		},
	}
	router := server.Router()

	// Request without auth header should fail when auth is enabled
	req := httptest.NewRequest("GET", "/indexes", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d when auth is missing, got %d", http.StatusUnauthorized, w.Code)
	}

	// Check WWW-Authenticate header
	if auth := w.Header().Get("WWW-Authenticate"); auth == "" {
		t.Error("Expected WWW-Authenticate header to be set")
	}
}

func TestServer_Authentication_Enabled_ValidAuth(t *testing.T) {
	mockEngine := &mockSearchEngine{
		indexes: []search.IndexInfo{
			{
				Name:     "test.index",
				DocCount: 1,
				Status:   "active",
			},
		},
	}

	// Server with auth config
	server := &Server{
		searchEngine: mockEngine,
		config: &config.Config{
			Server: config.ServerConfig{
				Host:     "0.0.0.0",
				Port:     8080,
				Username: "admin",
				Password: "secret",
			},
		},
	}
	router := server.Router()

	// Request with valid auth header should succeed
	req := httptest.NewRequest("GET", "/indexes", nil)
	req.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d with valid auth, got %d", http.StatusOK, w.Code)
	}
}

func TestServer_Authentication_Enabled_InvalidAuth(t *testing.T) {
	mockEngine := &mockSearchEngine{
		indexes: []search.IndexInfo{
			{
				Name:     "test.index",
				DocCount: 1,
				Status:   "active",
			},
		},
	}

	// Server with auth config
	server := &Server{
		searchEngine: mockEngine,
		config: &config.Config{
			Server: config.ServerConfig{
				Host:     "0.0.0.0",
				Port:     8080,
				Username: "admin",
				Password: "secret",
			},
		},
	}
	router := server.Router()

	// Request with invalid auth header should fail
	req := httptest.NewRequest("GET", "/indexes", nil)
	req.SetBasicAuth("admin", "wrongpassword")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d with invalid auth, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestServer_HealthEndpoint_AlwaysAccessible(t *testing.T) {
	mockEngine := &mockSearchEngine{}

	// Server with auth config
	server := &Server{
		searchEngine: mockEngine,
		config: &config.Config{
			Server: config.ServerConfig{
				Host:     "0.0.0.0",
				Port:     8080,
				Username: "admin",
				Password: "secret",
			},
		},
	}
	router := server.Router()

	// Health endpoint should be accessible without auth
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected health endpoint to be accessible without auth, got status %d", w.Code)
	}
}
