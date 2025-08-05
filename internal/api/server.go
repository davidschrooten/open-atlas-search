package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davidschrooten/open-atlas-search/config"
	"github.com/davidschrooten/open-atlas-search/internal/indexer"
	"github.com/davidschrooten/open-atlas-search/internal/search"
)

// ErrorResponse represents a structured API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Server represents the API server
type Server struct {
	searchEngine   search.SearchEngine
	indexerService *indexer.Service
	config         *config.Config
}

// NewServer creates a new API server
func NewServer(searchEngine search.SearchEngine, indexerService *indexer.Service, cfg *config.Config) *Server {
	return &Server{
		searchEngine:   searchEngine,
		indexerService: indexerService,
		config:         cfg,
	}
}

// Router setups the API routes
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(s.corsMiddleware)
	r.Use(s.methodNotAllowedMiddleware)

	r.Post("/indexes/{index}/search", s.handleSearch)
	r.Get("/indexes/{index}/status", s.handleStatus)
	r.Get("/indexes/{index}/mapping", s.handleMapping)
	r.Get("/indexes", s.handleListIndexes)
	r.Get("/health", s.handleHealth)
	r.Get("/ready", s.handleReady)

	return r
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	// Validate index parameter
	index := strings.TrimSpace(chi.URLParam(r, "index"))
	if index == "" {
		s.errorResponse(w, "bad_request", "Index parameter is required", http.StatusBadRequest)
		return
	}

	// Validate index exists
	if !s.indexExists(index) {
		s.errorResponse(w, "index_not_found", fmt.Sprintf("Index '%s' not found", index), http.StatusNotFound)
		return
	}

	// Validate request body
	if r.Body == nil {
		s.errorResponse(w, "bad_request", "Request body is required", http.StatusBadRequest)
		return
	}

	var searchReq struct {
		Query  map[string]interface{}         `json:"query"`
		Facets map[string]search.FacetRequest `json:"facets"`
		Size   int                            `json:"size"`
		From   int                            `json:"from"`
	}

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		log.Printf("Failed to decode search request: %v", err)
		s.errorResponse(w, "invalid_json", "Invalid JSON in request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate search parameters
	if searchReq.Size < 0 {
		s.errorResponse(w, "invalid_parameter", "Size parameter cannot be negative", http.StatusBadRequest)
		return
	}
	if searchReq.From < 0 {
		s.errorResponse(w, "invalid_parameter", "From parameter cannot be negative", http.StatusBadRequest)
		return
	}
	if searchReq.Size > 1000 {
		s.errorResponse(w, "invalid_parameter", "Size parameter cannot exceed 1000", http.StatusBadRequest)
		return
	}

	// Set defaults
	if searchReq.Size == 0 {
		searchReq.Size = 10
	}

	// Prepare the search request for the search engine
	sReq := search.SearchRequest{
		Index:  index,
		Query:  searchReq.Query,
		Facets: searchReq.Facets,
		Size:   searchReq.Size,
		From:   searchReq.From,
	}

	searchResult, err := s.searchEngine.Search(sReq)
	if err != nil {
		log.Printf("Search error for index '%s': %v", index, err)
		// Check if it's an index not found error
		if strings.Contains(err.Error(), "not found") {
			s.errorResponse(w, "index_not_found", fmt.Sprintf("Index '%s' not found", index), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "query") {
			s.errorResponse(w, "invalid_query", "Invalid search query: "+err.Error(), http.StatusBadRequest)
		} else {
			s.errorResponse(w, "search_failed", "Search operation failed", http.StatusInternalServerError)
		}
		return
	}

	s.successResponse(w, searchResult)
}

func (s *Server) handleListIndexes(w http.ResponseWriter, r *http.Request) {
	indexes, err := s.searchEngine.ListIndexes()
	if err != nil {
		log.Printf("Failed to list indexes: %v", err)
		s.errorResponse(w, "list_indexes_failed", "Failed to retrieve indexes", http.StatusInternalServerError)
		return
	}

	// Get sync states from indexer service and update indexes status
	if s.indexerService != nil {
		syncStates := s.indexerService.GetSyncStates()
		for i := range indexes {
			// Map index name to collection key for sync state lookup
			// Index name is now just the simple name, we need to find the matching collection
			indexName := indexes[i].Name
			collectionKey := s.findCollectionKeyForIndex(indexName)
			if collectionKey != "" {
				if syncState, exists := syncStates[collectionKey]; exists {
					if string(syncState.SyncStatus) == "in_progress" {
						indexes[i].Status = "syncing"
						indexes[i].SyncProgress = syncState.Progress
					} else {
						indexes[i].Status = "active"
					}
				}
			}
		}
	}

	s.successResponse(w, map[string]interface{}{
		"indexes": indexes,
		"total":   len(indexes),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Validate index parameter
	index := strings.TrimSpace(chi.URLParam(r, "index"))
	if index == "" {
		s.errorResponse(w, "bad_request", "Index parameter is required", http.StatusBadRequest)
		return
	}

	indexes, err := s.searchEngine.ListIndexes()
	if err != nil {
		log.Printf("Failed to list indexes for status check: %v", err)
		s.errorResponse(w, "internal_error", "Failed to retrieve index status", http.StatusInternalServerError)
		return
	}

	// Find the specific index
	var targetIndex *search.IndexInfo
	for i, idx := range indexes {
		if idx.Name == index {
			targetIndex = &indexes[i]
			break
		}
	}

	if targetIndex == nil {
		s.errorResponse(w, "index_not_found", fmt.Sprintf("Index '%s' not found", index), http.StatusNotFound)
		return
	}

	// Apply sync state to the specific index
	if s.indexerService != nil {
		syncStates := s.indexerService.GetSyncStates()
		collectionKey := s.findCollectionKeyForIndex(targetIndex.Name)
		if collectionKey != "" {
			if syncState, exists := syncStates[collectionKey]; exists {
				if string(syncState.SyncStatus) == "in_progress" {
					targetIndex.Status = "syncing"
					targetIndex.SyncProgress = syncState.Progress
				} else {
					targetIndex.Status = "active"
				}
			}
		}
	}

	// Create status response for the specific index
	status := map[string]interface{}{
		"service": "open-atlas-search",
		"status":  "running",
		"index":   *targetIndex,
	}

	s.successResponse(w, status)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Always return healthy for basic health check
	s.successResponse(w, map[string]interface{}{
		"status":  "healthy",
		"service": "open-atlas-search",
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{}

	// Check if search engine is initialized
	if s.searchEngine == nil {
		s.errorResponse(w, "service_unavailable", "Search engine not initialized", http.StatusServiceUnavailable)
		return
	}
	checks["searchEngine"] = "ok"

	// Check if indexer service is initialized
	if s.indexerService == nil {
		s.errorResponse(w, "service_unavailable", "Indexer service not initialized", http.StatusServiceUnavailable)
		return
	}
	checks["indexerService"] = "ok"

	// Verify that the search engine is working
	if _, err := s.searchEngine.ListIndexes(); err != nil {
		log.Printf("Readiness check failed - cannot list indexes: %v", err)
		s.errorResponse(w, "service_unavailable", "Search engine not ready", http.StatusServiceUnavailable)
		return
	}

	// If we have configured indexes, verify at least one exists
	if len(s.config.Indexes) > 0 {
		indexes, err := s.searchEngine.ListIndexes()
		if err != nil {
			log.Printf("Readiness check failed - error listing indexes: %v", err)
			s.errorResponse(w, "service_unavailable", "Cannot verify indexes", http.StatusServiceUnavailable)
			return
		}
		if len(indexes) == 0 {
			log.Printf("Readiness check failed - no indexes available")
			s.errorResponse(w, "service_unavailable", "No indexes available", http.StatusServiceUnavailable)
			return
		}
	}
	checks["indexes"] = "ok"

	s.successResponse(w, map[string]interface{}{
		"status":  "ready",
		"service": "open-atlas-search",
		"checks":  checks,
	})
}

func (s *Server) handleMapping(w http.ResponseWriter, r *http.Request) {
	// Validate index parameter
	index := strings.TrimSpace(chi.URLParam(r, "index"))
	if index == "" {
		s.errorResponse(w, "bad_request", "Index parameter is required", http.StatusBadRequest)
		return
	}

	// Validate index exists
	if !s.indexExists(index) {
		s.errorResponse(w, "index_not_found", fmt.Sprintf("Index '%s' not found", index), http.StatusNotFound)
		return
	}

	mapping, err := s.searchEngine.GetIndexMapping(index)
	if err != nil {
		log.Printf("Failed to get mapping for index '%s': %v", index, err)
		if strings.Contains(err.Error(), "not found") {
			s.errorResponse(w, "index_not_found", fmt.Sprintf("Index '%s' not found", index), http.StatusNotFound)
		} else {
			s.errorResponse(w, "mapping_failed", "Failed to retrieve index mapping", http.StatusInternalServerError)
		}
		return
	}

	s.successResponse(w, mapping)
}

// findCollectionKeyForIndex finds the collection key for a given index name
func (s *Server) findCollectionKeyForIndex(indexName string) string {
	for _, indexCfg := range s.config.Indexes {
		if indexCfg.Name == indexName {
			return fmt.Sprintf("%s.%s", indexCfg.Database, indexCfg.Collection)
		}
	}
	return ""
}

// successResponse writes a successful response in JSON
func (s *Server) successResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// errorResponse writes an error response in JSON
func (s *Server) errorResponse(w http.ResponseWriter, errorType, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errResp := ErrorResponse{
		Error:   errorType,
		Message: message,
		Code:    statusCode,
	}
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// indexExists checks if an index exists
func (s *Server) indexExists(indexName string) bool {
	indexes, err := s.searchEngine.ListIndexes()
	if err != nil {
		log.Printf("Error checking if index exists: %v", err)
		return false
	}
	for _, index := range indexes {
		if index.Name == indexName {
			return true
		}
	}
	return false
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// methodNotAllowedMiddleware handles unsupported HTTP methods
func (s *Server) methodNotAllowedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let chi handle the routing first
		next.ServeHTTP(w, r)
	})
}
